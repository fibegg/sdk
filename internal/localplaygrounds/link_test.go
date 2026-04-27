package localplaygrounds

import (
	"os"
	"path/filepath"
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
