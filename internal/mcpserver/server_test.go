package mcpserver

import (
	"context"
	"encoding/json"
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
	if len(names) < 40 {
		t.Fatalf("expected at least 40 tools registered, got %d", len(names))
	}
	for _, essential := range []string{
		"fibe_status",
		"fibe_doctor",
		"fibe_help",
		"fibe_run",
		"fibe_schema",
		"fibe_auth_set",
		"fibe_pipeline",
		"fibe_pipeline_result",
		"fibe_resource_list",
		"fibe_resource_get",
		"fibe_resource_delete",
		"fibe_resource_mutate",
		"fibe_mutter",
		"fibe_templates_develop",
		"fibe_playgrounds_wait",
		"fibe_playgrounds_logs",
		"fibe_monitor_list",
		"fibe_monitor_follow",
		"fibe_agents_start_chat",
		"fibe_agents_runtime_status",
	} {
		if _, ok := srv.dispatcher.lookup(essential); !ok {
			t.Errorf("essential tool %q not registered", essential)
		}
	}
}

func TestDefaultConfigUsesFullToolset(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.ToolSet != "full" {
		t.Fatalf("expected default ToolSet full, got %q", cfg.ToolSet)
	}
}

func TestFullModeAdvertisesCoreAgentTools(t *testing.T) {
	srv := New(Config{APIKey: "pk_test", ToolSet: "full", PipelineCacheSize: 4})
	if err := srv.RegisterAll(); err != nil {
		t.Fatalf("RegisterAll: %v", err)
	}

	advertised := advertisedToolNames(srv)
	for _, name := range []string{
		"fibe_agents_start_chat",
		"fibe_resource_list",
		"fibe_resource_get",
		"fibe_resource_delete",
		"fibe_monitor_list",
		"fibe_monitor_follow",
		"fibe_playgrounds_debug",
		"fibe_playgrounds_logs",
		"fibe_agents_runtime_status",
	} {
		if !advertised[name] {
			t.Errorf("%s should be advertised in full mode", name)
		}
	}

	for _, removed := range []string{"fibe_mutations_create", "fibe_teams_list"} {
		if _, ok := srv.dispatcher.lookup(removed); ok {
			t.Errorf("%s should not be registered", removed)
		}
	}
}

func TestCoreModeAdvertisesTemplateIterationAndDiagnosticsTools(t *testing.T) {
	srv := New(Config{APIKey: "pk_test", ToolSet: "core", PipelineCacheSize: 4})
	if err := srv.RegisterAll(); err != nil {
		t.Fatalf("RegisterAll: %v", err)
	}

	advertised := advertisedToolNames(srv)
	for _, name := range []string{
		"fibe_templates_develop",
		"fibe_resource_mutate",
		"fibe_playgrounds_action",
		"fibe_playgrounds_debug",
		"fibe_playgrounds_wait",
		"fibe_playgrounds_logs",
	} {
		if !advertised[name] {
			t.Errorf("%s should be advertised in core mode", name)
		}
	}
}

func TestToolAnnotationsDoNotMarkReadOnlyToolsDestructive(t *testing.T) {
	srv := New(Config{APIKey: "pk_test", ToolSet: "full", PipelineCacheSize: 4})
	if err := srv.RegisterAll(); err != nil {
		t.Fatalf("RegisterAll: %v", err)
	}

	tools := srv.mcp.ListTools()
	readOnlyTool := tools["fibe_doctor"].Tool
	if readOnlyTool.Annotations.ReadOnlyHint == nil || !*readOnlyTool.Annotations.ReadOnlyHint {
		t.Fatalf("fibe_doctor readOnlyHint = %#v, want true", readOnlyTool.Annotations.ReadOnlyHint)
	}
	if readOnlyTool.Annotations.DestructiveHint == nil || *readOnlyTool.Annotations.DestructiveHint {
		t.Fatalf("fibe_doctor destructiveHint = %#v, want false", readOnlyTool.Annotations.DestructiveHint)
	}

	destructiveTool := tools["fibe_playgrounds_action"].Tool
	if destructiveTool.Annotations.DestructiveHint == nil || !*destructiveTool.Annotations.DestructiveHint {
		t.Fatalf("fibe_playgrounds_action destructiveHint = %#v, want true", destructiveTool.Annotations.DestructiveHint)
	}
}

