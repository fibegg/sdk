package mcpserver

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCompanyRepositoryMutationInstallationSelection(t *testing.T) {
	calls := 0
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		var payload map[string]any
		json.NewDecoder(r.Body).Decode(&payload)
		if payload["github_installation_id"] != float64(501) {
			t.Errorf("selected installation lost: %+v", payload)
		}
		json.NewEncoder(w).Encode(map[string]any{"id": 7, "status": "active", "owner_type": "Team", "owner_id": 9})
	}))
	defer api.Close()
	server := New(Config{APIKey: "fixture", Domain: api.URL, ToolSet: "core"})
	if err := server.RegisterAll(); err != nil {
		t.Fatal(err)
	}
	for _, operation := range []string{"attach", "mirror"} {
		payload := map[string]any{"github_installation_id": 501}
		if operation == "attach" {
			payload["repo_full_name"] = "company-org/existing"
		} else {
			payload["source_url"] = "https://github.com/upstream/source"
		}
		_, err := server.dispatcher.dispatch(context.Background(), "fibe_resource_mutate", map[string]any{"resource": "repository", "operation": operation, "payload": payload})
		if err != nil {
			t.Fatalf("%s dispatch: %v", operation, err)
		}
	}
	if calls != 2 {
		t.Fatalf("expected two selected requests, got %d", calls)
	}
	_, err := server.dispatcher.dispatch(context.Background(), "fibe_resource_mutate", map[string]any{"resource": "repository", "operation": "attach", "payload": map[string]any{"repo_full_name": "company-org/existing", "github_installation_id": 0}})
	if err == nil || calls != 2 {
		t.Fatal("invalid installation reached the API")
	}
}
