package fibe

import (
	"context"
	"net/http"
)

// ServerInfoService provides access to the server's health/identity endpoint
// at /up. Unauthenticated.
type ServerInfoService struct {
	client *Client
}

// ServerInfo is the server's self-identification: its current UTC clock plus
// the build-time identity baked into the image at docker build.
//
// BuildTime and GitCommitSHA may be empty when the server was built without
// the FIBE_BUILD_TIME / FIBE_BUILD_GIT_COMMIT_SHA build-args (e.g. local dev).
type ServerInfo struct {
	Status       string `json:"status"`
	TimeUTC      string `json:"time_utc"`
	BuildTime    string `json:"build_time,omitempty"`
	GitCommitSHA string `json:"git_commit_sha,omitempty"`
}

// Get returns the server's current UTC time and build identity. Calls /up.json
// and does not require authentication.
func (s *ServerInfoService) Get(ctx context.Context) (*ServerInfo, error) {
	var result ServerInfo
	err := s.client.do(ctx, http.MethodGet, "/up.json", nil, &result)
	return &result, err
}
