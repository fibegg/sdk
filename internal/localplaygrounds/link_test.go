package localplaygrounds

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
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
	var current CurrentState
	if err := json.Unmarshal(state, &current); err != nil {
		t.Fatalf("state json: %v", err)
	}
	if current.Name != "pg-123" || current.Playspec != "tower-defence" || len(current.Repos) != 1 {
		t.Fatalf("state=%q want pg-123 json with one repo", state)
	}
	if current.Repos[0].Service != "web" || current.Repos[0].RepoRoot != result.Links[0].Target {
		t.Fatalf("unexpected repo state: %#v", current.Repos[0])
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

func TestScanMissingBaseDirReturnsStructuredError(t *testing.T) {
	_, err := Scan(filepath.Join(t.TempDir(), "missing"))
	var missing *BaseDirMissingError
	if !errors.As(err, &missing) {
		t.Fatalf("expected BaseDirMissingError, got %T: %v", err, err)
	}
	if missing.ErrorCode() != "LOCAL_PLAYGROUNDS_DIR_MISSING" || missing.ErrorStatus() != 404 {
		t.Fatalf("code=%q status=%d", missing.ErrorCode(), missing.ErrorStatus())
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
      fibe.gg/port: 3000
      fibe.gg/visibility: external
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
	if len(urls) != 1 || urls[0].Service != "api" || urls[0].URL != "https://api.example.test" {
		t.Fatalf("unexpected urls: %#v", urls)
	}
	mounts := Mounts(pg)
	if len(mounts) != 1 || mounts[0].Service != "api" || mounts[0].Branch != "main" || mounts[0].Prop != "mcp-test-dev" {
		t.Fatalf("unexpected mounts: %#v", mounts)
	}
}

func TestNamesExcludeJobModeAndStaticOnlyPlaygrounds(t *testing.T) {
	root := t.TempDir()
	normalDir := filepath.Join(root, "normal-app--1")
	staticDir := filepath.Join(root, "static-app--4")
	jobMapDir := filepath.Join(root, "ci-map--2")
	jobArrayDir := filepath.Join(root, "ci-array--3")
	jobMetadataDir := filepath.Join(root, "ci-metadata--5")
	for _, dir := range []string{normalDir, staticDir, jobMapDir, jobArrayDir, jobMetadataDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	normalCompose := `services:
  web:
    image: nginx
    labels:
      fibe.gg/playspec: normal-app
    volumes:
      - "/opt/fibe/playgrounds/normal-app--1/props/fibegg--normal-app--77/main:/app"
`
	staticCompose := `services:
  web:
    image: nginx
    labels:
      fibe.gg/playspec: static-app
`
	jobMapCompose := `services:
  test:
    image: alpine
    labels:
      fibe.gg/playspec: ci-map
      fibe.gg/job_watch: "true"
    volumes:
      - "/opt/fibe/playgrounds/ci-map--2/props/fibegg--ci-map--78/main:/app"
`
	jobArrayCompose := `services:
  test:
    image: alpine
    labels:
      - "fibe.gg/playspec=ci-array"
      - "fibe.gg/job_watch=true"
    volumes:
      - "/opt/fibe/playgrounds/ci-array--3/props/fibegg--ci-array--79/main:/app"
`
	jobMetadataCompose := `x-fibe.gg:
  metadata:
    job_mode: true
services:
  test:
    image: alpine
    labels:
      fibe.gg/playspec: ci-metadata
      fibe.gg/job_watch: "false"
    volumes:
      - "/opt/fibe/playgrounds/ci-metadata--5/props/fibegg--ci-metadata--80/main:/app"
`
	if err := os.WriteFile(filepath.Join(normalDir, "compose.yml"), []byte(normalCompose), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(staticDir, "compose.yml"), []byte(staticCompose), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(jobMapDir, "compose.yml"), []byte(jobMapCompose), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(jobArrayDir, "compose.yml"), []byte(jobArrayCompose), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(jobMetadataDir, "compose.yml"), []byte(jobMetadataCompose), 0o644); err != nil {
		t.Fatal(err)
	}

	playgrounds, err := Scan(root)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	names := Names(playgrounds)
	if len(names) != 1 || names[0].Name != "normal-app--1" {
		t.Fatalf("names=%#v want only normal-app--1", names)
	}

	mapJob, err := Find(playgrounds, "ci-map")
	if err != nil {
		t.Fatalf("Find ci-map: %v", err)
	}
	if !mapJob.JobMode || !mapJob.Services["test"].JobWatch {
		t.Fatalf("map job metadata not set: %#v", mapJob)
	}
	arrayJob, err := Find(playgrounds, "ci-array")
	if err != nil {
		t.Fatalf("Find ci-array: %v", err)
	}
	if !arrayJob.JobMode || !arrayJob.Services["test"].JobWatch {
		t.Fatalf("array job metadata not set: %#v", arrayJob)
	}
	metadataJob, err := Find(playgrounds, "ci-metadata")
	if err != nil {
		t.Fatalf("Find ci-metadata: %v", err)
	}
	if !metadataJob.JobMode || metadataJob.Services["test"].JobWatch {
		t.Fatalf("compose metadata job mode not set independently from service job_watch: %#v", metadataJob)
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

func TestLinkAdoptsLegacyDirectoryAndReplacesContents(t *testing.T) {
	parent := t.TempDir()
	linkDir := filepath.Join(parent, "playground")
	if err := os.MkdirAll(linkDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(linkDir, currentStateFilename), []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("/tmp/old-target", filepath.Join(linkDir, "old-link")); err != nil {
		t.Fatal(err)
	}

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
	if _, err := os.Lstat(filepath.Join(linkDir, "old-link")); !os.IsNotExist(err) {
		t.Fatalf("old link still exists, err=%v", err)
	}
	target, err := os.Readlink(filepath.Join(linkDir, "dynamic-app"))
	if err != nil {
		t.Fatalf("readlink: %v", err)
	}
	if target != hostMount {
		t.Fatalf("target=%s want %s", target, hostMount)
	}
	state, err := LoadCurrentState(linkDir)
	if err != nil {
		t.Fatalf("LoadCurrentState: %v", err)
	}
	if state.Name != "pg-dynamic" || len(state.Repos) != 1 {
		t.Fatalf("unexpected state: %#v", state)
	}
	if state.Repos[0].LinkPath != filepath.Join(linkDir, "dynamic-app") || state.Repos[0].RepoRoot != hostMount {
		t.Fatalf("unexpected repo state: %#v", state.Repos[0])
	}
}

func TestLinkStaticPlaygroundReplacesLegacyContentsAndWritesState(t *testing.T) {
	linkDir := filepath.Join(t.TempDir(), "playground")
	if err := os.MkdirAll(linkDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(linkDir, currentStateFilename), []byte("{}\n"), 0o644); err != nil {
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
	var current CurrentState
	if err := json.Unmarshal(state, &current); err != nil {
		t.Fatalf("state json: %v", err)
	}
	if current.Name != "bagg-app--24" || current.Playspec != "bagg-app" {
		t.Fatalf("state=%q want bagg-app--24 json", state)
	}

	entries, err := os.ReadDir(linkDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 || entries[0].Name() != currentStateFilename || entries[1].Name() != linkOwnerFilename {
		t.Fatalf("entries=%v want state and ownership marker", entries)
	}
}

func TestLinkRejectsUnownedDirectoryWithoutChangingIt(t *testing.T) {
	linkDir := filepath.Join(t.TempDir(), "playground")
	if err := os.MkdirAll(linkDir, 0o755); err != nil {
		t.Fatal(err)
	}
	keep := filepath.Join(linkDir, "keep.txt")
	if err := os.WriteFile(keep, []byte("important"), 0o644); err != nil {
		t.Fatal(err)
	}

	pg := &Playground{DirName: "pg", Playspec: "pg", Services: map[string]*Service{}}
	_, err := LinkPlayground(pg, linkDir)
	if err == nil || !strings.Contains(err.Error(), "non-empty unowned directory") {
		t.Fatalf("error=%v want unowned directory rejection", err)
	}
	if data, readErr := os.ReadFile(keep); readErr != nil || string(data) != "important" {
		t.Fatalf("existing data changed: data=%q err=%v", data, readErr)
	}
}

func TestLinkRejectsUnsafeComponents(t *testing.T) {
	for _, field := range []struct {
		name   string
		prop   string
		branch string
	}{
		{name: "prop traversal", prop: "../escape"},
		{name: "dot", prop: "."},
	} {
		t.Run(field.name, func(t *testing.T) {
			linkDir := filepath.Join(t.TempDir(), "playground")
			pg := &Playground{DirName: "pg", Services: map[string]*Service{
				"app": {Name: "app", HostMount: t.TempDir(), Prop: field.prop, Branch: field.branch},
			}}
			if _, err := LinkPlayground(pg, linkDir); err == nil {
				t.Fatal("LinkPlayground succeeded, want component error")
			}
			if _, err := os.Lstat(linkDir); !os.IsNotExist(err) {
				t.Fatalf("link directory created on validation failure: %v", err)
			}
		})
	}
}

func TestLinkSanitizesBranchSeparatorsWithoutLosingTheBranchSuffix(t *testing.T) {
	pg := &Playground{DirName: "pg", Services: map[string]*Service{
		"main":    {Name: "main", HostMount: "/sources/main", Prop: "app", Branch: "main", Repository: "github.com/acme/app"},
		"feature": {Name: "feature", HostMount: "/sources/feature", Prop: "app", Branch: "feature/x", Repository: "github.com/acme/app"},
	}}
	result, err := LinkPlayground(pg, filepath.Join(t.TempDir(), "playground"))
	if err != nil {
		t.Fatalf("LinkPlayground: %v", err)
	}
	names := []string{result.Links[0].Name, result.Links[1].Name}
	sort.Strings(names)
	if names[0] != "app-feature-x-217d2bf5" || names[1] != "app-main" {
		t.Fatalf("names=%v want stable sanitized branch suffix", names)
	}
}

func TestLinkRejectsDangerousDirectories(t *testing.T) {
	pg := &Playground{DirName: "pg", Services: map[string]*Service{}}
	if _, err := LinkPlayground(pg, string(filepath.Separator)); err == nil {
		t.Fatal("filesystem root accepted")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	if _, err := LinkPlayground(pg, home); err == nil {
		t.Fatal("home directory accepted")
	}
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := LinkPlayground(pg, cwd); err == nil {
		t.Fatal("current directory accepted")
	}
}

func TestLinkRejectsSymlinkedAncestor(t *testing.T) {
	parent := t.TempDir()
	realDir := filepath.Join(parent, "real")
	if err := os.Mkdir(realDir, 0o755); err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(parent, "alias")
	if err := os.Symlink(realDir, alias); err != nil {
		t.Fatal(err)
	}
	pg := &Playground{DirName: "pg", Services: map[string]*Service{}}
	if _, err := LinkPlayground(pg, filepath.Join(alias, "playground")); err == nil {
		t.Fatal("symlinked ancestor accepted")
	}
}

func TestLinkRejectsJobModeWithoutClearingTarget(t *testing.T) {
	linkDir := filepath.Join(t.TempDir(), "playground")
	if err := os.MkdirAll(linkDir, 0o755); err != nil {
		t.Fatal(err)
	}
	stalePath := filepath.Join(linkDir, "stale.txt")
	if err := os.WriteFile(stalePath, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}

	pg := &Playground{
		DirName:  "ci-fibeagent--24",
		Playspec: "ci-fibeagent",
		JobMode:  true,
		Services: map[string]*Service{
			"test": {
				Name:     "test",
				JobWatch: true,
			},
		},
	}

	_, err := LinkPlayground(pg, linkDir)
	if err == nil {
		t.Fatal("LinkPlayground succeeded, want error")
	}
	if !strings.Contains(err.Error(), "cannot link job-mode playground") {
		t.Fatalf("error=%q want job-mode error", err.Error())
	}
	if data, err := os.ReadFile(stalePath); err != nil || string(data) != "old" {
		t.Fatalf("stale target changed, data=%q err=%v", data, err)
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

func TestLinkNamingMatrixAndSharedCheckoutDeduplication(t *testing.T) {
	tests := []struct {
		name     string
		services map[string]*Service
		want     []string
	}{
		{
			name: "one repository and branch",
			services: map[string]*Service{
				"app":   {Name: "app", HostMount: "/sources/app-main", Prop: "app", Branch: "main", Repository: "github.com/acme/app"},
				"setup": {Name: "setup", HostMount: "/sources/app-main", Prop: "app", Branch: "main", Repository: "github.com/acme/app"},
			},
			want: []string{"app"},
		},
		{
			name: "several repositories on one branch",
			services: map[string]*Service{
				"app":    {Name: "app", HostMount: "/sources/app-main", Prop: "app", Branch: "main", Repository: "github.com/acme/app"},
				"worker": {Name: "worker", HostMount: "/sources/worker-main", Prop: "worker", Branch: "main", Repository: "github.com/acme/worker"},
			},
			want: []string{"app", "worker"},
		},
		{
			name: "more than one branch suffixes every link",
			services: map[string]*Service{
				"app":     {Name: "app", HostMount: "/sources/app-main", Prop: "app", Branch: "main", Repository: "github.com/acme/app"},
				"preview": {Name: "preview", HostMount: "/sources/app-preview", Prop: "app", Branch: "preview", Repository: "github.com/acme/app"},
				"worker":  {Name: "worker", HostMount: "/sources/worker-main", Prop: "worker", Branch: "main", Repository: "github.com/acme/worker"},
			},
			want: []string{"app-main", "app-preview", "worker-main"},
		},
		{
			name: "duplicate repository names qualify with owner",
			services: map[string]*Service{
				"first":  {Name: "first", HostMount: "/sources/acme-app", Prop: "app", Branch: "main", Repository: "github.com/acme/app"},
				"second": {Name: "second", HostMount: "/sources/other-app", Prop: "app", Branch: "main", Repository: "github.com/other/app"},
			},
			want: []string{"app-acme", "app-other"},
		},
		{
			name: "duplicate owners qualify with stable repository digest",
			services: map[string]*Service{
				"first":  {Name: "first", HostMount: "/sources/github-app", Prop: "app", Branch: "main", Repository: "github.com/acme/app"},
				"second": {Name: "second", HostMount: "/sources/gitlab-app", Prop: "app", Branch: "main", Repository: "gitlab.com/acme/app"},
			},
			want: []string{"app-acme-31d8e159", "app-acme-98d9fc12"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			linkDir := filepath.Join(t.TempDir(), "playground")
			result, err := LinkPlayground(&Playground{DirName: "matrix", Services: tt.services}, linkDir)
			if err != nil {
				t.Fatalf("LinkPlayground: %v", err)
			}
			got := make([]string, 0, len(result.Links))
			for _, link := range result.Links {
				got = append(got, link.Name)
			}
			sort.Strings(got)
			if strings.Join(got, ",") != strings.Join(tt.want, ",") {
				t.Fatalf("links=%v want %v", got, tt.want)
			}
		})
	}
}

func TestLinkRejectsDuplicatePhysicalCheckoutsForOneRepositoryBranch(t *testing.T) {
	pg := &Playground{DirName: "ambiguous", Services: map[string]*Service{
		"app":   {Name: "app", HostMount: "/sources/first", Prop: "app", Branch: "main", Repository: "github.com/acme/app"},
		"setup": {Name: "setup", HostMount: "/sources/second", Prop: "app", Branch: "main", Repository: "github.com/acme/app"},
	}}

	_, err := LinkPlayground(pg, filepath.Join(t.TempDir(), "playground"))
	if err == nil || !strings.Contains(err.Error(), "multiple physical checkouts") {
		t.Fatalf("error=%v want duplicate checkout rejection", err)
	}
}

func TestScanPrefersCoreSourcePlanManifest(t *testing.T) {
	root := t.TempDir()
	pgDir := filepath.Join(root, "core--42")
	if err := os.MkdirAll(pgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	compose := `services:
  app:
    image: app
    labels:
      fibe.gg/repo_url: https://token:secret@github.com/acme/app.git
      fibe.gg/source_mount: /rails
    volumes:
      - ./wrong:/rails
      - ./also-wrong:/rails/
  setup:
    image: app
    labels:
      fibe.gg/repo_url: git@github.com:acme/app.git
      fibe.gg/source_mount: /workspace
    volumes:
      - ./also-wrong:/workspace
  production:
    image: app
    labels:
      fibe.gg/repo_url: https://github.com/acme/app.git
      fibe.gg/source_mount: /rails
    volumes:
      - customer:/rails
`
	manifest := `{
  "version": 1,
  "sources": [{
    "repository": "github.com/acme/app",
    "branch": "main",
    "relative_path": "props/acme--app--0123456789/main--0123456789",
    "services": [
      {"service": "app", "production": false, "mount_target": "/rails"},
      {"service": "setup", "production": false, "mount_target": "/workspace"},
      {"service": "production", "production": true, "mount_target": "/rails"}
    ]
  }]
}`
	if err := os.WriteFile(filepath.Join(pgDir, "compose.yml"), []byte(compose), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pgDir, sourcePlanFilename), []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}

	playgrounds, err := Scan(root)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	pg := &playgrounds[0]
	wantCheckout := filepath.Join(pgDir, "props/acme--app--0123456789/main--0123456789")
	if pg.Services["app"].HostMount != wantCheckout || pg.Services["setup"].HostMount != wantCheckout {
		t.Fatalf("manifest checkout not applied: app=%q setup=%q", pg.Services["app"].HostMount, pg.Services["setup"].HostMount)
	}
	if pg.Services["production"].HostMount != "" {
		t.Fatalf("production mount=%q want empty", pg.Services["production"].HostMount)
	}
	if pg.Services["app"].Repository != "github.com/acme/app" || strings.Contains(pg.Services["app"].Repository, "secret") {
		t.Fatalf("repository metadata=%q", pg.Services["app"].Repository)
	}

	result, err := LinkPlayground(pg, filepath.Join(t.TempDir(), "playground"))
	if err != nil {
		t.Fatalf("LinkPlayground: %v", err)
	}
	if len(result.Links) != 1 || result.Links[0].Name != "app" {
		t.Fatalf("links=%#v want one app link", result.Links)
	}
	state, err := LoadCurrentState(result.LinkDir)
	if err != nil {
		t.Fatalf("LoadCurrentState: %v", err)
	}
	if state.Repos[0].NormalizedRepository != "github.com/acme/app" || strings.Join(state.Repos[0].Services, ",") != "app,production,setup" {
		t.Fatalf("repo metadata=%#v", state.Repos[0])
	}
	encoded, err := json.Marshal(result.Links[0])
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(encoded), `"normalized_repository":"github.com/acme/app"`) || !strings.Contains(string(encoded), `"services":["app","production","setup"]`) {
		t.Fatalf("linked path JSON=%s", encoded)
	}
}

func TestScanSelectsExactSourceMountTargetAndResolvesRelativeSource(t *testing.T) {
	root := t.TempDir()
	pgDir := filepath.Join(root, "relative--7")
	if err := os.MkdirAll(pgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	compose := `services:
  app:
    image: app
    labels:
      fibe.gg/repo_url: https://github.com/acme/app
      fibe.gg/branch: main
      fibe.gg/source_mount: /rails/
    volumes:
      - ./contains/props/unrelated:/cache
      - ${OTHER:-./wrong}:/other
      - type: bind
        source: ./source
        target: /rails/.
`
	if err := os.WriteFile(filepath.Join(pgDir, "compose.yml"), []byte(compose), 0o600); err != nil {
		t.Fatal(err)
	}

	playgrounds, err := Scan(root)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	service := playgrounds[0].Services["app"]
	if service.HostMount != filepath.Join(pgDir, "source") {
		t.Fatalf("HostMount=%q want relative exact-target source", service.HostMount)
	}
	if service.Prop != "app" || service.Branch != "main" {
		t.Fatalf("service metadata=%#v", service)
	}
}

func TestScanRejectsAmbiguousExactTargetMounts(t *testing.T) {
	root := t.TempDir()
	pgDir := filepath.Join(root, "ambiguous--8")
	if err := os.MkdirAll(pgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	compose := `services:
  app:
    labels:
      fibe.gg/repo_url: https://github.com/acme/app
      fibe.gg/source_mount: /rails
    volumes:
      - ./first:/rails
      - ./second:/rails/
`
	if err := os.WriteFile(filepath.Join(pgDir, "compose.yml"), []byte(compose), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := Scan(root)
	if err == nil || !strings.Contains(err.Error(), "multiple mounts target") {
		t.Fatalf("error=%v want exact-target ambiguity", err)
	}
}
