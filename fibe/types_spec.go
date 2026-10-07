package fibe

import (
	"encoding/json"
	"fmt"
	"time"
)

// Spec defines the service composition template.
type Spec struct {
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

type SpecCreateParams struct {
	Name            string           `json:"name"`
	Description     *string          `json:"description,omitempty"`
	BaseComposeYAML string           `json:"base_compose_yaml"`
	PersistVolumes  *bool            `json:"persist_volumes,omitempty"`
	JobMode         *bool            `json:"job_mode,omitempty"`
	Services        []SpecServiceDef `json:"services,omitempty"`
	TriggerConfig   map[string]any   `json:"trigger_config,omitempty"`
	MutiConfig      map[string]any   `json:"muti_config,omitempty"`
	ScheduleConfig  map[string]any   `json:"schedule_config,omitempty"`
}

func (p *SpecCreateParams) Validate() error {
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

// SpecServiceDef defines a service in a spec template.
type SpecServiceDef struct {
	Name                 string           `json:"name"`
	Type                 string           `json:"type"`
	RepositoryID         *int64           `json:"repository_id,omitempty"`
	RepositoryIdentifier string           `json:"-"`
	Workdir              string           `json:"workdir,omitempty"`
	Workflow             string           `json:"workflow,omitempty"`
	EnvFilePath          string           `json:"env_file_path,omitempty"`
	DockerfilePath       string           `json:"dockerfile_path,omitempty"`
	Image                string           `json:"image,omitempty"`
	Exposure             *ServiceExposure `json:"exposure,omitempty"`
	JobWatch             *bool            `json:"job_watch,omitempty"`
}

func (p SpecServiceDef) MarshalJSON() ([]byte, error) {
	type alias SpecServiceDef
	data, err := json.Marshal(alias(p))
	if err != nil {
		return nil, err
	}
	var body map[string]any
	if err := json.Unmarshal(data, &body); err != nil {
		return nil, err
	}
	if p.RepositoryIdentifier != "" {
		body["repository_id"] = p.RepositoryIdentifier
	}
	return json.Marshal(body)
}

type SpecUpdateParams struct {
	Name            *string          `json:"name,omitempty"`
	Description     *string          `json:"description,omitempty"`
	BaseComposeYAML *string          `json:"base_compose_yaml,omitempty"`
	PersistVolumes  *bool            `json:"persist_volumes,omitempty"`
	JobMode         *bool            `json:"job_mode,omitempty"`
	Services        []SpecServiceDef `json:"services,omitempty"`
	TriggerConfig   map[string]any   `json:"trigger_config,omitempty"`
	MutiConfig      map[string]any   `json:"muti_config,omitempty"`
	ScheduleConfig  map[string]any   `json:"schedule_config,omitempty"`
}

type ComposeValidation struct {
	Valid    bool     `json:"valid"`
	Services []any    `json:"services,omitempty"`
	Errors   []string `json:"errors,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
}

type ComposeValidateParams struct {
	ComposeYAML string `json:"compose_yaml"`
	TargetType  string `json:"target_type,omitempty"`
	JobMode     *bool  `json:"job_mode,omitempty"`
}

type MountedFileParams struct {
	MountPath      string   `json:"mount_path"`
	TargetServices []string `json:"target_services,omitempty"`
	ReadOnly       *bool    `json:"readonly,omitempty"`
	ArtefactID     *int64   `json:"artefact_id,omitempty"`
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

type SpecTemplateVersionSwitchParams struct {
	TargetTemplateVersionID      int64                      `json:"target_template_version_id"`
	Variables                    map[string]any             `json:"variables,omitempty"`
	RegenerateVariables          []string                   `json:"regenerate_variables,omitempty"`
	ConfirmWarnings              bool                       `json:"confirm_warnings,omitempty"`
	RolloutMode                  string                     `json:"rollout_mode,omitempty"`
	TargetPlaygroundID           *int64                     `json:"target_playground_id,omitempty"`
	TargetPlaygroundIdentifier   string                     `json:"-"`
	ResponseMode                 string                     `json:"response_mode,omitempty"`
	Summary                      bool                       `json:"summary,omitempty"`
	ProvisionMissingRepositories string                     `json:"provision_missing_repositories,omitempty"`
	ProvisionPrivate             *bool                      `json:"provision_private,omitempty"`
	ProvisionInputs              []ProvisionRepositoryInput `json:"provision_inputs,omitempty"`
	ReuseExistingRepositories    bool                       `json:"reuse_existing_repositories,omitempty"`
}

// ProvisionRepositoryInput optionally configures the per-URL provisioning of a
// fresh git repository + Repository when switching a spec/playground to a
// template that references a repo the player does not yet own. Used together
// with SpecTemplateVersionSwitchParams.ProvisionMissingRepositories.
type ProvisionRepositoryInput struct {
	SourceRepoURL string  `json:"source_repo_url"`
	NameOverride  *string `json:"name_override,omitempty"`
	DefaultBranch *string `json:"default_branch,omitempty"`
	Description   *string `json:"description,omitempty"`
	AutoInit      *bool   `json:"auto_init,omitempty"`
}

// ProvisionedRepositoryResult describes one Repository that the server provisioned during a
// template_version_switch when ProvisionMissingRepositories was set.
type ProvisionedRepositoryResult struct {
	SourceRepoURL string         `json:"source_repo_url,omitempty"`
	ServiceName   string         `json:"service_name,omitempty"`
	Provider      string         `json:"provider,omitempty"`
	Repo          map[string]any `json:"repo,omitempty"`
	RepositoryID  int64          `json:"repository_id,omitempty"`
	DefaultBranch string         `json:"default_branch,omitempty"`
}

func (p SpecTemplateVersionSwitchParams) MarshalJSON() ([]byte, error) {
	type alias SpecTemplateVersionSwitchParams
	data, err := json.Marshal(alias(p))
	if err != nil {
		return nil, err
	}
	var body map[string]any
	if err := json.Unmarshal(data, &body); err != nil {
		return nil, err
	}
	if p.TargetPlaygroundIdentifier != "" {
		body["target_playground_id"] = p.TargetPlaygroundIdentifier
	}
	return json.Marshal(body)
}

type SpecTemplateVersionSwitchResult struct {
	FromTemplateVersion         *TemplateVersionRef           `json:"from_template_version"`
	TargetTemplateVersion       *TemplateVersionRef           `json:"target_template_version"`
	SuggestedUpgrade            bool                          `json:"suggested_upgrade"`
	RequiredVariables           []TemplateSwitchVariable      `json:"required_variables"`
	TargetVariables             []TemplateSwitchVariable      `json:"target_variables"`
	Warnings                    []TemplateSwitchWarning       `json:"warnings"`
	Diff                        map[string]any                `json:"diff"`
	PlaygroundRolloutPlan       TemplateSwitchPlaygroundPlan  `json:"playground_rollout_plan"`
	NoOp                        bool                          `json:"no_op"`
	Spec                        *Spec                         `json:"spec"`
	ProvisionedRepositories     []ProvisionedRepositoryResult `json:"provisioned_repositories,omitempty"`
	RepositoryReusePlan         map[string]any                `json:"repository_reuse_plan,omitempty"`
	RepositoryResolutionPreview *RepositoryResolutionPreview  `json:"repository_resolution_preview,omitempty"`
}

// RepositoryResolutionPreview classifies each repo URL the target template version
// references against the player's existing Repositories. Populated only on preview
// (mode != apply) responses from /api/specs/:id/template_switches.
type RepositoryResolutionPreview struct {
	Existing          []RepositoryResolutionEntry `json:"existing,omitempty"`
	ExistingForks     []RepositoryResolutionEntry `json:"existing_forks,omitempty"`
	WouldCreatePublic []RepositoryResolutionEntry `json:"would_create_public,omitempty"`
	WouldProvision    []RepositoryResolutionEntry `json:"would_provision,omitempty"`
	MissingPrivate    []RepositoryResolutionEntry `json:"missing_private,omitempty"`
}

// RepositoryResolutionEntry is one row inside RepositoryResolutionPreview.
type RepositoryResolutionEntry struct {
	URL                  string `json:"url"`
	NormalizedURL        string `json:"normalized_url,omitempty"`
	ServiceName          string `json:"service_name,omitempty"`
	RepositoryID         int64  `json:"repository_id,omitempty"`
	ProvisionProvider    string `json:"provision_provider,omitempty"`
	RuntimeWritable      *bool  `json:"runtime_writable,omitempty"`
	RuntimeAccessSource  string `json:"runtime_access_source,omitempty"`
	RuntimeAccessMessage string `json:"runtime_access_message,omitempty"`
	RequiresFork         bool   `json:"requires_fork,omitempty"`
}

type SpecTemplateVersionSwitchPreview = SpecTemplateVersionSwitchResult

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

type SpecListParams struct {
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
