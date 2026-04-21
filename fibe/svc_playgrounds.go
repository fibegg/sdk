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

func (s *PlaygroundService) Rollout(ctx context.Context, id int64) (*Playground, error) {
	var result Playground
	err := s.client.do(ctx, http.MethodPost, fmt.Sprintf("/api/playgrounds/%d/rollout", id), nil, &result)
	return &result, err
}

func (s *PlaygroundService) HardRestart(ctx context.Context, id int64) (*Playground, error) {
	var result Playground
	err := s.client.do(ctx, http.MethodPost, fmt.Sprintf("/api/playgrounds/%d/hard_restart", id), nil, &result)
	return &result, err
}

func (s *PlaygroundService) RetryCompose(ctx context.Context, id int64, params *PlaygroundRetryComposeParams) (*PlaygroundStatus, error) {
	var result PlaygroundStatus
	err := s.client.do(ctx, http.MethodPost, fmt.Sprintf("/api/playgrounds/%d/retry_compose", id), params, &result)
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

func (s *PlaygroundService) Status(ctx context.Context, id int64) (*PlaygroundStatus, error) {
	var result PlaygroundStatus
	err := s.client.do(ctx, http.MethodGet, fmt.Sprintf("/api/playgrounds/%d/status", id), nil, &result)
	return &result, err
}

func (s *PlaygroundService) Compose(ctx context.Context, id int64) (*PlaygroundCompose, error) {
	var result PlaygroundCompose
	err := s.client.do(ctx, http.MethodGet, fmt.Sprintf("/api/playgrounds/%d/compose", id), nil, &result)
	return &result, err
}

func (s *PlaygroundService) Logs(ctx context.Context, id int64, service string, tail *int) (*PlaygroundLogs, error) {
	path := fmt.Sprintf("/api/playgrounds/%d/logs/%s", id, service)
	if tail != nil {
		path += fmt.Sprintf("?tail=%d", *tail)
	}
	var result PlaygroundLogs
	err := s.client.do(ctx, http.MethodGet, path, nil, &result)
	return &result, err
}

func (s *PlaygroundService) EnvMetadata(ctx context.Context, id int64) (*PlaygroundEnvMetadata, error) {
	var result PlaygroundEnvMetadata
	err := s.client.do(ctx, http.MethodGet, fmt.Sprintf("/api/playgrounds/%d/env_metadata", id), nil, &result)
	return &result, err
}

func (s *PlaygroundService) Debug(ctx context.Context, id int64) (map[string]any, error) {
	var result map[string]any
	err := s.client.do(ctx, http.MethodGet, fmt.Sprintf("/api/playgrounds/%d/debug", id), nil, &result)
	return result, err
}
