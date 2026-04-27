package mcpserver

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/fibegg/sdk/fibe"
	"github.com/fibegg/sdk/internal/resourceschema"
	"github.com/mark3labs/mcp-go/mcp"
)

func TestResourceListDispatchesToResourceService(t *testing.T) {
	apiKey, domain := requireRealServer(t)

	srv := New(Config{APIKey: apiKey, Domain: domain, ToolSet: "core"})
	if err := srv.RegisterAll(); err != nil {
		t.Fatalf("RegisterAll: %v", err)
	}

	res, err := srv.dispatcher.dispatch(context.Background(), "fibe_resource_list", map[string]any{
		"resource": "agents",
		"params": map[string]any{
			"per_page": 5,
		},
	})
	if err != nil {
		t.Fatalf("dispatch: %v", err)
	}
	m := res.(*fibe.ListResult[fibe.Agent])
	if m.Data == nil {
		t.Fatalf("expected data field in response")
	}
}

func TestResourceGetDispatchesWithAlias(t *testing.T) {
	apiKey, domain := requireRealServer(t)

	srv := New(Config{APIKey: apiKey, Domain: domain, ToolSet: "core"})
	if err := srv.RegisterAll(); err != nil {
		t.Fatalf("RegisterAll: %v", err)
	}

	agentName := fmt.Sprintf("test-agent-%d", time.Now().UnixNano())
	res, err := srv.dispatcher.dispatch(context.Background(), "fibe_resource_mutate", map[string]any{
		"resource":  "agents",
		"operation": "create",
		"payload": map[string]any{
			"name":     agentName,
			"provider": "openai-codex",
		},
	})
	if err != nil {
		t.Fatalf("create agent: %v", err)
	}
	m := res.(*fibe.Agent)
	agentID := int(m.ID)

	getRes, err := srv.dispatcher.dispatch(context.Background(), "fibe_resource_get", map[string]any{
		"resource": "agents",
		"id":       agentID,
	})
	if err != nil {
		t.Fatalf("get dispatch: %v", err)
	}
	getM := getRes.(*fibe.Agent)
	if getM.Name != agentName {
		t.Fatalf("expected name %s, got %s", agentName, getM.Name)
	}
}

func TestResourceDeleteDispatchesWithAlias(t *testing.T) {
	apiKey, domain := requireRealServer(t)

	srv := New(Config{APIKey: apiKey, Domain: domain, ToolSet: "core", Yolo: true})
	if err := srv.RegisterAll(); err != nil {
		t.Fatalf("RegisterAll: %v", err)
	}

	agentName := fmt.Sprintf("test-agent-%d", time.Now().UnixNano())
	res, err := srv.dispatcher.dispatch(context.Background(), "fibe_resource_mutate", map[string]any{
		"resource":  "agents",
		"operation": "create",
		"payload": map[string]any{
			"name":     agentName,
			"provider": "openai-codex",
		},
	})
	if err != nil {
		t.Fatalf("create agent: %v", err)
	}
	m := res.(*fibe.Agent)
	agentID := int(m.ID)

	if _, err := srv.dispatcher.dispatch(context.Background(), "fibe_resource_delete", map[string]any{
		"resource": "agents",
		"id":       agentID,
	}); err != nil {
		t.Fatalf("dispatch: %v", err)
	}
}

func TestResourceListDispatchesNewConsolidatedResources(t *testing.T) {
	apiKey, domain := requireRealServer(t)

	srv := New(Config{APIKey: apiKey, Domain: domain, ToolSet: "core"})
	if err := srv.RegisterAll(); err != nil {
		t.Fatalf("RegisterAll: %v", err)
	}

	cases := []map[string]any{
		{"resource": "artefact", "params": map[string]any{"q": "report"}},
		{"resource": "template_version", "params": map[string]any{"template_id": 4}},
		{"resource": "webhook_delivery", "params": map[string]any{"webhook_id": 9}},
	}
	for _, args := range cases {
		_, err := srv.dispatcher.dispatch(context.Background(), "fibe_resource_list", args)
		if err != nil && !strings.Contains(err.Error(), "404") && !strings.Contains(err.Error(), "status") {
			t.Fatalf("dispatch %#v failed with non-HTTP error: %v", args, err)
		}
	}
}

