package fibe

import (
	"context"
	"fmt"
	"io"
	"net/http"
)

type SpecService struct {
	client *Client
}

func (s *SpecService) List(ctx context.Context, params *SpecListParams) (*ListResult[Spec], error) {
	path := "/api/specs" + buildQuery(params)
	return doList[Spec](s.client, ctx, path)
}

func (s *SpecService) Get(ctx context.Context, id int64) (*Spec, error) {
	return s.GetByIdentifier(ctx, int64Identifier(id))
}

func (s *SpecService) GetByIdentifier(ctx context.Context, identifier string) (*Spec, error) {
	var result Spec
	err := s.client.do(ctx, http.MethodGet, identifierPath("/api/specs", identifier), nil, &result)
	return &result, err
}

func (s *SpecService) Create(ctx context.Context, params *SpecCreateParams) (*Spec, error) {
	if err := validateParams(params); err != nil {
		return nil, err
	}
	var result Spec
	body := map[string]any{"spec": params}
	err := s.client.do(ctx, http.MethodPost, "/api/specs", body, &result)
	return &result, err
}

func (s *SpecService) Update(ctx context.Context, id int64, params *SpecUpdateParams) (*Spec, error) {
	return s.UpdateByIdentifier(ctx, int64Identifier(id), params)
}

func (s *SpecService) UpdateByIdentifier(ctx context.Context, identifier string, params *SpecUpdateParams) (*Spec, error) {
	var result Spec
	body := map[string]any{"spec": params}
	err := s.client.do(ctx, http.MethodPatch, identifierPath("/api/specs", identifier), body, &result)
	return &result, err
}

func (s *SpecService) Delete(ctx context.Context, id int64) error {
	return s.DeleteByIdentifier(ctx, int64Identifier(id))
}

func (s *SpecService) DeleteByIdentifier(ctx context.Context, identifier string) error {
	return s.client.do(ctx, http.MethodDelete, identifierPath("/api/specs", identifier), nil, nil)
}

func (s *SpecService) Services(ctx context.Context, id int64) ([]any, error) {
	return s.ServicesByIdentifier(ctx, int64Identifier(id))
}

func (s *SpecService) ServicesByIdentifier(ctx context.Context, identifier string) ([]any, error) {
	var result []any
	err := s.client.do(ctx, http.MethodGet, identifierPath("/api/specs", identifier)+"/services", nil, &result)
	return result, err
}

func (s *SpecService) ValidateCompose(ctx context.Context, composeYAML string) (*ComposeValidation, error) {
	return s.ValidateComposeWithParams(ctx, &ComposeValidateParams{ComposeYAML: composeYAML})
}

func (s *SpecService) ValidateComposeWithParams(ctx context.Context, params *ComposeValidateParams) (*ComposeValidation, error) {
	if params == nil {
		params = &ComposeValidateParams{}
	}
	if errors, err := s.validateComposeSchema(ctx, params.ComposeYAML); err != nil {
		return nil, err
	} else if len(errors) > 0 {
		return &ComposeValidation{Valid: false, Errors: errors}, nil
	}

	var result ComposeValidation
	err := s.client.do(ctx, http.MethodPost, "/api/compose_validations", params, &result)
	return &result, err
}

func (s *SpecService) PreviewTemplateVersionSwitch(ctx context.Context, id int64, params *SpecTemplateVersionSwitchParams) (*SpecTemplateVersionSwitchPreview, error) {
	return s.PreviewTemplateVersionSwitchByIdentifier(ctx, int64Identifier(id), params)
}

func (s *SpecService) PreviewTemplateVersionSwitchByIdentifier(ctx context.Context, identifier string, params *SpecTemplateVersionSwitchParams) (*SpecTemplateVersionSwitchPreview, error) {
	var result SpecTemplateVersionSwitchPreview
	path := identifierPath("/api/specs", identifier) + "/template_switch_previews"
	err := s.client.do(ctx, http.MethodPost, path, params, &result)
	return &result, err
}

func (s *SpecService) SwitchTemplateVersion(ctx context.Context, id int64, params *SpecTemplateVersionSwitchParams) (*SpecTemplateVersionSwitchResult, error) {
	return s.SwitchTemplateVersionByIdentifier(ctx, int64Identifier(id), params)
}

