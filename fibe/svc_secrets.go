package fibe

import (
	"context"
	"net/http"
)

type SecretService struct {
	client *Client
}

func (s *SecretService) List(ctx context.Context, params *SecretListParams) (*ListResult[Secret], error) {
	path := "/api/secrets" + buildQuery(params)
	return doList[Secret](s.client, ctx, path)
}

func (s *SecretService) Get(ctx context.Context, id int64, reveal bool) (*Secret, error) {
	return s.GetByIdentifier(ctx, int64Identifier(id), reveal)
}

func (s *SecretService) GetByIdentifier(ctx context.Context, identifier string, reveal bool) (*Secret, error) {
	var result Secret
	path := identifierPath("/api/secrets", identifier)
	if reveal {
		path += "?reveal=true"
	}
	err := s.client.do(ctx, http.MethodGet, path, nil, &result)
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
	return s.UpdateByIdentifier(ctx, int64Identifier(id), params)
}

func (s *SecretService) UpdateByIdentifier(ctx context.Context, identifier string, params *SecretUpdateParams) (*Secret, error) {
	var result Secret
	body := map[string]any{"secret": params}
	err := s.client.do(ctx, http.MethodPatch, identifierPath("/api/secrets", identifier), body, &result)
	return &result, err
}

func (s *SecretService) Delete(ctx context.Context, id int64) error {
	return s.DeleteByIdentifier(ctx, int64Identifier(id))
}

func (s *SecretService) DeleteByIdentifier(ctx context.Context, identifier string) error {
	return s.client.do(ctx, http.MethodDelete, identifierPath("/api/secrets", identifier), nil, nil)
}