func TestResourceGetDispatchesArtefactAndAttachment(t *testing.T) {
	apiKey, domain := requireRealServer(t)

	srv := New(Config{APIKey: apiKey, Domain: domain, ToolSet: "core"})
	if err := srv.RegisterAll(); err != nil {
		t.Fatalf("RegisterAll: %v", err)
	}

	agentName := fmt.Sprintf("test-agent-%d", time.Now().UnixNano())
	res, err := srv.dispatcher.dispatch(context.Background(), "fibe_resource_mutate", map[string]any{
		"resource":  "agents",
		"operation": "create",
		"payload": map[string]any{
			"name":     agentName,
			"provider": "openai-codex",
		},
	})
	if err != nil {
		t.Fatalf("create agent: %v", err)
	}
	m := res.(*fibe.Agent)
	agentID := int(m.ID)

	os.Setenv("FIBE_AGENT_ID", fmt.Sprintf("%d", agentID))
	defer os.Unsetenv("FIBE_AGENT_ID")

	if _, err := srv.dispatcher.dispatch(context.Background(), "fibe_artefact_upload", map[string]any{
		"name":           "report.txt",
		"content_base64": "aGVsbG8=",
	}); err != nil {
		t.Fatalf("dispatch fibe_artefact_upload: %v", err)
	}

	artefactsRes, err := srv.dispatcher.dispatch(context.Background(), "fibe_resource_list", map[string]any{
		"resource": "artefacts",
		"params": map[string]any{
			"agent_id": fmt.Sprintf("%d", agentID),
		},
	})
	if err != nil {
		t.Fatalf("artefacts list dispatch: %v", err)
	}
	artefactsData := artefactsRes.(*fibe.ListResult[fibe.Artefact]).Data
	if len(artefactsData) == 0 {
		t.Fatalf("expected artefacts, got none")
	}
	artefactID := int(artefactsData[0].ID)

	if _, err := srv.dispatcher.dispatch(context.Background(), "fibe_resource_get", map[string]any{
		"resource": "artefact",
		"id":       artefactID,
	}); err != nil {
		t.Fatalf("artefact dispatch: %v", err)
	}
	out, err := srv.dispatcher.dispatch(context.Background(), "fibe_resource_get", map[string]any{
		"resource": "artefact_attachment",
		"id":       artefactID,
	})
	if err != nil {
		t.Fatalf("attachment dispatch: %v", err)
	}
	attachment := out.(map[string]any)
	if attachment["content_base64"] != "aGVsbG8=" {
		t.Fatalf("unexpected attachment payload: %#v", attachment)
	}
	if attachment["filename"] != "report.txt" {
		t.Fatalf("unexpected attachment metadata: %#v", attachment)
	}
}

func TestResourceDeleteDispatchesTemplateVersionAndSource(t *testing.T) {
	apiKey, domain := requireRealServer(t)

	srv := New(Config{APIKey: apiKey, Domain: domain, ToolSet: "core", Yolo: true})
	if err := srv.RegisterAll(); err != nil {
		t.Fatalf("RegisterAll: %v", err)
	}
	for _, args := range []map[string]any{
		{"resource": "template_version", "id": 6},
		{"resource": "template_source", "id": 4},
	} {
		_, err := srv.dispatcher.dispatch(context.Background(), "fibe_resource_delete", args)
		if err != nil && !strings.Contains(err.Error(), "404") && !strings.Contains(err.Error(), "status") {
			t.Fatalf("dispatch %#v failed with non-HTTP error: %v", args, err)
		}
	}
}

func TestPlaygroundActionDispatchesToActionEndpoint(t *testing.T) {
	apiKey, domain := requireRealServer(t)

	srv := New(Config{APIKey: apiKey, Domain: domain, ToolSet: "core", Yolo: true})
	if err := srv.RegisterAll(); err != nil {
		t.Fatalf("RegisterAll: %v", err)
	}

	_, err := srv.dispatcher.dispatch(context.Background(), "fibe_playgrounds_action", map[string]any{
		"playground_id": 7,
		"action_type":   "hard_restart",
		"force":         true,
	})
	if err != nil && !strings.Contains(err.Error(), "404") && !strings.Contains(err.Error(), "status") {
		t.Fatalf("dispatch error: %v", err)
	}
}

