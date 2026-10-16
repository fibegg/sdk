package fibe

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// ENV packs contain ordinary readable values; no reveal or masking flag applies.
type EnvPack struct {
	OwnershipMetadata
	ID        int64             `json:"id"`
	Name      string            `json:"name"`
	Env       map[string]string `json:"env"`
	Version   int64             `json:"version"`
	OwnerType string            `json:"owner_type"`
	OwnerID   int64             `json:"owner_id"`
	DeletedAt *time.Time        `json:"deleted_at,omitempty"`
	CreatedAt *time.Time        `json:"created_at,omitempty"`
	UpdatedAt *time.Time        `json:"updated_at,omitempty"`
}

type EnvPackListParams struct {
	Page    int `url:"page,omitempty"`
	PerPage int `url:"per_page,omitempty"`
}
type EnvPackCreateParams struct {
	Name string            `json:"name"`
	Env  map[string]string `json:"env"`
}
type EnvPackUpdateParams struct {
	Name    *string            `json:"name,omitempty"`
	Env     *map[string]string `json:"env,omitempty"`
	Version *int64             `json:"version,omitempty"`
}

var envPackKey = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func validateEnvPackValues(values map[string]string) error {
	for key := range values {
		if !envPackKey.MatchString(key) {
			return fmt.Errorf("invalid ENV key %q", key)
		}
	}
	return nil
}
func (p *EnvPackCreateParams) Validate() error {
	if p == nil || strings.TrimSpace(p.Name) == "" {
		return fmt.Errorf("ENV pack name is required")
	}
	if p.Env == nil {
		return fmt.Errorf("ENV pack env must be an object; use an empty map for no values")
	}
	return validateEnvPackValues(p.Env)
}
func (p *EnvPackUpdateParams) Validate() error {
	if p == nil {
		return fmt.Errorf("ENV pack update is required")
	}
	if p.Name != nil && strings.TrimSpace(*p.Name) == "" {
		return fmt.Errorf("ENV pack name is required")
	}
	if p.Env != nil {
		if *p.Env == nil {
			return fmt.Errorf("ENV pack env must be an object")
		}
		return validateEnvPackValues(*p.Env)
	}
	return nil
}

// Nil ServiceNames means all services. An explicit empty slice is invalid.
type EnvPackAttachmentInput struct {
	EnvPackID    int64     `json:"env_pack_id"`
	ServiceNames *[]string `json:"service_names,omitempty"`
}

// Nil Attachments preserves an existing list (or inherits Spec defaults on
// creation); a pointer to an empty slice explicitly detaches every pack.
type EnvPackAttachmentsParams struct {
	Attachments *[]EnvPackAttachmentInput `json:"env_pack_attachments,omitempty"`
}

func (p *EnvPackAttachmentsParams) Validate() error {
	if p == nil || p.Attachments == nil {
		return nil
	}
	seen := map[int64]bool{}
	for _, row := range *p.Attachments {
		if row.EnvPackID <= 0 || seen[row.EnvPackID] {
			return fmt.Errorf("ENV pack IDs must be positive and unique")
		}
		seen[row.EnvPackID] = true
		if row.ServiceNames != nil {
			if len(*row.ServiceNames) == 0 {
				return fmt.Errorf("service_names must be omitted or contain services")
			}
			names := map[string]bool{}
			for _, name := range *row.ServiceNames {
				if name == "" || names[name] {
					return fmt.Errorf("service_names must be nonempty and unique")
				}
				names[name] = true
			}
		}
	}
	return nil
}

type EnvPackAttachment struct {
	ID              int64          `json:"id"`
	EnvPackID       int64          `json:"env_pack_id"`
	Position        int            `json:"position"`
	ServiceNames    []string       `json:"service_names"`
	Name            string         `json:"name"`
	Owner           map[string]any `json:"owner"`
	GrantingSubject string         `json:"granting_subject"`
	GrantedAt       *time.Time     `json:"granted_at,omitempty"`
}
type EnvPackTarget struct {
	ID                 int64               `json:"id"`
	EnvPackAttachments []EnvPackAttachment `json:"env_pack_attachments"`
	EnvPacks           map[string]any      `json:"env_packs"`
}
type PlaygroundRerunParams struct {
	Name               *string                   `json:"name,omitempty"`
	EnvPackAttachments *[]EnvPackAttachmentInput `json:"env_pack_attachments,omitempty"`
}
