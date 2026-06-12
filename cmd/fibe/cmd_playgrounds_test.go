package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/fibegg/sdk/fibe"
)

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
		if r.Method != http.MethodPost || r.URL.Path != "/api/playgrounds/demo/operations" {
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
		if r.Method != http.MethodPost || r.URL.Path != "/api/playgrounds/demo/operations" {
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

func TestPlaygroundCreateAliasesAndServiceOverridesMapBody(t *testing.T) {
	setupAuthTest(t)

	var body map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/playgrounds" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"id": 42, "name": "demo", "status": "pending"})
	}))
	defer srv.Close()

	t.Setenv("FIBE_DOMAIN", srv.URL)
	t.Setenv("FIBE_API_KEY", "pk_test")

	cmd := playgroundsCmd()
	cmd.SetArgs([]string{
		"create",
		"--name", "demo",
		"--playspec", "starter",
		"--marquee", "next",
		"--service", "web.subdomain=demo",
		"--service", "web.exposure_port=3000",
		"--service", "web.exposure_visibility=external",
		"--service", "web.path_rule=PathPrefix(`/demo`)",
		"--service", "web.env_vars.RAILS_ENV=production",
		"--service", "web.git_config.branch_name=main",
		"--service", "web.git_config.create_branch=true",
	})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	playground := body["playground"].(map[string]any)
	if playground["playspec_id"] != "starter" {
		t.Fatalf("playspec_id = %#v, want starter", playground["playspec_id"])
	}
	if playground["marquee_id"] != "next" {
		t.Fatalf("marquee_id = %#v, want next", playground["marquee_id"])
	}
	web := playground["services"].(map[string]any)["web"].(map[string]any)
	if web["subdomain"] != "demo" || web["exposure_port"] != float64(3000) || web["exposure_visibility"] != "external" {
		t.Fatalf("web service exposure fields = %#v", web)
	}
	if web["path_rule"] != "PathPrefix(`/demo`)" {
		t.Fatalf("path_rule = %#v", web["path_rule"])
	}
	if web["env_vars"].(map[string]any)["RAILS_ENV"] != "production" {
		t.Fatalf("env_vars = %#v", web["env_vars"])
	}
	gitConfig := web["git_config"].(map[string]any)
	if gitConfig["branch_name"] != "main" || gitConfig["create_branch"] != true {
		t.Fatalf("git_config = %#v", gitConfig)
	}
}

func TestPlaygroundCreateRejectsConflictingAliases(t *testing.T) {
	setupAuthTest(t)

	cmd := playgroundsCmd()
	cmd.SetArgs([]string{"create", "--name", "demo", "--playspec-id", "one", "--playspec", "two", "--marquee", "next"})
	err := cmd.Execute()
	if err == nil {
		t.Fatalf("execute succeeded, want error")
	}
	if got := err.Error(); got != "conflicting values for --playspec-id and --playspec" {
		t.Fatalf("error = %q", got)
	}
}

func TestPlaygroundCreateRequiresMarqueeLocally(t *testing.T) {
	setupAuthTest(t)

	cmd := playgroundsCmd()
	cmd.SetArgs([]string{"create", "--name", "demo", "--playspec", "starter"})
	err := cmd.Execute()
	if err == nil {
		t.Fatalf("execute succeeded, want error")
	}
	if got := err.Error(); got != "required field 'marquee-id' not set" {
		t.Fatalf("error = %q", got)
	}
}

func TestPlaygroundServiceOverrideRejectsPortMappings(t *testing.T) {
	err := applyPlaygroundServiceOverride(map[string]*fibe.ServiceConfig{}, "web.port_mappings.3000=8080")
	if err == nil {
		t.Fatalf("override succeeded, want error")
	}
	if !strings.Contains(err.Error(), "does not support port_mappings") {
		t.Fatalf("error = %q", err)
	}
}
