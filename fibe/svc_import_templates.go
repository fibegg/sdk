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
		if params.TemplateID != nil {
			values.Set("template_id", fmt.Sprintf("%d", *params.TemplateID))
		}
		if params.Regex {
			values.Set("regex", "true")
		}
	}
	path := "/api/import_templates/search?" + values.Encode()
	return doList[ImportTemplate](s.client, ctx, path)
}

func (s *ImportTemplateService) Get(ctx context.Context, id int64) (*ImportTemplate, error) {
	var result ImportTemplate
	err := s.client.do(ctx, http.MethodGet, fmt.Sprintf("/api/import_templates/%d", id), nil, &result)
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
	var result ImportTemplate
	body := map[string]any{"import_template": params}
	err := s.client.do(ctx, http.MethodPatch, fmt.Sprintf("/api/import_templates/%d", id), body, &result)
	return &result, err
}

func (s *ImportTemplateService) Delete(ctx context.Context, id int64) error {
	return s.client.do(ctx, http.MethodDelete, fmt.Sprintf("/api/import_templates/%d", id), nil, nil)
}

func (s *ImportTemplateService) DestroyVersion(ctx context.Context, templateID, versionID int64) error {
	return s.client.do(ctx, http.MethodDelete, fmt.Sprintf("/api/import_template_versions/%d", versionID), nil, nil)
}

func (s *ImportTemplateService) ListVersions(ctx context.Context, id int64, params *ListParams) (*ListResult[ImportTemplateVersion], error) {
	path := fmt.Sprintf("/api/import_templates/%d/versions", id) + buildQuery(params)
	return doList[ImportTemplateVersion](s.client, ctx, path)
}

func (s *ImportTemplateService) CreateVersion(ctx context.Context, id int64, params *ImportTemplateVersionCreateParams) (*ImportTemplateVersion, error) {
	var result ImportTemplateVersion
	path := fmt.Sprintf("/api/import_templates/%d/versions", id)
	err := s.client.do(ctx, http.MethodPost, path, params, &result)
	return &result, err
}

func (s *ImportTemplateService) PatchPreview(ctx context.Context, id int64, params *TemplateVersionPatchParams) (*TemplateVersionPatchResult, error) {
	var result TemplateVersionPatchResult
	path := fmt.Sprintf("/api/import_templates/%d/versions/patch_preview", id)
	err := s.client.do(ctx, http.MethodPost, path, params, &result)
	return &result, err
}

func (s *ImportTemplateService) PatchCreate(ctx context.Context, id int64, params *TemplateVersionPatchParams) (*TemplateVersionPatchResult, error) {
	var result TemplateVersionPatchResult
	path := fmt.Sprintf("/api/import_templates/%d/versions/patch_create", id)
	err := s.client.do(ctx, http.MethodPost, path, params, &result)
	return &result, err
}

func (s *ImportTemplateService) SetSource(ctx context.Context, id int64, params *ImportTemplateSourceParams) (*ImportTemplate, error) {
	var result ImportTemplate
	body := map[string]any{"source": params}
	err := s.client.do(ctx, http.MethodPut, fmt.Sprintf("/api/import_templates/%d/source", id), body, &result)
	return &result, err
}

func (s *ImportTemplateService) ClearSource(ctx context.Context, id int64) (*ImportTemplate, error) {
	var result ImportTemplate
	err := s.client.do(ctx, http.MethodDelete, fmt.Sprintf("/api/import_templates/%d/source", id), nil, &result)
	return &result, err
}

func (s *ImportTemplateService) RefreshSource(ctx context.Context, id int64) (*ImportTemplateSourceRefreshResult, error) {
	var result ImportTemplateSourceRefreshResult
	err := s.client.do(ctx, http.MethodPost, fmt.Sprintf("/api/import_templates/%d/source/refresh", id), nil, &result)
	return &result, err
}

func (s *ImportTemplateService) UpgradeLinkedPlayspecs(ctx context.Context, templateID, versionID int64) (*ImportTemplateUpgradeLinkedResult, error) {
	var result ImportTemplateUpgradeLinkedResult
	path := fmt.Sprintf("/api/import_templates/%d/versions/%d/upgrade_linked_playspecs", templateID, versionID)
	err := s.client.do(ctx, http.MethodPost, path, nil, &result)
	return &result, err
}

func (s *ImportTemplateService) TogglePublic(ctx context.Context, templateID, versionID int64) (*ImportTemplateVersion, error) {
	var result ImportTemplateVersion
	path := fmt.Sprintf("/api/import_templates/%d/toggle_public", templateID)
	body := map[string]any{"version_id": versionID}
	err := s.client.do(ctx, http.MethodPatch, path, body, &result)
	return &result, err
}

func (s *ImportTemplateService) Launch(ctx context.Context, id int64) (*LaunchResult, error) {
	return s.LaunchWithParams(ctx, id, nil)
}

func (s *ImportTemplateService) LaunchWithParams(ctx context.Context, id int64, params *ImportTemplateLaunchParams) (*LaunchResult, error) {
	if params != nil {
		if err := validateParams(params); err != nil {
			return nil, err
		}
	}
	var result LaunchResult
	path := fmt.Sprintf("/api/import_templates/%d/launch", id)
	err := s.client.do(ctx, http.MethodPost, path, params, &result)
	return &result, err
}

func (s *ImportTemplateService) Fork(ctx context.Context, id int64) (*ImportTemplate, error) {
	var result ImportTemplate
	err := s.client.do(ctx, http.MethodPost, fmt.Sprintf("/api/import_templates/%d/fork", id), nil, &result)
	return &result, err
}

func (s *ImportTemplateService) UploadImage(ctx context.Context, id int64, params *UploadImageParams) (*ImportTemplate, error) {
	var result ImportTemplate
	err := s.client.do(ctx, http.MethodPost, fmt.Sprintf("/api/import_templates/%d/image", id), params, &result)
	return &result, err
}
