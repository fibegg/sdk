package mcpserver

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/fibegg/sdk/internal/localplaygrounds"
)

func TestLocalPlaygroundInfoUsesSharedDirectoryWithoutArgument(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HOST_ROOT", root)
	configured := t.TempDir()
	t.Setenv("PLAYGROUNDS_DIR", configured)
	explicit := t.TempDir()
	for dir, name := range map[string]string{configured: "shared", explicit: "explicit"} {
		if err := os.WriteFile(filepath.Join(dir, ".current_playground.json"), []byte(`{"name":"`+name+`","repos":[{"service":"web","link_path":"`+dir+`/web"}]}`), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	srv := New(Config{ToolSet: "full", PipelineCacheSize: 4})
	if err := srv.RegisterAll(); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ directory, name string }{{"", "shared"}, {explicit, "explicit"}} {
		args := map[string]any{"view": "current"}
		if tc.directory != "" {
			args["link_dir"] = tc.directory
		}
		result, err := srv.dispatcher.dispatch(context.Background(), "fibe_local_playgrounds_info", args)
		if err != nil {
			t.Fatal(err)
		}
		if state := result.(*localplaygrounds.CurrentState); state.Name != tc.name {
			t.Fatalf("state=%#v want %q", state, tc.name)
		}
		args["view"] = "repos"
		result, err = srv.dispatcher.dispatch(context.Background(), "fibe_local_playgrounds_info", args)
		if err != nil {
			t.Fatal(err)
		}
		repos := result.([]localplaygrounds.RepoEntry)
		if len(repos) != 1 || repos[0].Service != "web" {
			t.Fatalf("repos=%#v", repos)
		}
	}
}