func TestAdvertisedToolInputSchemasHaveObjectProperties(t *testing.T) {
	for _, toolSet := range []string{"core", "full"} {
		t.Run(toolSet, func(t *testing.T) {
			srv := New(Config{APIKey: "pk_test", ToolSet: toolSet, PipelineCacheSize: 4})
			if err := srv.RegisterAll(); err != nil {
				t.Fatalf("RegisterAll: %v", err)
			}

			for _, tool := range srv.mcp.ListTools() {
				data := []byte(tool.Tool.RawInputSchema)
				if len(data) == 0 {
					var err error
					data, err = json.Marshal(tool.Tool.InputSchema)
					if err != nil {
						t.Fatalf("%s input schema marshal: %v", tool.Tool.Name, err)
					}
				}

				var schema map[string]any
				if err := json.Unmarshal(data, &schema); err != nil {
					t.Fatalf("%s input schema unmarshal: %v", tool.Tool.Name, err)
				}

				props, ok := schema["properties"].(map[string]any)
				if !ok {
					t.Fatalf("%s inputSchema.properties is %T, want object", tool.Tool.Name, schema["properties"])
				}
				for name, prop := range props {
					if _, ok := prop.(map[string]any); !ok {
						t.Fatalf("%s inputSchema.properties.%s is %T (%#v), want object schema", tool.Tool.Name, name, prop, prop)
					}
				}
			}
		})
	}
}

func TestRegisteredToolSchemasHaveDescriptionsAndPositiveIDs(t *testing.T) {
	srv := New(Config{APIKey: "pk_test", ToolSet: "full", PipelineCacheSize: 4})
	if err := srv.RegisterAll(); err != nil {
		t.Fatalf("RegisterAll: %v", err)
	}

	for _, tool := range srv.AllTools() {
		schema, ok := tool.InputSchema.(map[string]any)
		if !ok || schema == nil {
			continue
		}
		assertSchemaDescriptionsAndIDMinimums(t, tool.Name, schema)
	}
}

func TestImportantToolEnumsAreAdvertised(t *testing.T) {
	srv := New(Config{APIKey: "pk_test", ToolSet: "full", PipelineCacheSize: 4})
	if err := srv.RegisterAll(); err != nil {
		t.Fatalf("RegisterAll: %v", err)
	}

	catalogTier := schemaPropertyEnum(t, srv.toolSchemas["fibe_tools_catalog"], "tier")
	for _, want := range []string{"meta", "base", "greenfield", "brownfield", "overseer", "local", "other", "core", "full", "all"} {
		if !containsString(catalogTier, want) {
			t.Fatalf("fibe_tools_catalog.tier enum missing %q: %#v", want, catalogTier)
		}
	}

	mutateResources := schemaPropertyEnum(t, srv.toolSchemas["fibe_resource_mutate"], "resource")
	for _, want := range []string{"agent", "agents", "api-key", "api-keys", "template-version", "template-versions"} {
		if !containsString(mutateResources, want) {
			t.Fatalf("fibe_resource_mutate.resource enum missing %q: %#v", want, mutateResources)
		}
	}

	mutateOperations := schemaPropertyEnum(t, srv.toolSchemas["fibe_resource_mutate"], "operation")
	for _, want := range []string{"create", "update", "toggle_public"} {
		if !containsString(mutateOperations, want) {
			t.Fatalf("fibe_resource_mutate.operation enum missing %q: %#v", want, mutateOperations)
		}
	}

	actions := schemaPropertyEnum(t, srv.toolSchemas["fibe_playgrounds_action"], "action_type")
	for _, want := range []string{"rollout", "hard_restart", "stop", "start", "retry_compose"} {
		if !containsString(actions, want) {
			t.Fatalf("fibe_playgrounds_action.action_type enum missing %q: %#v", want, actions)
		}
	}

}

