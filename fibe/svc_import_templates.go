package fibe

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

type ImportTemplateService struct {
	client *Client
}

func (s *ImportTemplateService) List(ctx context.Context, params *ImportTemplateListParams) (*ListResult[ImportTemplate], error) {
	path := "/api/import_templates" + buildQuery(params)
	return doList[ImportTemplate](s.client, ctx, path)
}

func (s *ImportTemplateService) Search(ctx context.Context, query string, templateID *int64) (*ListResult[ImportTemplate], error) {
	return s.SearchWithParams(ctx, &ImportTemplateSearchParams{Query: query, TemplateID: templateID})
}

func (s *ImportTemplateService) SearchWithParams(ctx context.Context, params *ImportTemplateSearchParams) (*ListResult[ImportTemplate], error) {
	values := url.Values{}
	if params != nil {
		values.Set("q", params.Query)
		if params.TemplateIdentifier != "" {
			values.Set("template_id", params.TemplateIdentifier)
		} else if params.TemplateID != nil {
			values.Set("template_id", fmt.Sprintf("%d", *params.TemplateID))
		}
		if params.Regex {
			values.Set("regex", "true")
		}
	}
	path := "/api/import_templates?" + values.Encode()
	return doList[ImportTemplate](s.client, ctx, path)
}

func (s *ImportTemplateService) Get(ctx context.Context, id int64) (*ImportTemplate, error) {
	return s.GetByIdentifier(ctx, int64Identifier(id))
}

func (s *ImportTemplateService) GetByIdentifier(ctx context.Context, identifier string) (*ImportTemplate, error) {
	var result ImportTemplate
	err := s.client.do(ctx, http.MethodGet, identifierPath("/api/import_templates", identifier), nil, &result)
	return &result, err
}

func (s *ImportTemplateService) Create(ctx context.Context, params *ImportTemplateCreateParams) (*ImportTemplate, error) {
	var result ImportTemplate
	body := map[string]any{
		"import_template": map[string]any{
			"name":        params.Name,
			"description": params.Description,
			"category_id": params.CategoryID,
		},
		"template_body": params.TemplateBody,
	}
	err := s.client.do(ctx, http.MethodPost, "/api/import_templates", body, &result)
	return &result, err
}

func (s *ImportTemplateService) Update(ctx context.Context, id int64, params *ImportTemplateUpdateParams) (*ImportTemplate, error) {
	return s.UpdateByIdentifier(ctx, int64Identifier(id), params)
}

func (s *ImportTemplateService) UpdateByIdentifier(ctx context.Context, identifier string, params *ImportTemplateUpdateParams) (*ImportTemplate, error) {
	var result ImportTemplate
	body := map[string]any{"import_template": params}
	err := s.client.do(ctx, http.MethodPatch, identifierPath("/api/import_templates", identifier), body, &result)
	return &result, err
}

func (s *ImportTemplateService) Delete(ctx context.Context, id int64) error {
	return s.DeleteByIdentifier(ctx, int64Identifier(id))
}

func (s *ImportTemplateService) DeleteByIdentifier(ctx context.Context, identifier string) error {
	return s.client.do(ctx, http.MethodDelete, identifierPath("/api/import_templates", identifier), nil, nil)
}

func (s *ImportTemplateService) DestroyVersion(ctx context.Context, templateID, versionID int64) error {
	return s.DestroyVersionByIdentifier(ctx, int64Identifier(templateID), versionID)
}

func (s *ImportTemplateService) DestroyVersionByIdentifier(ctx context.Context, templateIdentifier string, versionID int64) error {
	return s.client.do(ctx, http.MethodDelete, fmt.Sprintf("%s/versions/%d", identifierPath("/api/import_templates", templateIdentifier), versionID), nil, nil)
}

func (s *ImportTemplateService) ListVersions(ctx context.Context, id int64, params *ListParams) (*ListResult[ImportTemplateVersion], error) {
	return s.ListVersionsByIdentifier(ctx, int64Identifier(id), params)
}

func (s *ImportTemplateService) ListVersionsByIdentifier(ctx context.Context, identifier string, params *ListParams) (*ListResult[ImportTemplateVersion], error) {
	path := identifierPath("/api/import_templates", identifier) + "/versions" + buildQuery(params)
	return doList[ImportTemplateVersion](s.client, ctx, path)
}

func (s *ImportTemplateService) CreateVersion(ctx context.Context, id int64, params *ImportTemplateVersionCreateParams) (*ImportTemplateVersion, error) {
	return s.CreateVersionByIdentifier(ctx, int64Identifier(id), params)
}

func (s *ImportTemplateService) CreateVersionByIdentifier(ctx context.Context, identifier string, params *ImportTemplateVersionCreateParams) (*ImportTemplateVersion, error) {
	var result ImportTemplateVersion
	path := identifierPath("/api/import_templates", identifier) + "/versions"
	err := s.client.do(ctx, http.MethodPost, path, params, &result)
	return &result, err
}

func (s *ImportTemplateService) PatchPreview(ctx context.Context, id int64, params *TemplateVersionPatchParams) (*TemplateVersionPatchResult, error) {
	return s.PatchPreviewByIdentifier(ctx, int64Identifier(id), params)
}

func (s *ImportTemplateService) PatchPreviewByIdentifier(ctx context.Context, identifier string, params *TemplateVersionPatchParams) (*TemplateVersionPatchResult, error) {
	var result TemplateVersionPatchResult
	path := identifierPath("/api/import_templates", identifier) + "/patch_previews"
	err := s.client.do(ctx, http.MethodPost, path, params, &result)
	return &result, err
}

func (s *ImportTemplateService) PatchCreate(ctx context.Context, id int64, params *TemplateVersionPatchParams) (*TemplateVersionPatchResult, error) {
	return s.PatchCreateByIdentifier(ctx, int64Identifier(id), params)
}

