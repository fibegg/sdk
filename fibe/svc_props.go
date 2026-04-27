package fibe

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

type PropService struct {
	client *Client
}

func (s *PropService) List(ctx context.Context, params *PropListParams) (*ListResult[Prop], error) {
	path := "/api/props" + buildQuery(params)
	return doList[Prop](s.client, ctx, path)
}

func (s *PropService) WithDockerCompose(ctx context.Context, params *PropListParams) (*ListResult[Prop], error) {
	path := "/api/props/with_docker_compose" + buildQuery(params)
	return doList[Prop](s.client, ctx, path)
}

func (s *PropService) Get(ctx context.Context, id int64) (*Prop, error) {
	var result Prop
	err := s.client.do(ctx, http.MethodGet, fmt.Sprintf("/api/props/%d", id), nil, &result)
	return &result, err
}

func (s *PropService) Create(ctx context.Context, params *PropCreateParams) (*Prop, error) {
	if err := validateParams(params); err != nil {
		return nil, err
	}
	var result Prop
	body := map[string]any{"prop": params}
	err := s.client.do(ctx, http.MethodPost, "/api/props", body, &result)
	return &result, err
}

func (s *PropService) Attach(ctx context.Context, repoFullName string) (*Prop, error) {
	var result Prop
	body := map[string]any{"repo_full_name": repoFullName}
	err := s.client.do(ctx, http.MethodPost, "/api/props/attach", body, &result)
	return &result, err
}

func (s *PropService) Mirror(ctx context.Context, sourceURL string, name string) (*Prop, error) {
	var result Prop
	body := map[string]any{"source_url": sourceURL}
	if name != "" {
		body["name"] = name
	}
	err := s.client.do(ctx, http.MethodPost, "/api/props/mirror", body, &result)
	return &result, err
}


func (s *PropService) Update(ctx context.Context, id int64, params *PropUpdateParams) (*Prop, error) {
	var result Prop
	body := map[string]any{"prop": params}
	err := s.client.do(ctx, http.MethodPatch, fmt.Sprintf("/api/props/%d", id), body, &result)
	return &result, err
}

func (s *PropService) Delete(ctx context.Context, id int64) error {
	return s.client.do(ctx, http.MethodDelete, fmt.Sprintf("/api/props/%d", id), nil, nil)
}

func (s *PropService) Sync(ctx context.Context, id int64) error {
	var result map[string]any
	return s.client.do(ctx, http.MethodPost, fmt.Sprintf("/api/props/%d/sync", id), nil, &result)
}

func (s *PropService) Branches(ctx context.Context, id int64, query string, limit int) (*PropBranches, error) {
	path := fmt.Sprintf("/api/props/%d/branches", id)
	values := url.Values{}
	if query != "" {
		values.Set("query", query)
	}
	if limit > 0 {
		values.Set("limit", fmt.Sprintf("%d", limit))
	}
	if encoded := values.Encode(); encoded != "" {
		path += "?" + encoded
	}
	var result PropBranches
	err := s.client.do(ctx, http.MethodGet, path, nil, &result)
	return &result, err
}

func (s *PropService) EnvDefaults(ctx context.Context, id int64, branch string, envFilePath string) (*PropEnvDefaults, error) {
	values := url.Values{}
	values.Set("branch", branch)
	if envFilePath != "" {
		values.Set("env_file_path", envFilePath)
	}
	path := fmt.Sprintf("/api/props/%d/env_defaults?%s", id, values.Encode())
	var result PropEnvDefaults
	err := s.client.do(ctx, http.MethodGet, path, nil, &result)
	return &result, err
}
