package resourceschema

import "testing"

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

func TestResourceSelectorsIncludeHyphenAliases(t *testing.T) {
	selectors := ResourceSelectors()
	for _, want := range []string{"api-keys", "job-env", "import-templates"} {
		if !containsSelector(selectors, want) {
			t.Fatalf("ResourceSelectors missing %q: %#v", want, selectors)
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

	artefactCreate, _, op, ok := SchemaFor("artefacts", "create")
	if !ok || op != "create" {
		t.Fatalf("artefact.create schema missing")
	}
	artefactProps := artefactCreate.(map[string]any)["properties"].(map[string]any)
	for _, want := range []string{"agent_id", "name", "filename", "content_base64", "content_path", "description", "playground_id"} {
		if _, ok := artefactProps[want]; !ok {
			t.Fatalf("artefact.create missing property %q: %#v", want, artefactProps)
		}
	}

	mutterCreate, _, op, ok := SchemaFor("mutter", "create")
	if !ok || op != "create" {
		t.Fatalf("mutter.create schema missing")
	}
	mutterProps := mutterCreate.(map[string]any)["properties"].(map[string]any)
	for _, want := range []string{"agent_id", "type", "body", "playground_id"} {
		if _, ok := mutterProps[want]; !ok {
			t.Fatalf("mutter.create missing property %q: %#v", want, mutterProps)
		}
	}

	templateVersionCreate, _, op, ok := SchemaFor("template-version", "create")
	if !ok || op != "create" {
		t.Fatalf("template_version.create schema missing")
	}
	versionCreateProps := templateVersionCreate.(map[string]any)["properties"].(map[string]any)
	for _, want := range []string{"template_id", "template_body", "template_body_path", "public", "response_mode"} {
		if _, ok := versionCreateProps[want]; !ok {
			t.Fatalf("template_version.create missing property %q: %#v", want, versionCreateProps)
		}
	}

	templateUpdate, _, op, ok := SchemaFor("template", "update")
	if !ok || op != "update" {
		t.Fatalf("template.update schema missing")
	}
	templateProps := templateUpdate.(map[string]any)["properties"].(map[string]any)
	for _, want := range []string{"template_id", "name", "description", "category_id", "filename", "image_data", "content_base64", "content_path", "content_type"} {
		if _, ok := templateProps[want]; !ok {
			t.Fatalf("template.update missing property %q: %#v", want, templateProps)
		}
	}
	if _, ok := templateProps["content_type"].(map[string]any)["enum"]; !ok {
		t.Fatalf("template.update content_type enum missing: %#v", templateProps["content_type"])
	}

	templateDevelop, _, op, ok := SchemaFor("template", "develop")
	if !ok || op != "develop" {
		t.Fatalf("template.develop schema missing")
	}
	developProps := templateDevelop.(map[string]any)["properties"].(map[string]any)
	for _, want := range []string{"target_type", "target_id", "mode", "change_type", "post_apply", "template_body", "template_body_path"} {
		if _, ok := developProps[want]; !ok {
			t.Fatalf("template.develop missing property %q: %#v", want, developProps)
		}
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
	for _, want := range []string{"agent_id", "name", "mode", "model_options"} {
		if _, ok := props[want]; !ok {
			t.Fatalf("agent.update missing property %q: %#v", want, props)
		}
	}

	catalog := Catalog()
	foundAgent := false
	foundTemplateVersion := false
	for _, entry := range catalog {
		if entry.Name == "playground" && !containsSelector(entry.Operations, "action") {
			t.Fatalf("playground catalog operations missing action: %#v", entry.Operations)
		}
		if entry.Name == "agent" {
			foundAgent = true
			if !containsSelector(entry.Operations, "create") || !containsSelector(entry.Operations, "update") {
				t.Fatalf("agent catalog operations missing create/update: %#v", entry.Operations)
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
		{resource: "marquee", operation: "autoconnect_token", fields: []string{"email", "domain", "ip", "ssl_mode", "dns_provider", "dns_credentials"}},
		{resource: "marquee", operation: "generate_ssh_key", fields: []string{"marquee_id"}},
		{resource: "marquee", operation: "test_connection", fields: []string{"marquee_id"}},
		{resource: "prop", operation: "attach", fields: []string{"repo_full_name"}},
		{resource: "prop", operation: "mirror", fields: []string{"source_url", "name"}},
		{resource: "prop", operation: "sync", fields: []string{"prop_id"}},
		{resource: "template", operation: "fork", fields: []string{"template_id"}},
		{resource: "template", operation: "source_refresh", fields: []string{"template_id"}},
		{resource: "template", operation: "source_set", fields: []string{"template_id", "source_prop_id", "source_path", "source_ref", "source_auto_refresh", "source_auto_upgrade", "ci_enabled", "ci_marquee_id"}},
		{resource: "template", operation: "upgrade_playspecs", fields: []string{"template_id", "version_id"}},
		{resource: "template_version", operation: "toggle_public", fields: []string{"template_id", "version_id"}},
		{resource: "trick", operation: "trigger", fields: []string{"playspec_id", "marquee_id", "name"}},
		{resource: "trick", operation: "rerun", fields: []string{"trick_id"}},
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

	if _, _, err := ValidateMutationPayload("template", "source_set", map[string]any{"template_id": 1, "source_prop_id": 2, "source_path": "template.yml"}); err != nil {
		t.Fatalf("template.source_set should validate: %v", err)
	}
	if _, _, err := ValidateMutationPayload("template", "source_set", map[string]any{"template_id": 0, "source_prop_id": 2, "source_path": "template.yml"}); err == nil {
		t.Fatal("template.source_set should reject non-positive template_id")
	}
	if _, _, err := ValidateMutationPayload("marquee", "autoconnect_token", map[string]any{"ssl_mode": "bogus"}); err == nil {
		t.Fatal("marquee.autoconnect_token should reject unsupported ssl_mode")
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
	for _, want := range []string{"autoconnect_token", "source_set", "toggle_public", "trigger", "test"} {
		if !containsAnySelector(operationEnum, want) {
			t.Fatalf("mutation operation enum missing %q: %#v", want, operationEnum)
		}
	}
	if _, ok := props["dry_run"].(map[string]any); !ok {
		t.Fatalf("mutation tool schema missing dry_run: %#v", props)
	}

	if _, _, err := ValidateMutationPayload("agent", "update", map[string]any{"agent_id": 1}); err == nil {
		t.Fatal("expected empty update payload to be rejected")
	}
	if _, _, err := ValidateMutationPayload("api_key", "update", map[string]any{"api_key_id": 1, "label": "x"}); err == nil {
		t.Fatal("expected unsupported operation to be rejected")
	}
	if _, _, err := ValidateMutationPayload("template_version", "create", map[string]any{"template_id": 1, "template_body_path": "/tmp/template.yml"}); err != nil {
		t.Fatalf("template_version.create with template_body_path should validate: %v", err)
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
