package fibe

import "time"

type ImportTemplate struct {
	ID          *int64     `json:"id"`
	Name        string     `json:"name"`
	Description *string    `json:"description"`
	System      *bool      `json:"system"`
	Author      *string    `json:"author"`
	Category    *string    `json:"category"`
	ImageURL    *string    `json:"image_url"`
	CreatedAt   *time.Time `json:"created_at"`
	UpdatedAt   *time.Time `json:"updated_at"`

	// Detail fields
	Versions []ImportTemplateVersion `json:"versions,omitempty"`
}

type ImportTemplateVersion struct {
	ID           *int64     `json:"id"`
	Version      *int64     `json:"version"`
	Public       *bool      `json:"public"`
	Approved     *bool      `json:"approved"`
	TemplateBody string     `json:"template_body"`
	CreatedAt    *time.Time `json:"created_at"`
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
	TemplateBody string `json:"template_body"`
	Public       *bool  `json:"public,omitempty"`
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
