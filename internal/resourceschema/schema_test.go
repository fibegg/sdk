package resourceschema

import (
	"reflect"
	"strings"
	"testing"

	"github.com/fibegg/sdk/fibe"
)

func TestCanonicalResourceNormalizesAliases(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want string
	}{
		{in: "playgrounds", want: "playground"},
		{in: "API-KEYS", want: "api_key"},
		{in: "artefacts", want: "artefact"},
		{in: "import-templates", want: "template"},
		{in: "job-envs", want: "job_env"},
		{in: "template-versions", want: "template_version"},
	} {
		got, ok := CanonicalResource(tc.in)
		if !ok || got != tc.want {
			t.Fatalf("CanonicalResource(%q) = %q, %v; want %q, true", tc.in, got, ok, tc.want)
		}
	}
}

func TestSchemaOnlyResourcesDoNotEnterGenericResourceSelectors(t *testing.T) {
	generic := ResourceSelectors()
	for _, disallowed := range []string{"compose", "mutter", "mutters"} {
		if containsSelector(generic, disallowed) {
			t.Fatalf("generic resource selectors should not include schema-only resource %q: %#v", disallowed, generic)
		}
	}

	schema := SchemaResourceSelectors()
	for _, want := range []string{"artefact", "artefacts", "compose", "mutter", "mutters", "template-version", "template-versions"} {
		if !containsSelector(schema, want) {
			t.Fatalf("schema resource selectors missing %q: %#v", want, schema)
		}
	}

	mutation := MutationResourceSelectors()
	for _, disallowed := range []string{"mutter", "mutters"} {
		if containsSelector(mutation, disallowed) {
			t.Fatalf("mutation selectors should not include dedicated mutter tool resource %q: %#v", disallowed, mutation)
		}
	}
}

func TestOperationSelectorsAreOperationSpecific(t *testing.T) {
	deleteSelectors := ResourceSelectorsForOperation("delete")
	for _, want := range []string{"api-key", "api-keys", "playground", "playgrounds"} {
		if !containsSelector(deleteSelectors, want) {
			t.Fatalf("delete selectors missing %q: %#v", want, deleteSelectors)
		}
	}
	for _, disallowed := range []string{"category", "categories", "audit-log", "audit-logs"} {
		if containsSelector(deleteSelectors, disallowed) {
			t.Fatalf("delete selectors should not include %q: %#v", disallowed, deleteSelectors)
		}
	}

	getSelectors := ResourceSelectorsForOperation("get")
	if containsSelector(getSelectors, "api-key") || containsSelector(getSelectors, "api-keys") {
		t.Fatalf("get selectors should not include api_key aliases: %#v", getSelectors)
	}
	if !containsSelector(getSelectors, "agent-attachment") || !containsSelector(getSelectors, "agent-attachments") {
		t.Fatalf("get selectors should include agent attachment aliases: %#v", getSelectors)
	}

	watchSelectors := ResourceSelectorsForOperation("watch")
	if !containsSelector(watchSelectors, "agent") || !containsSelector(watchSelectors, "agents") {
		t.Fatalf("watch selectors should include agent aliases: %#v", watchSelectors)
	}
	if containsSelector(watchSelectors, "playground") {
		t.Fatalf("watch selectors should be operation-specific: %#v", watchSelectors)
	}
}

