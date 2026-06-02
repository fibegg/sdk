package mcpserver

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/fibegg/sdk/fibe"
)

func TestPlaygroundsLogsAllowsOmittedService(t *testing.T) {
	var body map[string]any
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/playgrounds/demo/logs" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode logs body: %v", err)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"service": nil,
			"lines":   []string{"[web] ready", "[worker] done"},
			"source":  "live",
			"entries": []map[string]any{
				{"service": "web", "line": "ready", "source": "live"},
				{"service": "worker", "line": "done", "source": "live"},
			},
		})
	}))
	defer api.Close()

	srv := New(Config{APIKey: "pk_test", Domain: api.URL, ToolSet: "core"})
	if err := srv.RegisterAll(); err != nil {
		t.Fatalf("RegisterAll: %v", err)
	}

	out, err := srv.dispatcher.dispatch(context.Background(), "fibe_playgrounds_logs", map[string]any{
		"id_or_name": "demo",
		"tail":       25,
	})
	if err != nil {
		t.Fatalf("playgrounds logs dispatch: %v", err)
	}
	if _, ok := body["service"]; ok {
		t.Fatalf("all-service logs should omit service: %#v", body)
	}
	if body["tail"] != float64(25) {
		t.Fatalf("unexpected logs body: %#v", body)
	}
	logs := out.(*fibe.PlaygroundLogs)
	if logs.Service != "" || len(logs.Entries) != 2 || logs.Entries[0].Service != "web" {
		t.Fatalf("unexpected all-service logs result: %#v", logs)
	}
}

func TestPlaygroundsLogsKeepsServiceSpecificBehavior(t *testing.T) {
	var body map[string]any
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode logs body: %v", err)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"service": "web",
			"lines":   []string{"ready"},
			"source":  "live",
		})
	}))
	defer api.Close()

	srv := New(Config{APIKey: "pk_test", Domain: api.URL, ToolSet: "core"})
	if err := srv.RegisterAll(); err != nil {
		t.Fatalf("RegisterAll: %v", err)
	}

	out, err := srv.dispatcher.dispatch(context.Background(), "fibe_playgrounds_logs", map[string]any{
		"id_or_name": "demo",
		"service":    "web",
	})
	if err != nil {
		t.Fatalf("playgrounds logs dispatch: %v", err)
	}
	if body["service"] != "web" {
		t.Fatalf("expected service-specific body, got %#v", body)
	}
	logs := out.(*fibe.PlaygroundLogs)
	if logs.Service != "web" || len(logs.Lines) != 1 {
		t.Fatalf("unexpected service-specific logs result: %#v", logs)
	}
}

func TestPlaygroundsLogsSchemaServiceIsOptional(t *testing.T) {
	srv := New(Config{APIKey: "pk_test", ToolSet: "core"})
	if err := srv.RegisterAll(); err != nil {
		t.Fatalf("RegisterAll: %v", err)
	}

	schema := srv.toolSchemas["fibe_playgrounds_logs"]
	props := schema["properties"].(map[string]any)
	if _, ok := props["service"]; !ok {
		t.Fatalf("fibe_playgrounds_logs schema missing service: %#v", schema)
	}
	required, _ := schema["required"].([]any)
	if containsAnyString(required, "service") {
		t.Fatalf("fibe_playgrounds_logs should not require service: %#v", schema)
	}
	if !containsAnyString(required, "id_or_name") {
		t.Fatalf("fibe_playgrounds_logs should require id_or_name: %#v", schema)
	}

	out, err := srv.dispatcher.dispatch(context.Background(), "fibe_tools_catalog", map[string]any{
		"tier":         "all",
		"name_pattern": "playgrounds_logs",
	})
	if err != nil {
		t.Fatalf("catalog dispatch: %v", err)
	}
	tool := catalogTool(catalogToolsFromResult(t, out), "fibe_playgrounds_logs")
	if tool == nil {
		t.Fatalf("catalog missing fibe_playgrounds_logs")
	}
	desc, _ := tool["description"].(string)
	if !strings.Contains(desc, "Omitting service returns all services") {
		t.Fatalf("catalog description should mention all-service default: %#v", tool)
	}
}
