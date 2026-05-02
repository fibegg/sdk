package resourceschema

import (
	"sort"
	"strings"
)

type MutationCase struct {
	Resource  string         `json:"resource"`
	Operation string         `json:"operation"`
	Schema    map[string]any `json:"payload_schema"`
}

var mutationCaseKeys = []struct {
	resource  string
	operation string
}{
	{resource: "agent", operation: "create"},
	{resource: "agent", operation: "update"},
	{resource: "agent", operation: "restart_chat"},
	{resource: "api_key", operation: "create"},
	{resource: "artefact", operation: "create"},
	{resource: "marquee", operation: "create"},
	{resource: "marquee", operation: "update"},
	{resource: "marquee", operation: "autoconnect_token"},
	{resource: "marquee", operation: "generate_ssh_key"},
	{resource: "marquee", operation: "test_connection"},
	{resource: "playground", operation: "create"},
	{resource: "playground", operation: "update"},
	{resource: "playground", operation: "action"},
	{resource: "playspec", operation: "create"},
	{resource: "playspec", operation: "update"},
	{resource: "prop", operation: "create"},
	{resource: "prop", operation: "update"},
	{resource: "prop", operation: "attach"},
	{resource: "prop", operation: "mirror"},
	{resource: "prop", operation: "sync"},
	{resource: "secret", operation: "create"},
	{resource: "secret", operation: "update"},
	{resource: "template", operation: "create"},
	{resource: "template", operation: "update"},
	{resource: "template", operation: "fork"},
	{resource: "template", operation: "source_refresh"},
	{resource: "template", operation: "source_set"},
	{resource: "template", operation: "upgrade_playspecs"},
	{resource: "template_version", operation: "create"},
	{resource: "template_version", operation: "toggle_public"},
	{resource: "trick", operation: "rerun"},
	{resource: "trick", operation: "trigger"},
	{resource: "webhook", operation: "create"},
	{resource: "webhook", operation: "update"},
	{resource: "webhook", operation: "test"},
	{resource: "job_env", operation: "create"},
	{resource: "job_env", operation: "update"},
}

func MutationCases() []MutationCase {
	out := make([]MutationCase, 0, len(mutationCaseKeys))
	for _, key := range mutationCaseKeys {
		schema, _, _, ok := SchemaFor(key.resource, key.operation)
		if !ok {
			continue
		}
		out = append(out, MutationCase{
			Resource:  key.resource,
			Operation: key.operation,
			Schema:    cloneMap(schemaMap(schema)),
		})
	}
	return out
}

func MutationResourceSelectors() []string {
	return resourceSelectorsFor(schemaResources(), func(r resourceDef) bool {
		for _, key := range mutationCaseKeys {
			if key.resource == r.name {
				return true
			}
		}
		return false
	})
}

func MutationOperationNames() []string {
	seen := map[string]bool{}
	for _, key := range mutationCaseKeys {
		seen[key.operation] = true
	}
	out := make([]string, 0, len(seen))
	for op := range seen {
		out = append(out, op)
	}
	sort.Strings(out)
	return out
}

func MutationOperationsForResource(rawResource string) ([]string, string, bool) {
	resource, ok := CanonicalResource(rawResource)
	if !ok {
		return nil, "", false
	}
	seen := map[string]bool{}
	for _, key := range mutationCaseKeys {
		if key.resource == resource {
			seen[key.operation] = true
		}
	}
	if len(seen) == 0 {
		return nil, resource, false
	}
	out := make([]string, 0, len(seen))
	for op := range seen {
		out = append(out, op)
	}
	sort.Strings(out)
	return out, resource, true
}

func MutationSchemaFor(rawResource, rawOperation string) (map[string]any, string, string, bool) {
	resource, ok := CanonicalResource(rawResource)
	if !ok {
		return nil, "", "", false
	}
	op := NormalizeResource(rawOperation)
	for _, key := range mutationCaseKeys {
		if key.resource == resource && key.operation == op {
			schema, _, _, ok := SchemaFor(resource, op)
			if !ok {
				return nil, "", "", false
			}
			return cloneMap(schemaMap(schema)), resource, op, true
		}
	}
	return nil, resource, op, false
}

func MutationResourceNamesString() string {
	names := map[string]bool{}
	for _, key := range mutationCaseKeys {
		names[key.resource] = true
	}
	out := make([]string, 0, len(names))
	for name := range names {
		out = append(out, name)
	}
	sort.Strings(out)
	return strings.Join(out, ", ")
}

func MutationToolInputSchema() map[string]any {
	return map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"required":             []string{"resource", "operation", "payload"},
		"properties": map[string]any{
			"resource": map[string]any{
				"type":        "string",
				"enum":        stringSliceToAny(MutationResourceSelectors()),
				"description": "Resource to mutate. Accepts canonical singular snake_case names and explicit aliases.",
			},
			"operation": map[string]any{
				"type":        "string",
				"enum":        stringSliceToAny(MutationOperationNames()),
				"description": "Mutation operation. Supported combinations are listed in fibe_schema(resource:list) and enforced by runtime validation.",
			},
			"payload": map[string]any{
				"type":        "object",
				"description": "Operation-specific payload. Validate shape with fibe_schema(resource:<name>, operation:<operation>); the server enforces that schema before any API request.",
			},
			"dry_run": map[string]any{
				"type":        "boolean",
				"description": "Validate the payload against fibe_schema and return without sending any API request.",
			},
			"confirm": map[string]any{
				"type":        "boolean",
				"description": "Set true for destructive routed operations such as playground.action unless the server runs with --yolo.",
			},
		},
	}
}