func TestRegistryCoversConcreteCreateUpdateSchemas(t *testing.T) {
	playgroundAction, _, op, ok := SchemaFor("playground", "action")
	if !ok || op != "action" {
		t.Fatalf("playground.action schema missing")
	}
	actionProps := playgroundAction.(map[string]any)["properties"].(map[string]any)
	if _, ok := actionProps["action_type"].(map[string]any)["enum"]; !ok {
		t.Fatalf("playground.action action_type enum missing: %#v", actionProps["action_type"])
	}

	if _, _, _, ok := SchemaFor("artefacts", "create"); ok {
		t.Fatalf("artefact.create should not be exposed through generic resource schema")
	}

	mutterCreate, _, op, ok := SchemaFor("mutter", "create")
	if !ok || op != "create" {
		t.Fatalf("mutter.create schema missing")
	}
	mutterProps := mutterCreate.(map[string]any)["properties"].(map[string]any)
	for _, want := range []string{"agent_id_or_name", "type", "body", "playground_id_or_name"} {
		if _, ok := mutterProps[want]; !ok {
			t.Fatalf("mutter.create missing property %q: %#v", want, mutterProps)
		}
	}

	templateVersionCreate, _, op, ok := SchemaFor("template-version", "create")
	if !ok || op != "create" {
		t.Fatalf("template_version.create schema missing")
	}
	versionCreateProps := templateVersionCreate.(map[string]any)["properties"].(map[string]any)
	for _, want := range []string{"template_id_or_name", "template_body", "template_body_path", "public", "response_mode"} {
		if _, ok := versionCreateProps[want]; !ok {
			t.Fatalf("template_version.create missing property %q: %#v", want, versionCreateProps)
		}
	}

	templateUpdate, _, op, ok := SchemaFor("template", "update")
	if !ok || op != "update" {
		t.Fatalf("template.update schema missing")
	}
	templateProps := templateUpdate.(map[string]any)["properties"].(map[string]any)
	for _, want := range []string{"id_or_name", "name", "description", "category_id", "filename", "image_data", "content_base64", "content_path", "content_type"} {
		if _, ok := templateProps[want]; !ok {
			t.Fatalf("template.update missing property %q: %#v", want, templateProps)
		}
	}
	if _, ok := templateProps["content_type"].(map[string]any)["enum"]; !ok {
		t.Fatalf("template.update content_type enum missing: %#v", templateProps["content_type"])
	}

	memoryMemorize, _, op, ok := SchemaFor("memory", "memorize")
	if !ok || op != "memorize" {
		t.Fatalf("memory.memorize schema missing")
	}
	memoryProps := memoryMemorize.(map[string]any)["properties"].(map[string]any)
	for _, want := range []string{"conversation_id", "content", "tags", "confidence", "groundings"} {
		if _, ok := memoryProps[want]; !ok {
			t.Fatalf("memory.memorize missing property %q: %#v", want, memoryProps)
		}
	}
	for _, disallowed := range []string{"conversation", "messages", "raw_content", "provider", "output_path"} {
		if _, ok := memoryProps[disallowed]; ok {
			t.Fatalf("memory.memorize should not expose %q: %#v", disallowed, memoryProps)
		}
	}
	if _, _, err := ValidatePayload("memory", "memorize", map[string]any{
		"conversation_id": "source-1",
		"content":         "Use refresh-token rotation.",
		"tags":            []any{"auth", "decision"},
		"groundings":      []any{map[string]any{"message_position": 0, "start_character": 4, "end_character": 25}},
	}); err != nil {
		t.Fatalf("memory.memorize should validate: %v", err)
	}

	templateChange, _, op, ok := SchemaFor("template", "change")
	if !ok || op != "change" {
		t.Fatalf("template.change schema missing")
	}
	changeProps := templateChange.(map[string]any)["properties"].(map[string]any)
	for _, want := range []string{"target_type", "target_id_or_name", "mode", "change_type", "confirm", "post_apply", "template_body", "template_body_path"} {
		if _, ok := changeProps[want]; !ok {
			t.Fatalf("template.change missing property %q: %#v", want, changeProps)
		}
	}
	if _, _, err := ValidatePayload("template", "change", map[string]any{
		"target_type":       "playground",
		"target_id_or_name": 42,
		"mode":              "apply",
		"change_type":       "patch",
		"post_apply":        "rollout_target",
		"wait":              true,
		"confirm":           true,
		"confirm_warnings":  true,
		"patches": []any{
			map[string]any{"op": "set", "path": "services.redis.image", "value": "redis:7-alpine", "create_missing": true},
		},
	}); err != nil {
		t.Fatalf("template.change should validate project-local rollout payload: %v", err)
	}
	if _, _, err := ValidatePayload("template", "change", map[string]any{
		"target_type":        "template",
		"target_id_or_name":  42,
		"mode":               "preview",
		"change_type":        "overwrite",
		"template_body":      "services: {}\n",
		"template_body_path": "/tmp/template.yml",
	}); err == nil {
		t.Fatalf("template.change should reject ambiguous template_body/template_body_path selectors")
	}
	if _, _, _, ok := SchemaFor("template", "develop"); ok {
		t.Fatalf("template.develop legacy schema should be removed")
	}
	if playgroundSwitchTemplate, _, op, ok := SchemaFor("playground", "switch_template"); !ok || op != "switch_template" {
		t.Fatalf("playground.switch_template schema missing")
	} else {
		props := playgroundSwitchTemplate.(map[string]any)["properties"].(map[string]any)
		for _, want := range []string{"id_or_name", "template_body", "template_body_path", "template_id_or_name", "template_version_id", "provision_missing_props"} {
			if _, ok := props[want]; !ok {
				t.Fatalf("playground.switch_template missing property %q: %#v", want, props)
			}
		}
	}
	if _, _, err := ValidatePayload("playground", "switch_template", map[string]any{
		"id_or_name": 42,
	}); err == nil {
		t.Fatalf("playground.switch_template should require a template selector")
	}
	if _, _, err := ValidatePayload("playground", "switch_template", map[string]any{
		"id_or_name":          42,
		"template_body":       "services: {}\n",
		"template_version_id": 123,
	}); err == nil {
		t.Fatalf("playground.switch_template should reject ambiguous template selectors")
	}
	marqueeCreate, _, op, ok := SchemaFor("marquee", "create")
	if !ok || op != "create" {
		t.Fatalf("marquee.create schema missing")
	}
	marqueeSchema := marqueeCreate.(map[string]any)
	if !containsSelector(marqueeSchema["required"].([]string), "port") {
		t.Fatalf("marquee.create should require port: %#v", marqueeSchema["required"])
	}
	port := marqueeSchema["properties"].(map[string]any)["port"].(map[string]any)
	if port["default"] != 22 {
		t.Fatalf("marquee.create port default = %#v, want 22", port["default"])
	}

	agentUpdate, _, op, ok := SchemaFor("agent", "update")
	if !ok || op != "update" {
		t.Fatalf("agent.update schema missing")
	}
	props := agentUpdate.(map[string]any)["properties"].(map[string]any)
	for _, want := range []string{"id_or_name", "name", "model_options", "prompt", "system_prompt_mode", "main_md", "main_md_mode", "mcp_json", "post_init_script", "custom_env", "cli_version", "provider_args", "skill_toggles"} {
		if _, ok := props[want]; !ok {
			t.Fatalf("agent.update missing property %q: %#v", want, props)
		}
	}

	agentCreate, _, op, ok := SchemaFor("agent", "create")
	if !ok || op != "create" {
		t.Fatalf("agent.create schema missing")
	}
	createProps := agentCreate.(map[string]any)["properties"].(map[string]any)
	for _, want := range []string{"name", "provider", "model_options", "prompt", "mcp_json", "post_init_script", "custom_env", "cli_version", "provider_args", "skill_toggles"} {
		if _, ok := createProps[want]; !ok {
			t.Fatalf("agent.create missing property %q: %#v", want, createProps)
		}
	}
	providerEnum, ok := createProps["provider"].(map[string]any)["enum"].([]string)
	if !ok {
		t.Fatalf("agent.create provider enum missing: %#v", createProps["provider"])
	}
	if !reflect.DeepEqual(providerEnum, fibe.ValidProviders) {
		t.Fatalf("agent.create provider enum = %#v, want %#v", providerEnum, fibe.ValidProviders)
	}
	modeDescription := createProps["mode"].(map[string]any)["description"].(string)
	if !strings.Contains(modeDescription, "does not inherit auth") || !strings.Contains(modeDescription, "fibe_agents_duplicate") {
		t.Fatalf("agent.create mode description should explain auth inheritance, got %q", modeDescription)
	}

	playspecCreate, _, op, ok := SchemaFor("playspec", "create")
	if !ok || op != "create" {
		t.Fatalf("playspec.create schema missing")
	}
	playspecCreateProps := playspecCreate.(map[string]any)["properties"].(map[string]any)
	for _, field := range []string{"schedule_config", "trigger_config", "muti_config"} {
		prop, ok := playspecCreateProps[field].(map[string]any)
		if !ok {
			t.Fatalf("playspec.create missing %q: %#v", field, playspecCreateProps)
		}
		if prop["type"] != "object" {
			t.Fatalf("playspec.create %s type = %#v, want object", field, prop["type"])
		}
	}
	triggerProps := playspecCreateProps["trigger_config"].(map[string]any)["properties"].(map[string]any)
	for _, field := range []string{"enabled", "event_type", "branch", "prop_id", "marquee_id", "agent_id", "max_retries", "prompt_template"} {
		if _, ok := triggerProps[field]; !ok {
			t.Fatalf("playspec.trigger_config missing %q: %#v", field, triggerProps)
		}
	}
	if _, ok := triggerProps["event_type"].(map[string]any)["enum"]; !ok {
		t.Fatalf("playspec.trigger_config.event_type enum missing: %#v", triggerProps["event_type"])
	}
	mutiProps := playspecCreateProps["muti_config"].(map[string]any)["properties"].(map[string]any)
	for _, field := range []string{"enabled", "language", "prop_id", "agent_id", "prompt_template"} {
		if _, ok := mutiProps[field]; !ok {
			t.Fatalf("playspec.muti_config missing %q: %#v", field, mutiProps)
		}
	}
	scheduleProps := playspecCreateProps["schedule_config"].(map[string]any)["properties"].(map[string]any)
	for _, field := range []string{"enabled", "cron", "marquee_id"} {
		if _, ok := scheduleProps[field]; !ok {
			t.Fatalf("playspec.schedule_config missing %q: %#v", field, scheduleProps)
		}
	}
	if _, _, err := ValidateMutationPayload("playspec", "create", map[string]any{
		"name":              "ci",
		"base_compose_yaml": "services:\n  job:\n    image: alpine\n",
		"job_mode":          true,
		"schedule_config": map[string]any{
			"enabled":    true,
			"cron":       "every 5 minutes",
			"marquee_id": "runner",
		},
		"trigger_config": map[string]any{
			"enabled":         true,
			"event_type":      "push",
			"branch":          "main",
			"prop_id":         "api",
			"marquee_id":      "runner",
			"agent_id":        "fixer",
			"max_retries":     2,
			"prompt_template": "Fix {{logs}}",
		},
		"muti_config": map[string]any{
			"enabled":         true,
			"language":        "ruby",
			"prop_id":         "api",
			"agent_id":        "fixer",
			"prompt_template": "Fix {{diff}}",
		},
	}); err != nil {
		t.Fatalf("playspec.create config payload should validate: %v", err)
	}

	catalog := Catalog()
	foundAgent := false
	foundAgentPoke := false
	foundTemplateVersion := false
	for _, entry := range catalog {
		if entry.Name == "playground" && !containsSelector(entry.Operations, "action") {
			t.Fatalf("playground catalog operations missing action: %#v", entry.Operations)
		}
		if entry.Name == "agent" {
			foundAgent = true
			if !containsSelector(entry.Operations, "create") ||
				!containsSelector(entry.Operations, "update") ||
				!containsSelector(entry.Operations, "restart_chat") ||
				!containsSelector(entry.Operations, "upload_attachment") ||
				!containsSelector(entry.Operations, "watch") {
				t.Fatalf("agent catalog operations missing expected operations: %#v", entry.Operations)
			}
		}
		if entry.Name == "agent_poke" {
			foundAgentPoke = true
			if !containsSelector(entry.Operations, "create") ||
				!containsSelector(entry.Operations, "update") ||
				!containsSelector(entry.Operations, "list") ||
				!containsSelector(entry.Operations, "get") ||
				!containsSelector(entry.Operations, "delete") {
				t.Fatalf("agent_poke catalog operations missing expected operations: %#v", entry.Operations)
			}
		}
		if entry.Name == "template_version" {
			foundTemplateVersion = true
			if !containsSelector(entry.Operations, "create") || !containsSelector(entry.Operations, "list") || !containsSelector(entry.Operations, "delete") {
				t.Fatalf("template_version catalog operations missing create/list/delete: %#v", entry.Operations)
			}
		}
	}
	if !foundAgent {
		t.Fatal("agent missing from catalog")
	}
	if !foundAgentPoke {
		t.Fatal("agent_poke missing from catalog")
	}
	if !foundTemplateVersion {
		t.Fatal("template_version missing from catalog")
	}
}

