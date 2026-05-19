package fibe

import (
	"context"
	"fmt"
	"net/http"
)

type APIKeyService struct {
	client *Client
}

func (s *APIKeyService) Me(ctx context.Context) (*Player, error) {
	var result Player
	err := s.client.do(ctx, http.MethodGet, "/api/me", nil, &result)
	return &result, err
}

func (s *APIKeyService) List(ctx context.Context, params *APIKeyListParams) (*ListResult[APIKey], error) {
	path := "/api/api_keys" + buildQuery(params)
	return doList[APIKey](s.client, ctx, path)
}

func (s *APIKeyService) Create(ctx context.Context, params *APIKeyCreateParams) (*APIKey, error) {
	var result APIKey
	body := map[string]any{"api_key": params}
	err := s.client.do(ctx, http.MethodPost, "/api/api_keys", body, &result)
	return &result, err
}

func (s *APIKeyService) Delete(ctx context.Context, id int64) error {
	return s.client.do(ctx, http.MethodDelete, fmt.Sprintf("/api/api_keys/%d", id), nil, nil)
}
