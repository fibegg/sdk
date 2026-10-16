package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestEnvPackCLIFileDetachmentAndCommands(t *testing.T) {
	setupAuthTest(t)
	hits := 0
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		if r.Method != "PATCH" || r.URL.Path != "/api/specs/7/env_pack_attachments" {
			t.Fatalf("request %s %s", r.Method, r.URL.Path)
		}
		var p map[string]any
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			t.Fatal(err)
		}
		rows, ok := p["env_pack_attachments"].([]any)
		if !ok || len(rows) != 0 {
			t.Fatal("explicit empty detach was lost")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":7,"env_pack_attachments":[]}`))
	}))
	defer api.Close()
	t.Setenv("FIBE_DOMAIN", api.URL)
	t.Setenv("FIBE_API_KEY", "fixture")
	file := filepath.Join(t.TempDir(), "attachments.json")
	if err := os.WriteFile(file, []byte(`{"env_pack_attachments":[]}`), 0600); err != nil {
		t.Fatal(err)
	}
	cmd := RootCmd()
	cmd.SetArgs([]string{"env-packs", "attachments", "set", "specs", "7", "--from-file", file})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if hits != 1 {
		t.Fatalf("hits %d", hits)
	}
	for _, args := range [][]string{{"env-packs", "create"}, {"env-packs", "attachments", "reorder"}, {"env-packs", "attachments", "retarget"}, {"env-packs", "attachments", "detach"}, {"env-packs", "attachments", "renew"}} {
		found, _, err := RootCmd().Find(args)
		if err != nil || found.Name() != args[len(args)-1] {
			t.Fatalf("missing command %v", args)
		}
	}
}

func TestEnvPackCLILaunchCarriesExplicitSelectionAndRejectsForgedFields(t *testing.T) {
	setupAuthTest(t)
	hits := 0
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/api/launches" {
			t.Fatalf("request %s %s", r.Method, r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		rows, ok := body["env_pack_attachments"].([]any)
		if !ok || len(rows) != 0 {
			t.Fatal("launch opt-out lost")
		}
		hits++
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"spec_id":7,"playground_id":8}`))
	}))
	defer api.Close()
	t.Setenv("FIBE_DOMAIN", api.URL)
	t.Setenv("FIBE_API_KEY", "fixture")
	command := RootCmd()
	command.SetArgs([]string{"launch", "--compose", "services:\n  web:\n    image: alpine\n", "--name", "fixture", "--host", "21", "--env-pack-attachments", "[]"})
	if err := command.Execute(); err != nil {
		t.Fatal(err)
	}
	if hits != 1 {
		t.Fatalf("hits %d", hits)
	}
	for _, raw := range []string{`[{"env_pack_id":1,"grantor_id":2}]`, `[{"env_pack_id":1,"service_names":[]}]`, `[] {}`} {
		if _, err := parseLaunchEnvPacks(raw, true); err == nil {
			t.Fatalf("invalid selection accepted: %s", raw)
		}
	}
	if hits != 1 {
		t.Fatal("invalid selection reached server")
	}
}