func TestRegistryCoversScopedMutationActionSchemas(t *testing.T) {
	for _, tc := range []struct {
		resource  string
		operation string
		fields    []string
	}{
		{resource: "agent", operation: "restart_chat", fields: []string{"id_or_name"}},
		{resource: "agent", operation: "upload_attachment", fields: []string{"id_or_name", "content_path", "content_base64", "filename", "conversation_id"}},
		{resource: "agent_poke", operation: "create", fields: []string{"agent_id_or_name", "schedule", "prompt", "conversation_id", "enabled"}},
		{resource: "agent_poke", operation: "update", fields: []string{"agent_id_or_name", "poke_id", "schedule", "prompt", "conversation_id", "enabled"}},
		{resource: "marquee", operation: "autoconnect_token", fields: []string{"email", "domain", "ip", "ssl_mode", "dns_provider", "dns_credentials"}},
		{resource: "marquee", operation: "generate_ssh_key", fields: []string{"id_or_name"}},
		{resource: "marquee", operation: "test_connection", fields: []string{"id_or_name"}},
		{resource: "prop", operation: "attach", fields: []string{"repo_full_name"}},
		{resource: "prop", operation: "mirror", fields: []string{"source_url", "name"}},
		{resource: "prop", operation: "sync", fields: []string{"id_or_name"}},
		{resource: "template", operation: "fork", fields: []string{"id_or_name"}},
		{resource: "template", operation: "source_refresh", fields: []string{"id_or_name"}},
		{resource: "template", operation: "source_set", fields: []string{"template_id_or_name", "source_prop_id_or_name", "source_path", "source_ref", "source_auto_refresh", "source_auto_upgrade", "ci_enabled", "ci_marquee_id_or_name"}},
		{resource: "template", operation: "upgrade_playspecs", fields: []string{"template_id_or_name", "version_id"}},
		{resource: "template_version", operation: "toggle_public", fields: []string{"template_id_or_name", "version_id"}},
		{resource: "trick", operation: "trigger", fields: []string{"playspec_id_or_name", "marquee_id_or_name", "name"}},
		{resource: "trick", operation: "rerun", fields: []string{"id_or_name"}},
		{resource: "webhook", operation: "test", fields: []string{"webhook_id"}},
	} {
		schema, _, op, ok := SchemaFor(tc.resource, tc.operation)
		if !ok || op != tc.operation {
			t.Fatalf("%s.%s schema missing", tc.resource, tc.operation)
		}
		props := schema.(map[string]any)["properties"].(map[string]any)
		for _, field := range tc.fields {
			prop, ok := props[field].(map[string]any)
			if !ok {
				t.Fatalf("%s.%s missing property %q: %#v", tc.resource, tc.operation, field, props)
			}
			if prop["description"] == "" {
				t.Fatalf("%s.%s.%s missing description: %#v", tc.resource, tc.operation, field, prop)
			}
			if schemaFieldIsNumericID(field, prop) {
				min, ok := numericMinimum(prop["minimum"])
				if !ok || min < 1 {
					t.Fatalf("%s.%s.%s missing minimum >= 1: %#v", tc.resource, tc.operation, field, prop)
				}
			}
		}
	}

	if _, _, err := ValidateMutationPayload("template", "source_set", map[string]any{"template_id_or_name": 1, "source_prop_id_or_name": 2, "source_path": "template.yml"}); err != nil {
		t.Fatalf("template.source_set should validate: %v", err)
	}
	if _, _, err := ValidateMutationPayload("template", "source_set", map[string]any{"template_id_or_name": 0, "source_prop_id_or_name": 2, "source_path": "template.yml"}); err == nil {
		t.Fatal("template.source_set should reject non-positive template_id_or_name")
	}
	if _, _, err := ValidateMutationPayload("marquee", "autoconnect_token", map[string]any{"ssl_mode": "bogus"}); err == nil {
		t.Fatal("marquee.autoconnect_token should reject unsupported ssl_mode")
	}
	if _, _, err := ValidateMutationPayload("agent", "upload_attachment", map[string]any{"id_or_name": "builder", "content_base64": "aGVsbG8=", "filename": "hello.txt"}); err != nil {
		t.Fatalf("agent.upload_attachment should validate: %v", err)
	}
	if _, _, err := ValidatePayload("agent_attachment", "get", map[string]any{"resource": "agent_attachment", "agent_id_or_name": "builder", "filename": "hello.txt"}); err != nil {
		t.Fatalf("agent_attachment.get should validate: %v", err)
	}
	if _, _, err := ValidatePayload("agent_poke", "list", map[string]any{"resource": "agent_poke", "params": map[string]any{"agent_id_or_name": "builder"}}); err != nil {
		t.Fatalf("agent_poke.list should validate: %v", err)
	}
	if _, _, err := ValidatePayload("agent_poke", "get", map[string]any{"resource": "agent_poke", "agent_id_or_name": "builder", "id": 1}); err != nil {
		t.Fatalf("agent_poke.get should validate: %v", err)
	}
}

