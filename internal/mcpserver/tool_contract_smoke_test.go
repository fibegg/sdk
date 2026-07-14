package mcpserver

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/fibegg/sdk/internal/resourceschema"
)

// TestToolContractSmoke invokes every registered MCP handler from its public
// schema. Exact schemas and ordering are covered by TestMCPToolContract; this
// test prevents a schema from drifting so far from its handler that binding
// panics or the registered handler becomes unreachable.
func TestToolContractSmoke(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		_, _ = io.Copy(io.Discard, r.Body)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data":       []any{},
			"meta":       map[string]any{"page": 1, "per_page": 25, "total": 0},
			"status":     "success",
			"request_id": "mcp-contract-smoke",
			"result":     map[string]any{},
		})
	}))
	defer api.Close()

	server := New(Config{
		APIKey:                 "contract-fixture",
		Domain:                 api.URL,
		ToolSet:                "full",
		Yolo:                   true,
		PipelineCacheSize:      8,
		PipelineCacheEntryMax:  1 << 20,
		PipelineMaxSteps:       25,
		PipelineMaxIterations:  50,
		PipelineMaxDepth:       8,
		PipelineMaxConcurrency: 8,
	})
	if err := server.RegisterAll(); err != nil {
		t.Fatal(err)
	}
	server.httpMode = true

	names := server.dispatcher.names()
	if len(names) != 60 {
		t.Fatalf("registered tools = %d, want frozen contract count 60", len(names))
	}
	for _, name := range names {
		name := name
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 25*time.Millisecond)
			defer cancel()
			args := schemaFixtureObject(server.toolSchemas[name], api.URL)
			args["confirm"] = true

			defer func() {
				if recovered := recover(); recovered != nil {
					t.Fatalf("registered MCP handler panicked for schema-derived input: %v", recovered)
				}
			}()
			_, _ = server.dispatcher.dispatch(ctx, name, args)
		})
	}
}

func TestAllResourceMutationHandlers(t *testing.T) {
	var requests atomic.Int64
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		defer r.Body.Close()
		_, _ = io.Copy(io.Discard, r.Body)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":                         1,
			"data":                       []any{},
			"meta":                       map[string]any{"page": 1, "per_page": 25, "total": 0},
			"status":                     "success",
			"request_id":                 "mutation-contract-smoke",
			"latest_version_id":          1,
			"source_template_version_id": 1,
			"source_template":            map[string]any{"id": 1},
			"playspec_id":                1,
			"marquee_id":                 1,
			"job_mode":                   true,
			"template":                   map[string]any{"id": 1},
			"version":                    map[string]any{"id": 1, "template_id": 1},
			"result":                     map[string]any{},
		})
	}))
	defer api.Close()

	server := New(Config{APIKey: "contract-fixture", Domain: api.URL, ToolSet: "full", Yolo: true})
	if err := server.RegisterAll(); err != nil {
		t.Fatal(err)
	}
	client, err := server.resolveClient(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	cases := resourceschema.MutationCases()
	if len(cases) != 41 {
		t.Fatalf("mutation cases = %d, want frozen contract count 41", len(cases))
	}
	for _, mutation := range cases {
		mutation := mutation
		t.Run(mutation.Resource+"/"+mutation.Operation, func(t *testing.T) {
			payload := schemaFixtureObject(mutation.Schema, api.URL)
			completeMutationFixture(mutation.Resource, mutation.Operation, payload)
			canonicalResource, canonicalOperation, err := resourceschema.ValidateMutationPayload(mutation.Resource, mutation.Operation, payload)
			if err != nil {
				t.Fatalf("schema-derived payload is invalid: %v\npayload: %#v", err, payload)
			}

			before := requests.Load()
			_, callErr := dispatchResourceMutation(context.Background(), client, canonicalResource, canonicalOperation, payload)
			if requests.Load() == before {
				t.Fatalf("handler returned before reaching the API: %v\npayload: %#v", callErr, payload)
			}
		})
	}
}

func TestAllFlatResourceHandlers(t *testing.T) {
	var requests atomic.Int64
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		defer r.Body.Close()
		_, _ = io.Copy(io.Discard, r.Body)
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Disposition", `attachment; filename="fixture.txt"`)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":     1,
			"data":   []any{},
			"meta":   map[string]any{"page": 1, "per_page": 25, "total": 0},
			"status": "success",
		})
	}))
	defer api.Close()

	server := New(Config{APIKey: "contract-fixture", Domain: api.URL, ToolSet: "full", Yolo: true})
	client, err := server.resolveClient(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	resources := flatResourceTools()
	if len(resources) != 21 {
		t.Fatalf("flat resources = %d, want frozen contract count 21", len(resources))
	}
	for name, tool := range resources {
		name, tool := name, tool
		if tool.list != nil {
			t.Run(name+"/list", func(t *testing.T) {
				args := map[string]any{}
				switch name {
				case "agent_poke":
					args["agent_id_or_name"] = "fixture"
				case "template_version":
					args["template_id_or_name"] = "fixture"
				case "webhook_delivery":
					args["webhook_id"] = float64(1)
				}
				before := requests.Load()
				_, callErr := tool.list(context.Background(), client, args)
				if requests.Load() == before {
					t.Fatalf("list handler returned before reaching the API: %v", callErr)
				}
			})
		}
		if tool.get != nil || tool.getWithArgs != nil {
			t.Run(name+"/get", func(t *testing.T) {
				before := requests.Load()
				var callErr error
				if tool.getWithArgs != nil {
					args := map[string]any{"agent_id_or_name": "fixture", "id": float64(1), "filename": "fixture.txt"}
					_, callErr = tool.getWithArgs(context.Background(), client, args)
				} else {
					_, callErr = tool.get(context.Background(), client, "1")
				}
				if requests.Load() == before {
					t.Fatalf("get handler returned before reaching the API: %v", callErr)
				}
			})
		}
		if tool.delete != nil || tool.deleteWithArgs != nil {
			t.Run(name+"/delete", func(t *testing.T) {
				before := requests.Load()
				var callErr error
				if tool.deleteWithArgs != nil {
					args := map[string]any{"agent_id_or_name": "fixture", "template_id_or_name": "fixture", "id": float64(1)}
					callErr = tool.deleteWithArgs(context.Background(), client, args)
				} else {
					callErr = tool.delete(context.Background(), client, "1")
				}
				if requests.Load() == before {
					t.Fatalf("delete handler returned before reaching the API: %v", callErr)
				}
			})
		}
	}
}

