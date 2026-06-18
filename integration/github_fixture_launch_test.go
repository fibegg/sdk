package integration

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/fibegg/sdk/fibe"
)

const (
	githubFixtureBackendRepo  = "https://github.com/fibegg-fixtures/backend"
	githubFixtureFrontendRepo = "https://github.com/fibegg-fixtures/frontend"
	githubFixtureConfigPath   = "docker-compose.yml"
	githubFixtureRef          = "main"
	githubFixtureComposeURL   = "https://raw.githubusercontent.com/fibegg-fixtures/backend/main/docker-compose.yml"
)

var githubFixtureRepos = []string{githubFixtureBackendRepo, githubFixtureFrontendRepo}

type githubFixtureAccess struct {
	writable bool
	reason   string
}

func TestDockerE2EGitHubFixtureLaunchWorkflow(t *testing.T) {
	if !envBool("FIBE_E2E_BOOTSTRAP") && !envBool("SDK_E2E_BOOTSTRAP") && !envBool("FIBE_GITHUB_FIXTURE_E2E") {
		t.Skip("Docker E2E GitHub fixture launch test requires FIBE_E2E_BOOTSTRAP, SDK_E2E_BOOTSTRAP, or FIBE_GITHUB_FIXTURE_E2E=1")
	}

	pat := githubFixturePAT(t)
	c := userClient(t)
	marqueeID := testMarqueeID(t)
	if marqueeID == 0 {
		t.Skip("FIBE_TEST_MARQUEE_ID or an active Marquee is required for GitHub fixture launch tests")
	}

	attachGitHubPATToCurrentPlayer(t, pat)
	writeAccess := githubFixturesWriteAccess(t, pat)

	t.Run("repository source launch uses branch path variables subdomains env and service overrides", func(t *testing.T) {
		create := true
		persist := false
		apiSubdomain := uniqueName("sdk-fixture-api")
		frontendSubdomain := uniqueName("sdk-fixture-frontend")
		params := &fibe.LaunchParams{
			Name:             uniqueName("sdk-fixture-repo"),
			RepositoryURL:    githubFixtureBackendRepo,
			ConfigPath:       githubFixtureConfigPath,
			GitHubRef:        githubFixtureRef,
			MarqueeID:        &marqueeID,
			CreatePlayground: &create,
			PersistVolumes:   &persist,
			Variables: map[string]string{
				"API_SUBDOMAIN": apiSubdomain,
				"SUBDOMAIN":     frontendSubdomain,
				"INTERVAL":      "8",
				"VITE_API":      "https://" + apiSubdomain + ".fixture.e2e.invalid",
			},
			EnvOverrides: map[string]string{
				"FIBE_FIXTURE_MODE":  "sdk-repo-source",
				"FIBE_FIXTURE_SUITE": "sdk",
			},
			ServiceSubdomains: map[string]string{
				"api":      apiSubdomain,
				"frontend": frontendSubdomain,
			},
			Services: map[string]any{
				"api":      map[string]any{"fixture_marker": "sdk-repo-api"},
				"frontend": map[string]any{"fixture_marker": "sdk-repo-frontend"},
			},
		}
		result, err := c.Launch.Create(ctx(), params)
		if !writeAccess.writable {
			requireRuntimeWriteRejected(t, err, "launch GitHub fixture by repository URL", writeAccess.reason)
			requireRepoRequiresFork(t, c, githubFixtureRepos)
			return
		}
		requireNoError(t, err, "launch GitHub fixture by repository URL")
		cleanupLaunchResult(t, c, result)
		requireLaunchIDs(t, result)

		status := waitForPlaygroundStatusWithin(t, c, result.PlaygroundID, []string{"running", "completed", "error", "failed"}, 8*time.Minute)
		if status != "running" && status != "completed" {
			t.Fatalf("repository-source fixture playground reached %q, want running/completed", status)
		}
		pg, err := c.Playgrounds.Get(ctx(), result.PlaygroundID)
		requireNoError(t, err, "get repository-source fixture playground")
		if pg.EnvOverrides["FIBE_FIXTURE_MODE"] != "sdk-repo-source" {
			t.Fatalf("env override not persisted: %#v", pg.EnvOverrides)
		}
		requireRepoRuntimeWritable(t, c, githubFixtureRepos)
	})

	t.Run("pasted compose requires writable fixture props and rejects read-only public attachments", func(t *testing.T) {
		composeYAML := fetchGitHubFixtureCompose(t, pat)

		if !writeAccess.writable {
			requireFixturePropCreateRejected(t, c, githubFixtureBackendRepo, "sdk-fixture-backend-readonly", writeAccess.reason)
			create := false
			persist := false
			_, err := c.Launch.Create(ctx(), &fibe.LaunchParams{
				Name:             uniqueName("sdk-fixture-pasted-readonly"),
				ComposeYAML:      composeYAML,
				MarqueeID:        &marqueeID,
				CreatePlayground: &create,
				PersistVolumes:   &persist,
				Variables: map[string]string{
					"API_SUBDOMAIN": uniqueName("sdk-pasted-readonly-api"),
					"SUBDOMAIN":     uniqueName("sdk-pasted-readonly-frontend"),
					"INTERVAL":      "9",
					"VITE_API":      "https://sdk-pasted-readonly.fixture.e2e.invalid",
				},
			})
			requireRuntimeWriteRejected(t, err, "launch pasted compose with read-only public repos", writeAccess.reason)
			requireRepoRequiresFork(t, c, githubFixtureRepos)
			return
		}

		createFixturePropWithoutCredentials(t, c, githubFixtureBackendRepo, "sdk-fixture-backend-prop")
		createFixturePropWithoutCredentials(t, c, githubFixtureFrontendRepo, "sdk-fixture-frontend-prop")
		requireRepoRuntimeWritable(t, c, githubFixtureRepos)

		create := false
		persist := false
		result, err := c.Launch.Create(ctx(), &fibe.LaunchParams{
			Name:             uniqueName("sdk-fixture-pasted"),
			ComposeYAML:      composeYAML,
			MarqueeID:        &marqueeID,
			CreatePlayground: &create,
			PersistVolumes:   &persist,
			Variables: map[string]string{
				"API_SUBDOMAIN": uniqueName("sdk-pasted-api"),
				"SUBDOMAIN":     uniqueName("sdk-pasted-frontend"),
				"INTERVAL":      "9",
				"VITE_API":      "https://sdk-pasted.fixture.e2e.invalid",
			},
			EnvOverrides: map[string]string{
				"FIBE_FIXTURE_MODE":  "sdk-pasted-compose",
				"FIBE_FIXTURE_SUITE": "sdk",
			},
		})
		requireNoError(t, err, "launch GitHub fixture by pasted compose")
		cleanupLaunchResult(t, c, result)
		requirePlayspecID(t, result)
		requireRepoRuntimeWritable(t, c, githubFixtureRepos)

		pg, err := c.Playgrounds.Create(ctx(), &fibe.PlaygroundCreateParams{
			Name:       uniqueName("sdk-fixture-pasted-pg"),
			PlayspecID: result.PlayspecID,
			MarqueeID:  &marqueeID,
		})
		requireNoError(t, err, "create playground from pasted-compose fixture playspec")
		t.Cleanup(func() { _ = c.Playgrounds.Delete(ctx(), pg.ID) })

		status := waitForPlaygroundStatusWithin(t, c, pg.ID, []string{"running", "completed", "error", "failed"}, 8*time.Minute)
		if status != "running" && status != "completed" {
			t.Fatalf("pasted-compose fixture playground reached %q, want running/completed", status)
		}
	})
}

