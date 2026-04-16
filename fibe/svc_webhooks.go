package fibe

import (
	"context"
	"fmt"
	"net/http"
)

type WebhookEndpointService struct {
	client *Client
}

func (s *WebhookEndpointService) List(ctx context.Context, params *WebhookEndpointListParams) (*ListResult[WebhookEndpoint], error) {
	path := "/api/webhook_endpoints" + buildQuery(params)
	return doList[WebhookEndpoint](s.client, ctx, path)
}

func (s *WebhookEndpointService) Get(ctx context.Context, id int64) (*WebhookEndpoint, error) {
	var result WebhookEndpoint
	err := s.client.do(ctx, http.MethodGet, fmt.Sprintf("/api/webhook_endpoints/%d", id), nil, &result)
	return &result, err
}

func (s *WebhookEndpointService) Create(ctx context.Context, params *WebhookEndpointCreateParams) (*WebhookEndpoint, error) {
	if err := validateParams(params); err != nil {
		return nil, err
	}
	var result WebhookEndpoint
	body := map[string]any{"webhook_endpoint": params}
	err := s.client.do(ctx, http.MethodPost, "/api/webhook_endpoints", body, &result)
	return &result, err
}

func (s *WebhookEndpointService) Update(ctx context.Context, id int64, params *WebhookEndpointUpdateParams) (*WebhookEndpoint, error) {
	var result WebhookEndpoint
	body := map[string]any{"webhook_endpoint": params}
	err := s.client.do(ctx, http.MethodPatch, fmt.Sprintf("/api/webhook_endpoints/%d", id), body, &result)
	return &result, err
}

func (s *WebhookEndpointService) Delete(ctx context.Context, id int64) error {
	return s.client.do(ctx, http.MethodDelete, fmt.Sprintf("/api/webhook_endpoints/%d", id), nil, nil)
}

func (s *WebhookEndpointService) Test(ctx context.Context, id int64) error {
	var result map[string]any
	return s.client.do(ctx, http.MethodPost, fmt.Sprintf("/api/webhook_endpoints/%d/test", id), nil, &result)
}

func (s *WebhookEndpointService) ListDeliveries(ctx context.Context, id int64, params *ListParams) (*ListResult[WebhookDelivery], error) {
	path := fmt.Sprintf("/api/webhook_endpoints/%d/deliveries", id) + buildQuery(params)
	return doList[WebhookDelivery](s.client, ctx, path)
}

func (s *WebhookEndpointService) EventTypes(ctx context.Context) ([]string, error) {
	var result struct {
		EventTypes []string `json:"event_types"`
	}
	err := s.client.do(ctx, http.MethodGet, "/api/webhook_endpoints/event_types", nil, &result)
	return result.EventTypes, err
}
