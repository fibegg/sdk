package mcpserver

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestMutterToolCreatesAgentMutter(t *testing.T) {
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/agents/7/mutter" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body["type"] != "proof" || body["body"] != "Verified rollout completed." || body["playground_id"].(float64) != 42 {
			t.Fatalf("unexpected body: %#v", body)
		}
		_, _ = w.Write([]byte(`{"agent_id":7,"data":[{"type":"proof","body":"Verified rollout completed."}]}`))
	}))
	defer api.Close()

	srv := New(Config{APIKey: "pk_test", Domain: api.URL, ToolSet: "core"})
	if err := srv.RegisterAll(); err != nil {
		t.Fatalf("RegisterAll: %v", err)
	}

	os.Setenv("FIBE_AGENT_ID", "7")
	defer os.Unsetenv("FIBE_AGENT_ID")

	if _, err := srv.dispatcher.dispatch(context.Background(), "fibe_mutter", map[string]any{
		"type":          "proof",
		"body":          "Verified rollout completed.",
		"playground_id": 42,
	}); err != nil {
		t.Fatalf("fibe_mutter dispatch: %v", err)
	}
}

func TestMutterToolValidatesBeforeAPI(t *testing.T) {
	var calls int
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		t.Fatalf("unexpected API call: %s %s", r.Method, r.URL.Path)
	}))
	defer api.Close()

	srv := New(Config{APIKey: "pk_test", Domain: api.URL, ToolSet: "core"})
	if err := srv.RegisterAll(); err != nil {
		t.Fatalf("RegisterAll: %v", err)
	}

	os.Setenv("FIBE_AGENT_ID", "bad")
	defer os.Unsetenv("FIBE_AGENT_ID")

	_, err := srv.dispatcher.dispatch(context.Background(), "fibe_mutter", map[string]any{
		"type":     "proof",
		"body":     "bad id",
	})
	if err == nil || !strings.Contains(err.Error(), "missing or invalid") {
		t.Fatalf("expected missing or invalid environment variable error, got %v", err)
	}

	os.Setenv("FIBE_AGENT_ID", "7")

	_, err = srv.dispatcher.dispatch(context.Background(), "fibe_mutter", map[string]any{
		"type":     "proof",
		"body":     "extra",
		"extra":    true,
	})
	if err == nil || !strings.Contains(err.Error(), "unsupported field") {
		t.Fatalf("expected local unsupported field error, got %v", err)
	}

	if calls != 0 {
		t.Fatalf("invalid payloads should not hit API, got %d call(s)", calls)
	}
}

func TestMuttersGetSchemaRequiresAgentID(t *testing.T) {
	srv := New(Config{APIKey: "pk_test", ToolSet: "core"})
	if err := srv.RegisterAll(); err != nil {
		t.Fatalf("RegisterAll: %v", err)
	}

	schema := srv.toolSchemas["fibe_mutters_get"]
	props := schema["properties"].(map[string]any)
	if _, ok := props["agent_id"]; !ok {
		t.Fatalf("fibe_mutters_get schema missing agent_id: %#v", schema)
	}
	for _, bad := range []string{"PlaygroundID", "Query", "PerPage"} {
		if _, ok := props[bad]; ok {
			t.Fatalf("fibe_mutters_get schema should use snake_case, found %q in %#v", bad, schema)
		}
	}
	if playground, ok := props["playground_id"].(map[string]any); !ok {
		t.Fatalf("fibe_mutters_get playground_id should be integer with minimum: %#v", props["playground_id"])
	} else if minimum, ok := numericMinimum(playground["minimum"]); !ok || minimum < 1 {
		t.Fatalf("fibe_mutters_get playground_id should have minimum >= 1: %#v", playground)
	}
	required, _ := schema["required"].([]any)
	if !containsAnyString(required, "agent_id") {
		t.Fatalf("fibe_mutters_get schema should require agent_id: %#v", schema)
	}
	if len(required) != 1 {
		t.Fatalf("fibe_mutters_get should only require agent_id: %#v", schema)
	}
}

func TestMuttersGetUsesAgentIDAndFilters(t *testing.T) {
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/agents/7/mutter" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		q := r.URL.Query()
		for key, want := range map[string]string{
			"playground_id": "42",
			"q":             "deploy",
			"status":        "open",
			"severity":      "high",
			"page":          "2",
			"per_page":      "10",
		} {
			if got := q.Get(key); got != want {
				t.Fatalf("query %s = %q, want %q (full query %s)", key, got, want, r.URL.RawQuery)
			}
		}
		_, _ = w.Write([]byte(`{"agent_id":7,"data":[]}`))
	}))
	defer api.Close()

	srv := New(Config{APIKey: "pk_test", Domain: api.URL, ToolSet: "core"})
	if err := srv.RegisterAll(); err != nil {
		t.Fatalf("RegisterAll: %v", err)
	}

	if _, err := srv.dispatcher.dispatch(context.Background(), "fibe_mutters_get", map[string]any{
		"agent_id":      7,
		"playground_id": 42,
		"query":         "deploy",
		"status":        "open",
		"severity":      "high",
		"page":          2,
		"per_page":      10,
	}); err != nil {
		t.Fatalf("fibe_mutters_get dispatch: %v", err)
	}
}

func containsAnyString(values []any, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
