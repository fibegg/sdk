package fibe

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

type RepositoryService struct {
	client *Client
}

func (s *RepositoryService) List(ctx context.Context, params *RepositoryListParams) (*ListResult[Repository], error) {
	path := "/api/repositories" + buildQuery(params)
	return doList[Repository](s.client, ctx, path)
}

func (s *RepositoryService) Get(ctx context.Context, id int64) (*Repository, error) {
	return s.GetByIdentifier(ctx, int64Identifier(id))
}

func (s *RepositoryService) GetByIdentifier(ctx context.Context, identifier string) (*Repository, error) {
	var result Repository
	err := s.client.do(ctx, http.MethodGet, identifierPath("/api/repositories", identifier), nil, &result)
	return &result, err
}

func (s *RepositoryService) Create(ctx context.Context, params *RepositoryCreateParams) (*Repository, error) {
	if err := validateParams(params); err != nil {
		return nil, err
	}
	var result Repository
	body := map[string]any{"repository": params}
	err := s.client.do(ctx, http.MethodPost, "/api/repositories", body, &result)
	return &result, err
}

func (s *RepositoryService) Attach(ctx context.Context, repoFullName string) (*Repository, error) {
	return s.AttachWithParams(ctx, &RepositoryAttachParams{RepoFullName: repoFullName})
}

func (s *RepositoryService) AttachWithParams(ctx context.Context, params *RepositoryAttachParams) (*Repository, error) {
	if params == nil {
		return nil, fmt.Errorf("fibe: attachment parameters are required")
	}
	if err := params.Validate(); err != nil {
		return nil, err
	}
	var result Repository
	err := s.client.do(ctx, http.MethodPost, "/api/repositories/attachments", params, &result)
	return &result, err
}

func (s *RepositoryService) Mirror(ctx context.Context, sourceURL string, name string) (*Repository, error) {
	result, err := s.MirrorWithParams(ctx, &RepositoryMirrorParams{SourceURL: sourceURL, Name: name})
	if result == nil {
		return nil, err
	}
	return &result.Repository, err
}

// MirrorWithParams imports through the selected owner's provider and waits for Company completion.
func (s *RepositoryService) MirrorWithParams(ctx context.Context, params *RepositoryMirrorParams) (*RepositoryMirrorState, error) {
	if params == nil {
		return nil, fmt.Errorf("fibe: mirror parameters are required")
	}
	if err := params.Validate(); err != nil {
		return nil, err
	}
	var wire struct {
		RepositoryMirrorState
		Imported *RepositoryMirrorState `json:"repository"`
	}
	err := s.client.doAsync(WithFields(ctx), http.MethodPost, "/api/repositories/mirrors", "/api/company_repository_operations/%s", params, &wire)
	if err != nil {
		return nil, err
	}
	result := &wire.RepositoryMirrorState
	if wire.Imported != nil {
		result = wire.Imported
	}
	return ProjectFields(result, fieldsFromCtx(ctx)), nil
}

func (s *RepositoryService) Update(ctx context.Context, id int64, params *RepositoryUpdateParams) (*Repository, error) {
	return s.UpdateByIdentifier(ctx, int64Identifier(id), params)
}

func (s *RepositoryService) UpdateByIdentifier(ctx context.Context, identifier string, params *RepositoryUpdateParams) (*Repository, error) {
	var result Repository
	body := map[string]any{"repository": params}
	err := s.client.do(ctx, http.MethodPatch, identifierPath("/api/repositories", identifier), body, &result)
	return &result, err
}

func (s *RepositoryService) Delete(ctx context.Context, id int64) error {
	return s.DeleteByIdentifier(ctx, int64Identifier(id))
}

func (s *RepositoryService) DeleteByIdentifier(ctx context.Context, identifier string) error {
	return s.client.do(ctx, http.MethodDelete, identifierPath("/api/repositories", identifier), nil, nil)
}

func (s *RepositoryService) Sync(ctx context.Context, id int64) error {
	return s.SyncByIdentifier(ctx, int64Identifier(id))
}

func (s *RepositoryService) SyncByIdentifier(ctx context.Context, identifier string) error {
	var result map[string]any
	return s.client.do(ctx, http.MethodPost, identifierPath("/api/repositories", identifier)+"/syncs", nil, &result)
}

func (s *RepositoryService) Branches(ctx context.Context, id int64, query string, limit int) (*RepositoryBranches, error) {
	return s.BranchesByIdentifier(ctx, int64Identifier(id), query, limit)
}

func (s *RepositoryService) BranchesByIdentifier(ctx context.Context, identifier string, query string, limit int) (*RepositoryBranches, error) {
	path := identifierPath("/api/repositories", identifier) + "/branches"
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
	var result RepositoryBranches
	err := s.client.do(ctx, http.MethodGet, path, nil, &result)
	return &result, err
}

func (s *RepositoryService) EnvDefaults(ctx context.Context, id int64, branch string, envFilePath string) (*RepositoryEnvDefaults, error) {
	return s.EnvDefaultsByIdentifier(ctx, int64Identifier(id), branch, envFilePath)
}

func (s *RepositoryService) EnvDefaultsByIdentifier(ctx context.Context, identifier string, branch string, envFilePath string) (*RepositoryEnvDefaults, error) {
	values := url.Values{}
	values.Set("branch", branch)
	if envFilePath != "" {
		values.Set("env_file_path", envFilePath)
	}
	path := identifierPath("/api/repositories", identifier) + "/env_defaults?" + values.Encode()
	var result RepositoryEnvDefaults
	err := s.client.do(ctx, http.MethodGet, path, nil, &result)
	return &result, err
}
