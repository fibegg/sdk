package fibe

import (
	"encoding/json"
	"time"
)

type ImportTemplate struct {
	ID               *int64                `json:"id"`
	Name             string                `json:"name"`
	Description      *string               `json:"description"`
	System           *bool                 `json:"system"`
	Author           *string               `json:"author"`
	Category         *string               `json:"category"`
	ImageURL         *string               `json:"image_url"`
	Source           *ImportTemplateSource `json:"source,omitempty"`
	LatestVersion    *int64                `json:"latest_version,omitempty"`
	LatestVersionID  *int64                `json:"latest_version_id,omitempty"`
	LatestVersionTag *string               `json:"latest_version_tag,omitempty"`
	VersionTags      []string              `json:"version_tags,omitempty"`
	Variables        map[string]any        `json:"variables,omitempty"`
	CreatedAt        *time.Time            `json:"created_at"`
	UpdatedAt        *time.Time            `json:"updated_at"`

	// Detail fields
	Versions []ImportTemplateVersion `json:"versions,omitempty"`
}

func (t *ImportTemplate) UnmarshalJSON(data []byte) error {
	cleaned, parsedTimes, err := stripFlexibleTimeFields(data, "created_at", "updated_at")
	if err != nil {
		return err
	}
	type alias ImportTemplate
	var out alias
	if err := json.Unmarshal(cleaned, &out); err != nil {
		return err
	}
	out.CreatedAt = parsedTimes["created_at"]
	out.UpdatedAt = parsedTimes["updated_at"]
	*t = ImportTemplate(out)
	return nil
}

type ImportTemplateVersion struct {
	ID                *int64                       `json:"id"`
	Version           *int64                       `json:"version"`
	Public            *bool                        `json:"public"`
	Approved          *bool                        `json:"approved"`
	GreenfieldDefault *bool                        `json:"greenfield_default"`
	TemplateBody      string                       `json:"template_body"`
	Changelog         *string                      `json:"changelog,omitempty"`
	Source            *ImportTemplateVersionSource `json:"source,omitempty"`
	Variables         map[string]any               `json:"variables,omitempty"`
	CreatedAt         *time.Time                   `json:"created_at"`
}

func (v *ImportTemplateVersion) UnmarshalJSON(data []byte) error {
	cleaned, parsedTimes, err := stripFlexibleTimeFields(data, "created_at")
	if err != nil {
		return err
	}
	type alias ImportTemplateVersion
	var out alias
	if err := json.Unmarshal(cleaned, &out); err != nil {
		return err
	}
	out.CreatedAt = parsedTimes["created_at"]
	*v = ImportTemplateVersion(out)
	return nil
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
	CIEnabled         *bool      `json:"ci_enabled,omitempty"`
	CIMarqueeID       *int64     `json:"ci_marquee_id,omitempty"`
	CIMarqueeName     *string    `json:"ci_marquee_name,omitempty"`
	LastStatus        *string    `json:"last_status,omitempty"`
	LastCommitSHA     *string    `json:"last_commit_sha,omitempty"`
	LastContentSHA    *string    `json:"last_content_sha,omitempty"`
	LastRefreshedAt   *time.Time `json:"last_refreshed_at,omitempty"`
	LastError         *string    `json:"last_error,omitempty"`
}

func (s *ImportTemplateSource) UnmarshalJSON(data []byte) error {
	cleaned, parsedTimes, err := stripFlexibleTimeFields(data, "last_refreshed_at")
	if err != nil {
		return err
	}
	type alias ImportTemplateSource
	var out alias
	if err := json.Unmarshal(cleaned, &out); err != nil {
		return err
	}
	out.LastRefreshedAt = parsedTimes["last_refreshed_at"]
	*s = ImportTemplateSource(out)
	return nil
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
	ResponseMode string  `json:"response_mode,omitempty"`
}

type TemplatePatchEdit struct {
	Path          string `json:"path,omitempty"`
	Op            string `json:"op,omitempty"`
	Value         any    `json:"value,omitempty"`
	Expect        any    `json:"expect,omitempty"`
	Search        string `json:"search,omitempty"`
	Replace       string `json:"replace,omitempty"`
	AllowMultiple *bool  `json:"allow_multiple,omitempty"`
	CreateMissing *bool  `json:"create_missing,omitempty"`
	AllowMissing  *bool  `json:"allow_missing,omitempty"`
}

