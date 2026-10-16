package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSpecCreateCommandMapsJobConfigFlags(t *testing.T) {
	setupAuthTest(t)
	resetFromFileFlagsForTest(t)

	var body map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/specs" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"id": 7, "name": "ci"})
	}))
	defer srv.Close()

	t.Setenv("FIBE_DOMAIN", srv.URL)
	t.Setenv("FIBE_API_KEY", "pk_test")

	cmd := specsCmd()
	cmd.SetArgs([]string{
		"create",
		"--name", "ci",
		"--compose", "services:\n  job:\n    image: alpine\n",
		"--job-mode",
		"--schedule-enabled",
		"--schedule-cron", "every 5 minutes",
		"--schedule-host", "runner",
		"--trigger-enabled",
		"--trigger-event-type", "push",
		"--trigger-branch", "main",
		"--trigger-repository", "api",
		"--trigger-host", "runner",
		"--trigger-agent-id", "fixer",
		"--trigger-max-retries", "2",
		"--trigger-prompt-template", "Fix {{logs}}",
		"--muti-enabled",
		"--muti-language", "ruby",
		"--muti-repository", "api",
		"--muti-agent-id", "fixer",
		"--muti-prompt-template", "Fix {{diff}}",
	})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	payload := body["spec"].(map[string]any)
	if payload["job_mode"] != true {
		t.Fatalf("job_mode = %#v, want true", payload["job_mode"])
	}
	schedule := payload["schedule_config"].(map[string]any)
	if schedule["enabled"] != true || schedule["cron"] != "every 5 minutes" || schedule["host_id"] != "runner" {
		t.Fatalf("unexpected schedule_config: %#v", schedule)
	}
	trigger := payload["trigger_config"].(map[string]any)
	if trigger["enabled"] != true ||
		trigger["event_type"] != "push" ||
		trigger["branch"] != "main" ||
		trigger["repository_id"] != "api" ||
		trigger["host_id"] != "runner" ||
		trigger["agent_id"] != "fixer" ||
		trigger["prompt_template"] != "Fix {{logs}}" ||
		trigger["max_retries"] != float64(2) {
		t.Fatalf("unexpected trigger_config: %#v", trigger)
	}
	muti := payload["muti_config"].(map[string]any)
	if muti["enabled"] != true ||
		muti["language"] != "ruby" ||
		muti["repository_id"] != "api" ||
		muti["agent_id"] != "fixer" ||
		muti["prompt_template"] != "Fix {{diff}}" {
		t.Fatalf("unexpected muti_config: %#v", muti)
	}
}

func TestSpecUpdateCommandMergesConfigFlags(t *testing.T) {
	setupAuthTest(t)
	resetFromFileFlagsForTest(t)

	var patched map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/specs/ci":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":   7,
				"name": "ci",
				"trigger_config": map[string]any{
					"enabled":       true,
					"event_type":    "push",
					"branch":        "main",
					"repository_id": 3,
					"host_id":       4,
				},
			})
		case r.Method == http.MethodPatch && r.URL.Path == "/api/specs/ci":
			if err := json.NewDecoder(r.Body).Decode(&patched); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 7, "name": "ci"})
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer srv.Close()

	t.Setenv("FIBE_DOMAIN", srv.URL)
	t.Setenv("FIBE_API_KEY", "pk_test")

	cmd := specsCmd()
	cmd.SetArgs([]string{
		"update", "ci",
		"--trigger-agent-id", "fixer",
		"--trigger-prompt-template", "Fix {{logs}}",
	})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	trigger := patched["spec"].(map[string]any)["trigger_config"].(map[string]any)
	if trigger["enabled"] != true ||
		trigger["event_type"] != "push" ||
		trigger["branch"] != "main" ||
		trigger["repository_id"] != float64(3) ||
		trigger["host_id"] != float64(4) ||
		trigger["agent_id"] != "fixer" ||
		trigger["prompt_template"] != "Fix {{logs}}" {
		t.Fatalf("unexpected merged trigger_config: %#v", trigger)
	}
}

func resetFromFileFlagsForTest(t *testing.T) {
	t.Helper()
	oldFlag := flagFromFile
	oldRaw := rawPayload
	flagFromFile = ""
	rawPayload = nil
	t.Cleanup(func() {
		flagFromFile = oldFlag
		rawPayload = oldRaw
	})
}
