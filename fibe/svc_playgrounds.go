package fibe

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

type PlaygroundService struct {
	client *Client
}

func (s *PlaygroundService) List(ctx context.Context, params *PlaygroundListParams) (*ListResult[Playground], error) {
	path := "/api/playgrounds" + buildQuery(params)
	return doList[Playground](s.client, ctx, path)
}

func (s *PlaygroundService) Get(ctx context.Context, id int64) (*Playground, error) {
	return s.GetByIdentifier(ctx, int64Identifier(id))
}

func (s *PlaygroundService) GetByIdentifier(ctx context.Context, identifier string) (*Playground, error) {
	var result Playground
	err := s.client.do(ctx, http.MethodGet, identifierPath("/api/playgrounds", identifier), nil, &result)
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
	return s.UpdateByIdentifier(ctx, int64Identifier(id), params)
}

func (s *PlaygroundService) UpdateByIdentifier(ctx context.Context, identifier string, params *PlaygroundUpdateParams) (*Playground, error) {
	var result Playground
	body := map[string]any{"playground": params}
	err := s.client.do(ctx, http.MethodPatch, identifierPath("/api/playgrounds", identifier), body, &result)
	return &result, err
}

func (s *PlaygroundService) Delete(ctx context.Context, id int64) error {
	return s.DeleteByIdentifier(ctx, int64Identifier(id))
}

func (s *PlaygroundService) DeleteByIdentifier(ctx context.Context, identifier string) error {
	return s.client.do(ctx, http.MethodDelete, identifierPath("/api/playgrounds", identifier), nil, nil)
}

// Action performs a lifecycle action (rollout, hard_restart, stop, start, retry_compose).
// The API returns 202 Accepted; this method polls until the operation completes.
func (s *PlaygroundService) Action(ctx context.Context, id int64, params *PlaygroundActionParams) (*PlaygroundStatus, error) {
	return s.ActionByIdentifier(ctx, int64Identifier(id), params)
}

func (s *PlaygroundService) ActionByIdentifier(ctx context.Context, identifier string, params *PlaygroundActionParams) (*PlaygroundStatus, error) {
	if params == nil {
		params = &PlaygroundActionParams{}
	}
	if err := validateParams(params); err != nil {
		return nil, err
	}
	var result PlaygroundStatus
	path := identifierPath("/api/playgrounds", identifier)
	err := s.client.doAsync(ctx, http.MethodPost, path+"/action", path+"/action/%s", params, &result)
	return &result, err
}

func (s *PlaygroundService) ExtendExpiration(ctx context.Context, id int64, durationHours *int) (*PlaygroundExtendResult, error) {
	return s.ExtendExpirationByIdentifier(ctx, int64Identifier(id), durationHours)
}

func (s *PlaygroundService) ExtendExpirationByIdentifier(ctx context.Context, identifier string, durationHours *int) (*PlaygroundExtendResult, error) {
	var result PlaygroundExtendResult
	var body any
	if durationHours != nil {
		body = map[string]any{"duration_hours": *durationHours}
	}
	err := s.client.do(ctx, http.MethodPost, identifierPath("/api/playgrounds", identifier)+"/extend_expiration", body, &result)
	return &result, err
}

// Status returns the cached build status without triggering a remote refresh.
func (s *PlaygroundService) Status(ctx context.Context, id int64) (*PlaygroundStatus, error) {
	return s.StatusByIdentifier(ctx, int64Identifier(id))
}

func (s *PlaygroundService) StatusByIdentifier(ctx context.Context, identifier string) (*PlaygroundStatus, error) {
	var result PlaygroundStatus
	err := s.client.do(ctx, http.MethodGet, identifierPath("/api/playgrounds", identifier)+"/status", nil, &result)
	return &result, err
}

// StatusRefresh triggers an async remote status refresh and polls for the result.
func (s *PlaygroundService) StatusRefresh(ctx context.Context, id int64) (*PlaygroundStatus, error) {
	return s.StatusRefreshByIdentifier(ctx, int64Identifier(id))
}

func (s *PlaygroundService) StatusRefreshByIdentifier(ctx context.Context, identifier string) (*PlaygroundStatus, error) {
	var result PlaygroundStatus
	path := identifierPath("/api/playgrounds", identifier)
	err := s.client.doAsync(ctx, http.MethodPost, path+"/status_refresh", path+"/status_refresh/%s", nil, &result)
	return &result, err
}

func (s *PlaygroundService) Compose(ctx context.Context, id int64) (*PlaygroundCompose, error) {
	return s.ComposeByIdentifier(ctx, int64Identifier(id))
}

func (s *PlaygroundService) ComposeByIdentifier(ctx context.Context, identifier string) (*PlaygroundCompose, error) {
	var result PlaygroundCompose
	err := s.client.do(ctx, http.MethodGet, identifierPath("/api/playgrounds", identifier)+"/compose", nil, &result)
	return &result, err
}

// Logs triggers an async log snapshot and polls for the result.
func (s *PlaygroundService) Logs(ctx context.Context, id int64, service string, tail *int) (*PlaygroundLogs, error) {
	return s.LogsByIdentifier(ctx, int64Identifier(id), service, tail)
}

func (s *PlaygroundService) LogsByIdentifier(ctx context.Context, identifier string, service string, tail *int) (*PlaygroundLogs, error) {
	base := identifierPath("/api/playgrounds", identifier)
	path := base + "/logs/" + url.PathEscape(service)
	if tail != nil {
		path += fmt.Sprintf("?tail=%d", *tail)
	}
	var result PlaygroundLogs
	err := s.client.doAsync(ctx, http.MethodGet, path, base+"/logs_status/%s", nil, &result)
	return &result, err
}

func (s *PlaygroundService) EnvMetadata(ctx context.Context, id int64) (*PlaygroundEnvMetadata, error) {
	return s.EnvMetadataByIdentifier(ctx, int64Identifier(id))
}

func (s *PlaygroundService) EnvMetadataByIdentifier(ctx context.Context, identifier string) (*PlaygroundEnvMetadata, error) {
	var result PlaygroundEnvMetadata
	err := s.client.do(ctx, http.MethodGet, identifierPath("/api/playgrounds", identifier)+"/env_metadata", nil, &result)
	return &result, err
}

func (s *PlaygroundService) Debug(ctx context.Context, id int64) (map[string]any, error) {
	return s.DebugWithParams(ctx, id, nil)
}

// DebugWithParams retrieves playground diagnostics. When refresh=true, the API
// returns 202 and this method auto-polls for the final result.
func (s *PlaygroundService) DebugWithParams(ctx context.Context, id int64, params *PlaygroundDebugParams) (map[string]any, error) {
	return s.DebugWithParamsByIdentifier(ctx, int64Identifier(id), params)
}

func (s *PlaygroundService) DebugWithParamsByIdentifier(ctx context.Context, identifier string, params *PlaygroundDebugParams) (map[string]any, error) {
	var result map[string]any
	base := identifierPath("/api/playgrounds", identifier)
	path := base + "/debug" + buildQuery(params)
	err := s.client.doAsync(ctx, http.MethodGet, path, base+"/debug/%s", nil, &result)
	return result, err
}
