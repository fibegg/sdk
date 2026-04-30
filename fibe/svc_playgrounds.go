package fibe

import (
	"context"
	"fmt"
	"net/http"
)

type PlaygroundService struct {
	client *Client
}

func (s *PlaygroundService) List(ctx context.Context, params *PlaygroundListParams) (*ListResult[Playground], error) {
	path := "/api/playgrounds" + buildQuery(params)
	return doList[Playground](s.client, ctx, path)
}

func (s *PlaygroundService) Get(ctx context.Context, id int64) (*Playground, error) {
	var result Playground
	err := s.client.do(ctx, http.MethodGet, fmt.Sprintf("/api/playgrounds/%d", id), nil, &result)
	return &result, err
}

func (s *PlaygroundService) Create(ctx context.Context, params *PlaygroundCreateParams) (*Playground, error) {
	if err := validateParams(params); err != nil {
		return nil, err
	}
	var result Playground
	body := map[string]any{"playground": params}
	err := s.client.do(ctx, http.MethodPost, "/api/playgrounds", body, &result)
	return &result, err
}

func (s *PlaygroundService) Update(ctx context.Context, id int64, params *PlaygroundUpdateParams) (*Playground, error) {
	var result Playground
	body := map[string]any{"playground": params}
	err := s.client.do(ctx, http.MethodPatch, fmt.Sprintf("/api/playgrounds/%d", id), body, &result)
	return &result, err
}

func (s *PlaygroundService) Delete(ctx context.Context, id int64) error {
	return s.client.do(ctx, http.MethodDelete, fmt.Sprintf("/api/playgrounds/%d", id), nil, nil)
}

// Action performs a lifecycle action (rollout, hard_restart, stop, start, retry_compose).
// The API returns 202 Accepted; this method polls until the operation completes.
func (s *PlaygroundService) Action(ctx context.Context, id int64, params *PlaygroundActionParams) (*PlaygroundStatus, error) {
	if params == nil {
		params = &PlaygroundActionParams{}
	}
	if err := validateParams(params); err != nil {
		return nil, err
	}
	var result PlaygroundStatus
	statusFmt := fmt.Sprintf("/api/playgrounds/%d/action/%%s", id)
	err := s.client.doAsync(ctx, http.MethodPost, fmt.Sprintf("/api/playgrounds/%d/action", id), statusFmt, params, &result)
	return &result, err
}

func (s *PlaygroundService) ExtendExpiration(ctx context.Context, id int64, durationHours *int) (*PlaygroundExtendResult, error) {
	var result PlaygroundExtendResult
	var body any
	if durationHours != nil {
		body = map[string]any{"duration_hours": *durationHours}
	}
	err := s.client.do(ctx, http.MethodPost, fmt.Sprintf("/api/playgrounds/%d/extend_expiration", id), body, &result)
	return &result, err
}

// Status returns the cached build status without triggering a remote refresh.
func (s *PlaygroundService) Status(ctx context.Context, id int64) (*PlaygroundStatus, error) {
	var result PlaygroundStatus
	err := s.client.do(ctx, http.MethodGet, fmt.Sprintf("/api/playgrounds/%d/status", id), nil, &result)
	return &result, err
}

// StatusRefresh triggers an async remote status refresh and polls for the result.
func (s *PlaygroundService) StatusRefresh(ctx context.Context, id int64) (*PlaygroundStatus, error) {
	var result PlaygroundStatus
	statusFmt := fmt.Sprintf("/api/playgrounds/%d/status_refresh/%%s", id)
	err := s.client.doAsync(ctx, http.MethodPost, fmt.Sprintf("/api/playgrounds/%d/status_refresh", id), statusFmt, nil, &result)
	return &result, err
}

func (s *PlaygroundService) Compose(ctx context.Context, id int64) (*PlaygroundCompose, error) {
	var result PlaygroundCompose
	err := s.client.do(ctx, http.MethodGet, fmt.Sprintf("/api/playgrounds/%d/compose", id), nil, &result)
	return &result, err
}

// Logs triggers an async log snapshot and polls for the result.
func (s *PlaygroundService) Logs(ctx context.Context, id int64, service string, tail *int) (*PlaygroundLogs, error) {
	path := fmt.Sprintf("/api/playgrounds/%d/logs/%s", id, service)
	if tail != nil {
		path += fmt.Sprintf("?tail=%d", *tail)
	}
	var result PlaygroundLogs
	statusFmt := fmt.Sprintf("/api/playgrounds/%d/logs_status/%%s", id)
	err := s.client.doAsync(ctx, http.MethodGet, path, statusFmt, nil, &result)
	return &result, err
}

func (s *PlaygroundService) EnvMetadata(ctx context.Context, id int64) (*PlaygroundEnvMetadata, error) {
	var result PlaygroundEnvMetadata
	err := s.client.do(ctx, http.MethodGet, fmt.Sprintf("/api/playgrounds/%d/env_metadata", id), nil, &result)
	return &result, err
}

func (s *PlaygroundService) Debug(ctx context.Context, id int64) (map[string]any, error) {
	return s.DebugWithParams(ctx, id, nil)
}

// DebugWithParams retrieves playground diagnostics. When refresh=true, the API
// returns 202 and this method auto-polls for the final result.
func (s *PlaygroundService) DebugWithParams(ctx context.Context, id int64, params *PlaygroundDebugParams) (map[string]any, error) {
	var result map[string]any
	path := fmt.Sprintf("/api/playgrounds/%d/debug", id) + buildQuery(params)
	statusFmt := fmt.Sprintf("/api/playgrounds/%d/debug/%%s", id)
	err := s.client.doAsync(ctx, http.MethodGet, path, statusFmt, nil, &result)
	return result, err
}
