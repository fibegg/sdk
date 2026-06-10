package mcpserver

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fibegg/sdk/fibe"
)

func TestLaunchArgsAcceptComposeYAML(t *testing.T) {
	params, err := launchArgs(context.Background(), nil, map[string]any{
		"name":            "todo",
		"compose_yaml":    "services:\n  web:\n    image: nginx\n",
		"persist_volumes": true,
	})
	if err != nil {
		t.Fatalf("launchArgs: %v", err)
	}
	if params.Name != "todo" || params.ComposeYAML == "" {
		t.Fatalf("unexpected params: %#v", params)
	}
	if params.PersistVolumes == nil || *params.PersistVolumes != true {
		t.Fatalf("persist volumes = %#v, want true", params.PersistVolumes)
	}
}

func TestLaunchArgsAcceptGitHubRepository(t *testing.T) {
	client := githubInstallationTestClient(t)
	params, err := launchArgs(context.Background(), client, map[string]any{
		"repository_url": "owner/repo@feature/foo",
		"config_path":    "deploy/fibe.yml",
	})
	if err != nil {
		t.Fatalf("launchArgs: %v", err)
	}
	if params.Name != "repo" || params.RepositoryURL != "https://github.com/owner/repo" {
		t.Fatalf("unexpected repository params: %#v", params)
	}
	if params.GitHubRef != "feature/foo" || params.ConfigPath != "deploy/fibe.yml" {
		t.Fatalf("unexpected config selector params: %#v", params)
	}
	if params.GitHubInstallationID == nil || *params.GitHubInstallationID != 123 {
		t.Fatalf("unexpected github installation: %#v", params.GitHubInstallationID)
	}
}

func TestLaunchToolRegisteredAsGreenfield(t *testing.T) {
	srv := New(Config{APIKey: "pk_test", ToolSet: "core"})
	if err := srv.RegisterAll(); err != nil {
		t.Fatalf("RegisterAll: %v", err)
	}
	tool, ok := srv.dispatcher.lookup("fibe_launch_create")
	if !ok {
		t.Fatal("fibe_launch_create not registered")
	}
	if tool.tier != tierGreenfield {
		t.Fatalf("tier=%v want greenfield", tool.tier)
	}
}

func githubInstallationTestClient(t *testing.T) *fibe.Client {
	t.Helper()
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/installations" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"id":7,"provider":"github","installation_id":123,"installation_account":"owner"}],"meta":{"total":1}}`))
	}))
	t.Cleanup(api.Close)
	return fibe.NewClient(fibe.WithBaseURL(api.URL), fibe.WithAPIKey("pk_test"), fibe.WithMaxRetries(0))
}