func githubFixturePAT(t *testing.T) string {
	t.Helper()
	pat := strings.TrimSpace(os.Getenv("GITHUB_PAT"))
	if pat == "" {
		t.Skip("GITHUB_PAT is required for real GitHub fixture launch tests")
	}
	return pat
}

func attachGitHubPATToCurrentPlayer(t *testing.T, pat string) {
	t.Helper()
	baseURL := strings.TrimRight(firstEnv("FIBE_DOMAIN", "FIBE_URL", "FIBE_BASE_URL"), "/")
	if baseURL == "" {
		baseURL = "http://localhost:3000"
	}
	adminToken := firstEnv("FIBE_ADMIN_API_KEY", "E2E_ADMIN_API_KEY")
	if adminToken == "" {
		adminToken = defaultE2EAdminAPIKey
	}
	client := &e2eBootstrapClient{baseURL: baseURL, adminToken: adminToken, http: &http.Client{Timeout: 120 * time.Second}}

	player, err := client.currentPlayer(os.Getenv("FIBE_API_KEY"))
	requireNoError(t, err, "resolve current SDK e2e player")

	var payload struct {
		Success bool `json:"success"`
	}
	err = client.requestJSON(http.MethodPost, "/e2e_backdoor/operation", adminToken, map[string]any{
		"operation":         "attach_github_token",
		"player_id":         player.ID,
		"access_token":      pat,
		"provider_username": "fibegg-fixtures",
		"provider_user_id":  fmt.Sprintf("fibegg-fixtures-sdk-%d", player.ID),
	}, &payload, http.StatusOK)
	requireNoError(t, err, "attach GitHub PAT to current SDK e2e player")
	if !payload.Success {
		t.Fatal("attach_github_token returned success=false")
	}
}

