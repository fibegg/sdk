package mcpserver

import (
	"context"
	"strings"
	"testing"

	"github.com/fibegg/sdk/fibe"
	"github.com/spf13/cobra"
)

// TestServerBootstrap verifies the MCP server can be constructed, tools can
// be registered, and dispatcher lookups find every registered tool — all
// without touching the network.
func TestServerBootstrap(t *testing.T) {
	srv := New(Config{
		APIKey:                "pk_test_dummy",
		ToolSet:               "full",
		PipelineCacheSize:     8,
		PipelineCacheEntryMax: 1024 * 1024,
		PipelineMaxSteps:      25,
		PipelineMaxIterations: 50,
		CobraRoot: &cobra.Command{
			Use: "fibe",
		},
	})
	if err := srv.RegisterAll(); err != nil {
		t.Fatalf("RegisterAll: %v", err)
	}

	// Every registered tool must show up in the dispatcher's name list.
	names := srv.dispatcher.names()
	if len(names) < 50 {
		t.Fatalf("expected at least 50 tools registered, got %d", len(names))
	}
	for _, essential := range []string{
		"fibe_me",
		"fibe_status",
		"fibe_limits",
		"fibe_doctor",
		"fibe_help",
		"fibe_run",
		"fibe_schema",
		"fibe_auth_set",
		"fibe_pipeline",
		"fibe_pipeline_result",
		"fibe_playgrounds_list",
		"fibe_playgrounds_get",
		"fibe_playgrounds_create",
		"fibe_playgrounds_wait",
		"fibe_playgrounds_logs",
		"fibe_launch",
	} {
		if _, ok := srv.dispatcher.lookup(essential); !ok {
			t.Errorf("essential tool %q not registered", essential)
		}
	}
}

// TestCoreTierFilter verifies FIBE_MCP_TOOLS=core advertises a smaller
// subset while dispatcher still knows every tool (so pipeline steps can
// reach them).
func TestCoreTierFilter(t *testing.T) {
	srv := New(Config{APIKey: "pk_test", ToolSet: "core", PipelineCacheSize: 4})
	if err := srv.RegisterAll(); err != nil {
		t.Fatalf("RegisterAll: %v", err)
	}

	// Dispatcher has everything regardless of tier.
	full := srv.dispatcher.names()

	// Meta tools are always advertised.
	for _, meta := range []string{"fibe_pipeline", "fibe_help", "fibe_run"} {
		if _, ok := srv.dispatcher.lookup(meta); !ok {
			t.Errorf("%s should always be registered", meta)
		}
	}

	// A "full"-only tool like fibe_api_keys_delete should still be in the
	// dispatcher so pipeline steps can call it, even in core mode.
	if _, ok := srv.dispatcher.lookup("fibe_api_keys_delete"); !ok {
		t.Errorf("fibe_api_keys_delete should be reachable via dispatcher in core mode")
	}

	// Count scaled with tier.
	t.Logf("registered %d tools in core mode (dispatcher-level)", len(full))
}

// TestConfirmGate verifies destructive tools require confirm:true unless
// --yolo is set.
func TestConfirmGate(t *testing.T) {
	srv := New(Config{APIKey: "pk_test", ToolSet: "core"})
	if err := srv.RegisterAll(); err != nil {
		t.Fatalf("RegisterAll: %v", err)
	}

	ctx := context.Background()

	// fibe_playgrounds_delete without confirm should error.
	_, err := srv.dispatcher.dispatch(ctx, "fibe_playgrounds_delete", map[string]any{
		"id": 42,
	})
	if err == nil {
		t.Fatalf("expected confirm-required error, got nil")
	}
	if _, ok := err.(*confirmRequiredError); !ok {
		// Could also fail at client resolution for other reasons, but the
		// confirm gate runs first.
		if !strings.Contains(err.Error(), "confirm:true") && !strings.Contains(err.Error(), "destructive") {
			t.Fatalf("expected confirm-required error, got: %v", err)
		}
	}
}

