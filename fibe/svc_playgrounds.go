package fibe

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const (
	PlaygroundWaitReadinessLifecycle = "lifecycle"
	PlaygroundWaitReadinessServices  = "services"
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

// Action performs a lifecycle action (rollout, hard_restart, stop, start, retry_compose, enable_maintenance, disable_maintenance).
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
	err := s.client.doAsync(ctx, http.MethodPost, path+"/operations", "/api/async_requests/%s", params, &result)
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
	err := s.client.do(ctx, http.MethodPost, identifierPath("/api/playgrounds", identifier)+"/expiration", body, &result)
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
	err := s.client.doAsync(ctx, http.MethodPost, path+"/status", "/api/async_requests/%s", nil, &result)
	return &result, err
}

func (s *PlaygroundService) WaitForStatus(ctx context.Context, id int64, target string, timeout time.Duration, interval time.Duration) (*Playground, error) {
	return s.WaitForStatusByIdentifier(ctx, int64Identifier(id), target, timeout, interval)
}

func (s *PlaygroundService) WaitForStatusByIdentifier(ctx context.Context, identifier string, target string, timeout time.Duration, interval time.Duration) (*Playground, error) {
	return s.WaitForStatusWithReadinessByIdentifier(ctx, identifier, target, "", timeout, interval)
}

func (s *PlaygroundService) WaitForStatusWithReadinessByIdentifier(ctx context.Context, identifier string, target string, readiness string, timeout time.Duration, interval time.Duration) (*Playground, error) {
	if target == "" {
		target = "running"
	}
	readiness, err := NormalizePlaygroundWaitReadiness(readiness, target)
	if err != nil {
		return nil, err
	}
	if timeout <= 0 {
		timeout = 10 * time.Minute
	}
	if interval <= 0 {
		interval = 3 * time.Second
	}
	deadline := time.After(timeout)
	for {
		status, err := s.StatusByIdentifier(ctx, identifier)
		if err != nil {
			return nil, err
		}
		ready, pendingReason := PlaygroundStatusMatchesWaitTarget(status, target, readiness)
		if ready {
			return s.GetByIdentifier(ctx, identifier)
		}
		if status.Status == "error" || status.Status == "failed" || status.Status == "destroyed" {
			return nil, NewPlaygroundTerminalStateError(status)
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-deadline:
			if pendingReason == "" {
				pendingReason = fmt.Sprintf("last status: %s", status.Status)
			}
			return nil, fmt.Errorf("timeout after %s — %s", timeout, pendingReason)
		case <-time.After(interval):
		}
	}
}

func NormalizePlaygroundWaitReadiness(readiness string, target string) (string, error) {
	readiness = strings.TrimSpace(strings.ToLower(readiness))
	if readiness == "" {
		if strings.TrimSpace(strings.ToLower(target)) == "running" || target == "" {
			return PlaygroundWaitReadinessServices, nil
		}
		return PlaygroundWaitReadinessLifecycle, nil
	}
	switch readiness {
	case PlaygroundWaitReadinessLifecycle, PlaygroundWaitReadinessServices:
		return readiness, nil
	default:
		return "", fmt.Errorf("unsupported readiness %q; expected %q or %q", readiness, PlaygroundWaitReadinessLifecycle, PlaygroundWaitReadinessServices)
	}
}

func PlaygroundStatusMatchesWaitTarget(status *PlaygroundStatus, target string, readiness string) (bool, string) {
	if status == nil {
		return false, "status unavailable"
	}
	if target == "" {
		target = "running"
	}
	if status.Status != target {
		return false, fmt.Sprintf("last status: %s", status.Status)
	}
	if readiness != PlaygroundWaitReadinessServices || target != "running" {
		return true, ""
	}
	return PlaygroundServicesReady(status.Services)
}

func PlaygroundServicesReady(services []PlaygroundServiceInfo) (bool, string) {
	if len(services) == 0 {
		return false, "service readiness unavailable: no services reported"
	}
	pending := make([]string, 0)
	for _, service := range services {
		if playgroundServiceReady(service) {
			continue
		}
		pending = append(pending, playgroundServiceSummary(service))
	}
	if len(pending) > 0 {
		return false, "services not ready: " + strings.Join(pending, ", ")
	}
	return true, ""
}

func PlaygroundServiceReadinessSummary(services []PlaygroundServiceInfo) string {
	if len(services) == 0 {
		return "none reported"
	}
	parts := make([]string, 0, len(services))
	for _, service := range services {
		parts = append(parts, playgroundServiceSummary(service))
	}
	return strings.Join(parts, ", ")
}

func playgroundServiceReady(service PlaygroundServiceInfo) bool {
	status := strings.ToLower(strings.TrimSpace(service.Status))
	health := strings.ToLower(strings.TrimSpace(service.Health))
	if service.ExitCode != nil && *service.ExitCode != 0 {
		return false
	}
	if health == "unhealthy" || health == "starting" {
		return false
	}
	if service.Running {
		return health == "" || health == "healthy"
	}
	return status == "running" && (health == "" || health == "healthy")
}

func playgroundServiceSummary(service PlaygroundServiceInfo) string {
	name := strings.TrimSpace(service.Name)
	if name == "" {
		name = "<unnamed>"
	}
	status := strings.TrimSpace(service.Status)
	if status == "" {
		status = "unknown"
	}
	parts := []string{name + "=" + status}
	if health := strings.TrimSpace(service.Health); health != "" {
		parts = append(parts, "health="+health)
	}
	if service.Running {
		parts = append(parts, "running")
	} else {
		parts = append(parts, "not-running")
	}
	if service.ExitCode != nil {
		parts = append(parts, fmt.Sprintf("exit=%d", *service.ExitCode))
	}
	return strings.Join(parts, "/")
}

func (s *PlaygroundService) Compose(ctx context.Context, id int64) (*PlaygroundCompose, error) {
	return s.ComposeByIdentifier(ctx, int64Identifier(id))
}

func (s *PlaygroundService) ComposeByIdentifier(ctx context.Context, identifier string) (*PlaygroundCompose, error) {
	var result PlaygroundCompose
	err := s.client.do(ctx, http.MethodGet, identifierPath("/api/playgrounds", identifier)+"/compose", nil, &result)
	return &result, err
}

// Logs triggers an async log snapshot and polls for the result. Empty service returns all services.
func (s *PlaygroundService) Logs(ctx context.Context, id int64, service string, tail *int) (*PlaygroundLogs, error) {
	return s.LogsByIdentifier(ctx, int64Identifier(id), service, tail)
}

func (s *PlaygroundService) LogsByIdentifier(ctx context.Context, identifier string, service string, tail *int) (*PlaygroundLogs, error) {
	base := identifierPath("/api/playgrounds", identifier)
	body := map[string]any{}
	if service != "" {
		body["service"] = service
	}
	if tail != nil {
		body["tail"] = *tail
	}
	var result PlaygroundLogs
	err := s.client.doAsync(ctx, http.MethodPost, base+"/logs", "/api/async_requests/%s", body, &result)
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
	err := s.client.doAsync(ctx, http.MethodGet, path, "/api/async_requests/%s", nil, &result)
	return result, err
}
