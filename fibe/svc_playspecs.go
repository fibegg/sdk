package fibe

import (
	"context"
	"fmt"
	"io"
	"net/http"
)

type PlayspecService struct {
	client *Client
}

func (s *PlayspecService) List(ctx context.Context, params *PlayspecListParams) (*ListResult[Playspec], error) {
	path := "/api/playspecs" + buildQuery(params)
	return doList[Playspec](s.client, ctx, path)
}

func (s *PlayspecService) Get(ctx context.Context, id int64) (*Playspec, error) {
	var result Playspec
	err := s.client.do(ctx, http.MethodGet, fmt.Sprintf("/api/playspecs/%d", id), nil, &result)
	return &result, err
}

func (s *PlayspecService) Create(ctx context.Context, params *PlayspecCreateParams) (*Playspec, error) {
	if err := validateParams(params); err != nil {
		return nil, err
	}
	var result Playspec
	body := map[string]any{"playspec": params}
	err := s.client.do(ctx, http.MethodPost, "/api/playspecs", body, &result)
	return &result, err
}

func (s *PlayspecService) Update(ctx context.Context, id int64, params *PlayspecUpdateParams) (*Playspec, error) {
	var result Playspec
	body := map[string]any{"playspec": params}
	err := s.client.do(ctx, http.MethodPatch, fmt.Sprintf("/api/playspecs/%d", id), body, &result)
	return &result, err
}

func (s *PlayspecService) Delete(ctx context.Context, id int64) error {
	return s.client.do(ctx, http.MethodDelete, fmt.Sprintf("/api/playspecs/%d", id), nil, nil)
}

func (s *PlayspecService) Services(ctx context.Context, id int64) ([]any, error) {
	var result []any
	err := s.client.do(ctx, http.MethodGet, fmt.Sprintf("/api/playspecs/%d/services", id), nil, &result)
	return result, err
}

func (s *PlayspecService) ValidateCompose(ctx context.Context, composeYAML string) (*ComposeValidation, error) {
	var result ComposeValidation
	body := map[string]any{"compose_yaml": composeYAML}
	err := s.client.do(ctx, http.MethodPost, "/api/playspecs/validate_compose", body, &result)
	return &result, err
}

func (s *PlayspecService) AddMountedFile(ctx context.Context, id int64, file io.Reader, fileName string, params *MountedFileParams) error {
	fields := map[string]string{
		"mount_path": params.MountPath,
	}
	if params.ReadOnly != nil {
		if *params.ReadOnly {
			fields["readonly"] = "true"
		} else {
			fields["readonly"] = "false"
		}
	}
	path := fmt.Sprintf("/api/playspecs/%d/add_mounted_file", id)
	return s.client.doMultipart(ctx, http.MethodPost, path, fields, "file", fileName, file, nil)
}

func (s *PlayspecService) UpdateMountedFile(ctx context.Context, id int64, params *MountedFileUpdateParams) error {
	path := fmt.Sprintf("/api/playspecs/%d/update_mounted_file", id)
	return s.client.do(ctx, http.MethodPatch, path, params, nil)
}

func (s *PlayspecService) RemoveMountedFile(ctx context.Context, id int64, filename string) error {
	path := fmt.Sprintf("/api/playspecs/%d/remove_mounted_file", id)
	body := map[string]any{"filename": filename}
	return s.client.do(ctx, http.MethodDelete, path, body, nil)
}

func (s *PlayspecService) AddRegistryCredential(ctx context.Context, id int64, params *RegistryCredentialParams) (*RegistryCredentialResult, error) {
	path := fmt.Sprintf("/api/playspecs/%d/add_registry_credential", id)
	var result RegistryCredentialResult
	err := s.client.do(ctx, http.MethodPost, path, params, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (s *PlayspecService) RemoveRegistryCredential(ctx context.Context, id int64, credentialID string) error {
	path := fmt.Sprintf("/api/playspecs/%d/remove_registry_credential", id)
	body := map[string]any{"credential_id": credentialID}
	return s.client.do(ctx, http.MethodDelete, path, body, nil)
}