func TestResourceUnsupportedOperations(t *testing.T) {
	srv := New(Config{APIKey: "pk_test", ToolSet: "core"})
	if err := srv.RegisterAll(); err != nil {
		t.Fatalf("RegisterAll: %v", err)
	}

	_, err := srv.dispatcher.dispatch(context.Background(), "fibe_resource_get", map[string]any{
		"resource": "audit_log",
		"id":       1,
	})
	if err == nil || !strings.Contains(err.Error(), "does not support get") {
		t.Fatalf("expected unsupported get error, got %v", err)
	}

	_, err = srv.dispatcher.dispatch(context.Background(), "fibe_resource_list", map[string]any{
		"resource": "unknown",
	})
	if err == nil || !strings.Contains(err.Error(), "supported flat resources") {
		t.Fatalf("expected supported flat resources error, got %v", err)
	}
}

func TestResourceGetRejectsReveal(t *testing.T) {
	srv := New(Config{APIKey: "pk_test", ToolSet: "core"})
	if err := srv.RegisterAll(); err != nil {
		t.Fatalf("RegisterAll: %v", err)
	}

	_, err := srv.dispatcher.dispatch(context.Background(), "fibe_resource_get", map[string]any{
		"resource": "secret",
		"id":       9,
		"reveal":   true,
	})
	if err == nil || !strings.Contains(err.Error(), "reveal") {
		t.Fatalf("expected reveal rejection, got %v", err)
	}
}

func TestOldFlatResourceToolsAreNotRegistered(t *testing.T) {
	srv := New(Config{APIKey: "pk_test", ToolSet: "full"})
	if err := srv.RegisterAll(); err != nil {
		t.Fatalf("RegisterAll: %v", err)
	}

	for _, name := range []string{
		"fibe_playgrounds_list",
		"fibe_playgrounds_get",
		"fibe_playgrounds_delete",
		"fibe_playgrounds_create",
		"fibe_playgrounds_update",
		"fibe_agents_create",
		"fibe_agents_update",
		"fibe_artefacts_create",
		"fibe_mutters_create",
		"fibe_templates_versions_create",
		"fibe_templates_versions_patch_create",
		"fibe_job_env_list",
		"fibe_job_env_get",
		"fibe_job_env_delete",
		"fibe_job_env_set",
		"fibe_job_env_update",
		"fibe_secrets_get",
		"fibe_api_keys_delete",
		"fibe_playgrounds_rollout",
		"fibe_playgrounds_hard_restart",
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
		"fibe_agents_raw_providers_get",
		"fibe_agents_raw_providers_update",
		"fibe_templates_lineage",
		"fibe_marquees_generate_ssh_key_status",
		"fibe_marquees_test_connection_status",
		"fibe_feedbacks_create",
		"fibe_feedbacks_delete",
		"fibe_feedbacks_update",
		"fibe_artefacts_list",
		"fibe_artefacts_download",
		"fibe_templates_versions_list",
		"fibe_webhooks_deliveries_list",
		"fibe_webhooks_event_types",
		"fibe_playspecs_validate_compose",
		"fibe_templates_patch_apply",
		"fibe_templates_versions_patch_preview",
		"fibe_playspecs_switch_version",
		"fibe_playspecs_switch_version_preview",
	} {
		if _, ok := srv.dispatcher.lookup(name); ok {
			t.Fatalf("%s should not be registered", name)
		}
	}

	for _, name := range []string{
		"fibe_resource_mutate",
		"fibe_mutter",
		"fibe_templates_develop",
		"fibe_playgrounds_action",
		"fibe_feedbacks_get",
		"fibe_templates_launch",
		"fibe_feedbacks_list",
	} {
		if _, ok := srv.dispatcher.lookup(name); !ok {
			t.Fatalf("%s should remain registered", name)
		}
	}
}

