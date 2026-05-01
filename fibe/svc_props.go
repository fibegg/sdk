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
	return s.GetByIdentifier(ctx, int64Identifier(id))
}

func (s *PropService) GetByIdentifier(ctx context.Context, identifier string) (*Prop, error) {
	var result Prop
	err := s.client.do(ctx, http.MethodGet, identifierPath("/api/props", identifier), nil, &result)
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
	return s.UpdateByIdentifier(ctx, int64Identifier(id), params)
}

func (s *PropService) UpdateByIdentifier(ctx context.Context, identifier string, params *PropUpdateParams) (*Prop, error) {
	var result Prop
	body := map[string]any{"prop": params}
	err := s.client.do(ctx, http.MethodPatch, identifierPath("/api/props", identifier), body, &result)
	return &result, err
}

func (s *PropService) Delete(ctx context.Context, id int64) error {
	return s.DeleteByIdentifier(ctx, int64Identifier(id))
}

func (s *PropService) DeleteByIdentifier(ctx context.Context, identifier string) error {
	return s.client.do(ctx, http.MethodDelete, identifierPath("/api/props", identifier), nil, nil)
}

func (s *PropService) Sync(ctx context.Context, id int64) error {
	return s.SyncByIdentifier(ctx, int64Identifier(id))
}

func (s *PropService) SyncByIdentifier(ctx context.Context, identifier string) error {
	var result map[string]any
	return s.client.do(ctx, http.MethodPost, identifierPath("/api/props", identifier)+"/sync", nil, &result)
}

func (s *PropService) Branches(ctx context.Context, id int64, query string, limit int) (*PropBranches, error) {
	return s.BranchesByIdentifier(ctx, int64Identifier(id), query, limit)
}

func (s *PropService) BranchesByIdentifier(ctx context.Context, identifier string, query string, limit int) (*PropBranches, error) {
	path := identifierPath("/api/props", identifier) + "/branches"
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
	return s.EnvDefaultsByIdentifier(ctx, int64Identifier(id), branch, envFilePath)
}

func (s *PropService) EnvDefaultsByIdentifier(ctx context.Context, identifier string, branch string, envFilePath string) (*PropEnvDefaults, error) {
	values := url.Values{}
	values.Set("branch", branch)
	if envFilePath != "" {
		values.Set("env_file_path", envFilePath)
	}
	path := identifierPath("/api/props", identifier) + "/env_defaults?" + values.Encode()
	var result PropEnvDefaults
	err := s.client.do(ctx, http.MethodGet, path, nil, &result)
	return &result, err
}