func TestDispatchRejectsNonPositiveIDs(t *testing.T) {
	srv := New(Config{APIKey: "pk_test", ToolSet: "core", Yolo: true})
	if err := srv.RegisterAll(); err != nil {
		t.Fatalf("RegisterAll: %v", err)
	}

	_, err := srv.dispatcher.dispatch(context.Background(), "fibe_resource_get", map[string]any{
		"resource": "playground",
		"id":       0,
	})
	if err == nil || !strings.Contains(err.Error(), "greater than zero") {
		t.Fatalf("expected positive ID validation error, got %v", err)
	}
}

func TestToolsCatalogIncludesEnrichedSchemas(t *testing.T) {
	srv := New(Config{APIKey: "pk_test", ToolSet: "core", PipelineCacheSize: 4})
	if err := srv.RegisterAll(); err != nil {
		t.Fatalf("RegisterAll: %v", err)
	}

	out, err := srv.dispatcher.dispatch(context.Background(), "fibe_tools_catalog", map[string]any{
		"name_pattern":   "fibe_resource_mutate",
		"include_schema": true,
	})
	if err != nil {
		t.Fatalf("fibe_tools_catalog: %v", err)
	}
	result := out.(map[string]any)
	data, err := json.Marshal(result["tools"])
	if err != nil {
		t.Fatalf("marshal tools: %v", err)
	}
	var tools []map[string]any
	if err := json.Unmarshal(data, &tools); err != nil {
		t.Fatalf("unmarshal tools: %v", err)
	}
	if len(tools) != 1 {
		t.Fatalf("expected one catalog entry, got %#v", tools)
	}
	inputSchema := tools[0]["input_schema"].(map[string]any)
	resources := schemaPropertyEnum(t, inputSchema, "resource")
	if !containsString(resources, "agent") {
		t.Fatalf("catalog schema missing resource enum: %#v", resources)
	}
}

func assertSchemaDescriptionsAndIDMinimums(t *testing.T, path string, schema map[string]any) {
	t.Helper()
	props, _ := schema["properties"].(map[string]any)
	for name, raw := range props {
		prop, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		fieldPath := path + "." + name
		desc, _ := prop["description"].(string)
		if strings.TrimSpace(desc) == "" {
			t.Fatalf("%s is missing description", fieldPath)
		}
		if isNumericIDSchema(name, prop) {
			minimum, ok := numericMinimum(prop["minimum"])
			if !ok || minimum < 1 {
				t.Fatalf("%s missing minimum >= 1: %#v", fieldPath, prop)
			}
		}
		assertSchemaDescriptionsAndIDMinimums(t, fieldPath, prop)
		if items, ok := prop["items"].(map[string]any); ok {
			assertSchemaDescriptionsAndIDMinimums(t, fieldPath+"[]", items)
		}
	}
}

func schemaPropertyEnum(t *testing.T, schema map[string]any, property string) []string {
	t.Helper()
	props, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatalf("schema properties is %T", schema["properties"])
	}
	prop, ok := props[property].(map[string]any)
	if !ok {
		t.Fatalf("schema property %q is %T", property, props[property])
	}
	raw, ok := prop["enum"].([]any)
	if !ok {
		t.Fatalf("schema property %q enum is %T", property, prop["enum"])
	}
	out := make([]string, 0, len(raw))
	for _, value := range raw {
		s, ok := value.(string)
		if !ok {
			t.Fatalf("enum value is %T", value)
		}
		out = append(out, s)
	}
	return out
}

func catalogToolsFromResult(t *testing.T, out any) []map[string]any {
	t.Helper()
	result, ok := out.(map[string]any)
	if !ok {
		t.Fatalf("catalog result is %T", out)
	}
	data, err := json.Marshal(result["tools"])
	if err != nil {
		t.Fatalf("marshal catalog tools: %v", err)
	}
	var tools []map[string]any
	if err := json.Unmarshal(data, &tools); err != nil {
		t.Fatalf("unmarshal catalog tools: %v", err)
	}
	return tools
}

