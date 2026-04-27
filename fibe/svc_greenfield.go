package fibe

import (
	"context"
	"net/http"
)

type GreenfieldService struct {
	client *Client
}

func (s *GreenfieldService) Create(ctx context.Context, params *GreenfieldCreateParams) (*GreenfieldResult, error) {
	if err := validateParams(params); err != nil {
		return nil, err
	}
	var result GreenfieldResult
	err := s.client.do(ctx, http.MethodPost, "/api/greenfield", params, &result)
	return &result, err
}
