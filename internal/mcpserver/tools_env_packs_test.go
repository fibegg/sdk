package mcpserver

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestEnvPackMCPDiscoveryAndMutation(t *testing.T) {
	hits := 0
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		if r.URL.Path != "/api/playgrounds/7/env_pack_attachments" {
			t.Errorf("path %s", r.URL.Path)
		}
		if r.Method == http.MethodPatch {
			var p map[string]any
			if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
				t.Fatal(err)
			}
			if len(p["env_pack_attachments"].([]any)) != 0 {
				t.Fatal("detach list changed")
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":7,"env_pack_attachments":[],"env_packs":{"pending":true}}`))
	}))
	defer api.Close()
	srv := New(Config{APIKey: "fixture", Domain: api.URL, ToolSet: "core"})
	if err := srv.RegisterAll(); err != nil {
		t.Fatal(err)
	}
	if _, err := srv.dispatcher.dispatch(context.Background(), "fibe_env_pack_attachments_get", map[string]any{"target": "playgrounds", "target_id": 7}); err != nil {
		t.Fatal(err)
	}
	if _, err := srv.dispatcher.dispatch(context.Background(), "fibe_resource_mutate", map[string]any{"resource": "env_packs", "operation": "attachments_replace", "payload": map[string]any{"target": "playgrounds", "target_id": 7, "env_pack_attachments": []any{}}}); err != nil {
		t.Fatal(err)
	}
	if hits != 2 {
		t.Fatalf("hits %d", hits)
	}
	if _, err := srv.dispatcher.dispatch(context.Background(), "fibe_resource_mutate", map[string]any{"resource": "env_pack", "operation": "attachments_replace", "payload": map[string]any{"target": "playgrounds", "target_id": 7, "env_pack_attachments": []any{map[string]any{"env_pack_id": 1, "grantor_id": 2}}}}); err == nil {
		t.Fatal("forged grant accepted")
	}
	if hits != 2 {
		t.Fatal("invalid request reached server")
	}
}

func TestEnvPackMCPLaunchCarriesSelectionAcrossSourceKinds(t *testing.T) {
	var bodies []map[string]any
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Fatalf("unexpected method %s", r.Method)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		bodies = append(bodies, body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":8,"spec_id":7,"playground_id":8,"playground":{"id":8}}`))
	}))
	defer api.Close()
	srv := New(Config{APIKey: "fixture", Domain: api.URL, ToolSet: "full"})
	if err := srv.RegisterAll(); err != nil {
		t.Fatal(err)
	}
	sourceCases := []map[string]any{
		{"compose_yaml": "services:\n  web:\n    image: alpine\n", "name": "fixture", "create_playground": false},
		{"template_id_or_name": "12", "host_id_or_name": "21"},
		{"template_version_id": float64(22), "host_id_or_name": "21", "name": "fixture"},
		{"spec_id_or_name": "7", "host_id_or_name": "21", "name": "fixture"},
	}
	for _, args := range sourceCases {
		args["env_pack_attachments"] = []any{}
		if _, err := srv.dispatcher.dispatch(context.Background(), "fibe_launch", args); err != nil {
			t.Fatal(err)
		}
	}
	if len(bodies) != 4 {
		t.Fatalf("requests %d", len(bodies))
	}
	for _, body := range bodies {
		// Spec creation uses the normal Playground envelope.
		if nested, ok := body["playground"].(map[string]any); ok {
			body = nested
		}
		rows, ok := body["env_pack_attachments"].([]any)
		if !ok || len(rows) != 0 {
			t.Fatal("explicit opt-out lost")
		}
	}
	tool := srv.mcp.GetTool("fibe_launch")
	var schema map[string]any
	if err := json.Unmarshal(tool.Tool.RawInputSchema, &schema); err != nil {
		t.Fatal(err)
	}
	if _, exists := schema["properties"].(map[string]any)["env_pack_attachments"]; !exists {
		t.Fatal("launch schema omitted recipient selection")
	}
	forged := map[string]any{"compose_yaml": "services: {}", "name": "forged", "env_pack_attachments": []any{map[string]any{"env_pack_id": 1, "grantor_id": 2}}}
	if _, err := srv.dispatcher.dispatch(context.Background(), "fibe_launch", forged); err == nil {
		t.Fatal("forged launch grant accepted")
	}
	if len(bodies) != 4 {
		t.Fatal("invalid grant reached server")
	}
}
