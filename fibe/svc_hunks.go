package fibe

import (
	"context"
	"fmt"
	"net/http"
)

type HunkService struct {
	client *Client
}

func (s *HunkService) List(ctx context.Context, propID int64, params *HunkListParams) (*ListResult[Hunk], error) {
	path := fmt.Sprintf("/api/props/%d/hunks", propID)
	if params != nil {
		path += buildQuery(params)
	}
	return doList[Hunk](s.client, ctx, path)
}

func (s *HunkService) Get(ctx context.Context, propID, id int64) (*Hunk, error) {
	var result Hunk
	path := fmt.Sprintf("/api/props/%d/hunks/%d", propID, id)
	err := s.client.do(ctx, http.MethodGet, path, nil, &result)
	return &result, err
}

func (s *HunkService) Update(ctx context.Context, propID, id int64, params *HunkUpdateParams) (*Hunk, error) {
	var result Hunk
	path := fmt.Sprintf("/api/props/%d/hunks/%d", propID, id)
	err := s.client.do(ctx, http.MethodPatch, path, params, &result)
	return &result, err
}

func (s *HunkService) Ingest(ctx context.Context, propID int64, force bool) error {
	body := map[string]any{}
	if force {
		body["force"] = true
	}
	var result map[string]any
	path := fmt.Sprintf("/api/props/%d/hunks/ingest", propID)
	return s.client.do(ctx, http.MethodPost, path, body, &result)
}

func (s *HunkService) Next(ctx context.Context, propID int64, processorName string) (*Hunk, error) {
	var result Hunk
	body := map[string]any{"processor_name": processorName}
	path := fmt.Sprintf("/api/props/%d/hunks/next", propID)
	err := s.client.do(ctx, http.MethodPost, path, body, &result)
	return &result, err
}
