package fibe

import (
	"context"
	"net/http"
)

type GitHubAppService struct {
	client *Client
}

type GitHubAppConnectInfo struct {
	AppSlug    string `json:"app_slug"`
	InstallURL string `json:"install_url"`
}

func (s *GitHubAppService) ConnectInfo(ctx context.Context) (*GitHubAppConnectInfo, error) {
	var result GitHubAppConnectInfo
	err := s.client.do(ctx, http.MethodGet, "/api/github_apps/connect", nil, &result)
	return &result, err
}
