package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLaunchCommandMapsPersistVolumesFlag(t *testing.T) {
	setupAuthTest(t)
	resetFromFileFlagsForTest(t)

	var body map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/launches" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"playspec_id": 7})
	}))
	defer srv.Close()

	t.Setenv("FIBE_DOMAIN", srv.URL)
	t.Setenv("FIBE_API_KEY", "pk_test")

	cmd := launchCmd()
	cmd.SetArgs([]string{
		"--name", "ci",
		"--compose", "services:\n  job:\n    image: alpine\n",
		"--job-mode",
		"--marquee-id", "runner",
		"--persist-volumes",
	})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	if body["persist_volumes"] != true {
		t.Fatalf("persist_volumes = %#v, want true in %#v", body["persist_volumes"], body)
	}
}

func TestLaunchHelpExposesPersistVolumesFlag(t *testing.T) {
	help := commandHelp(t, launchCmd())
	if !strings.Contains(help, "--persist-volumes") {
		t.Fatalf("launch help missing --persist-volumes:\n%s", help)
	}
}