func TestResourceSchemaCatalog(t *testing.T) {
	srv := New(Config{APIKey: "pk_test", ToolSet: "core"})
	if err := srv.RegisterAll(); err != nil {
		t.Fatalf("RegisterAll: %v", err)
	}

	out, err := srv.dispatcher.dispatch(context.Background(), "fibe_schema", map[string]any{
		"resource": "list",
	})
	if err != nil {
		t.Fatalf("fibe_schema resource=list: %v", err)
	}
	catalog, ok := out.(map[string]any)
	if !ok {
		t.Fatalf("catalog is %T, want map", out)
	}
	resources, ok := catalog["resources"].([]resourceschema.CatalogEntry)
	if !ok {
		t.Fatalf("resources is %T, want []CatalogEntry", catalog["resources"])
	}
	var foundPlayground bool
	var foundAgent bool
	for _, entry := range resources {
		if entry.Name == "playground" {
			foundPlayground = true
			if !containsString(entry.Aliases, "playgrounds") || !containsString(entry.Operations, "list") || !containsString(entry.Operations, "action") {
				t.Fatalf("unexpected playground catalog entry: %#v", entry)
			}
		}
		if entry.Name == "agent" {
			foundAgent = true
			if !containsString(entry.Operations, "create") || !containsString(entry.Operations, "update") {
				t.Fatalf("agent catalog should include create/update operations: %#v", entry)
			}
		}
	}
	if !foundPlayground {
		t.Fatal("playground missing from resource catalog")
	}
	if !foundAgent {
		t.Fatal("agent missing from resource catalog")
	}

	schema, err := srv.dispatcher.dispatch(context.Background(), "fibe_schema", map[string]any{
		"resource":  "playgrounds",
		"operation": "list",
	})
	if err != nil {
		t.Fatalf("fibe_schema playground list: %v", err)
	}
	m, ok := schema.(map[string]any)
	if !ok {
		t.Fatalf("schema is %T, want map", schema)
	}
	props := m["properties"].(map[string]any)
	params := props["params"].(map[string]any)
	paramProps := params["properties"].(map[string]any)
	if _, ok := paramProps["per_page"]; !ok {
		t.Fatalf("expected per_page in params schema, got %#v", paramProps)
	}

	updateSchema, err := srv.dispatcher.dispatch(context.Background(), "fibe_schema", map[string]any{
		"resource":  "agent",
		"operation": "update",
	})
	if err != nil {
		t.Fatalf("fibe_schema agent update: %v", err)
	}
	updateProps := updateSchema.(map[string]any)["properties"].(map[string]any)
	for _, want := range []string{"agent_id", "name", "mode", "model_options"} {
		if _, ok := updateProps[want]; !ok {
			t.Fatalf("expected %s in agent.update schema, got %#v", want, updateProps)
		}
	}

	actionSchema, err := srv.dispatcher.dispatch(context.Background(), "fibe_schema", map[string]any{
		"resource":  "playground",
		"operation": "action",
	})
	if err != nil {
		t.Fatalf("fibe_schema playground action: %v", err)
	}
	actionProps := actionSchema.(map[string]any)["properties"].(map[string]any)
	if _, ok := actionProps["playground_id"]; !ok {
		t.Fatalf("expected playground_id in playground.action schema, got %#v", actionProps)
	}
	actionEnum, ok := actionProps["action_type"].(map[string]any)["enum"].([]string)
	if !ok || !containsString(actionEnum, "retry_compose") {
		t.Fatalf("expected action_type enum in playground.action schema, got %#v", actionProps["action_type"])
	}

	composeSchema, err := srv.dispatcher.dispatch(context.Background(), "fibe_schema", map[string]any{
		"resource":  "compose",
		"operation": "validate",
	})
	if err != nil {
		t.Fatalf("fibe_schema compose validate: %v", err)
	}
	composeProps := composeSchema.(map[string]any)["properties"].(map[string]any)
	targetEnum := composeProps["target_type"].(map[string]any)["enum"].([]string)
	if !containsString(targetEnum, "trick") {
		t.Fatalf("compose.validate target_type enum missing trick: %#v", targetEnum)
	}

	eventSchema, err := srv.dispatcher.dispatch(context.Background(), "fibe_schema", map[string]any{
		"resource":  "webhook",
		"operation": "event_types",
	})
	if err != nil {
		t.Fatalf("fibe_schema webhook event_types: %v", err)
	}
	if len(eventSchema.(map[string]any)["event_types"].([]string)) == 0 {
		t.Fatalf("expected webhook event types in schema: %#v", eventSchema)
	}

	developSchema, err := srv.dispatcher.dispatch(context.Background(), "fibe_schema", map[string]any{
		"resource":  "template",
		"operation": "develop",
	})
	if err != nil {
		t.Fatalf("fibe_schema template develop: %v", err)
	}
	developProps := developSchema.(map[string]any)["properties"].(map[string]any)
	postApplyEnum := developProps["post_apply"].(map[string]any)["enum"].([]string)
	if !containsString(postApplyEnum, "trigger_trick") {
		t.Fatalf("template.develop post_apply enum missing trigger_trick: %#v", postApplyEnum)
	}
}