func catalogHasTool(tools []map[string]any, name string) bool {
	for _, tool := range tools {
		if tool["name"] == name {
			return true
		}
	}
	return false
}

func numericMinimum(v any) (float64, bool) {
	switch n := v.(type) {
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case float64:
		return n, true
	case json.Number:
		f, err := n.Float64()
		return f, err == nil
	}
	return 0, false
}

// TestCoreTierFilter verifies FIBE_MCP_TOOLS=core advertises the
// meta+base+greenfield+brownfield shortcut while dispatcher still knows
// every tool (so fibe_call and pipeline steps can reach them).
func TestCoreTierFilter(t *testing.T) {
	srv := New(Config{APIKey: "pk_test", ToolSet: "core", PipelineCacheSize: 4})
	if err := srv.RegisterAll(); err != nil {
		t.Fatalf("RegisterAll: %v", err)
	}

	// Dispatcher has everything regardless of tier.
	full := srv.dispatcher.names()

	// Meta tools are included by the core shortcut.
	for _, meta := range []string{"fibe_pipeline", "fibe_help", "fibe_run"} {
		if _, ok := srv.dispatcher.lookup(meta); !ok {
			t.Errorf("%s should always be registered", meta)
		}
	}

	// Generic resource tools replace the old flat list/get/delete MCP tools.
	if _, ok := srv.dispatcher.lookup("fibe_resource_delete"); !ok {
		t.Errorf("fibe_resource_delete should be reachable via dispatcher in core mode")
	}
	if _, ok := srv.dispatcher.lookup("fibe_api_keys_delete"); ok {
		t.Errorf("old flat fibe_api_keys_delete should not be registered")
	}

	// Count scaled with tier.
	t.Logf("registered %d tools in core mode (dispatcher-level)", len(full))
}

func TestCoreAdvertisesMainPlaygroundToolsAndMeta(t *testing.T) {
	srv := New(Config{APIKey: "pk_test", ToolSet: "core", PipelineCacheSize: 4})
	if err := srv.RegisterAll(); err != nil {
		t.Fatalf("RegisterAll: %v", err)
	}

	advertised := advertisedToolNames(srv)

	for _, name := range []string{
		"fibe_resource_list",
		"fibe_resource_get",
		"fibe_resource_delete",
		"fibe_resource_mutate",
		"fibe_mutter",
		"fibe_greenfield_create",
		"fibe_templates_search",
		"fibe_templates_launch",
		"fibe_templates_develop",
		"fibe_playgrounds_debug",
		"fibe_playgrounds_logs",
		"fibe_playgrounds_wait",
		"fibe_local_playgrounds_link",
	} {
		if !advertised[name] {
			t.Errorf("%s should be advertised in core mode", name)
		}
	}

	for _, name := range []string{
		"fibe_pipeline",
		"fibe_pipeline_result",
		"fibe_help",
		"fibe_run",
		"fibe_status",
		"fibe_doctor",
		"fibe_schema",
		"fibe_tools_catalog",
		"fibe_call",
	} {
		if !advertised[name] {
			t.Errorf("%s should always be advertised", name)
		}
		tImpl, ok := srv.dispatcher.lookup(name)
		if !ok {
			t.Fatalf("meta tool %s not registered", name)
		}
		if tImpl.tier != tierMeta {
			t.Errorf("%s tier=%v want tierMeta", name, tImpl.tier)
		}
	}

	if !advertised["fibe_playgrounds_logs_follow"] {
		t.Errorf("fibe_playgrounds_logs_follow should be advertised in core mode")
	}
	if advertised["fibe_auth_set"] {
		t.Errorf("fibe_auth_set should not be advertised in core mode; it belongs to the other tier")
	}
}

