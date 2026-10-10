package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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
		{"rerun", "example"},
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
			"spec_id":             7,
			"spec_name":           "starter",
			"host_id":             9,
			"host_name":           "edge",
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
					"service":         "web",
					"repository_name": "app",
					"branch":          "main",
					"repository_url":  "https://github.com/acme/app",
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
		if r.Method == http.MethodGet && r.URL.Path == "/api/specs/starter" {
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
		"--spec", "starter",
		"--host", "next",
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
	if playground["spec_id"] != "starter" {
		t.Fatalf("spec_id = %#v, want starter", playground["spec_id"])
	}
	if playground["host_id"] != "next" {
		t.Fatalf("host_id = %#v, want next", playground["host_id"])
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

	retiredFlag := "--spec" + "-id"
	cmd := playgroundsCmd()
	cmd.SetArgs([]string{"create", "--name", "demo", retiredFlag, "one", "--host", "next"})
	err := cmd.Execute()
	if err == nil {
		t.Fatalf("execute succeeded, want error")
	}
	if got := err.Error(); !strings.Contains(got, "unknown flag: "+retiredFlag) {
		t.Fatalf("error = %q", got)
	}
}

func TestPlaygroundCreateRequiresHostWhenInferenceFails(t *testing.T) {
	setupAuthTest(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/hosts" {
			t.Errorf("unexpected request %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"data": []any{}})
	}))
	defer server.Close()
	flagDomain, flagAPIKey = server.URL, "synthetic-host-inference"

	cmd := playgroundsCmd()
	cmd.SetArgs([]string{"create", "--name", "demo", "--spec", "starter"})
	err := cmd.Execute()
	if err == nil {
		t.Fatalf("execute succeeded, want error")
	}
	if got := err.Error(); !strings.Contains(got, "--host is required; no launchable Hosts are available") {
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

// playgroundRerunServer resolves the source by name, then expects one POST to
// the numeric rerun path. It records every request body keyed by "METHOD path".
func playgroundRerunServer(t *testing.T, requests *[]string, bodies map[string]map[string]any, rerunStatus int, rerunResponse any) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.Method + " " + r.URL.Path
		*requests = append(*requests, key)
		body := map[string]any{}
		if r.Body != nil {
			_ = json.NewDecoder(r.Body).Decode(&body)
		}
		bodies[key] = body
		w.Header().Set("Content-Type", "application/json")
		switch key {
		case "GET /api/playgrounds/demo":
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 42, "name": "demo", "status": "running"})
		case "POST /api/playgrounds/42/rerun":
			w.WriteHeader(rerunStatus)
			_ = json.NewEncoder(w).Encode(rerunResponse)
		default:
			t.Errorf("unexpected request %s", key)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

func TestPlaygroundRerunPreservesSourceReferencesByDefault(t *testing.T) {
	setupAuthTest(t)
	requests := []string{}
	bodies := map[string]map[string]any{}
	srv := playgroundRerunServer(t, &requests, bodies, http.StatusCreated, map[string]any{"id": 43, "name": "demo-rerun-abc123", "status": "pending"})
	defer srv.Close()
	t.Setenv("FIBE_DOMAIN", srv.URL)
	t.Setenv("FIBE_API_KEY", "pk_test")

	out, err := captureStdout(func() error {
		cmd := RootCmd()
		cmd.SetArgs([]string{"pg", "rerun", "demo"})
		return cmd.Execute()
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if got, want := strings.Join(requests, "|"), "GET /api/playgrounds/demo|POST /api/playgrounds/42/rerun"; got != want {
		t.Fatalf("requests = %s, want %s", got, want)
	}
	body := bodies["POST /api/playgrounds/42/rerun"]
	if _, ok := body["env_pack_attachments"]; ok {
		t.Fatalf("rerun fabricated an attachment selection: %#v", body)
	}
	if _, ok := body["name"]; ok {
		t.Fatalf("rerun sent a name nobody asked for: %#v", body)
	}
	for _, want := range []string{"43", "demo-rerun-abc123", "demo", "pending"} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected %q in output:\n%s", want, out)
		}
	}
}

func TestPlaygroundRerunSendsNameAndExplicitEmptyAttachments(t *testing.T) {
	setupAuthTest(t)
	requests := []string{}
	bodies := map[string]map[string]any{}
	srv := playgroundRerunServer(t, &requests, bodies, http.StatusCreated, map[string]any{"id": 43, "name": "copy", "status": "pending"})
	defer srv.Close()
	t.Setenv("FIBE_DOMAIN", srv.URL)
	t.Setenv("FIBE_API_KEY", "pk_test")
	file := filepath.Join(t.TempDir(), "rerun.json")
	if err := os.WriteFile(file, []byte(`{"env_pack_attachments":[]}`), 0o600); err != nil {
		t.Fatal(err)
	}

	out, err := captureStdout(func() error {
		cmd := RootCmd()
		cmd.SetArgs([]string{"pg", "rerun", "demo", "--name", "copy", "--from-file", file, "-o", "json"})
		return cmd.Execute()
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	body := bodies["POST /api/playgrounds/42/rerun"]
	if body["name"] != "copy" {
		t.Fatalf("name = %#v, want copy", body["name"])
	}
	rows, ok := body["env_pack_attachments"].([]any)
	if !ok || len(rows) != 0 {
		t.Fatalf("explicit empty detach was lost: %#v", body)
	}
	var got map[string]any
	if err := json.Unmarshal([]byte(out), &got); err != nil || got["id"] != float64(43) {
		t.Fatalf("JSON output = %q (%v)", out, err)
	}
}

func TestPlaygroundRerunSendsOrderedAttachmentSelection(t *testing.T) {
	setupAuthTest(t)
	requests := []string{}
	bodies := map[string]map[string]any{}
	srv := playgroundRerunServer(t, &requests, bodies, http.StatusCreated, map[string]any{"id": 43, "name": "demo-rerun", "status": "pending"})
	defer srv.Close()
	t.Setenv("FIBE_DOMAIN", srv.URL)
	t.Setenv("FIBE_API_KEY", "pk_test")
	file := filepath.Join(t.TempDir(), "rerun.yaml")
	if err := os.WriteFile(file, []byte("env_pack_attachments:\n  - env_pack_id: 7\n    service_names: [web]\n  - env_pack_id: 5\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := captureStdout(func() error {
		cmd := RootCmd()
		cmd.SetArgs([]string{"pg", "rerun", "demo", "--from-file", file})
		return cmd.Execute()
	}); err != nil {
		t.Fatalf("execute: %v", err)
	}
	rows := bodies["POST /api/playgrounds/42/rerun"]["env_pack_attachments"].([]any)
	if len(rows) != 2 {
		t.Fatalf("attachments = %#v", rows)
	}
	first, second := rows[0].(map[string]any), rows[1].(map[string]any)
	if first["env_pack_id"] != float64(7) || second["env_pack_id"] != float64(5) {
		t.Fatalf("order lost: %#v", rows)
	}
	if names, _ := first["service_names"].([]any); len(names) != 1 || names[0] != "web" {
		t.Fatalf("service_names lost: %#v", first)
	}
	if _, ok := second["service_names"]; ok {
		t.Fatalf("all-services row gained service_names: %#v", second)
	}
}

func TestPlaygroundRerunRejectsInvalidSelectionBeforeRequest(t *testing.T) {
	setupAuthTest(t)
	requests := []string{}
	bodies := map[string]map[string]any{}
	srv := playgroundRerunServer(t, &requests, bodies, http.StatusCreated, map[string]any{})
	defer srv.Close()
	t.Setenv("FIBE_DOMAIN", srv.URL)
	t.Setenv("FIBE_API_KEY", "pk_test")
	file := filepath.Join(t.TempDir(), "rerun.json")
	if err := os.WriteFile(file, []byte(`{"env_pack_attachments":[{"env_pack_id":7,"service_names":[]}]}`), 0o600); err != nil {
		t.Fatal(err)
	}

	cmd := RootCmd()
	cmd.SetArgs([]string{"pg", "rerun", "demo", "--from-file", file})
	cmd.SilenceErrors, cmd.SilenceUsage = true, true
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "service_names") {
		t.Fatalf("expected local service_names validation error, got %v", err)
	}
	if hits := len(bodies["POST /api/playgrounds/42/rerun"]); hits != 0 {
		t.Fatalf("invalid selection reached the server: %v", requests)
	}
}

func TestPlaygroundRerunSurfacesServerErrorCode(t *testing.T) {
	setupAuthTest(t)
	requests := []string{}
	bodies := map[string]map[string]any{}
	srv := playgroundRerunServer(t, &requests, bodies, http.StatusUnprocessableEntity, map[string]any{
		"error": map[string]any{"code": "env_pack_grant_invalid", "message": "env pack grant invalid"},
	})
	defer srv.Close()
	t.Setenv("FIBE_DOMAIN", srv.URL)
	t.Setenv("FIBE_API_KEY", "pk_test")

	cmd := RootCmd()
	cmd.SetArgs([]string{"pg", "rerun", "demo"})
	cmd.SilenceErrors, cmd.SilenceUsage = true, true
	err := cmd.Execute()
	var apiErr *fibe.APIError
	if !errors.As(err, &apiErr) || apiErr.Code != "env_pack_grant_invalid" || apiErr.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("error = %#v, want env_pack_grant_invalid/422", err)
	}
}

func TestPlaygroundRerunMissingSourceStopsBeforeRerun(t *testing.T) {
	setupAuthTest(t)
	requests := []string{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]any{"code": "NOT_FOUND", "message": "not found"}})
	}))
	defer srv.Close()
	t.Setenv("FIBE_DOMAIN", srv.URL)
	t.Setenv("FIBE_API_KEY", "pk_test")

	cmd := RootCmd()
	cmd.SetArgs([]string{"pg", "rerun", "missing"})
	cmd.SilenceErrors, cmd.SilenceUsage = true, true
	err := cmd.Execute()
	var apiErr *fibe.APIError
	if !errors.As(err, &apiErr) || apiErr.Code != "NOT_FOUND" {
		t.Fatalf("error = %v, want wrapped NOT_FOUND", err)
	}
	if len(requests) != 1 || requests[0] != "GET /api/playgrounds/missing" {
		t.Fatalf("requests = %v, want only the source lookup", requests)
	}
}

func TestPlaygroundRerunIsMutatingAndRequiresOneArgument(t *testing.T) {
	root := RootCmd()
	found, _, err := root.Find([]string{"playgrounds", "rerun"})
	if err != nil || found.Name() != "rerun" {
		t.Fatalf("playgrounds rerun is not registered: %v", err)
	}
	if got := found.Annotations[commandSafetyAnnotation]; got != "mutating" {
		t.Fatalf("safety = %q, want mutating", got)
	}
	if err := found.Args(found, nil); err == nil {
		t.Fatal("rerun accepted zero arguments")
	}
	if err := found.Args(found, []string{"a", "b"}); err == nil {
		t.Fatal("rerun accepted two arguments")
	}
}
