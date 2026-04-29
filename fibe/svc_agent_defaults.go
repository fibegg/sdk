package fibe

import (
	"context"
	"net/http"
)

type AgentDefaultsService struct {
	client *Client
}

func (s *AgentDefaultsService) Get(ctx context.Context) (*AgentDefaultsPayload, error) {
	var result AgentDefaultsPayload
	err := s.client.do(ctx, http.MethodGet, "/api/agent_defaults", nil, &result)
	return &result, err
}

func (s *AgentDefaultsService) Update(ctx context.Context, defaults AgentDefaults) (*AgentDefaultsPayload, error) {
	var result AgentDefaultsPayload
	body := map[string]any{"agent_defaults": defaults}
	err := s.client.do(ctx, http.MethodPatch, "/api/agent_defaults", body, &result)
	return &result, err
}

func (s *AgentDefaultsService) Reset(ctx context.Context) (*AgentDefaultsPayload, error) {
	var result AgentDefaultsPayload
	err := s.client.do(ctx, http.MethodDelete, "/api/agent_defaults", nil, &result)
	return &result, err
}
