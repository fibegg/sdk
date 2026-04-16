package fibe

import (
	"context"
	"fmt"
	"net/http"
)

type MutterService struct {
	client *Client
}

func (s *MutterService) Get(ctx context.Context, agentID int64, params *MutterListParams) (*Mutter, error) {
	path := fmt.Sprintf("/api/agents/%d/mutter", agentID)
	if params != nil {
		path += buildQuery(params)
	}
	var result Mutter
	err := s.client.do(ctx, http.MethodGet, path, nil, &result)
	return &result, err
}

func (s *MutterService) CreateItem(ctx context.Context, agentID int64, params *MutterItemParams) (*Mutter, error) {
	path := fmt.Sprintf("/api/agents/%d/mutter", agentID)
	var result Mutter
	err := s.client.do(ctx, http.MethodPost, path, params, &result)
	return &result, err
}

type AuditLogService struct {
	client *Client
}

func (s *AuditLogService) List(ctx context.Context, params *AuditLogListParams) (*ListResult[AuditLog], error) {
	path := "/api/audit_logs"
	if params != nil {
		path += buildQuery(params)
	}
	return doList[AuditLog](s.client, ctx, path)
}

type GitHubRepoService struct {
	client *Client
}

func (s *GitHubRepoService) Create(ctx context.Context, params *GitHubRepoCreateParams) (*GitHubRepo, error) {
	var result GitHubRepo
	err := s.client.do(ctx, http.MethodPost, "/api/github_repos", params, &result)
	return &result, err
}

type GiteaRepoService struct {
	client *Client
}

func (s *GiteaRepoService) Create(ctx context.Context, params *GiteaRepoCreateParams) (*GiteaRepo, error) {
	var result GiteaRepo
	err := s.client.do(ctx, http.MethodPost, "/api/gitea_repos", params, &result)
	return &result, err
}

type LaunchService struct {
	client *Client
}

func (s *LaunchService) Create(ctx context.Context, params *LaunchParams) (*LaunchResult, error) {
	if err := validateParams(params); err != nil {
		return nil, err
	}
	var result LaunchResult
	err := s.client.do(ctx, http.MethodPost, "/api/launch", params, &result)
	return &result, err
}

type RepoStatusService struct {
	client *Client
}

func (s *RepoStatusService) Check(ctx context.Context, githubURLs []string) (*RepoStatus, error) {
	var result RepoStatus
	body := map[string]any{"github_urls": githubURLs}
	err := s.client.do(ctx, http.MethodPost, "/api/repo_status", body, &result)
	return &result, err
}

type TemplateCategoryService struct {
	client *Client
}

func (s *TemplateCategoryService) List(ctx context.Context, params *ListParams) (*ListResult[TemplateCategory], error) {
	path := "/api/template_categories" + buildQuery(params)
	return doList[TemplateCategory](s.client, ctx, path)
}
