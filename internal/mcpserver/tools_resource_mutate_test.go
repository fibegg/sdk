package mcpserver

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/fibegg/sdk/fibe"
)

func TestResourceMutateDispatchesCreateAndUpdate(t *testing.T) {
	apiKey, domain := requireRealServer(t)
	srv := New(Config{APIKey: apiKey, Domain: domain, ToolSet: "core"})
	if err := srv.RegisterAll(); err != nil {
		t.Fatalf("RegisterAll: %v", err)
	}

	name := fmt.Sprintf("test-builder-%d", time.Now().UnixNano())
	res, err := srv.dispatcher.dispatch(context.Background(), "fibe_resource_mutate", map[string]any{
		"resource":  "agents",
		"operation": "create",
		"payload": map[string]any{
			"name":     name,
			"provider": "openai-codex",
		},
	})
	if err != nil {
		t.Fatalf("create dispatch: %v", err)
	}
	m := res.(*fibe.Agent)
	idFloat := m.ID

	if _, err := srv.dispatcher.dispatch(context.Background(), "fibe_resource_mutate", map[string]any{
		"resource":  "agent",
		"operation": "update",
		"payload": map[string]any{
			"agent_id": int(idFloat),
			"name":     name + "-renamed",
		},
	}); err != nil {
		t.Fatalf("update dispatch: %v", err)
	}
}

func TestResourceMutateRejectsInvalidPayloadBeforeAPI(t *testing.T) {
	srv := New(mockServerConfig())
	if err := srv.RegisterAll(); err != nil {
		t.Fatalf("RegisterAll: %v", err)
	}

	_, err := srv.dispatcher.dispatch(context.Background(), "fibe_resource_mutate", map[string]any{
		"resource":  "api_key",
		"operation": "update",
		"payload": map[string]any{
			"api_key_id": 1,
			"label":      "bad",
		},
	})
	if err == nil || !strings.Contains(err.Error(), "does not support operation") {
		t.Fatalf("expected unsupported operation error, got %v", err)
	}

	_, err = srv.dispatcher.dispatch(context.Background(), "fibe_resource_mutate", map[string]any{
		"resource":  "agent",
		"operation": "create",
		"payload": map[string]any{
			"name": "missing provider",
		},
	})
	if err == nil || !strings.Contains(err.Error(), "provider is required") {
		t.Fatalf("expected local required-field error, got %v", err)
	}
}

func TestResourceMutateDryRunValidatesWithoutAPI(t *testing.T) {
	srv := New(mockServerConfig())
	if err := srv.RegisterAll(); err != nil {
		t.Fatalf("RegisterAll: %v", err)
	}

	out, err := srv.dispatcher.dispatch(context.Background(), "fibe_resource_mutate", map[string]any{
		"resource":  "props",
		"operation": "attach",
		"dry_run":   true,
		"payload": map[string]any{
			"repo_full_name": "octocat/Hello-World",
		},
	})
	if err != nil {
		t.Fatalf("dry run dispatch: %v", err)
	}
	result := out.(map[string]any)
	if result["resource"] != "prop" || result["operation"] != "attach" || result["dry_run"] != true || result["valid"] != true {
		t.Fatalf("unexpected dry run result: %#v", result)
	}
}

func TestResourceMutateDispatchesScopedActions(t *testing.T) {
	apiKey, domain := requireRealServer(t)
	srv := New(Config{APIKey: apiKey, Domain: domain, ToolSet: "core"})
	if err := srv.RegisterAll(); err != nil {
		t.Fatalf("RegisterAll: %v", err)
	}

	cases := []struct {
		resource  string
		operation string
		payload   map[string]any
	}{
		{resource: "marquee", operation: "autoconnect_token", payload: map[string]any{"ssl_mode": "http"}},
		{resource: "marquee", operation: "generate_ssh_key", payload: map[string]any{"marquee_id": 3}},
		{resource: "marquee", operation: "test_connection", payload: map[string]any{"marquee_id": 3}},
		{resource: "prop", operation: "attach", payload: map[string]any{"repo_full_name": "octocat/Hello-World"}},
		{resource: "prop", operation: "mirror", payload: map[string]any{"source_url": "https://github.com/octocat/Hello-World", "name": "mirror"}},
		{resource: "prop", operation: "sync", payload: map[string]any{"prop_id": 3}},
		{resource: "template", operation: "fork", payload: map[string]any{"template_id": 4}},
		{resource: "template", operation: "source_refresh", payload: map[string]any{"template_id": 4}},
		{resource: "template", operation: "source_set", payload: map[string]any{"template_id": 4, "source_prop_id": 3, "source_path": "template.yml", "ci_marquee_id": 3}},
		{resource: "template", operation: "upgrade_playspecs", payload: map[string]any{"template_id": 4, "version_id": 5}},
		{resource: "template_version", operation: "toggle_public", payload: map[string]any{"template_id": 4, "version_id": 5}},
		{resource: "trick", operation: "trigger", payload: map[string]any{"playspec_id": 7, "marquee_id": 3, "name": "trick"}},
		{resource: "trick", operation: "rerun", payload: map[string]any{"trick_id": 8}},
		{resource: "webhook", operation: "test", payload: map[string]any{"webhook_id": 9}},
	}

	for _, tc := range cases {
		t.Run(tc.resource+"."+tc.operation, func(t *testing.T) {
			_, err := srv.dispatcher.dispatch(context.Background(), "fibe_resource_mutate", map[string]any{
				"resource":  tc.resource,
				"operation": tc.operation,
				"payload":   tc.payload,
			})
			if err != nil && !strings.Contains(err.Error(), "404") && !strings.Contains(err.Error(), "403") && !strings.Contains(err.Error(), "422") && !strings.Contains(err.Error(), "status") {
				t.Fatalf("dispatch error not standard: %v", err)
			}
		})
	}
}