func completeMutationFixture(resource, operation string, payload map[string]any) {
	if operation == "update" {
		switch resource {
		case "agent_poke":
			payload["prompt"] = "fixture"
		case "agent", "marquee", "playground", "prop":
			payload["name"] = "fixture-updated"
		case "playspec", "template", "webhook":
			payload["description"] = "fixture-updated"
		case "secret", "job_env":
			payload["value"] = "fixture-updated"
		}
	}
	if resource == "agent" && operation == "upload_attachment" {
		payload["id_or_name"] = 1
		payload["content_base64"] = "Zml4dHVyZQ=="
		delete(payload, "content_path")
	}
	if resource == "template" && operation == "change" {
		payload["patches"] = []any{map[string]any{"op": "replace", "path": "/name", "value": "fixture"}}
	}
}

func schemaFixtureObject(schema map[string]any, domain string) map[string]any {
	value := schemaFixtureValue(schema, "", domain, 0)
	object, _ := value.(map[string]any)
	if object == nil {
		object = map[string]any{}
	}
	return object
}

func schemaFixtureValue(schema map[string]any, fieldName, domain string, depth int) any {
	if depth > 8 {
		return nil
	}
	if _, hasType := schema["type"]; !hasType && schema["properties"] == nil && schema["required"] == nil {
		for _, keyword := range []string{"oneOf", "anyOf"} {
			variants, _ := schema[keyword].([]any)
			if len(variants) > 0 {
				if selected, ok := variants[0].(map[string]any); ok {
					return schemaFixtureValue(selected, fieldName, domain, depth+1)
				}
			}
		}
	}
	switch enum := schema["enum"].(type) {
	case []any:
		if len(enum) > 0 {
			return enum[0]
		}
	case []string:
		if len(enum) > 0 {
			return enum[0]
		}
	}

	typeName, _ := schema["type"].(string)
	switch typeName {
	case "object", "":
		object := map[string]any{}
		properties, _ := schema["properties"].(map[string]any)
		required := schemaStringSet(schema["required"])
		for _, keyword := range []string{"oneOf", "anyOf"} {
			variants, _ := schema[keyword].([]any)
			if len(variants) == 0 {
				continue
			}
			selected, _ := variants[0].(map[string]any)
			for name := range schemaStringSet(selected["required"]) {
				required[name] = struct{}{}
			}
			if selectedProperties, ok := selected["properties"].(map[string]any); ok {
				if properties == nil {
					properties = map[string]any{}
				}
				for name, property := range selectedProperties {
					properties[name] = property
				}
			}
		}
		for name := range required {
			property, _ := properties[name].(map[string]any)
			object[name] = schemaFixtureValue(property, name, domain, depth+1)
		}
		return object
	case "array":
		items, _ := schema["items"].(map[string]any)
		return []any{schemaFixtureValue(items, fieldName, domain, depth+1)}
	case "boolean":
		return false
	case "integer", "number":
		return float64(1)
	case "string":
		switch strings.ToLower(fieldName) {
		case "api_key":
			return "contract-fixture"
		case "base_compose_yaml", "compose_yaml", "template_body":
			return "services: {}"
		case "content_base64":
			return "Zml4dHVyZQ=="
		case "domain", "base_url":
			return domain
		case "url":
			return "https://example.com/fixture"
		case "key":
			return "FIXTURE_KEY"
		case "repo_full_name":
			return "example/fixture"
		case "schedule":
			return "0 * * * *"
		case "tool":
			return "fibe_status"
		case "status":
			return "running"
		case "action_type", "action":
			return "stop"
		case "repository_url", "repo", "github_urls":
			return "https://github.com/example/fixture"
		case "view":
			return "summary"
		default:
			return "fixture"
		}
	default:
		return nil
	}
}

func schemaStringSet(value any) map[string]struct{} {
	result := map[string]struct{}{}
	switch items := value.(type) {
	case []any:
		for _, item := range items {
			if text, ok := item.(string); ok {
				result[text] = struct{}{}
			}
		}
	case []string:
		for _, text := range items {
			result[text] = struct{}{}
		}
	}
	return result
}