type TemplateVersionPatchParams struct {
	BaseVersionID              int64               `json:"base_version_id"`
	TemplateBody               string              `json:"template_body,omitempty"`
	Patches                    []TemplatePatchEdit `json:"patches,omitempty"`
	Edits                      []TemplatePatchEdit `json:"edits,omitempty"`
	Public                     *bool               `json:"public,omitempty"`
	Changelog                  *string             `json:"changelog,omitempty"`
	TargetPlayspecID           *int64              `json:"target_playspec_id,omitempty"`
	TargetPlayspecIdentifier   string              `json:"-"`
	RolloutMode                string              `json:"rollout_mode,omitempty"`
	TargetPlaygroundID         *int64              `json:"target_playground_id,omitempty"`
	TargetPlaygroundIdentifier string              `json:"-"`
	SwitchVariables            map[string]any      `json:"switch_variables,omitempty"`
	RegenerateVariables        []string            `json:"regenerate_variables,omitempty"`
	ConfirmWarnings            *bool               `json:"confirm_warnings,omitempty"`
	AutoSwitch                 *bool               `json:"auto_switch,omitempty"`
	ResponseMode               string              `json:"response_mode,omitempty"`
}

func (p TemplateVersionPatchParams) MarshalJSON() ([]byte, error) {
	type alias TemplateVersionPatchParams
	data, err := json.Marshal(alias(p))
	if err != nil {
		return nil, err
	}
	var body map[string]any
	if err := json.Unmarshal(data, &body); err != nil {
		return nil, err
	}
	if p.TargetPlayspecIdentifier != "" {
		body["target_playspec_id"] = p.TargetPlayspecIdentifier
	}
	if p.TargetPlaygroundIdentifier != "" {
		body["target_playground_id"] = p.TargetPlaygroundIdentifier
	}
	return json.Marshal(body)
}

type TemplateVersionPatchResult map[string]any

type ImportTemplateSourceParams struct {
	SourcePropID         int64  `json:"source_prop_id"`
	SourcePropIdentifier string `json:"-"`
	SourcePath           string `json:"source_path"`
	SourceRef            string `json:"source_ref,omitempty"`
	SourceAutoRefresh    *bool  `json:"source_auto_refresh,omitempty"`
	SourceAutoUpgrade    *bool  `json:"source_auto_upgrade,omitempty"`
	CIEnabled            *bool  `json:"ci_enabled,omitempty"`
	CIMarqueeID          *int64 `json:"ci_marquee_id,omitempty"`
	CIMarqueeIdentifier  string `json:"-"`
	MarqueeID            *int64 `json:"marquee_id,omitempty"`
	MarqueeIdentifier    string `json:"-"`
}

func (p *ImportTemplateSourceParams) Validate() error {
	v := &validator{}
	v.requiredIDOrIdentifier("source_prop_id", p.SourcePropID, p.SourcePropIdentifier)
	v.required("source_path", p.SourcePath)
	return v.err()
}

func (p ImportTemplateSourceParams) MarshalJSON() ([]byte, error) {
	type alias ImportTemplateSourceParams
	data, err := json.Marshal(alias(p))
	if err != nil {
		return nil, err
	}
	var body map[string]any
	if err := json.Unmarshal(data, &body); err != nil {
		return nil, err
	}
	if p.SourcePropIdentifier != "" {
		body["source_prop_id"] = p.SourcePropIdentifier
	}
	if p.CIMarqueeIdentifier != "" {
		body["ci_marquee_id"] = p.CIMarqueeIdentifier
	}
	if p.MarqueeIdentifier != "" {
		body["marquee_id"] = p.MarqueeIdentifier
	}
	return json.Marshal(body)
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
	MarqueeIdentifier string            `json:"-"`
	Version           *int64            `json:"version,omitempty"`
	Name              string            `json:"name,omitempty"`
	Variables         map[string]any    `json:"variables,omitempty"`
	EnvOverrides      map[string]string `json:"env_overrides,omitempty"`
	ServiceSubdomains map[string]string `json:"service_subdomains,omitempty"`
	Services          map[string]any    `json:"services,omitempty"`
}

func (p *ImportTemplateLaunchParams) Validate() error {
	v := &validator{}
	v.requiredIDOrIdentifier("marquee_id", p.MarqueeID, p.MarqueeIdentifier)
	return v.err()
}

func (p ImportTemplateLaunchParams) MarshalJSON() ([]byte, error) {
	type alias ImportTemplateLaunchParams
	data, err := json.Marshal(alias(p))
	if err != nil {
		return nil, err
	}
	var body map[string]any
	if err := json.Unmarshal(data, &body); err != nil {
		return nil, err
	}
	if p.MarqueeIdentifier != "" {
		body["marquee_id"] = p.MarqueeIdentifier
	}
	return json.Marshal(body)
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

type ImportTemplateSearchParams struct {
	Query      string
	TemplateID *int64
	Regex      bool
}

type TemplateCategory struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}
