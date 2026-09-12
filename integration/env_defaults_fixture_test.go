package integration

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/fibegg/sdk/fibe"
)

// Own a real repository for the lifetime of these subtests. Borrowing another
// parallel test's Prop races its cleanup, and synthetic branch indexes can be
// overwritten by the application's normal background reindexing.
func ownedEnvDefaultsFixture(t *testing.T, c *fibe.Client) (fibe.Prop, string) {
	t.Helper()
	token, err := os.ReadFile(os.Getenv("GITEA_ADMIN_TOKEN_FILE"))
	requireNoError(t, err, "read local Gitea fixture token")
	host := strings.TrimRight(os.Getenv("GITEA_HOST"), "/")
	httpClient := &http.Client{Timeout: 30 * time.Second}
	request := func(method, path string, body any) []byte {
		t.Helper()
		var reader io.Reader
		if body != nil {
			encoded, err := json.Marshal(body)
			requireNoError(t, err)
			reader = bytes.NewReader(encoded)
		}
		req, err := http.NewRequestWithContext(ctx(), method, host+path, reader)
		requireNoError(t, err)
		req.Header.Set("Authorization", "token "+strings.TrimSpace(string(token)))
		req.Header.Set("Content-Type", "application/json")
		res, err := httpClient.Do(req)
		requireNoError(t, err, "local Gitea fixture request")
		defer res.Body.Close()
		data, err := io.ReadAll(res.Body)
		requireNoError(t, err)
		if method == http.MethodDelete && res.StatusCode == http.StatusNotFound {
			return nil
		}
		if res.StatusCode < 200 || res.StatusCode >= 300 {
			t.Fatalf("local Gitea fixture %s %s returned HTTP %d", method, path, res.StatusCode)
		}
		return data
	}
	var repo struct {
		FullName      string `json:"full_name"`
		CloneURL      string `json:"clone_url"`
		DefaultBranch string `json:"default_branch"`
	}
	data := request(http.MethodPost, "/api/v1/user/repos", map[string]any{
		"name": uniqueName("sdk-env-defaults"), "private": false, "auto_init": true,
	})
	requireNoError(t, json.Unmarshal(data, &repo))
	if repo.FullName == "" || repo.CloneURL == "" || repo.DefaultBranch == "" {
		t.Fatal("local Gitea fixture response is incomplete")
	}
	t.Cleanup(func() { request(http.MethodDelete, "/api/v1/repos/"+repo.FullName, nil) })
	request(http.MethodPost, "/api/v1/repos/"+repo.FullName+"/contents/"+seededPropEnvFile, map[string]any{
		"branch":  repo.DefaultBranch,
		"content": base64.StdEncoding.EncodeToString([]byte("FIBE_E2E=1\nRAILS_ENV=e2e\n")),
		"message": "Create environment defaults fixture",
	})
	prop, err := c.Props.Create(ctx(), &fibe.PropCreateParams{Name: ptr(uniqueName(seededPropNamePrefix)), RepositoryURL: repo.CloneURL})
	requireNoError(t, err, "attach owned environment defaults fixture")
	t.Cleanup(func() { requireNoError(t, c.Props.Delete(ctx(), prop.ID), "delete owned environment defaults fixture") })
	deadline := time.Now().Add(time.Minute)
	for time.Now().Before(deadline) {
		defaults, err := c.Props.EnvDefaults(ctx(), prop.ID, repo.DefaultBranch, seededPropEnvFile)
		requireNoError(t, err, "read owned environment defaults fixture")
		if defaults.Defaults["FIBE_E2E"] == "1" {
			return *prop, repo.DefaultBranch
		}
		time.Sleep(time.Second)
	}
	t.Fatal("owned environment defaults fixture did not become ready")
	return fibe.Prop{}, ""
}
