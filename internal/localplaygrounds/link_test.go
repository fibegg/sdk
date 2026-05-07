package localplaygrounds

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLinkCreatesSymlinksAndStateFile(t *testing.T) {
	root := t.TempDir()
	t.Setenv("MARQUEE_ROOT", root)
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

func TestBaseDirResolvesMarqueeRootPlaygroundsSubdirectory(t *testing.T) {
	root := t.TempDir()
	playgroundsRoot := filepath.Join(root, "playgrounds")
	pgDir := filepath.Join(playgroundsRoot, "pg-123")
	if err := os.MkdirAll(pgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pgDir, "compose.yml"), []byte("services: {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("MARQUEE_ROOT", root)

	if got := BaseDir(); got != playgroundsRoot {
		t.Fatalf("BaseDir()=%q want %q", got, playgroundsRoot)
	}
}

func TestScanViewsAndIDResolution(t *testing.T) {
	root := t.TempDir()
	pgDir := filepath.Join(root, "mcp-test-dev--42")
	if err := os.MkdirAll(pgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	compose := `services:
  api:
    image: ruby:latest
    labels:
      fibe.gg/playspec: mcp-test-dev
      fibe.gg/playground: mcp-test-dev--42
      fibe.gg/subdomain: api
      fibe.gg/expose: public:3000
      fibe.gg/start_command: "bin/dev"
      traefik.enable: "true"
    volumes:
      - type: bind
        source: /opt/fibe/playgrounds/mcp-test-dev--42/props/viktorvsk--mcp-test-dev--5/main
        target: /app
  worker:
    image: alpine
    labels:
      - "fibe.gg/playspec=mcp-test-dev"
      - "traefik.enable=false"
`
	if err := os.WriteFile(filepath.Join(pgDir, "compose.yml"), []byte(compose), 0o644); err != nil {
		t.Fatal(err)
	}

	playgrounds, err := Scan(root)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(playgrounds) != 1 {
		t.Fatalf("playgrounds=%d want 1", len(playgrounds))
	}
	pg, err := Find(playgrounds, "42")
	if err != nil {
		t.Fatalf("Find by ID: %v", err)
	}
	if pg.ID != "42" || pg.DirName != "mcp-test-dev--42" || pg.Playspec != "mcp-test-dev" {
		t.Fatalf("unexpected playground: %#v", pg)
	}

	names := Names(playgrounds)
	if len(names) != 1 || names[0].ID != "42" || names[0].Name != "mcp-test-dev--42" || names[0].Path != pgDir {
		t.Fatalf("unexpected names: %#v", names)
	}
	urls := URLs(pg, "example.test")
	if len(urls) != 1 || urls[0].Service != "api" || urls[0].URL != "api.example.test" {
		t.Fatalf("unexpected urls: %#v", urls)
	}
	mounts := Mounts(pg)
	if len(mounts) != 1 || mounts[0].Service != "api" || mounts[0].Branch != "main" || mounts[0].Prop != "mcp-test-dev" {
		t.Fatalf("unexpected mounts: %#v", mounts)
	}
}

func TestFindUsesPlaygroundLabelIDFallback(t *testing.T) {
	root := t.TempDir()
	pgDir := filepath.Join(root, "compose-without-id")
	if err := os.MkdirAll(pgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	compose := `services:
  web:
    image: nginx
    labels:
      fibe.gg/playspec: fallback-app
      fibe.gg/playground: fallback-app--77
`
	if err := os.WriteFile(filepath.Join(pgDir, "compose.yml"), []byte(compose), 0o644); err != nil {
		t.Fatal(err)
	}

	playgrounds, err := Scan(root)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	pg, err := Find(playgrounds, "77")
	if err != nil {
		t.Fatalf("Find by fallback ID: %v", err)
	}
	if pg.DirName != "compose-without-id" {
		t.Fatalf("DirName=%s want compose-without-id", pg.DirName)
	}
}

func TestFindRejectsAmbiguousPlayspecPrefix(t *testing.T) {
	playgrounds := []Playground{
		{ID: "1", DirName: "alpha--1", Playspec: "suite-app"},
		{ID: "2", DirName: "beta--2", Playspec: "suite-api"},
	}
	_, err := Find(playgrounds, "suite")
	if err == nil {
		t.Fatal("Find succeeded, want ambiguous error")
	}
	if !strings.Contains(err.Error(), "multiple playgrounds found matching") {
		t.Fatalf("error=%q want ambiguous match", err.Error())
	}
}

func TestMountsReturnsOnlyMountableServices(t *testing.T) {
	pg := &Playground{
		DirName:  "static-site--24",
		Playspec: "static-site",
		Services: map[string]*Service{
			"web": {Name: "web", Image: "nginx"},
		},
	}
	if mounts := Mounts(pg); len(mounts) != 0 {
		t.Fatalf("mounts=%#v want empty", mounts)
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
