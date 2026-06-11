package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWaitPlaygroundUsesIdentifierEndpoint(t *testing.T) {
	setupAuthTest(t)

	paths := []string{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.EscapedPath()
		paths = append(paths, path)
		if r.Method != http.MethodGet {
			t.Fatalf("unexpected request %s %s", r.Method, path)
		}
		switch path {
		case "/api/playgrounds/next/status":
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 129, "status": "running"})
		case "/api/playgrounds/next":
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 129, "name": "next", "status": "running"})
		default:
			t.Fatalf("unexpected request %s %s", r.Method, path)
		}
	}))
	defer srv.Close()

	t.Setenv("FIBE_DOMAIN", srv.URL)
	t.Setenv("FIBE_API_KEY", "pk_test")

	cmd := waitCmd()
	cmd.SetArgs([]string{"playground", "next", "--status", "running", "--timeout", "1s"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	if len(paths) != 2 || paths[0] != "/api/playgrounds/next/status" || paths[1] != "/api/playgrounds/next" {
		t.Fatalf("paths=%#v", paths)
	}
}

func TestWaitTrickUsesIdentifierEndpoint(t *testing.T) {
	setupAuthTest(t)

	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.EscapedPath()
		if r.Method != http.MethodGet || gotPath != "/api/playgrounds/nightly-build/status" {
			t.Fatalf("unexpected request %s %s", r.Method, gotPath)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"id": 77, "name": "nightly-build", "status": "completed", "job_mode": true})
	}))
	defer srv.Close()

	t.Setenv("FIBE_DOMAIN", srv.URL)
	t.Setenv("FIBE_API_KEY", "pk_test")

	cmd := waitCmd()
	cmd.SetArgs([]string{"trick", "nightly-build", "--status", "completed", "--timeout", "1s"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	if gotPath != "/api/playgrounds/nightly-build/status" {
		t.Fatalf("path=%q", gotPath)
	}
}

func TestWaitTrickFailsOnFailedJobResult(t *testing.T) {
	setupAuthTest(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.EscapedPath() != "/api/playgrounds/nightly-build/status" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.EscapedPath())
		}
		success := false
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":            77,
			"status":        "completed",
			"job_mode":      true,
			"result_status": "failed",
			"job_result":    map[string]any{"success": success},
		})
	}))
	defer srv.Close()

	t.Setenv("FIBE_DOMAIN", srv.URL)
	t.Setenv("FIBE_API_KEY", "pk_test")

	cmd := waitCmd()
	cmd.SetArgs([]string{"trick", "nightly-build", "--status", "completed", "--timeout", "1s"})
	if err := cmd.Execute(); err == nil {
		t.Fatal("expected failed job result error")
	}
}
