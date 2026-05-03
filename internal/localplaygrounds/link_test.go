package localplaygrounds

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLinkCreatesSymlinksAndStateFile(t *testing.T) {
	root := t.TempDir()
	t.Setenv("PLAYROOMS_ROOT", root)
	pgDir := filepath.Join(root, "pg-123")
	if err := os.MkdirAll(pgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	compose := `services:
  web:
    image: nginx
    labels:
      fibe.gg/playspec: tower-defence
    volumes:
      - "/opt/fibe/playgrounds/pg-123/props/fibegg--tower-defence--77/main:/app"
`
	if err := os.WriteFile(filepath.Join(pgDir, "compose.yml"), []byte(compose), 0o644); err != nil {
		t.Fatal(err)
	}

	linkDir := filepath.Join(t.TempDir(), "playground")
	result, err := Link("tower-defence", linkDir)
	if err != nil {
		t.Fatalf("Link: %v", err)
	}

	if result.LinkDir != linkDir || result.Playground != "pg-123" {
		t.Fatalf("unexpected result: %#v", result)
	}
	if len(result.Links) != 1 {
		t.Fatalf("links=%d want 1", len(result.Links))
	}
	target, err := os.Readlink(result.Links[0].Path)
	if err != nil {
		t.Fatalf("readlink: %v", err)
	}
	if target != result.Links[0].Target {
		t.Fatalf("target=%s want %s", target, result.Links[0].Target)
	}
	state, err := os.ReadFile(result.StateFile)
	if err != nil {
		t.Fatalf("state: %v", err)
	}
	if string(state) != "pg-123" {
		t.Fatalf("state=%q want pg-123", state)
	}
}

func TestLinkPreservesExistingDirectoryAndClearsContents(t *testing.T) {
	parent := t.TempDir()
	linkDir := filepath.Join(parent, "playground")
	if err := os.MkdirAll(linkDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(linkDir, "stale.txt"), []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(linkDir, "stale-dir"), 0o755); err != nil {
		t.Fatal(err)
	}

	if err := os.Chmod(parent, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(parent, 0o755)
	})

	hostMount := filepath.Join(t.TempDir(), "main")
	pg := &Playground{
		DirName:  "pg-dynamic",
		Playspec: "dynamic-app",
		Services: map[string]*Service{
			"app": {
				Name:      "app",
				HostMount: hostMount,
				Prop:      "dynamic-app",
				Branch:    "main",
			},
		},
	}

	result, err := LinkPlayground(pg, linkDir)
	if err != nil {
		t.Fatalf("LinkPlayground: %v", err)
	}
	if len(result.Links) != 1 {
		t.Fatalf("links=%d want 1", len(result.Links))
	}
	if _, err := os.Stat(linkDir); err != nil {
		t.Fatalf("link dir was not preserved: %v", err)
	}
	if _, err := os.Lstat(filepath.Join(linkDir, "stale.txt")); !os.IsNotExist(err) {
		t.Fatalf("stale file still exists, err=%v", err)
	}
	if _, err := os.Lstat(filepath.Join(linkDir, "stale-dir")); !os.IsNotExist(err) {
		t.Fatalf("stale directory still exists, err=%v", err)
	}
	target, err := os.Readlink(filepath.Join(linkDir, "dynamic-app"))
	if err != nil {
		t.Fatalf("readlink: %v", err)
	}
	if target != hostMount {
		t.Fatalf("target=%s want %s", target, hostMount)
	}
}

func TestLinkStaticPlaygroundClearsContentsAndWritesState(t *testing.T) {
	linkDir := filepath.Join(t.TempDir(), "playground")
	if err := os.MkdirAll(linkDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(linkDir, "stale.txt"), []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("/tmp/old-target", filepath.Join(linkDir, "old-link")); err != nil {
		t.Fatal(err)
	}

	pg := &Playground{
		DirName:  "bagg-app--24",
		Playspec: "bagg-app",
		Services: map[string]*Service{
			"app": {
				Name:      "app",
				Image:     "bagg-app:phase0",
				Traefik:   true,
				Subdomain: "app",
			},
		},
	}

	result, err := LinkPlayground(pg, linkDir)
	if err != nil {
		t.Fatalf("LinkPlayground: %v", err)
	}
	if len(result.Links) != 0 {
		t.Fatalf("links=%d want 0", len(result.Links))
	}
	state, err := os.ReadFile(result.StateFile)
	if err != nil {
		t.Fatalf("state: %v", err)
	}
	if string(state) != "bagg-app--24" {
		t.Fatalf("state=%q want bagg-app--24", state)
	}

	entries, err := os.ReadDir(linkDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != ".current_playground" {
		t.Fatalf("entries=%v want only .current_playground", entries)
	}
}

func TestLinkFailsWhenTargetIsNotDirectory(t *testing.T) {
	linkDir := filepath.Join(t.TempDir(), "playground")
	if err := os.WriteFile(linkDir, []byte("not a directory"), 0o644); err != nil {
		t.Fatal(err)
	}

	pg := &Playground{DirName: "pg", Playspec: "pg", Services: map[string]*Service{}}
	_, err := LinkPlayground(pg, linkDir)
	if err == nil {
		t.Fatal("LinkPlayground succeeded, want error")
	}
	if !strings.Contains(err.Error(), "must be a directory") {
		t.Fatalf("error=%q want directory error", err.Error())
	}
}