func fetchGitHubFixtureCompose(t *testing.T, pat string) string {
	t.Helper()
	req, err := http.NewRequestWithContext(ctx(), http.MethodGet, githubFixtureComposeURL, nil)
	requireNoError(t, err, "build fixture compose request")
	req.Header.Set("Accept", "text/plain")
	req.Header.Set("Authorization", "Bearer "+pat)

	resp, err := http.DefaultClient.Do(req)
	requireNoError(t, err, "fetch fixture compose")
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	requireNoError(t, err, "read fixture compose")
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		t.Fatalf("fetch fixture compose returned HTTP %d: %s", resp.StatusCode, string(raw))
	}
	body := string(raw)
	if !strings.Contains(body, "fibegg-fixtures/backend") || !strings.Contains(body, "fibegg-fixtures/frontend") || !strings.Contains(body, "x-fibe.gg:") {
		t.Fatalf("fixture compose did not contain expected Fibe fixture markers")
	}
	return body
}

func githubFixturesWriteAccess(t *testing.T, pat string) githubFixtureAccess {
	t.Helper()
	reasons := make([]string, 0, len(githubFixtureRepos))
	allWritable := true
	for _, repoURL := range githubFixtureRepos {
		access := githubFixtureWriteAccess(t, pat, repoURL)
		if !access.writable {
			allWritable = false
		}
		reasons = append(reasons, fmt.Sprintf("%s: %s", repoURL, access.reason))
	}
	return githubFixtureAccess{writable: allWritable, reason: strings.Join(reasons, "; ")}
}

func githubFixtureWriteAccess(t *testing.T, pat, repoURL string) githubFixtureAccess {
	t.Helper()
	fullName := strings.TrimSuffix(strings.TrimPrefix(repoURL, "https://github.com/"), ".git")
	req, err := http.NewRequestWithContext(ctx(), http.MethodGet, "https://api.github.com/repos/"+fullName, nil)
	requireNoError(t, err, "build GitHub permission request")
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", "Bearer "+pat)
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("User-Agent", "Fibe-SDK-E2E")

	resp, err := http.DefaultClient.Do(req)
	requireNoError(t, err, "check GitHub fixture write access")
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	requireNoError(t, err, "read GitHub permission response")
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return githubFixtureAccess{writable: false, reason: fmt.Sprintf("GitHub returned HTTP %d for %s", resp.StatusCode, fullName)}
	}

	var payload struct {
		Permissions map[string]any `json:"permissions"`
	}
	requireNoError(t, json.Unmarshal(raw, &payload), "parse GitHub permission response")
	writable := permissionTruthy(payload.Permissions["push"]) ||
		permissionTruthy(payload.Permissions["admin"]) ||
		permissionTruthy(payload.Permissions["maintain"])
	if writable {
		return githubFixtureAccess{writable: true, reason: "token has push permission"}
	}
	return githubFixtureAccess{writable: false, reason: "token can read " + fullName + " but cannot push"}
}

func permissionTruthy(value any) bool {
	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		return strings.EqualFold(typed, "true")
	default:
		return false
	}
}

func createFixturePropWithoutCredentials(t *testing.T, c *fibe.Client, repoURL, prefix string) *fibe.Prop {
	t.Helper()
	name := uniqueName(prefix)
	provider := "github"
	branch := githubFixtureRef
	private := false
	prop, err := c.Props.Create(ctx(), &fibe.PropCreateParams{
		RepositoryURL: repoURL,
		Name:          &name,
		Provider:      &provider,
		DefaultBranch: &branch,
		Private:       &private,
	})
	requireNoError(t, err, "create fixture prop without credentials")
	t.Cleanup(func() { _ = c.Props.Delete(ctx(), prop.ID) })
	return prop
}

func requireFixturePropCreateRejected(t *testing.T, c *fibe.Client, repoURL, prefix, accessReason string) {
	t.Helper()
	name := uniqueName(prefix)
	provider := "github"
	branch := githubFixtureRef
	private := false
	_, err := c.Props.Create(ctx(), &fibe.PropCreateParams{
		RepositoryURL: repoURL,
		Name:          &name,
		Provider:      &provider,
		DefaultBranch: &branch,
		Private:       &private,
	})
	requireRuntimeWriteRejected(t, err, "create fixture prop without runtime write permission", accessReason)
}

