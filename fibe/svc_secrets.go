package fibe

import (
	"context"
	"fmt"
	"net/http"
)

type SecretService struct {
	client *Client
}

func (s *SecretService) List(ctx context.Context, params *SecretListParams) (*ListResult[Secret], error) {
	path := "/api/secrets" + buildQuery(params)
	return doList[Secret](s.client, ctx, path)
}

func (s *SecretService) Get(ctx context.Context, id int64) (*Secret, error) {
	var result Secret
	err := s.client.do(ctx, http.MethodGet, fmt.Sprintf("/api/secrets/%d", id), nil, &result)
	return &result, err
}

func (s *SecretService) Create(ctx context.Context, params *SecretCreateParams) (*Secret, error) {
	if err := validateParams(params); err != nil {
		return nil, err
	}
	var result Secret
	body := map[string]any{"secret": params}
	err := s.client.do(ctx, http.MethodPost, "/api/secrets", body, &result)
	return &result, err
}

func (s *SecretService) Update(ctx context.Context, id int64, params *SecretUpdateParams) (*Secret, error) {
	var result Secret
	body := map[string]any{"secret": params}
	err := s.client.do(ctx, http.MethodPatch, fmt.Sprintf("/api/secrets/%d", id), body, &result)
	return &result, err
}

func (s *SecretService) Delete(ctx context.Context, id int64) error {
	return s.client.do(ctx, http.MethodDelete, fmt.Sprintf("/api/secrets/%d", id), nil, nil)
}