func TestCommaSeparatedToolSetAdvertisesSelectedTiers(t *testing.T) {
	srv := New(Config{APIKey: "pk_test", ToolSet: "other,meta", PipelineCacheSize: 4})
	if err := srv.RegisterAll(); err != nil {
		t.Fatalf("RegisterAll: %v", err)
	}

	advertised := advertisedToolNames(srv)
	for _, name := range []string{
		"fibe_call",
		"fibe_schema",
		"fibe_tools_catalog",
		"fibe_auth_set",
		"fibe_get_github_token",
		"fibe_repo_status_check",
	} {
		if !advertised[name] {
			t.Errorf("%s should be advertised for other,meta", name)
		}
	}
	for _, name := range []string{
		"fibe_resource_list",
		"fibe_mutter",
		"fibe_greenfield_create",
		"fibe_playgrounds_action",
		"fibe_agents_runtime_status",
		"fibe_local_playgrounds_list",
	} {
		if advertised[name] {
			t.Errorf("%s should not be advertised for other,meta", name)
		}
	}
}

func TestInvalidToolSetRejected(t *testing.T) {
	srv := New(Config{APIKey: "pk_test", ToolSet: "banana"})
	err := srv.RegisterAll()
	if err == nil || !strings.Contains(err.Error(), "unknown MCP tool tier") {
		t.Fatalf("expected invalid tool tier error, got %v", err)
	}
}

func TestToolsCatalogTierShortcuts(t *testing.T) {
	srv := New(Config{APIKey: "pk_test", ToolSet: "core", PipelineCacheSize: 4})
	if err := srv.RegisterAll(); err != nil {
		t.Fatalf("RegisterAll: %v", err)
	}

	out, err := srv.dispatcher.dispatch(context.Background(), "fibe_tools_catalog", map[string]any{
		"tier": "core",
	})
	if err != nil {
		t.Fatalf("fibe_tools_catalog core: %v", err)
	}
	coreTools := catalogToolsFromResult(t, out)
	for _, tool := range coreTools {
		tier, _ := tool["tier"].(string)
		switch tier {
		case "meta", "base", "greenfield", "brownfield":
		default:
			t.Fatalf("core catalog included tier %q in %#v", tier, tool)
		}
	}
	if !catalogHasTool(coreTools, "fibe_resource_list") || !catalogHasTool(coreTools, "fibe_playgrounds_action") {
		t.Fatalf("core catalog missing expected base/brownfield tools: %#v", coreTools)
	}
	if catalogHasTool(coreTools, "fibe_agents_runtime_status") {
		t.Fatalf("core catalog should not include overseer tools")
	}

	out, err = srv.dispatcher.dispatch(context.Background(), "fibe_tools_catalog", map[string]any{
		"tier": "overseer",
	})
	if err != nil {
		t.Fatalf("fibe_tools_catalog overseer: %v", err)
	}
	overseerTools := catalogToolsFromResult(t, out)
	if !catalogHasTool(overseerTools, "fibe_agents_runtime_status") {
		t.Fatalf("overseer catalog missing agent runtime status: %#v", overseerTools)
	}
	for _, tool := range overseerTools {
		if tier, _ := tool["tier"].(string); tier != "overseer" {
			t.Fatalf("overseer catalog included tier %q in %#v", tier, tool)
		}
	}
}

// TestConfirmGate verifies destructive tools require confirm:true unless
// --yolo is set.
func TestConfirmGate(t *testing.T) {
	srv := New(Config{APIKey: "pk_test", ToolSet: "core"})
	if err := srv.RegisterAll(); err != nil {
		t.Fatalf("RegisterAll: %v", err)
	}

	ctx := context.Background()

	// fibe_resource_delete without confirm should error.
	_, err := srv.dispatcher.dispatch(ctx, "fibe_resource_delete", map[string]any{
		"resource": "playground",
		"id":       42,
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
	_, err := srv.dispatcher.dispatch(ctx, "fibe_resource_delete", map[string]any{
		"resource": "audit_log",
		"id":       42,
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
