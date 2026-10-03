package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTemplateCreateReadsBodyFileAndRejectsMissingFile(t *testing.T) {
	setupAuthTest(t)
	body := "services:\n  web:\n    image: nginx:alpine\n"
	path := filepath.Join(t.TempDir(), "compose.yaml")
	if err := os.WriteFile(path, []byte(body), 0600); err != nil {
		t.Fatal(err)
	}
	requests := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.Method != http.MethodPost || r.URL.Path != "/api/import_templates" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Error(err)
		}
		if payload["template_body"] != body {
			t.Errorf("body = %#v", payload["template_body"])
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"id": 10, "name": "demo"})
	}))
	defer srv.Close()
	t.Setenv("FIBE_DOMAIN", srv.URL)
	t.Setenv("FIBE_API_KEY", "pk_test")
	for _, file := range []string{path, path + ".missing"} {
		cmd := RootCmd()
		cmd.SetErr(&bytes.Buffer{})
		cmd.SetArgs([]string{"templates", "create", "--name", "demo", "--body", "@" + file, "--output", "json"})
		_, err := captureStdout(cmd.Execute)
		if file == path && err != nil {
			t.Fatal(err)
		}
		if file != path && (err == nil || !strings.Contains(err.Error(), "read text file")) {
			t.Fatalf("missing-file error = %v", err)
		}
	}
	if requests != 1 {
		t.Fatalf("requests = %d", requests)
	}
}

func TestTemplateVersionReadsCanonicalStructuredInput(t *testing.T) {
	setupAuthTest(t)
	path := filepath.Join(t.TempDir(), "input.json")
	body := "services: {}\n"
	data, _ := json.Marshal(map[string]any{"template_body": body, "public": false})
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/import_templates/10/versions" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Error(err)
		}
		if payload["template_body"] != body {
			t.Errorf("payload = %#v", payload)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"id": 11})
	}))
	defer srv.Close()
	t.Setenv("FIBE_DOMAIN", srv.URL)
	t.Setenv("FIBE_API_KEY", "pk_test")
	cmd := RootCmd()
	cmd.SetArgs([]string{"templates", "versions", "create", "10", "--from-file", path})
	if _, err := captureStdout(cmd.Execute); err != nil {
		t.Fatal(err)
	}
}

func TestComposeValidationReadsStructuredInputAndFlagOverrides(t *testing.T) {
	setupAuthTest(t)
	path := filepath.Join(t.TempDir(), "input.json")
	if err := os.WriteFile(path, []byte(`{"compose_yaml":"services: {}"}`), 0600); err != nil {
		t.Fatal(err)
	}
	wantBody := "services: {}"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/schema.json" {
			_ = json.NewEncoder(w).Encode(map[string]any{"type": "object"})
			return
		}
		if r.Method != http.MethodPost || r.URL.Path != "/api/compose_validations" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Error(err)
		}
		if payload["compose_yaml"] != wantBody {
			t.Errorf("body = %#v", payload)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"valid": true})
	}))
	defer srv.Close()
	t.Setenv("FIBE_DOMAIN", srv.URL)
	t.Setenv("FIBE_API_KEY", "pk_test")
	for _, override := range []string{"", "services: {web: {image: nginx}}"} {
		args := []string{"playspecs", "validate-compose", "--from-file", path}
		if override != "" {
			wantBody = override
			args = append(args, "--compose", override)
		}
		cmd := RootCmd()
		cmd.SetArgs(args)
		if _, err := captureStdout(cmd.Execute); err != nil {
			t.Fatal(err)
		}
	}
}

func TestMutationCommandsRespectJSONOutput(t *testing.T) {
	for _, args := range [][]string{
		{"props", "mirror", "--url", "https://github.com/example/app"},
		{"props", "attach", "--repo", "example/app"},
		{"pg", "stop", "demo"}, {"pg", "start", "demo"},
		{"pg", "rollout", "demo"}, {"pg", "hard-restart", "demo"},
	} {
		t.Run(strings.Join(args[:2], "/"), func(t *testing.T) {
			setupAuthTest(t)
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					t.Errorf("method = %s", r.Method)
				}
				_ = json.NewEncoder(w).Encode(map[string]any{"id": 42, "name": "demo", "status": "stopped"})
			}))
			defer srv.Close()
			t.Setenv("FIBE_DOMAIN", srv.URL)
			t.Setenv("FIBE_API_KEY", "pk_test")
			cmd := RootCmd()
			cmd.SetErr(&bytes.Buffer{})
			cmd.SetArgs(append(args, "--output", "json"))
			out, err := captureStdout(cmd.Execute)
			if err != nil {
				t.Fatal(err)
			}
			var payload map[string]any
			if err := json.Unmarshal([]byte(out), &payload); err != nil {
				t.Fatalf("non-JSON output %q: %v", out, err)
			}
			if payload["id"] != float64(42) {
				t.Fatalf("payload = %#v", payload)
			}
		})
	}
}

func TestWaitRunningRequiresServicesUnlessLifecycleRequested(t *testing.T) {
	for _, readiness := range []string{"", "lifecycle"} {
		t.Run("readiness="+readiness, func(t *testing.T) {
			setupAuthTest(t)
			polls := 0
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if strings.HasSuffix(r.URL.Path, "/status") {
					polls++
					health := "starting"
					if polls > 1 {
						health = "healthy"
					}
					_ = json.NewEncoder(w).Encode(map[string]any{"id": 42, "status": "running", "services": []map[string]any{{"name": "web", "status": "running", "health": health}}})
					return
				}
				_ = json.NewEncoder(w).Encode(map[string]any{"id": 42, "status": "running"})
			}))
			defer srv.Close()
			t.Setenv("FIBE_DOMAIN", srv.URL)
			t.Setenv("FIBE_API_KEY", "pk_test")
			args := []string{"wait", "playground", "demo", "--interval", "1ms", "--timeout", "1s"}
			if readiness != "" {
				args = append(args, "--readiness", readiness)
			}
			cmd := RootCmd()
			cmd.SetErr(&bytes.Buffer{})
			cmd.SetArgs(args)
			if _, err := captureStdout(cmd.Execute); err != nil {
				t.Fatal(err)
			}
			want := 2
			if readiness == "lifecycle" {
				want = 1
			}
			if polls != want {
				t.Fatalf("polls = %d; want %d", polls, want)
			}
		})
	}
}

func TestWaitPropReportsAnAsynchronousMirrorFailure(t *testing.T) {
	setupAuthTest(t)
	requests := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.Method != http.MethodGet || r.URL.Path != "/api/props/42" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		status := "retrying"
		if requests > 1 {
			status = "failed"
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"id": 42, "mirror_status": status, "mirror_error": "Repository access denied"})
	}))
	defer srv.Close()
	t.Setenv("FIBE_DOMAIN", srv.URL)
	t.Setenv("FIBE_API_KEY", "pk_test")
	cmd := RootCmd()
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"wait", "prop", "42", "--interval", "1ms", "--timeout", "1s"})
	_, err := captureStdout(cmd.Execute)
	if err == nil || !strings.Contains(err.Error(), "Repository access denied") || requests != 2 {
		t.Fatalf("requests=%d error=%v", requests, err)
	}
}
