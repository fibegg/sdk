package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPlaygroundsHelpMatchesLifecycleSurface(t *testing.T) {
	help := commandHelp(t, playgroundsCmd())

	for _, want := range []string{
		"start <id-or-name>",
		"stop <id-or-name>",
		"has_changes",
		"completed",
		"stopping",
		"stopped",
		"destroying",
		"maintenance enable <id-or-name>",
		"maintenance disable <id-or-name>",
	} {
		if !strings.Contains(help, want) {
			t.Fatalf("playgrounds help missing %q:\n%s", want, help)
		}
	}
}

func TestPlaygroundActionCommandsAreRegistered(t *testing.T) {
	cmd := playgroundsCmd()

	for _, args := range [][]string{
		{"start", "example"},
		{"stop", "example"},
		{"rollout", "example"},
		{"hard-restart", "example"},
		{"maintenance", "enable", "example"},
		{"maintenance", "disable", "example"},
	} {
		found, _, err := cmd.Find(args)
		if err != nil {
			t.Fatalf("find %v: %v", args, err)
		}
		if found == nil {
			t.Fatalf("find %v returned nil command", args)
		}
		if found.Use == "" {
			t.Fatalf("find %v returned command without use", args)
		}
	}
}

func TestPlaygroundMaintenanceEnableCommandMapsActionBody(t *testing.T) {
	setupAuthTest(t)

	var body map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/playgrounds/demo/action" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"id": 42, "status": "stopped", "maintenance_enabled": true})
	}))
	defer srv.Close()

	t.Setenv("FIBE_DOMAIN", srv.URL)
	t.Setenv("FIBE_API_KEY", "pk_test")

	cmd := playgroundsCmd()
	cmd.SetArgs([]string{"maintenance", "enable", "demo"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	if body["action_type"] != "enable_maintenance" {
		t.Fatalf("body action_type = %#v, want enable_maintenance", body["action_type"])
	}
}

func TestPlaygroundMaintenanceDisableCommandMapsActionBody(t *testing.T) {
	setupAuthTest(t)

	var body map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/playgrounds/demo/action" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"id": 42, "status": "stopped", "maintenance_enabled": false})
	}))
	defer srv.Close()

	t.Setenv("FIBE_DOMAIN", srv.URL)
	t.Setenv("FIBE_API_KEY", "pk_test")

	cmd := playgroundsCmd()
	cmd.SetArgs([]string{"maintenance", "disable", "demo"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	if body["action_type"] != "disable_maintenance" {
		t.Fatalf("body action_type = %#v, want disable_maintenance", body["action_type"])
	}
}