func TestResourceMutateScopedActionValidationBeforeAPI(t *testing.T) {
	srv := New(mockServerConfig())
	if err := srv.RegisterAll(); err != nil {
		t.Fatalf("RegisterAll: %v", err)
	}

	for _, tc := range []struct {
		name      string
		resource  string
		operation string
		payload   map[string]any
		want      string
	}{
		{name: "bad id", resource: "template", operation: "source_refresh", payload: map[string]any{"template_id": 0}, want: "greater than or equal to 1"},
		{name: "unknown field", resource: "prop", operation: "mirror", payload: map[string]any{"source_url": "https://github.com/o/r", "extra": true}, want: "unsupported field"},
		{name: "old prop alias rejected", resource: "prop", operation: "attach", payload: map[string]any{"repository_url": "https://github.com/o/r"}, want: "repo_full_name is required"},
		{name: "old template id alias rejected", resource: "template", operation: "source_set", payload: map[string]any{"id": 1, "source_prop_id": 2, "source_path": "template.yml"}, want: "template_id is required"},
		{name: "unsupported pair", resource: "webhook", operation: "source_set", payload: map[string]any{"webhook_id": 1}, want: "does not support operation"},
		{name: "dedicated mutter tool", resource: "mutter", operation: "create", payload: map[string]any{"type": "proof", "body": "done"}, want: "does not support mutation operations"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := srv.dispatcher.dispatch(context.Background(), "fibe_resource_mutate", map[string]any{
				"resource":  tc.resource,
				"operation": tc.operation,
				"payload":   tc.payload,
			})
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("expected %q error, got %v", tc.want, err)
			}
		})
	}
}

func TestResourceMutateListedConcreteToolsRemoved(t *testing.T) {
	srv := New(Config{APIKey: "pk_test", ToolSet: "full"})
	if err := srv.RegisterAll(); err != nil {
		t.Fatalf("RegisterAll: %v", err)
	}

	for _, name := range []string{
		"fibe_marquees_autoconnect_token",
		"fibe_marquees_generate_ssh_key",
		"fibe_marquees_test_connection",
		"fibe_props_attach",
		"fibe_props_mirror",
		"fibe_props_sync",
		"fibe_templates_fork",
		"fibe_templates_source_refresh",
		"fibe_templates_source_set",
		"fibe_templates_upgrade_playspecs",
		"fibe_templates_versions_toggle_public",
		"fibe_tricks_rerun",
		"fibe_tricks_trigger",
		"fibe_webhooks_test",
		"fibe_job_env_set",
		"fibe_agents_raw_providers_get",
		"fibe_agents_raw_providers_update",
		"fibe_templates_lineage",
		"fibe_marquees_generate_ssh_key_status",
		"fibe_marquees_test_connection_status",
		"fibe_feedbacks_create",
		"fibe_feedbacks_delete",
		"fibe_feedbacks_update",
		"fibe_mutters_create",
	} {
		if _, ok := srv.dispatcher.lookup(name); ok {
			t.Fatalf("%s should not be registered", name)
		}
	}

	for _, name := range []string{
		"fibe_templates_launch",
		"fibe_feedbacks_get",
		"fibe_feedbacks_list",
		"fibe_mutter",
	} {
		if _, ok := srv.dispatcher.lookup(name); !ok {
			t.Fatalf("%s should remain registered", name)
		}
	}
}
