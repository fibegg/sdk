package fibe

import "time"

type ImportTemplate struct {
	ID          *int64                `json:"id"`
	Name        string                `json:"name"`
	Description *string               `json:"description"`
	System      *bool                 `json:"system"`
	Author      *string               `json:"author"`
	Category    *string               `json:"category"`
	ImageURL    *string               `json:"image_url"`
	Source      *ImportTemplateSource `json:"source,omitempty"`
	CreatedAt   *time.Time            `json:"created_at"`
	UpdatedAt   *time.Time            `json:"updated_at"`

	// Detail fields
	Versions []ImportTemplateVersion `json:"versions,omitempty"`
}

type ImportTemplateVersion struct {
	ID           *int64                       `json:"id"`
	Version      *int64                       `json:"version"`
	Public       *bool                        `json:"public"`
	Approved     *bool                        `json:"approved"`
	TemplateBody string                       `json:"template_body"`
	Changelog    *string                      `json:"changelog,omitempty"`
	Source       *ImportTemplateVersionSource `json:"source,omitempty"`
	CreatedAt    *time.Time                   `json:"created_at"`
}

type ImportTemplateSource struct {
	ID                *int64     `json:"id,omitempty"`
	PropID            *int64     `json:"prop_id,omitempty"`
	PropName          *string    `json:"prop_name,omitempty"`
	PropRepositoryURL *string    `json:"prop_repository_url,omitempty"`
	Path              string     `json:"path"`
	Ref               string     `json:"ref"`
	AutoRefresh       *bool      `json:"auto_refresh,omitempty"`
	AutoUpgrade       *bool      `json:"auto_upgrade,omitempty"`
	LastStatus        *string    `json:"last_status,omitempty"`
	LastCommitSHA     *string    `json:"last_commit_sha,omitempty"`
	LastContentSHA    *string    `json:"last_content_sha,omitempty"`
	LastRefreshedAt   *time.Time `json:"last_refreshed_at,omitempty"`
	LastError         *string    `json:"last_error,omitempty"`
}

type ImportTemplateVersionSource struct {
	PropID            *int64  `json:"prop_id,omitempty"`
	PropName          *string `json:"prop_name,omitempty"`
	PropRepositoryURL *string `json:"prop_repository_url,omitempty"`
	Path              string  `json:"path,omitempty"`
	Ref               string  `json:"ref,omitempty"`
	CommitSHA         *string `json:"commit_sha,omitempty"`
	ContentSHA        *string `json:"content_sha,omitempty"`
}

type ImportTemplateCreateParams struct {
	Name         string `json:"name"`
	Description  string `json:"description,omitempty"`
	CategoryID   int64  `json:"category_id"`
	TemplateBody string `json:"template_body"`
}

type ImportTemplateUpdateParams struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	CategoryID  *int64  `json:"category_id,omitempty"`
}

type UploadImageParams struct {
	Filename    string `json:"filename"`
	ImageData   string `json:"image_data"`
	ContentType string `json:"content_type,omitempty"`
}

type ImportTemplateVersionCreateParams struct {
	TemplateBody string  `json:"template_body"`
	Public       *bool   `json:"public,omitempty"`
	Changelog    *string `json:"changelog,omitempty"`
}

type ImportTemplateSourceParams struct {
	SourcePropID      int64  `json:"source_prop_id"`
	SourcePath        string `json:"source_path"`
	SourceRef         string `json:"source_ref,omitempty"`
	SourceAutoRefresh *bool  `json:"source_auto_refresh,omitempty"`
	SourceAutoUpgrade *bool  `json:"source_auto_upgrade,omitempty"`
}

type ImportTemplateSourceRefreshResult struct {
	Success        bool                   `json:"success"`
	VersionCreated bool                   `json:"version_created"`
	Version        *ImportTemplateVersion `json:"version,omitempty"`
	Upgrade        map[string]any         `json:"upgrade,omitempty"`
	Errors         []string               `json:"errors,omitempty"`
	Payload        map[string]any         `json:"payload,omitempty"`
	Template       *ImportTemplate        `json:"template,omitempty"`
}

type ImportTemplateUpgradeLinkedResult struct {
	Success       bool             `json:"success"`
	UpgradedCount int64            `json:"upgraded_count"`
	FailedCount   int64            `json:"failed_count"`
	Results       []map[string]any `json:"results,omitempty"`
	Errors        []string         `json:"errors,omitempty"`
}

type ImportTemplateLaunchParams struct {
	MarqueeID         int64             `json:"marquee_id"`
	Version           *int64            `json:"version,omitempty"`
	Name              string            `json:"name,omitempty"`
	Variables         map[string]any    `json:"variables,omitempty"`
	EnvOverrides      map[string]string `json:"env_overrides,omitempty"`
	ServiceSubdomains map[string]string `json:"service_subdomains,omitempty"`
	Services          map[string]any    `json:"services,omitempty"`
}

func (p *ImportTemplateLaunchParams) Validate() error {
	v := &validator{}
	v.requiredInt("marquee_id", p.MarqueeID)
	return v.err()
}

type ImportTemplateListParams struct {
	Q          string `url:"q,omitempty"`
	CategoryID int64  `url:"category_id,omitempty"`
	Name       string `url:"name,omitempty"`
	System     *bool  `url:"system,omitempty"`
	Sort       string `url:"sort,omitempty"`
	Page       int    `url:"page,omitempty"`
	PerPage    int    `url:"per_page,omitempty"`
}

type TemplateCategory struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}
