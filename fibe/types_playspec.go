package fibe

import (
	"fmt"
	"time"
)

// Playspec defines the service composition template.
type Playspec struct {
	ID                        *int64              `json:"id"`
	Name                      string              `json:"name"`
	Description               *string             `json:"description"`
	Locked                    *bool               `json:"locked"`
	PersistVolumes            *bool               `json:"persist_volumes"`
	JobMode                   *bool               `json:"job_mode"`
	PlaygroundCount           *int64              `json:"playground_count"`
	TriggerEnabled            *bool               `json:"trigger_enabled"`
	MutiMode                  *bool               `json:"muti_mode"`
	CreatedAt                 *time.Time          `json:"created_at"`
	UpdatedAt                 *time.Time          `json:"updated_at"`
	SourceTemplateVersionID   *int64              `json:"source_template_version_id"`
	SourceTemplateVersion     *TemplateVersionRef `json:"source_template_version"`
	SourceTemplate            *TemplateRef        `json:"source_template"`
	TemplateVersionSwitchable *bool               `json:"template_version_switchable"`
	SuggestedTemplateVersion  *TemplateVersionRef `json:"suggested_template_version"`

	// Detail fields
	Services       []any             `json:"services,omitempty"`
	MountedFiles   []MountedFileInfo `json:"mounted_files,omitempty"`
	Credentials    any               `json:"credentials,omitempty"`
	TriggerConfig  map[string]any    `json:"trigger_config,omitempty"`
	MutiConfig     map[string]any    `json:"muti_config,omitempty"`
	ScheduleConfig map[string]any    `json:"schedule_config,omitempty"`
}

type TemplateRef struct {
	ID     *int64  `json:"id"`
	Name   string  `json:"name"`
	Author *string `json:"author,omitempty"`
	System *bool   `json:"system,omitempty"`
}

type TemplateVersionRef struct {
	ID        *int64       `json:"id"`
	Version   *int64       `json:"version"`
	Public    *bool        `json:"public"`
	Approved  *bool        `json:"approved"`
	CreatedAt *time.Time   `json:"created_at,omitempty"`
	Template  *TemplateRef `json:"template,omitempty"`
}

type MountedFileInfo struct {
	Filename    string `json:"filename"`
	ByteSize    int64  `json:"byte_size"`
	ContentType string `json:"content_type"`
}

type PlayspecCreateParams struct {
	Name            string               `json:"name"`
	Description     *string              `json:"description,omitempty"`
	BaseComposeYAML string               `json:"base_compose_yaml"`
	PersistVolumes  *bool                `json:"persist_volumes,omitempty"`
	JobMode         *bool                `json:"job_mode,omitempty"`
	Services        []PlayspecServiceDef `json:"services,omitempty"`
	TriggerConfig   map[string]any       `json:"trigger_config,omitempty"`
	MutiConfig      map[string]any       `json:"muti_config,omitempty"`
}

func (p *PlayspecCreateParams) Validate() error {
	v := &validator{}
	v.required("name", p.Name)
	v.required("base_compose_yaml", p.BaseComposeYAML)
	names := map[string]bool{}
	for i, svc := range p.Services {
		prefix := fmt.Sprintf("services[%d]", i)
		v.required(prefix+".name", svc.Name)
		v.oneOf(prefix+".type", svc.Type, []string{ServiceTypeStatic, ServiceTypeDynamic})
		if names[svc.Name] {
			v.errors = append(v.errors, ValidationError{Field: prefix + ".name", Message: "must be unique"})
		}
		names[svc.Name] = true
		if svc.Exposure != nil && svc.Exposure.Enabled {
			v.port(prefix+".exposure.port", svc.Exposure.Port)
			v.subdomain(prefix+".exposure.subdomain", svc.Exposure.Subdomain)
			v.oneOf(prefix+".exposure.visibility", svc.Exposure.Visibility, []string{"internal", "external"})
		}
	}
	return v.err()
}

const (
	ServiceTypeStatic  = "static"
	ServiceTypeDynamic = "dynamic"
)

// PlayspecServiceDef defines a service in a playspec template.
type PlayspecServiceDef struct {
	Name           string           `json:"name"`
	Type           string           `json:"type"`
	PropID         *int64           `json:"prop_id,omitempty"`
	Workdir        string           `json:"workdir,omitempty"`
	Workflow       string           `json:"workflow,omitempty"`
	EnvFilePath    string           `json:"env_file_path,omitempty"`
	DockerfilePath string           `json:"dockerfile_path,omitempty"`
	Image          string           `json:"image,omitempty"`
	Exposure       *ServiceExposure `json:"exposure,omitempty"`
	JobWatch       *bool            `json:"job_watch,omitempty"`
}

