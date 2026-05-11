package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWaitPlaygroundUsesIdentifierEndpoint(t *testing.T) {
	setupAuthTest(t)

	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.EscapedPath()
		if r.Method != http.MethodGet || gotPath != "/api/playgrounds/next/status" {
			t.Fatalf("unexpected request %s %s", r.Method, gotPath)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"id": 129, "status": "running"})
	}))
	defer srv.Close()

	t.Setenv("FIBE_DOMAIN", srv.URL)
	t.Setenv("FIBE_API_KEY", "pk_test")

	cmd := waitCmd()
	cmd.SetArgs([]string{"playground", "next", "--status", "running", "--timeout", "1s"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	if gotPath != "/api/playgrounds/next/status" {
		t.Fatalf("path=%q", gotPath)
	}
}

func TestWaitTrickUsesIdentifierEndpoint(t *testing.T) {
	setupAuthTest(t)

	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.EscapedPath()
		if r.Method != http.MethodGet || gotPath != "/api/playgrounds/nightly-build" {
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
	if gotPath != "/api/playgrounds/nightly-build" {
		t.Fatalf("path=%q", gotPath)
	}
}
