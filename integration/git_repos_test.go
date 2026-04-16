package integration

import (
	"testing"

	"github.com/fibegg/sdk/fibe"
)

func TestGitHubRepos_CreateFailure(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	repo, err := c.GitHubRepos.Create(ctx(), &fibe.GitHubRepoCreateParams{
		Name: "fibegg-invalid",
	})
	if err == nil {
		// API accepted the request — GitHub integration is connected and created the repo
		t.Logf("GitHub repo creation succeeded (name=%q) — API has active GitHub integration", repo.Name)
		return
	}

	apiErr, ok := err.(*fibe.APIError)
	if !ok {
		t.Fatalf("expected API error, got: %v", err)
	}
	validCodes := []int{400, 401, 403, 404, 422}
	found := false
	for _, code := range validCodes {
		if apiErr.StatusCode == code {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected one of %v, got %d: %v", validCodes, apiErr.StatusCode, apiErr)
	}
}

func TestGiteaRepos_CreateFailure(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	_, err := c.GiteaRepos.Create(ctx(), &fibe.GiteaRepoCreateParams{
		Name: "fibegg-invalid",
	})
	if err == nil {
		t.Logf("Gitea repo creation succeeded — API has active Gitea integration")
		return
	}

	apiErr, ok := err.(*fibe.APIError)
	if !ok {
		t.Fatalf("expected API error, got: %v", err)
	}
	validCodes := []int{400, 401, 403, 422, 503}
	found := false
	for _, code := range validCodes {
		if apiErr.StatusCode == code {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected one of %v, got %d: %v", validCodes, apiErr.StatusCode, apiErr)
	}
}
