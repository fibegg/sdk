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

func TestPlaygroundGetTableShowsServiceURLsAndHidesPassword(t *testing.T) {
	setupAuthTest(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/playgrounds/demo" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		running := true
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":                  42,
			"name":                "demo",
			"status":              "running",
			"maintenance_enabled": false,
			"job_mode":            false,
			"playspec_id":         7,
			"playspec_name":       "starter",
			"marquee_id":          9,
			"marquee_name":        "edge",
			"compose_project":     "starter--42",
			"root_domain":         "example.test",
			"routing_scheme":      "https",
			"internal_password":   "super-secret",
			"service_urls": []map[string]any{
				{
					"name":          "web",
					"type":          "static",
					"url":           "https://demo.example.test",
					"visibility":    "internal",
					"auth_required": true,
					"status":        "running",
					"health":        "healthy",
					"running":       running,
				},
			},
			"services": []map[string]any{
				{
					"name":    "web",
					"status":  "running",
					"health":  "healthy",
					"running": running,
					"image":   "nginx",
				},
			},
			"service_sources": []map[string]any{
				{
					"service":        "web",
					"prop_name":      "app",
					"branch":         "main",
					"repository_url": "https://github.com/acme/app",
				},
			},
			"build_statuses": []map[string]any{
				{
					"service_name": "web",
					"branch":       "main",
					"active": map[string]any{
						"id":               1,
						"status":           "built",
						"commit_sha":       "abcdef1234567890",
						"short_commit_sha": "abcdef1",
					},
				},
			},
		})
	}))
	defer srv.Close()

	t.Setenv("FIBE_DOMAIN", srv.URL)
	t.Setenv("FIBE_API_KEY", "pk_test")

	out, err := captureStdout(func() error {
		cmd := RootCmd()
		cmd.SetArgs([]string{"pg", "get", "demo"})
		return cmd.Execute()
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	for _, want := range []string{
		"Service URLs:",
		"https://demo.example.test",
		"HTTP basic auth: username playground",
		"Services:",
		"Sources:",
		"Builds:",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected %q in output:\n%s", want, out)
		}
	}
	if strings.Contains(out, "super-secret") {
		t.Fatalf("table output leaked internal password:\n%s", out)
	}
}

func TestPlaygroundGetJSONOnlyServiceURLs(t *testing.T) {
	setupAuthTest(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/playgrounds/demo" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":     42,
			"name":   "demo",
			"status": "running",
			"service_urls": []map[string]any{
				{
					"name":          "web",
					"type":          "static",
					"url":           "https://demo.example.test",
					"visibility":    "external",
					"auth_required": false,
				},
			},
		})
	}))
	defer srv.Close()

	t.Setenv("FIBE_DOMAIN", srv.URL)
	t.Setenv("FIBE_API_KEY", "pk_test")

	out, err := captureStdout(func() error {
		cmd := RootCmd()
		cmd.SetArgs([]string{"pg", "get", "demo", "-o", "json", "--only", "service_urls"})
		return cmd.Execute()
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("decode output: %v\n%s", err, out)
	}
	if len(got) != 1 {
		t.Fatalf("projected output keys = %#v, want only service_urls", got)
	}
	urls, ok := got["service_urls"].([]any)
	if !ok || len(urls) != 1 {
		t.Fatalf("service_urls = %#v", got["service_urls"])
	}
	first := urls[0].(map[string]any)
	if first["url"] != "https://demo.example.test" {
		t.Fatalf("projected service URL = %#v", first)
	}
}

func TestPlaygroundCreateServiceOverridesMapBody(t *testing.T) {
	setupAuthTest(t)

	var body map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/api/playspecs/starter" {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":   7,
				"name": "starter",
				"services": []map[string]any{
					{"name": "web"},
				},
			})
			return
		}
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

func TestPlaygroundCreateRejectsRetiredIDFlags(t *testing.T) {
	setupAuthTest(t)

	retiredFlag := "--playspec" + "-id"
	cmd := playgroundsCmd()
	cmd.SetArgs([]string{"create", "--name", "demo", retiredFlag, "one", "--marquee", "next"})
	err := cmd.Execute()
	if err == nil {
		t.Fatalf("execute succeeded, want error")
	}
	if got := err.Error(); !strings.Contains(got, "unknown flag: "+retiredFlag) {
		t.Fatalf("error = %q", got)
	}
}

func TestPlaygroundCreateRequiresMarqueeWhenInferenceFails(t *testing.T) {
	setupAuthTest(t)

	cmd := playgroundsCmd()
	cmd.SetArgs([]string{"create", "--name", "demo", "--playspec", "starter"})
	err := cmd.Execute()
	if err == nil {
		t.Fatalf("execute succeeded, want error")
	}
	if got := err.Error(); !strings.Contains(got, "Authentication required") && !strings.Contains(got, "API key") {
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
