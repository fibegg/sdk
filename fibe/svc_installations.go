package fibe

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// InstallationService provides access to GitHub App installations linked to the
// authenticated player. Installations are how Fibe accesses GitHub repositories.
type InstallationService struct {
	client *Client
}

// Installation represents a GitHub App installation connected to the player.
type Installation struct {
	ID                  int64   `json:"id"`
	Provider            string  `json:"provider"`
	InstallationID      int64   `json:"installation_id"`
	InstallationAccount *string `json:"installation_account"`
	CreatedAt           string  `json:"created_at"`
}

// InstallationRepo represents a repository accessible through an installation.
type InstallationRepo struct {
	ID            int64   `json:"id"`
	Name          string  `json:"name"`
	FullName      string  `json:"full_name"`
	Private       bool    `json:"private"`
	Description   *string `json:"description"`
	HTMLURL       string  `json:"html_url"`
	CloneURL      string  `json:"clone_url"`
	SSHURL        string  `json:"ssh_url"`
	DefaultBranch string  `json:"default_branch"`
}

// InstallationReposParams controls listing/searching repos in an installation.
type InstallationReposParams struct {
	Q       string `url:"q,omitempty"`
	Page    int    `url:"page,omitempty"`
	PerPage int    `url:"per_page,omitempty"`
}

// List returns all GitHub App installations linked to the authenticated player.
func (s *InstallationService) List(ctx context.Context) (*ListResult[Installation], error) {
	return doList[Installation](s.client, ctx, "/api/installations")
}

// Repos returns repositories accessible through the given installation.
// Optionally filter by query string (uses GitHub repo search when q is set,
// otherwise lists all repos accessible to the installation).
func (s *InstallationService) Repos(ctx context.Context, id int64, params *InstallationReposParams) (*ListResult[InstallationRepo], error) {
	path := fmt.Sprintf("/api/installations/%d/repos", id) + buildQuery(params)
	return doList[InstallationRepo](s.client, ctx, path)
}

// FindGitHubRepos searches across ALL of the player's GitHub App installations.
// Rails aggregates results in parallel and deduplicates by full_name.
func (s *InstallationService) FindGitHubRepos(ctx context.Context, params *InstallationReposParams) (*ListResult[InstallationRepo], error) {
	path := "/api/github_repos/search" + buildQuery(params)
	return doList[InstallationRepo](s.client, ctx, path)
}

// Token returns a fresh installation access token. If repo is non-empty, the
// token is scoped to the specific repository (owner/name format).
func (s *InstallationService) Token(ctx context.Context, id int64, repo string) (*GitHubToken, error) {
	path := fmt.Sprintf("/api/installations/%d/token", id)
	if repo != "" {
		values := url.Values{}
		values.Set("repo", repo)
		path += "?" + values.Encode()
	}
	var result GitHubToken
	err := s.client.do(ctx, http.MethodGet, path, nil, &result)
	return &result, err
}

// GetGitHubToken returns a fresh GitHub token by auto-resolving the correct
// installation for the given repo. No installation ID needed.
func (s *InstallationService) GetGitHubToken(ctx context.Context, repo string) (*GitHubToken, error) {
	values := url.Values{}
	values.Set("repo", repo)
	path := "/api/github_token?" + values.Encode()
	var result GitHubToken
	err := s.client.do(ctx, http.MethodGet, path, nil, &result)
	return &result, err
}
