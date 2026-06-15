package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLaunchWaitPrintsFinalReadyPlayground(t *testing.T) {
	setupAuthTest(t)

	statusCalls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/playspecs/starter":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":       299,
				"name":     "starter",
				"services": []map[string]any{{"name": "api"}},
			})
		case "/api/playgrounds":
			if r.Method != http.MethodPost {
				t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":     515,
				"name":   "demo",
				"status": "pending",
			})
		case "/api/playgrounds/515/status":
			statusCalls++
			payload := map[string]any{
				"id":     515,
				"status": "running",
				"services": []map[string]any{{
					"name":    "api",
					"status":  "stopped",
					"running": false,
				}},
			}
			if statusCalls >= 2 {
				payload["services"] = []map[string]any{{
					"name":    "api",
					"status":  "running",
					"health":  "healthy",
					"running": true,
				}}
			}
			_ = json.NewEncoder(w).Encode(payload)
		case "/api/playgrounds/515":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":                  515,
				"name":                "demo",
				"status":              "running",
				"maintenance_enabled": false,
				"job_mode":            false,
				"playspec_id":         299,
				"playspec_name":       "starter",
				"services": []map[string]any{{
					"name":    "api",
					"status":  "running",
					"health":  "healthy",
					"running": true,
				}},
			})
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer srv.Close()

	t.Setenv("FIBE_DOMAIN", srv.URL)
	t.Setenv("FIBE_API_KEY", "pk_test")

	cmd := launchCmd()
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--playspec", "starter", "--name", "demo", "--marquee", "office", "--wait", "--wait-timeout", "1s"})

	out, err := captureStdout(func() error {
		return cmd.Execute()
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if statusCalls != 2 {
		t.Fatalf("statusCalls=%d want 2", statusCalls)
	}
	for _, want := range []string{"Status:          running", "Services:", "api      running  healthy  yes"} {
		if !strings.Contains(out, want) {
			t.Fatalf("output missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "stopped") {
		t.Fatalf("output should use final ready playground, got:\n%s", out)
	}
}