func TestMutationToolSchemaIsCompactAndRuntimeValidated(t *testing.T) {
	schema := MutationToolInputSchema()
	if _, ok := schema["oneOf"]; ok {
		t.Fatalf("mutation tool schema should not embed oneOf payload variants: %#v", schema)
	}
	props := schema["properties"].(map[string]any)
	resourceEnum := props["resource"].(map[string]any)["enum"].([]any)
	if !containsAnySelector(resourceEnum, "template-version") {
		t.Fatalf("mutation resource enum missing template-version: %#v", resourceEnum)
	}
	operationEnum := props["operation"].(map[string]any)["enum"].([]any)
	if containsAnySelector(operationEnum, "patch_create") {
		t.Fatalf("mutation operation enum should not include patch_create: %#v", operationEnum)
	}
	for _, want := range []string{"action", "autoconnect_token", "restart_chat", "source_set", "toggle_public", "trigger", "test", "upload_attachment"} {
		if !containsAnySelector(operationEnum, want) {
			t.Fatalf("mutation operation enum missing %q: %#v", want, operationEnum)
		}
	}
	if _, ok := props["dry_run"].(map[string]any); !ok {
		t.Fatalf("mutation tool schema missing dry_run: %#v", props)
	}
	if _, ok := props["confirm"].(map[string]any); !ok {
		t.Fatalf("mutation tool schema missing confirm: %#v", props)
	}

	if _, _, err := ValidateMutationPayload("agent", "update", map[string]any{"id_or_name": 1}); err == nil {
		t.Fatal("expected empty update payload to be rejected")
	}
	if _, _, err := ValidateMutationPayload("job_env", "create", map[string]any{"key": "SERVICES_ONLY", "value": ""}); err != nil {
		t.Fatalf("job_env.create should allow an empty value: %v", err)
	}
	if _, _, err := ValidateMutationPayload("job_env", "create", map[string]any{"key": "SERVICES_ONLY"}); err == nil {
		t.Fatal("job_env.create should require the value field to be present")
	}
	if _, _, err := ValidateMutationPayload("job_env", "update", map[string]any{"job_env_id": 1, "value": ""}); err != nil {
		t.Fatalf("job_env.update should allow an empty value update: %v", err)
	}
	if _, _, err := ValidateMutationPayload("api_key", "update", map[string]any{"api_key_id": 1, "label": "x"}); err == nil {
		t.Fatal("expected unsupported operation to be rejected")
	}
	if _, _, err := ValidateMutationPayload("template_version", "create", map[string]any{"template_id_or_name": 1, "template_body_path": "/tmp/template.yml"}); err != nil {
		t.Fatalf("template_version.create with template_body_path should validate: %v", err)
	}
}

