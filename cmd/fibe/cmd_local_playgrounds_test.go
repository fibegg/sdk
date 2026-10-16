package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/fibegg/sdk/internal/localplaygrounds"
)

func TestLocalPlaygroundCommandsShareConfiguredLinkDirectory(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HOST_ROOT", root)
	linkDir := filepath.Join(t.TempDir(), "data", "agent-77", "playground")
	t.Setenv("PLAYGROUNDS_DIR", linkDir)
	source := filepath.Join(root, "app--77", "repositories", "team--app--1", "main")
	if err := os.MkdirAll(source, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "source.txt"), []byte("agent edit"), 0o644); err != nil {
		t.Fatal(err)
	}
	compose := fmt.Sprintf("services:\n  web:\n    labels:\n      fibe.gg/spec: app\n    volumes:\n      - %q\n", source+":/app")
	if err := os.WriteFile(filepath.Join(root, "app--77", "compose.yml"), []byte(compose), 0o644); err != nil {
		t.Fatal(err)
	}
	oldOutput := flagOutput
	flagOutput = "json"
	t.Cleanup(func() { flagOutput = oldOutput })
	run := func(args ...string) string {
		t.Helper()
		cmd := localPlaygroundsCmd()
		cmd.SetArgs(args)
		stdout, err := captureLocalPathsStdout(t, cmd.Execute)
		if err != nil {
			t.Fatalf("%v: %v", args, err)
		}
		return stdout
	}
	for range 2 {
		run("link", "77") // No --link-dir: tools inherit the agent's shared path.
		var state localplaygrounds.CurrentState
		if err := json.Unmarshal([]byte(run("info", "--view", "current")), &state); err != nil {
			t.Fatal(err)
		}
		if state.ID != "77" || state.LinkDir != linkDir {
			t.Fatalf("current state=%#v", state)
		}
		var repos []localplaygrounds.RepoEntry
		if err := json.Unmarshal([]byte(run("info", "--view", "repos")), &repos); err != nil {
			t.Fatal(err)
		}
		if len(repos) != 1 || repos[0].LinkPath != filepath.Join(linkDir, "app") {
			t.Fatalf("repos=%#v", repos)
		}
		data, err := os.ReadFile(filepath.Join(repos[0].LinkPath, "source.txt"))
		if err != nil || string(data) != "agent edit" {
			t.Fatalf("linked source=%q err=%v", data, err)
		}
	}
	explicit := filepath.Join(t.TempDir(), "explicit")
	run("link", "77", "--link-dir", explicit)
	var state localplaygrounds.CurrentState
	if err := json.Unmarshal([]byte(run("info", "--view", "current", "--link-dir", explicit)), &state); err != nil {
		t.Fatal(err)
	}
	if state.LinkDir != explicit {
		t.Fatalf("explicit link directory=%q", state.LinkDir)
	}
}
