package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPlayspecCreateCommandMapsJobConfigFlags(t *testing.T) {
	setupAuthTest(t)
	resetFromFileFlagsForTest(t)

	var body map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/playspecs" {
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

	cmd := playspecsCmd()
	cmd.SetArgs([]string{
		"create",
		"--name", "ci",
		"--compose", "services:\n  job:\n    image: alpine\n",
		"--job-mode",
		"--schedule-enabled",
		"--schedule-cron", "every 5 minutes",
		"--schedule-marquee-id", "runner",
		"--trigger-enabled",
		"--trigger-event-type", "push",
		"--trigger-branch", "main",
		"--trigger-prop-id", "api",
		"--trigger-marquee-id", "runner",
		"--trigger-agent-id", "fixer",
		"--trigger-max-retries", "2",
		"--trigger-prompt-template", "Fix {{logs}}",
		"--muti-enabled",
		"--muti-language", "ruby",
		"--muti-prop-id", "api",
		"--muti-agent-id", "fixer",
		"--muti-prompt-template", "Fix {{diff}}",
	})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	payload := body["playspec"].(map[string]any)
	if payload["job_mode"] != true {
		t.Fatalf("job_mode = %#v, want true", payload["job_mode"])
	}
	schedule := payload["schedule_config"].(map[string]any)
	if schedule["enabled"] != true || schedule["cron"] != "every 5 minutes" || schedule["marquee_id"] != "runner" {
		t.Fatalf("unexpected schedule_config: %#v", schedule)
	}
	trigger := payload["trigger_config"].(map[string]any)
	if trigger["enabled"] != true ||
		trigger["event_type"] != "push" ||
		trigger["branch"] != "main" ||
		trigger["prop_id"] != "api" ||
		trigger["marquee_id"] != "runner" ||
		trigger["agent_id"] != "fixer" ||
		trigger["prompt_template"] != "Fix {{logs}}" ||
		trigger["max_retries"] != float64(2) {
		t.Fatalf("unexpected trigger_config: %#v", trigger)
	}
	muti := payload["muti_config"].(map[string]any)
	if muti["enabled"] != true ||
		muti["language"] != "ruby" ||
		muti["prop_id"] != "api" ||
		muti["agent_id"] != "fixer" ||
		muti["prompt_template"] != "Fix {{diff}}" {
		t.Fatalf("unexpected muti_config: %#v", muti)
	}
}

func TestPlayspecUpdateCommandMergesConfigFlags(t *testing.T) {
	setupAuthTest(t)
	resetFromFileFlagsForTest(t)

	var patched map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/playspecs/ci":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":   7,
				"name": "ci",
				"trigger_config": map[string]any{
					"enabled":    true,
					"event_type": "push",
					"branch":     "main",
					"prop_id":    3,
					"marquee_id": 4,
				},
			})
		case r.Method == http.MethodPatch && r.URL.Path == "/api/playspecs/ci":
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

	cmd := playspecsCmd()
	cmd.SetArgs([]string{
		"update", "ci",
		"--trigger-agent-id", "fixer",
		"--trigger-prompt-template", "Fix {{logs}}",
	})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	trigger := patched["playspec"].(map[string]any)["trigger_config"].(map[string]any)
	if trigger["enabled"] != true ||
		trigger["event_type"] != "push" ||
		trigger["branch"] != "main" ||
		trigger["prop_id"] != float64(3) ||
		trigger["marquee_id"] != float64(4) ||
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
