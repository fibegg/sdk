package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/fibegg/sdk/fibe"
)

func TestMarqueeIDFromEnv(t *testing.T) {
	t.Setenv("FIBE_MARQUEE_ID", "42")

	id, err := marqueeIDFromEnv()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != 42 {
		t.Fatalf("id=%d want 42", id)
	}
}

func TestParseGreenfieldVars(t *testing.T) {
	vars, err := parseGreenfieldVars([]string{"app_name=Tower", "tier=dev", "--subdomain=x1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if vars["app_name"] != "Tower" || vars["tier"] != "dev" || vars["subdomain"] != "x1" {
		t.Fatalf("unexpected vars: %#v", vars)
	}
}

func TestGreenfieldHelpShowsOnlyPublicUXFlags(t *testing.T) {
	cmd := greenfieldCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--help"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("help failed: %v", err)
	}

	got := out.String()
	for _, want := range []string{
		"--name string",
		"Repository/app name (required, must be unique)",
		"--git-provider string",
		"Destination git provider: gitea or github (optional, default: gitea)",
		"--template-id int",
		"Template to use (optional, default: base template)",
		"--version string",
		"Template version tag or number when --template-id is used (e.g. v1, optional, default: latest version)",
		"--template-body string",
		"Template YAML body to use directly (optional)",
		"--marquee-id int",
		"Target marquee ID (optional, default: current Marquee)",
		"--var strings",
		"Set template variables (e.g., --var app_name=Tower, optional)",
		"--wait-timeout duration",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("help output does not contain %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "--link-dir") {
		t.Fatalf("help output exposes --link-dir:\n%s", got)
	}
}

func TestApplyGreenfieldFromFileTreatsRawComposeAsTemplateBody(t *testing.T) {
	oldFlag := flagFromFile
	oldRaw := rawPayload
	defer func() {
		flagFromFile = oldFlag
		rawPayload = oldRaw
	}()

	body := "services:\n  web:\n    image: nginx\n"
	path := filepath.Join(t.TempDir(), "template.yml")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("write temp template: %v", err)
	}
	flagFromFile = path

	params := &fibe.GreenfieldCreateParams{Name: "todo"}
	if err := applyGreenfieldFromFile(params); err != nil {
		t.Fatalf("applyGreenfieldFromFile: %v", err)
	}
	if params.TemplateBody != body {
		t.Fatalf("template_body=%q want %q", params.TemplateBody, body)
	}
}

func TestNormalizeTemplateBodyValueExpandsNewlines(t *testing.T) {
	got := normalizeTemplateBodyValue(`services:\n  web:\n    image: nginx\n`)
	want := "services:\n  web:\n    image: nginx\n"
	if got != want {
		t.Fatalf("body=%q want %q", got, want)
	}
}

func TestWaitForPlaygroundReachesRunning(t *testing.T) {
	statusCalls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/playgrounds/7/status":
			statusCalls++
			status := "in_progress"
			if statusCalls >= 2 {
				status = "running"
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 7, "status": status})
		case "/api/playgrounds/7":
			_ = json.NewEncoder(w).Encode(fibe.Playground{ID: 7, Name: "tower-defence", Status: "running"})
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	c := fibe.NewClient(fibe.WithDomain(srv.URL), fibe.WithAPIKey("pk_test"), fibe.WithMaxRetries(0))
	pg, err := waitForPlayground(context.Background(), c, 7, "running", time.Second, time.Millisecond, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pg.ID != 7 || pg.Status != "running" {
		t.Fatalf("unexpected playground: %#v", pg)
	}
	if statusCalls != 2 {
		t.Fatalf("statusCalls=%d want 2", statusCalls)
	}
}

func TestWaitForPlaygroundTerminalStateIncludesDetails(t *testing.T) {
	message := "compose up failed"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/playgrounds/7/status":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":                  7,
				"status":              "error",
				"error_message":       message,
				"error_step":          "compose_up",
				"error_step_label":    "Starting Containers",
				"failure_diagnostics": map[string]any{"compose_error": "bad image"},
			})
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	c := fibe.NewClient(fibe.WithDomain(srv.URL), fibe.WithAPIKey("pk_test"), fibe.WithMaxRetries(0))
	_, err := waitForPlayground(context.Background(), c, 7, "running", time.Second, time.Millisecond, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	got := err.Error()
	for _, want := range []string{"playground reached terminal state: error", "error_message: compose up failed", "error_step: compose_up (Starting Containers)", "failure_diagnostics", "bad image"} {
		if !strings.Contains(got, want) {
			t.Fatalf("error %q does not contain %q", got, want)
		}
	}
}