func (s *SpecService) SwitchTemplateVersionByIdentifier(ctx context.Context, identifier string, params *SpecTemplateVersionSwitchParams) (*SpecTemplateVersionSwitchResult, error) {
	var result SpecTemplateVersionSwitchResult
	path := identifierPath("/api/specs", identifier) + "/template_switches"
	err := s.client.doAsync(ctx, http.MethodPost, path, "/api/async_requests/%s", params, &result)
	return &result, err
}

func VerifyTemplateVersionSwitchResult(result *SpecTemplateVersionSwitchResult, targetVersionID int64) error {
	if result == nil {
		return fmt.Errorf("template version switch did not return a result")
	}
	if result.TargetTemplateVersion == nil || result.TargetTemplateVersion.ID == nil {
		return fmt.Errorf("template version switch did not return target_template_version")
	}
	if *result.TargetTemplateVersion.ID != targetVersionID {
		return fmt.Errorf("template version switch returned target_template_version %d, expected %d", *result.TargetTemplateVersion.ID, targetVersionID)
	}
	if result.NoOp {
		return nil
	}
	if result.Spec == nil {
		return fmt.Errorf("template version switch did not return the updated spec")
	}
	if result.Spec.SourceTemplateVersionID == nil {
		return fmt.Errorf("template version switch result missing spec.source_template_version_id")
	}
	if *result.Spec.SourceTemplateVersionID != targetVersionID {
		return fmt.Errorf("template version switch did not apply target version %d; spec has source_template_version_id %d", targetVersionID, *result.Spec.SourceTemplateVersionID)
	}
	return nil
}

func (s *SpecService) AddMountedFile(ctx context.Context, id int64, file io.Reader, fileName string, params *MountedFileParams) error {
	return s.AddMountedFileByIdentifier(ctx, int64Identifier(id), file, fileName, params)
}

func (s *SpecService) AddMountedFileByIdentifier(ctx context.Context, identifier string, file io.Reader, fileName string, params *MountedFileParams) error {
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
	path := identifierPath("/api/specs", identifier) + "/mounts"
	return s.client.doMultipart(ctx, http.MethodPost, path, fields, "file", fileName, file, nil)
}

func (s *SpecService) UpdateMountedFile(ctx context.Context, id int64, params *MountedFileUpdateParams) error {
	return s.UpdateMountedFileByIdentifier(ctx, int64Identifier(id), params)
}

func (s *SpecService) UpdateMountedFileByIdentifier(ctx context.Context, identifier string, params *MountedFileUpdateParams) error {
	path := identifierPath("/api/specs", identifier) + "/mounts"
	return s.client.do(ctx, http.MethodPatch, path, params, nil)
}

func (s *SpecService) RemoveMountedFile(ctx context.Context, id int64, filename string) error {
	return s.RemoveMountedFileByIdentifier(ctx, int64Identifier(id), filename)
}

func (s *SpecService) RemoveMountedFileByIdentifier(ctx context.Context, identifier string, filename string) error {
	path := identifierPath("/api/specs", identifier) + "/mounts"
	body := map[string]any{"filename": filename}
	return s.client.do(ctx, http.MethodDelete, path, body, nil)
}

func (s *SpecService) AddRegistryCredential(ctx context.Context, id int64, params *RegistryCredentialParams) (*RegistryCredentialResult, error) {
	return s.AddRegistryCredentialByIdentifier(ctx, int64Identifier(id), params)
}

func (s *SpecService) AddRegistryCredentialByIdentifier(ctx context.Context, identifier string, params *RegistryCredentialParams) (*RegistryCredentialResult, error) {
	path := identifierPath("/api/specs", identifier) + "/registry_credentials"
	var result RegistryCredentialResult
	err := s.client.do(ctx, http.MethodPost, path, params, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (s *SpecService) RemoveRegistryCredential(ctx context.Context, id int64, credentialID string) error {
	return s.RemoveRegistryCredentialByIdentifier(ctx, int64Identifier(id), credentialID)
}

func (s *SpecService) RemoveRegistryCredentialByIdentifier(ctx context.Context, identifier string, credentialID string) error {
	path := identifierPath("/api/specs", identifier) + "/registry_credentials"
	body := map[string]any{"credential_id": credentialID}
	return s.client.do(ctx, http.MethodDelete, path, body, nil)
}