func requireRuntimeWriteRejected(t *testing.T, err error, label, accessReason string) {
	t.Helper()
	if err == nil {
		t.Fatalf("%s unexpectedly succeeded with read-only fixture token; access=%s", label, accessReason)
	}
	apiErr, ok := err.(*fibe.APIError)
	if !ok {
		t.Fatalf("%s returned non-API error %T: %v", label, err, err)
	}
	if apiErr.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("%s returned HTTP %d, want 422: %v", label, apiErr.StatusCode, apiErr)
	}
	lower := strings.ToLower(apiErr.Error())
	if !strings.Contains(lower, "push") && !strings.Contains(lower, "writable") && !strings.Contains(lower, "fork") {
		t.Fatalf("%s error did not explain write/fork requirement: %v; access=%s", label, apiErr, accessReason)
	}
}

func requireRepoRuntimeWritable(t *testing.T, c *fibe.Client, urls []string) {
	t.Helper()
	requireRepoRuntimeWritableValue(t, c, urls, true)
}

func requireRepoRuntimeWritableValue(t *testing.T, c *fibe.Client, urls []string, want bool) {
	t.Helper()
	status, err := c.RepoStatus.Check(ctx(), urls)
	requireNoError(t, err, "check fixture repo status")
	byURL := map[string]fibe.RepoStatusEntry{}
	for _, entry := range status.Repos {
		byURL[entry.URL] = entry
		if entry.GitHubURL != "" {
			byURL[entry.GitHubURL] = entry
		}
	}
	for _, url := range urls {
		entry, ok := byURL[url]
		if !ok {
			t.Fatalf("repo_status did not include %s: %#v", url, status.Repos)
		}
		if entry.Status != "ready" {
			t.Fatalf("%s status=%q, want ready; entry=%#v", url, entry.Status, entry)
		}
		if entry.RuntimeWritable == nil {
			t.Fatalf("%s runtime_writable missing: %#v", url, entry)
		}
		if *entry.RuntimeWritable != want {
			t.Fatalf("%s runtime_writable=%v, want %v; entry=%#v", url, *entry.RuntimeWritable, want, entry)
		}
		if want && entry.RequiresFork {
			t.Fatalf("%s requires_fork=true after credentialed launch: %#v", url, entry)
		}
	}
}

func requireRepoRequiresFork(t *testing.T, c *fibe.Client, urls []string) {
	t.Helper()
	status, err := c.RepoStatus.Check(ctx(), urls)
	requireNoError(t, err, "check fixture repo status")
	byURL := map[string]fibe.RepoStatusEntry{}
	for _, entry := range status.Repos {
		byURL[entry.URL] = entry
		if entry.GitHubURL != "" {
			byURL[entry.GitHubURL] = entry
		}
	}
	for _, url := range urls {
		entry, ok := byURL[url]
		if !ok {
			t.Fatalf("repo_status did not include %s: %#v", url, status.Repos)
		}
		if entry.Status != "needs_fork" && entry.Status != "not_writable" {
			t.Fatalf("%s status=%q, want needs_fork/not_writable for read-only fixture token; entry=%#v", url, entry.Status, entry)
		}
		if entry.RuntimeWritable == nil || *entry.RuntimeWritable {
			t.Fatalf("%s runtime_writable=%v, want false; entry=%#v", url, entry.RuntimeWritable, entry)
		}
		if !entry.RequiresFork {
			t.Fatalf("%s requires_fork=false, want true; entry=%#v", url, entry)
		}
	}
}

func requireLaunchIDs(t *testing.T, result *fibe.LaunchResult) {
	t.Helper()
	if result == nil || result.PlayspecID == 0 || result.PlaygroundID == 0 {
		t.Fatalf("launch result missing playspec/playground IDs: %#v", result)
	}
}

func requirePlayspecID(t *testing.T, result *fibe.LaunchResult) {
	t.Helper()
	if result == nil || result.PlayspecID == 0 {
		t.Fatalf("launch result missing playspec ID: %#v", result)
	}
	if result.PlaygroundID != 0 {
		t.Fatalf("launch result unexpectedly created playground when create_playground=false: %#v", result)
	}
}

func cleanupLaunchResult(t *testing.T, c *fibe.Client, result *fibe.LaunchResult) {
	t.Helper()
	t.Cleanup(func() {
		if result == nil {
			return
		}
		if result.PlaygroundID != 0 {
			_ = c.Playgrounds.Delete(ctx(), result.PlaygroundID)
		}
		if result.PlayspecID != 0 {
			_ = c.Playspecs.Delete(ctx(), result.PlayspecID)
		}
	})
}
