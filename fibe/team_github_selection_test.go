package fibe

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

func TestTeamGithubSelectionReachesEachEndpoint(t *testing.T) {
	seen := map[string]bool{}
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		seen[r.Method+" "+r.URL.Path] = true
		if r.Method == http.MethodGet {
			if r.URL.Query().Get("github_installation_id") != "501" {
				t.Errorf("Team search lost selected installation: %s", r.URL.RawQuery)
			}
			json.NewEncoder(w).Encode(map[string]any{"data": []any{map[string]any{"id": 7, "full_name": "team-org/existing"}}, "meta": map[string]any{"total": 1}})
			return
		}
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		if body["github_installation_id"] != float64(501) {
			t.Errorf("%s lost selected Team installation: %+v", r.URL.Path, body)
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{"id": 7, "owner_type": "Team", "owner_id": 9, "full_name": "team-org/existing"})
	})
	if result, err := c.Repositories.AttachWithParams(context.Background(), &RepositoryAttachParams{RepoFullName: "team-org/existing", GitHubInstallationID: 501}); err != nil || result.ID != 7 {
		t.Fatalf("Team attachment: %+v %v", result, err)
	}
	if result, err := c.GitHubRepos.Create(context.Background(), &GitHubRepoCreateParams{Name: "existing", GitHubInstallationID: 501}); err != nil || result.ID != 7 {
		t.Fatalf("Team create: %+v %v", result, err)
	}
	if result, err := c.Installations.FindGitHubRepos(context.Background(), &InstallationReposParams{GitHubInstallationID: 501, Q: "existing"}); err != nil || len(result.Data) != 1 {
		t.Fatalf("Team search: %+v %v", result, err)
	}
	if len(seen) != 3 {
		t.Fatalf("not all selected endpoints executed: %+v", seen)
	}
}

func TestTeamAttachmentPreservesLegacySignature(t *testing.T) {
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		if body["repo_full_name"] != "personal/repo" || body["github_installation_id"] != nil {
			t.Errorf("legacy attachment changed: %+v", body)
		}
		json.NewEncoder(w).Encode(map[string]any{"id": 8})
	})
	if result, err := c.Repositories.Attach(context.Background(), "personal/repo"); err != nil || result.ID != 8 {
		t.Fatalf("legacy attachment: %+v %v", result, err)
	}
}
