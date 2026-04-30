package fibe

import (
	"context"
	"fmt"
	"net/http"
)

type MemoryService struct {
	client *Client
}

func (s *MemoryService) List(ctx context.Context, params *MemoryListParams) (*ListResult[Memory], error) {
	path := "/api/memories"
	if params != nil {
		path += buildQuery(params)
	}
	return doList[Memory](s.client, ctx, path)
}

func (s *MemoryService) Get(ctx context.Context, id int64) (*Memory, error) {
	var result Memory
	path := fmt.Sprintf("/api/memories/%d", id)
	err := s.client.do(ctx, http.MethodGet, path, nil, &result)
	return &result, err
}

func (s *MemoryService) Memorize(ctx context.Context, payload map[string]any) (*MemoryMemorizeResult, error) {
	var result MemoryMemorizeResult
	err := s.client.do(ctx, http.MethodPost, "/api/memories/memorize", payload, &result)
	return &result, err
}

func (s *MemoryService) Delete(ctx context.Context, id int64) error {
	path := fmt.Sprintf("/api/memories/%d", id)
	return s.client.do(ctx, http.MethodDelete, path, nil, nil)
}