func TestResourceToolSchemasUseOperationSpecificEnums(t *testing.T) {
	srv := New(Config{APIKey: "pk_test", ToolSet: "core"})
	if err := srv.RegisterAll(); err != nil {
		t.Fatalf("RegisterAll: %v", err)
	}
	tools := srv.mcp.ListTools()

	listEnum := toolPropertyEnum(t, tools["fibe_resource_list"].Tool, "resource")
	for _, want := range []string{"category", "categories", "audit-log"} {
		if !containsString(listEnum, want) {
			t.Fatalf("list enum missing %q: %#v", want, listEnum)
		}
	}
	for _, want := range []string{"artefact", "artefacts", "template-version", "template-versions", "webhook-delivery"} {
		if !containsString(listEnum, want) {
			t.Fatalf("list enum missing %q: %#v", want, listEnum)
		}
	}

	getEnum := toolPropertyEnum(t, tools["fibe_resource_get"].Tool, "resource")
	for _, disallowed := range []string{"api-key", "api-keys", "category", "audit-log"} {
		if containsString(getEnum, disallowed) {
			t.Fatalf("get enum should not include %q: %#v", disallowed, getEnum)
		}
	}

	deleteEnum := toolPropertyEnum(t, tools["fibe_resource_delete"].Tool, "resource")
	for _, want := range []string{"api-key", "api-keys", "playground", "playgrounds"} {
		if !containsString(deleteEnum, want) {
			t.Fatalf("delete enum missing %q: %#v", want, deleteEnum)
		}
	}
	for _, disallowed := range []string{"category", "categories", "audit-log"} {
		if containsString(deleteEnum, disallowed) {
			t.Fatalf("delete enum should not include %q: %#v", disallowed, deleteEnum)
		}
	}
}

func TestSchemaToolAdvertisesResourceAndOperationEnums(t *testing.T) {
	srv := New(Config{APIKey: "pk_test", ToolSet: "core"})
	if err := srv.RegisterAll(); err != nil {
		t.Fatalf("RegisterAll: %v", err)
	}
	tool := srv.mcp.ListTools()["fibe_schema"].Tool

	resourceEnum := toolPropertyEnum(t, tool, "resource")
	for _, want := range []string{"list", "agent", "agents", "api-key", "api-keys", "artefact", "artefacts", "template-version", "template-versions"} {
		if !containsString(resourceEnum, want) {
			t.Fatalf("fibe_schema resource enum missing %q: %#v", want, resourceEnum)
		}
	}

	operationEnum := toolPropertyEnum(t, tool, "operation")
	for _, want := range []string{"list", "get", "delete", "create", "update", "action", "develop", "validate", "event_types"} {
		if !containsString(operationEnum, want) {
			t.Fatalf("fibe_schema operation enum missing %q: %#v", want, operationEnum)
		}
	}
}

func containsString(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}

func toolPropertyEnum(t *testing.T, tool mcp.Tool, property string) []string {
	t.Helper()
	rawSchema := toolInputSchemaToMap(tool)
	schema, ok := rawSchema.(map[string]any)
	if !ok {
		t.Fatalf("input schema is %T, want map", rawSchema)
	}
	props, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatalf("schema properties is %T, want map", schema["properties"])
	}
	prop, ok := props[property].(map[string]any)
	if !ok {
		t.Fatalf("schema property %q is %T, want map", property, props[property])
	}
	raw, ok := prop["enum"].([]any)
	if !ok {
		t.Fatalf("schema property %q enum is %T, want []any", property, prop["enum"])
	}
	values := make([]string, 0, len(raw))
	for _, value := range raw {
		s, ok := value.(string)
		if !ok {
			t.Fatalf("enum value is %T, want string", value)
		}
		values = append(values, s)
	}
	return values
}