type PlayspecUpdateParams struct {
	Name            *string              `json:"name,omitempty"`
	Description     *string              `json:"description,omitempty"`
	BaseComposeYAML *string              `json:"base_compose_yaml,omitempty"`
	PersistVolumes  *bool                `json:"persist_volumes,omitempty"`
	JobMode         *bool                `json:"job_mode,omitempty"`
	Services        []PlayspecServiceDef `json:"services,omitempty"`
	TriggerConfig   map[string]any       `json:"trigger_config,omitempty"`
	MutiConfig      map[string]any       `json:"muti_config,omitempty"`
}

type ComposeValidation struct {
	Valid    bool     `json:"valid"`
	Errors   []string `json:"errors,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
}

type MountedFileParams struct {
	MountPath      string   `json:"mount_path"`
	TargetServices []string `json:"target_services,omitempty"`
	ReadOnly       *bool    `json:"readonly,omitempty"`
}

type MountedFileUpdateParams struct {
	Filename       string   `json:"filename"`
	MountPath      string   `json:"mount_path"`
	TargetServices []string `json:"target_services,omitempty"`
	ReadOnly       *bool    `json:"readonly,omitempty"`
}

type RegistryCredentialParams struct {
	RegistryType string `json:"registry_type"`
	RegistryURL  string `json:"registry_url"`
	Username     string `json:"username"`
	Secret       string `json:"secret"`
}

type RegistryCredentialInfo struct {
	ID           string `json:"id"`
	RegistryType string `json:"registry_type"`
	RegistryURL  string `json:"registry_url"`
	Username     string `json:"username"`
}

type RegistryCredentialResult struct {
	Credentials []RegistryCredentialInfo `json:"credentials"`
}

type PlayspecTemplateVersionSwitchParams struct {
	TargetTemplateVersionID int64          `json:"target_template_version_id"`
	Variables               map[string]any `json:"variables,omitempty"`
	RegenerateVariables     []string       `json:"regenerate_variables,omitempty"`
	ConfirmWarnings         bool           `json:"confirm_warnings,omitempty"`
}

type PlayspecTemplateVersionSwitchResult struct {
	FromTemplateVersion   *TemplateVersionRef          `json:"from_template_version"`
	TargetTemplateVersion *TemplateVersionRef          `json:"target_template_version"`
	SuggestedUpgrade      bool                         `json:"suggested_upgrade"`
	RequiredVariables     []TemplateSwitchVariable     `json:"required_variables"`
	TargetVariables       []TemplateSwitchVariable     `json:"target_variables"`
	Warnings              []TemplateSwitchWarning      `json:"warnings"`
	Diff                  map[string]any               `json:"diff"`
	PlaygroundRolloutPlan TemplateSwitchPlaygroundPlan `json:"playground_rollout_plan"`
	NoOp                  bool                         `json:"no_op"`
	Playspec              *Playspec                    `json:"playspec"`
}

type PlayspecTemplateVersionSwitchPreview = PlayspecTemplateVersionSwitchResult

type TemplateSwitchVariable struct {
	Name       string `json:"name"`
	Label      string `json:"label,omitempty"`
	Required   bool   `json:"required,omitempty"`
	Random     bool   `json:"random,omitempty"`
	HasDefault bool   `json:"has_default,omitempty"`
	Stored     bool   `json:"stored,omitempty"`
	Validation string `json:"validation,omitempty"`
}

type TemplateSwitchWarning struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Items   []any  `json:"items,omitempty"`
}

type TemplateSwitchPlaygroundPlan struct {
	Blocked   []int64 `json:"blocked"`
	Rollout   []int64 `json:"rollout"`
	Unchanged []int64 `json:"unchanged"`
}

type PlayspecListParams struct {
	Q             string `url:"q,omitempty"`
	JobMode       *bool  `url:"job_mode,omitempty"`
	Locked        *bool  `url:"locked,omitempty"`
	Name          string `url:"name,omitempty"`
	CreatedAfter  string `url:"created_after,omitempty"`
	CreatedBefore string `url:"created_before,omitempty"`
	Sort          string `url:"sort,omitempty"`
	Page          int    `url:"page,omitempty"`
	PerPage       int    `url:"per_page,omitempty"`
}