func (s *ImportTemplateService) PatchCreateByIdentifier(ctx context.Context, identifier string, params *TemplateVersionPatchParams) (*TemplateVersionPatchResult, error) {
	var result TemplateVersionPatchResult
	path := identifierPath("/api/import_templates", identifier) + "/patches"
	err := s.client.do(ctx, http.MethodPost, path, params, &result)
	return &result, err
}

func (s *ImportTemplateService) SetSource(ctx context.Context, id int64, params *ImportTemplateSourceParams) (*ImportTemplate, error) {
	return s.SetSourceByIdentifier(ctx, int64Identifier(id), params)
}

func (s *ImportTemplateService) SetSourceByIdentifier(ctx context.Context, identifier string, params *ImportTemplateSourceParams) (*ImportTemplate, error) {
	if err := validateParams(params); err != nil {
		return nil, err
	}
	var result ImportTemplate
	body := map[string]any{"source": params}
	err := s.client.do(ctx, http.MethodPut, identifierPath("/api/import_templates", identifier)+"/source", body, &result)
	return &result, err
}

func (s *ImportTemplateService) ClearSource(ctx context.Context, id int64) (*ImportTemplate, error) {
	return s.ClearSourceByIdentifier(ctx, int64Identifier(id))
}

func (s *ImportTemplateService) ClearSourceByIdentifier(ctx context.Context, identifier string) (*ImportTemplate, error) {
	var result ImportTemplate
	err := s.client.do(ctx, http.MethodDelete, identifierPath("/api/import_templates", identifier)+"/source", nil, &result)
	return &result, err
}

func (s *ImportTemplateService) RefreshSource(ctx context.Context, id int64) (*ImportTemplateSourceRefreshResult, error) {
	return s.RefreshSourceByIdentifier(ctx, int64Identifier(id))
}

func (s *ImportTemplateService) RefreshSourceByIdentifier(ctx context.Context, identifier string) (*ImportTemplateSourceRefreshResult, error) {
	var result ImportTemplateSourceRefreshResult
	path := identifierPath("/api/import_templates", identifier) + "/source"
	err := s.client.doAsync(ctx, http.MethodPost, path, "/api/async_requests/%s", nil, &result)
	return &result, err
}

func (s *ImportTemplateService) UpgradeLinkedPlayspecs(ctx context.Context, templateID, versionID int64) (*ImportTemplateUpgradeLinkedResult, error) {
	return s.UpgradeLinkedPlayspecsByIdentifier(ctx, int64Identifier(templateID), versionID)
}

func (s *ImportTemplateService) UpgradeLinkedPlayspecsByIdentifier(ctx context.Context, templateIdentifier string, versionID int64) (*ImportTemplateUpgradeLinkedResult, error) {
	var result ImportTemplateUpgradeLinkedResult
	path := fmt.Sprintf("%s/versions/%d/upgrades", identifierPath("/api/import_templates", templateIdentifier), versionID)
	err := s.client.doAsync(ctx, http.MethodPost, path, "/api/async_requests/%s", nil, &result)
	return &result, err
}

func (s *ImportTemplateService) TogglePublic(ctx context.Context, templateID, versionID int64) (*ImportTemplateVersion, error) {
	return s.TogglePublicByIdentifier(ctx, int64Identifier(templateID), versionID)
}

func (s *ImportTemplateService) TogglePublicByIdentifier(ctx context.Context, templateIdentifier string, versionID int64) (*ImportTemplateVersion, error) {
	var result ImportTemplateVersion
	path := fmt.Sprintf("%s/versions/%d/publication", identifierPath("/api/import_templates", templateIdentifier), versionID)
	err := s.client.do(ctx, http.MethodPatch, path, nil, &result)
	return &result, err
}

func (s *ImportTemplateService) Launch(ctx context.Context, id int64) (*LaunchResult, error) {
	return s.LaunchWithParamsByIdentifier(ctx, int64Identifier(id), nil)
}

func (s *ImportTemplateService) LaunchWithParams(ctx context.Context, id int64, params *ImportTemplateLaunchParams) (*LaunchResult, error) {
	return s.LaunchWithParamsByIdentifier(ctx, int64Identifier(id), params)
}

func (s *ImportTemplateService) LaunchWithParamsByIdentifier(ctx context.Context, identifier string, params *ImportTemplateLaunchParams) (*LaunchResult, error) {
	if params != nil {
		if err := validateParams(params); err != nil {
			return nil, err
		}
	}
	var result LaunchResult
	path := identifierPath("/api/import_templates", identifier) + "/launches"
	err := s.client.do(ctx, http.MethodPost, path, params, &result)
	return &result, err
}

func (s *ImportTemplateService) Fork(ctx context.Context, id int64) (*ImportTemplate, error) {
	return s.ForkByIdentifier(ctx, int64Identifier(id))
}

func (s *ImportTemplateService) ForkByIdentifier(ctx context.Context, identifier string) (*ImportTemplate, error) {
	var result ImportTemplate
	err := s.client.do(ctx, http.MethodPost, identifierPath("/api/import_templates", identifier)+"/forks", nil, &result)
	return &result, err
}

func (s *ImportTemplateService) UploadImage(ctx context.Context, id int64, params *UploadImageParams) (*ImportTemplate, error) {
	return s.UploadImageByIdentifier(ctx, int64Identifier(id), params)
}

func (s *ImportTemplateService) UploadImageByIdentifier(ctx context.Context, identifier string, params *UploadImageParams) (*ImportTemplate, error) {
	var result ImportTemplate
	err := s.client.do(ctx, http.MethodPost, identifierPath("/api/import_templates", identifier)+"/images", params, &result)
	return &result, err
}
