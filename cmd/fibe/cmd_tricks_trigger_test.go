package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTricksTriggerSendsRunOverrides(t *testing.T) {
	setupAuthTest(t)

	var body map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.EscapedPath() {
		case "/api/playgrounds":
			if r.Method != http.MethodPost {
				t.Fatalf("unexpected method %s", r.Method)
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 77, "name": "run", "status": "pending", "job_mode": true})
		case "/api/playspecs/nightly-build":
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 12, "name": "nightly-build", "job_mode": true})
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.EscapedPath())
		}
	}))
	defer srv.Close()

	t.Setenv("FIBE_DOMAIN", srv.URL)
	t.Setenv("FIBE_API_KEY", "pk_test")

	cmd := tricksCmd()
	cmd.SetArgs([]string{
		"trigger",
		"--playspec", "nightly-build",
		"--marquee", "lyke",
		"--name", "run",
		"--env-overrides", `{"GH_TOKEN":"secret"}`,
		"--only-service", "tests",
		"--except-service", "lint",
	})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	playground := body["playground"].(map[string]any)
	if playground["playspec_id"] != "nightly-build" || playground["marquee_id"] != "lyke" {
		t.Fatalf("unexpected identifiers: %#v", playground)
	}
	env := playground["env_overrides"].(map[string]any)
	if env["GH_TOKEN"] != "secret" {
		t.Fatalf("env override missing: %#v", env)
	}
	if got := playground["only_services"].([]any)[0]; got != "tests" {
		t.Fatalf("only_services = %#v", playground["only_services"])
	}
	if got := playground["except_services"].([]any)[0]; got != "lint" {
		t.Fatalf("except_services = %#v", playground["except_services"])
	}
}
