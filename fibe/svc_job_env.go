package fibe

import (
	"context"
	"fmt"
	"net/http"
)

type JobEnvService struct {
	client *Client
}

func (s *JobEnvService) List(ctx context.Context, params *JobEnvListParams) (*ListResult[JobEnvEntry], error) {
	path := "/api/job_env" + buildQuery(params)
	return doList[JobEnvEntry](s.client, ctx, path)
}

func (s *JobEnvService) Get(ctx context.Context, id int64, reveal bool) (*JobEnvEntry, error) {
	var result JobEnvEntry
	path := fmt.Sprintf("/api/job_env/%d", id)
	if reveal {
		path += "?reveal=true"
	}
	err := s.client.do(ctx, http.MethodGet, path, nil, &result)
	return &result, err
}

func (s *JobEnvService) Set(ctx context.Context, params *JobEnvSetParams) (*JobEnvEntry, error) {
	if err := validateParams(params); err != nil {
		return nil, err
	}
	var result JobEnvEntry
	body := map[string]any{"job_env": params}
	err := s.client.do(ctx, http.MethodPost, "/api/job_env", body, &result)
	return &result, err
}

func (s *JobEnvService) Update(ctx context.Context, id int64, params *JobEnvUpdateParams) (*JobEnvEntry, error) {
	var result JobEnvEntry
	body := map[string]any{"job_env": params}
	err := s.client.do(ctx, http.MethodPatch, fmt.Sprintf("/api/job_env/%d", id), body, &result)
	return &result, err
}

func (s *JobEnvService) Delete(ctx context.Context, id int64) error {
	return s.client.do(ctx, http.MethodDelete, fmt.Sprintf("/api/job_env/%d", id), nil, nil)
}