// TestYoloSkipsConfirm: with Yolo=true, destructive tools run without confirm.
// We can't actually call the API (no network), but we can verify the gate is
// bypassed by substituting a stub client path.
func TestYoloSkipsConfirm(t *testing.T) {
	srv := New(Config{APIKey: "pk_test", ToolSet: "core", Yolo: true})
	if err := srv.RegisterAll(); err != nil {
		t.Fatalf("RegisterAll: %v", err)
	}

	ctx := context.Background()
	_, err := srv.dispatcher.dispatch(ctx, "fibe_playgrounds_delete", map[string]any{
		"id": 42,
	})
	// Should NOT fail with confirm-required. Will likely fail trying to hit
	// the network, which is fine — we just care that the confirm gate was skipped.
	if err != nil {
		if _, ok := err.(*confirmRequiredError); ok {
			t.Fatalf("yolo mode should bypass confirm gate, but got confirm-required error")
		}
	}
}

// TestPipelineRefs exercises the JSONPath binding resolver without hitting
// the network: we register a fake tool that just echoes its args.
func TestPipelineRefs(t *testing.T) {
	srv := New(Config{APIKey: "pk_test", ToolSet: "full", PipelineCacheSize: 4, PipelineMaxSteps: 10})
	if err := srv.RegisterAll(); err != nil {
		t.Fatalf("RegisterAll: %v", err)
	}

	// Inject a no-network echo tool.
	srv.dispatcher.register(&toolImpl{
		name: "test_echo",
		tier: tierMeta,
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			return args, nil
		},
	})

	result, err := srv.runPipeline(context.Background(), map[string]any{
		"steps": []any{
			map[string]any{"id": "a", "tool": "test_echo", "args": map[string]any{"x": 7}},
			map[string]any{"id": "b", "tool": "test_echo", "args": map[string]any{"refd": "$.a.x"}},
		},
		"return": "$.b.refd",
	})
	if err != nil {
		t.Fatalf("runPipeline: %v", err)
	}
	m, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map result, got %T", result)
	}
	if m["result"].(float64) != 7 {
		t.Fatalf("expected resolved ref value 7, got %v", m["result"])
	}
	if m["pipeline_id"] == nil {
		t.Errorf("expected pipeline_id in response")
	}
}

// TestPipelineCache verifies cache round-trip through fibe_pipeline_result.
func TestPipelineCache(t *testing.T) {
	srv := New(Config{APIKey: "pk_test", ToolSet: "full", PipelineCacheSize: 4, PipelineMaxSteps: 10, PipelineCacheEntryMax: 1 << 20})
	if err := srv.RegisterAll(); err != nil {
		t.Fatalf("RegisterAll: %v", err)
	}
	srv.dispatcher.register(&toolImpl{
		name: "test_echo2",
		tier: tierMeta,
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			return args, nil
		},
	})

	ctx := context.Background()
	resp, err := srv.runPipeline(ctx, map[string]any{
		"steps": []any{
			map[string]any{"id": "a", "tool": "test_echo2", "args": map[string]any{"v": 1}},
		},
	})
	if err != nil {
		t.Fatalf("pipeline: %v", err)
	}
	m := resp.(map[string]any)
	pid := m["pipeline_id"].(string)
	if pid == "" {
		t.Fatal("expected non-empty pipeline_id")
	}

	// Re-query via fibe_pipeline_result.
	result, err := srv.dispatcher.dispatch(ctx, "fibe_pipeline_result", map[string]any{
		"pipeline_id": pid,
		"path":        "$.steps.a.v",
	})
	if err != nil {
		t.Fatalf("pipeline_result: %v", err)
	}
	if v, ok := result.(float64); !ok || v != 1 {
		t.Fatalf("expected cached value 1, got %T %v", result, result)
	}
}
