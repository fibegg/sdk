package fibe

import (
	"context"
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
	return s.GetByIdentifier(ctx, int64Identifier(id))
}

func (s *PlayspecService) GetByIdentifier(ctx context.Context, identifier string) (*Playspec, error) {
	var result Playspec
	err := s.client.do(ctx, http.MethodGet, identifierPath("/api/playspecs", identifier), nil, &result)
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
	return s.UpdateByIdentifier(ctx, int64Identifier(id), params)
}

func (s *PlayspecService) UpdateByIdentifier(ctx context.Context, identifier string, params *PlayspecUpdateParams) (*Playspec, error) {
	var result Playspec
	body := map[string]any{"playspec": params}
	err := s.client.do(ctx, http.MethodPatch, identifierPath("/api/playspecs", identifier), body, &result)
	return &result, err
}

func (s *PlayspecService) Delete(ctx context.Context, id int64) error {
	return s.DeleteByIdentifier(ctx, int64Identifier(id))
}

func (s *PlayspecService) DeleteByIdentifier(ctx context.Context, identifier string) error {
	return s.client.do(ctx, http.MethodDelete, identifierPath("/api/playspecs", identifier), nil, nil)
}

func (s *PlayspecService) Services(ctx context.Context, id int64) ([]any, error) {
	return s.ServicesByIdentifier(ctx, int64Identifier(id))
}

func (s *PlayspecService) ServicesByIdentifier(ctx context.Context, identifier string) ([]any, error) {
	var result []any
	err := s.client.do(ctx, http.MethodGet, identifierPath("/api/playspecs", identifier)+"/services", nil, &result)
	return result, err
}

func (s *PlayspecService) ValidateCompose(ctx context.Context, composeYAML string) (*ComposeValidation, error) {
	return s.ValidateComposeWithParams(ctx, &ComposeValidateParams{ComposeYAML: composeYAML})
}

func (s *PlayspecService) ValidateComposeWithParams(ctx context.Context, params *ComposeValidateParams) (*ComposeValidation, error) {
	if params == nil {
		params = &ComposeValidateParams{}
	}
	if errors, err := s.validateComposeSchema(ctx, params.ComposeYAML); err != nil {
		return nil, err
	} else if len(errors) > 0 {
		return &ComposeValidation{Valid: false, Errors: errors}, nil
	}

	var result ComposeValidation
	err := s.client.do(ctx, http.MethodPost, "/api/playspecs/validate_compose", params, &result)
	return &result, err
}

func (s *PlayspecService) PreviewTemplateVersionSwitch(ctx context.Context, id int64, params *PlayspecTemplateVersionSwitchParams) (*PlayspecTemplateVersionSwitchPreview, error) {
	return s.PreviewTemplateVersionSwitchByIdentifier(ctx, int64Identifier(id), params)
}

func (s *PlayspecService) PreviewTemplateVersionSwitchByIdentifier(ctx context.Context, identifier string, params *PlayspecTemplateVersionSwitchParams) (*PlayspecTemplateVersionSwitchPreview, error) {
	var result PlayspecTemplateVersionSwitchPreview
	path := identifierPath("/api/playspecs", identifier) + "/template_version_switch/preview"
	err := s.client.do(ctx, http.MethodPost, path, params, &result)
	return &result, err
}

func (s *PlayspecService) SwitchTemplateVersion(ctx context.Context, id int64, params *PlayspecTemplateVersionSwitchParams) (*PlayspecTemplateVersionSwitchResult, error) {
	return s.SwitchTemplateVersionByIdentifier(ctx, int64Identifier(id), params)
}

func (s *PlayspecService) SwitchTemplateVersionByIdentifier(ctx context.Context, identifier string, params *PlayspecTemplateVersionSwitchParams) (*PlayspecTemplateVersionSwitchResult, error) {
	var result PlayspecTemplateVersionSwitchResult
	path := identifierPath("/api/playspecs", identifier) + "/template_version_switch"
	err := s.client.do(ctx, http.MethodPost, path, params, &result)
	return &result, err
}

func (s *PlayspecService) AddMountedFile(ctx context.Context, id int64, file io.Reader, fileName string, params *MountedFileParams) error {
	return s.AddMountedFileByIdentifier(ctx, int64Identifier(id), file, fileName, params)
}

func (s *PlayspecService) AddMountedFileByIdentifier(ctx context.Context, identifier string, file io.Reader, fileName string, params *MountedFileParams) error {
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
	path := identifierPath("/api/playspecs", identifier) + "/add_mounted_file"
	return s.client.doMultipart(ctx, http.MethodPost, path, fields, "file", fileName, file, nil)
}

func (s *PlayspecService) UpdateMountedFile(ctx context.Context, id int64, params *MountedFileUpdateParams) error {
	return s.UpdateMountedFileByIdentifier(ctx, int64Identifier(id), params)
}

func (s *PlayspecService) UpdateMountedFileByIdentifier(ctx context.Context, identifier string, params *MountedFileUpdateParams) error {
	path := identifierPath("/api/playspecs", identifier) + "/update_mounted_file"
	return s.client.do(ctx, http.MethodPatch, path, params, nil)
}

func (s *PlayspecService) RemoveMountedFile(ctx context.Context, id int64, filename string) error {
	return s.RemoveMountedFileByIdentifier(ctx, int64Identifier(id), filename)
}

func (s *PlayspecService) RemoveMountedFileByIdentifier(ctx context.Context, identifier string, filename string) error {
	path := identifierPath("/api/playspecs", identifier) + "/remove_mounted_file"
	body := map[string]any{"filename": filename}
	return s.client.do(ctx, http.MethodDelete, path, body, nil)
}

func (s *PlayspecService) AddRegistryCredential(ctx context.Context, id int64, params *RegistryCredentialParams) (*RegistryCredentialResult, error) {
	return s.AddRegistryCredentialByIdentifier(ctx, int64Identifier(id), params)
}

func (s *PlayspecService) AddRegistryCredentialByIdentifier(ctx context.Context, identifier string, params *RegistryCredentialParams) (*RegistryCredentialResult, error) {
	path := identifierPath("/api/playspecs", identifier) + "/add_registry_credential"
	var result RegistryCredentialResult
	err := s.client.do(ctx, http.MethodPost, path, params, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (s *PlayspecService) RemoveRegistryCredential(ctx context.Context, id int64, credentialID string) error {
	return s.RemoveRegistryCredentialByIdentifier(ctx, int64Identifier(id), credentialID)
}

func (s *PlayspecService) RemoveRegistryCredentialByIdentifier(ctx context.Context, identifier string, credentialID string) error {
	path := identifierPath("/api/playspecs", identifier) + "/remove_registry_credential"
	body := map[string]any{"credential_id": credentialID}
	return s.client.do(ctx, http.MethodDelete, path, body, nil)
}
