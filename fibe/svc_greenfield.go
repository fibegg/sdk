package fibe

import (
	"context"
	"net/http"
)

type GreenfieldService struct {
	client *Client
}

// Create initializes a greenfield environment. The API returns 202 Accepted;
// this method automatically polls for the final result.
func (s *GreenfieldService) Create(ctx context.Context, params *GreenfieldCreateParams) (*GreenfieldResult, error) {
	if err := validateParams(params); err != nil {
		return nil, err
	}
	var result GreenfieldResult
	err := s.client.doAsync(ctx, http.MethodPost, "/api/greenfield", "/api/greenfield/%s", params, &result)
	return &result, err
}
