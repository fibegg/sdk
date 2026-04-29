package fibe

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

func TestAgentDefaultsServiceGetAndUpdate(t *testing.T) {
	var gotMethod string
	var gotPath string
	var gotBody map[string]any

	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		if r.Method == http.MethodPatch {
			if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
				t.Fatalf("decode body: %v", err)
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"agent_defaults": map[string]any{
				"system_prompt": "profile prompt",
				"provider_overrides": map[string]any{
					"gemini": map[string]any{"model_options": "gemini-pro"},
				},
			},
		})
	})

	payload, err := c.AgentDefaults.Get(context.Background())
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if gotMethod != http.MethodGet || gotPath != "/api/agent_defaults" {
		t.Fatalf("Get sent %s %s", gotMethod, gotPath)
	}
	if payload.AgentDefaults["system_prompt"] != "profile prompt" {
		t.Fatalf("unexpected defaults: %#v", payload.AgentDefaults)
	}

	_, err = c.AgentDefaults.Update(context.Background(), AgentDefaults{
		"custom_env": "SDK_ENV=true",
		"skill_toggles": map[string]bool{
			"fibe-hunks.md": false,
		},
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if gotMethod != http.MethodPatch || gotPath != "/api/agent_defaults" {
		t.Fatalf("Update sent %s %s", gotMethod, gotPath)
	}
	if gotBody["agent_defaults"] == nil {
		t.Fatalf("Update body missing agent_defaults: %#v", gotBody)
	}

	_, err = c.AgentDefaults.Reset(context.Background())
	if err != nil {
		t.Fatalf("Reset: %v", err)
	}
	if gotMethod != http.MethodDelete || gotPath != "/api/agent_defaults" {
		t.Fatalf("Reset sent %s %s", gotMethod, gotPath)
	}
}
