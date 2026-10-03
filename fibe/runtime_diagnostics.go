package fibe

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// Runtime diagnostics extend the original SDK response types without changing
// their public struct layouts.
type PlaygroundRuntimeStatus struct {
	PlaygroundStatus
	RuntimeServices  []PlaygroundRuntimeServiceInfo `json:"services,omitempty"`
	RuntimeOperation map[string]any                 `json:"runtime_operation,omitempty"`
	BuildWarnings    []string                       `json:"build_warnings,omitempty"`
}

type PlaygroundRuntimeServiceInfo struct {
	PlaygroundServiceInfo
	CompletionExpected bool `json:"completion_expected,omitempty"`
}

type PlaygroundBuildLogDebugParams struct {
	PlaygroundDebugParams
	IncludeBuildLogs *bool `url:"include_build_logs,omitempty"`
}

type PropMirrorState struct {
	Prop
	MirrorStatus string `json:"mirror_status,omitempty"`
	MirrorError  string `json:"mirror_error,omitempty"`
}

func (s *PlaygroundService) RuntimeStatus(ctx context.Context, id int64) (*PlaygroundRuntimeStatus, error) {
	return s.RuntimeStatusByIdentifier(ctx, int64Identifier(id))
}

func (s *PlaygroundService) RuntimeStatusByIdentifier(ctx context.Context, identifier string) (*PlaygroundRuntimeStatus, error) {
	var result PlaygroundRuntimeStatus
	err := s.client.do(ctx, http.MethodGet, identifierPath("/api/playgrounds", identifier)+"/status", nil, &result)
	return &result, err
}

func (s *PlaygroundService) DebugWithBuildLogsByIdentifier(ctx context.Context, identifier string, params *PlaygroundBuildLogDebugParams) (map[string]any, error) {
	var result map[string]any
	err := s.client.doAsync(ctx, http.MethodGet, identifierPath("/api/playgrounds", identifier)+"/debug"+buildQuery(params), "/api/async_requests/%s", nil, &result)
	return result, err
}

func (s *PropService) GetMirrorStateByIdentifier(ctx context.Context, identifier string) (*PropMirrorState, error) {
	var result PropMirrorState
	err := s.client.do(ctx, http.MethodGet, identifierPath("/api/props", identifier), nil, &result)
	return &result, err
}

func (s *PropService) MirrorWithState(ctx context.Context, sourceURL, name string) (*PropMirrorState, error) {
	var result PropMirrorState
	body := map[string]any{"source_url": sourceURL}
	if name != "" {
		body["name"] = name
	}
	err := s.client.do(ctx, http.MethodPost, "/api/props/mirrors", body, &result)
	return &result, err
}

func (p *PlaygroundRuntimeStatus) UnmarshalJSON(data []byte) error {
	normalized, err := normalizeBuildWarningJSON(data)
	if err != nil {
		return err
	}
	type base PlaygroundStatus
	var wire struct {
		base
		Services         []PlaygroundRuntimeServiceInfo `json:"services"`
		RuntimeOperation map[string]any                 `json:"runtime_operation"`
		BuildWarnings    []string                       `json:"build_warnings"`
	}
	if err := json.Unmarshal(normalized, &wire); err != nil {
		return err
	}
	p.PlaygroundStatus = PlaygroundStatus(wire.base)
	p.RuntimeServices, p.RuntimeOperation, p.BuildWarnings = wire.Services, wire.RuntimeOperation, wire.BuildWarnings
	for _, service := range wire.Services {
		p.Services = append(p.Services, service.PlaygroundServiceInfo)
	}
	return nil
}

// Older clients expose warning text. Accept the server's structured warnings
// as well as the original string form.
func (p *Playground) UnmarshalJSON(data []byte) error {
	normalized, err := normalizeBuildWarningJSON(data)
	if err != nil {
		return err
	}
	type wire Playground
	return json.Unmarshal(normalized, (*wire)(p))
}

func normalizeBuildWarningJSON(data []byte) ([]byte, error) {
	var object map[string]json.RawMessage
	if err := json.Unmarshal(data, &object); err != nil {
		return nil, err
	}
	raw := object["build_warnings"]
	if len(raw) == 0 {
		return data, nil
	}
	var warnings []string
	if json.Unmarshal(raw, &warnings) == nil {
		return data, nil
	}
	var structured []map[string]string
	if err := json.Unmarshal(raw, &structured); err != nil {
		return nil, fmt.Errorf("decode build warnings: %w", err)
	}
	for _, warning := range structured {
		text, _ := json.Marshal(warning)
		warnings = append(warnings, string(text))
	}
	object["build_warnings"], _ = json.Marshal(warnings)
	return json.Marshal(object)
}

func PlaygroundRuntimeStatusMatchesWaitTarget(status *PlaygroundRuntimeStatus, target, readiness string) (bool, string) {
	if status == nil {
		return false, "status unavailable"
	}
	if target == "" {
		target = "running"
	}
	if status.Status != target || readiness != PlaygroundWaitReadinessServices || target != "running" {
		return PlaygroundStatusMatchesWaitTarget(&status.PlaygroundStatus, target, readiness)
	}
	if status.NeedsRecreation != nil && *status.NeedsRecreation {
		return PlaygroundStatusMatchesWaitTarget(&status.PlaygroundStatus, target, readiness)
	}
	if len(status.BuildWarnings) > 0 {
		return false, "deployment has build warnings; inspect playground debug --build-logs"
	}
	if len(status.RuntimeServices) == 0 {
		return PlaygroundServicesReady(status.Services)
	}
	for _, service := range status.RuntimeServices {
		if service.CompletionExpected {
			if service.Status == "exited" && service.ExitCode != nil && *service.ExitCode == 0 {
				continue
			}
		} else if playgroundServiceReady(service.PlaygroundServiceInfo) {
			continue
		}
		return false, "services not ready: " + playgroundServiceSummary(service.PlaygroundServiceInfo)
	}
	return true, ""
}