func TestNamedResourceMutationSchemasDoNotExposeLegacyTargetAliases(t *testing.T) {
	disallowed := []string{
		"agent_id",
		"playground_id",
		"playground_identifier",
		"template_id",
		"secret_id",
		"trick_id",
		"playspec_id",
		"prop_id",
		"marquee_id",
		"source_prop_id",
		"ci_marquee_id",
		"target_playspec_id",
		"target_playground_id",
		"build_in_public_playground_id",
		"target_id",
	}
	for _, tc := range []struct {
		resource  string
		operation string
	}{
		{resource: "agent", operation: "create"},
		{resource: "agent", operation: "update"},
		{resource: "agent", operation: "restart_chat"},
		{resource: "agent", operation: "upload_attachment"},
		{resource: "agent_poke", operation: "create"},
		{resource: "agent_poke", operation: "update"},
		{resource: "mutter", operation: "create"},
		{resource: "playground", operation: "create"},
		{resource: "playground", operation: "update"},
		{resource: "playground", operation: "action"},
		{resource: "playground", operation: "switch_template"},
		{resource: "playspec", operation: "update"},
		{resource: "prop", operation: "update"},
		{resource: "prop", operation: "sync"},
		{resource: "marquee", operation: "update"},
		{resource: "marquee", operation: "generate_ssh_key"},
		{resource: "marquee", operation: "test_connection"},
		{resource: "secret", operation: "update"},
		{resource: "template", operation: "update"},
		{resource: "template", operation: "change"},
		{resource: "template", operation: "fork"},
		{resource: "template", operation: "source_refresh"},
		{resource: "template", operation: "source_set"},
		{resource: "template", operation: "upgrade_playspecs"},
		{resource: "template_version", operation: "create"},
		{resource: "template_version", operation: "toggle_public"},
		{resource: "trick", operation: "trigger"},
		{resource: "trick", operation: "rerun"},
	} {
		schema, _, _, ok := SchemaFor(tc.resource, tc.operation)
		if !ok {
			t.Fatalf("%s.%s schema missing", tc.resource, tc.operation)
		}
		props := schema.(map[string]any)["properties"].(map[string]any)
		for _, field := range disallowed {
			if _, ok := props[field]; ok {
				t.Fatalf("%s.%s exposes legacy field %q: %#v", tc.resource, tc.operation, field, props)
			}
		}
	}
}

func containsSelector(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func containsAnySelector(values []any, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
