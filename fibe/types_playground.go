package fibe

import (
	"encoding/json"
	"time"
)

// Playground represents a running environment instance.
type Playground struct {
	ID              int64          `json:"id"`
	Name            string         `json:"name"`
	Status          string         `json:"status"`
	JobMode         bool           `json:"job_mode"`
	PlayspecID      *int64         `json:"playspec_id"`
	PlayspecName    *string        `json:"playspec_name"`
	MarqueeID       *int64         `json:"marquee_id,omitempty"`
	ServiceBranches map[string]any `json:"service_branches"`
	ExpiresAt       *time.Time     `json:"expires_at"`
	CreatedAt       time.Time      `json:"created_at"`

	// Detail fields (only present on Get, not List)
	ComposeProject       *string                 `json:"compose_project,omitempty"`
	InternalPassword     *string                 `json:"internal_password,omitempty"`
	EnvOverrides         map[string]string       `json:"env_overrides,omitempty"`
	LastAppliedAt        *time.Time              `json:"last_applied_at,omitempty"`
	ErrorMessage         *string                 `json:"error_message,omitempty"`
	NeedsRecreation      *bool                   `json:"needs_recreation,omitempty"`
	TimeRemaining        *float64                `json:"time_remaining,omitempty"`
	ExpirationPercentage *float64                `json:"expiration_percentage,omitempty"`
	BuildWarnings        []string                `json:"build_warnings,omitempty"`
	Services             []PlaygroundServiceInfo `json:"services,omitempty"`
	JobResult            *JobResult              `json:"job_result,omitempty"`
}

type PlaygroundServiceInfo struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Image  string `json:"image,omitempty"`
}

type JobResult struct {
	ID             *int64         `json:"id"`
	Success        *bool          `json:"success"`
	CompletedAt    *time.Time     `json:"completed_at"`
	ServiceResults map[string]any `json:"service_results"`
}

type PlaygroundCreateParams struct {
	Name               string                    `json:"name"`
	PlayspecID         int64                     `json:"playspec_id"`
	PlayspecIdentifier string                    `json:"-"`
	MarqueeID          *int64                    `json:"marquee_id,omitempty"`
	MarqueeIdentifier  string                    `json:"-"`
	ExpiresAt          *time.Time                `json:"expires_at,omitempty"`
	NeverExpire        *bool                     `json:"never_expire,omitempty"`
	Services           map[string]*ServiceConfig `json:"services,omitempty"`
	BuildOverridesYAML map[string]any            `json:"build_overrides_yaml,omitempty"`
}

func (p *PlaygroundCreateParams) Validate() error {
	v := &validator{}
	v.required("name", p.Name)
	v.requiredIDOrIdentifier("playspec_id", p.PlayspecID, p.PlayspecIdentifier)
	for name, svc := range p.Services {
		if svc != nil && svc.Exposure != nil {
			v.subdomain(name+".exposure.subdomain", svc.Exposure.Subdomain)
			v.port(name+".exposure.port", svc.Exposure.Port)
		}
	}
	return v.err()
}

func (p PlaygroundCreateParams) MarshalJSON() ([]byte, error) {
	type alias PlaygroundCreateParams
	data, err := json.Marshal(alias(p))
	if err != nil {
		return nil, err
	}
	var body map[string]any
	if err := json.Unmarshal(data, &body); err != nil {
		return nil, err
	}
	if p.PlayspecIdentifier != "" {
		body["playspec_id"] = p.PlayspecIdentifier
	}
	if p.MarqueeIdentifier != "" {
		body["marquee_id"] = p.MarqueeIdentifier
	}
	return json.Marshal(body)
}

// ServiceConfig configures a single service within a playground.
type ServiceConfig struct {
	Subdomain          string            `json:"subdomain,omitempty"`
	ExposureVisibility string            `json:"exposure_visibility,omitempty"`
	StartCommand       string            `json:"start_command,omitempty"`
	DockerfilePath     string            `json:"dockerfile_path,omitempty"`
	EnvFilePath        string            `json:"env_file_path,omitempty"`
	HealthcheckPath    string            `json:"healthcheck_path,omitempty"`
	Image              string            `json:"image,omitempty"`
	EnvVars            map[string]string `json:"env_vars,omitempty"`
	Exposure           *ServiceExposure  `json:"exposure,omitempty"`
	GitConfig          *GitConfig        `json:"git_config,omitempty"`
	PortMappings       []PortMapping     `json:"port_mappings,omitempty"`
	ExposurePort       *int              `json:"exposure_port,omitempty"`
}

type ServiceExposure struct {
	Enabled    bool   `json:"enabled"`
	Port       int    `json:"port,omitempty"`
	Subdomain  string `json:"subdomain,omitempty"`
	Visibility string `json:"visibility,omitempty"`
	PathRule   string `json:"path_rule,omitempty"`
}

type GitConfig struct {
	BranchName     string `json:"branch_name,omitempty"`
	BaseBranchName string `json:"base_branch_name,omitempty"`
	CreateBranch   bool   `json:"create_branch,omitempty"`
}

type PortMapping struct {
	Container string `json:"container"`
	Host      string `json:"host"`
}

// PlaygroundListParams controls filtering and pagination for playground list.
type PlaygroundListParams struct {
	Q                  string `url:"q,omitempty"`
	Status             string `url:"status,omitempty"`
	JobMode            *bool  `url:"job_mode,omitempty"`
	PlayspecID         int64  `url:"playspec_id,omitempty"`
	PlayspecIdentifier string `url:"playspec_id,omitempty"`
	MarqueeID          int64  `url:"marquee_id,omitempty"`
	MarqueeIdentifier  string `url:"marquee_id,omitempty"`
	Name               string `url:"name,omitempty"`
	CreatedAfter       string `url:"created_after,omitempty"`
	CreatedBefore      string `url:"created_before,omitempty"`
	Sort               string `url:"sort,omitempty"`
	Page               int    `url:"page,omitempty"`
	PerPage            int    `url:"per_page,omitempty"`
}

type PlaygroundUpdateParams struct {
	Name               *string                   `json:"name,omitempty"`
	PlayspecID         *int64                    `json:"playspec_id,omitempty"`
	PlayspecIdentifier string                    `json:"-"`
	MarqueeID          *int64                    `json:"marquee_id,omitempty"`
	MarqueeIdentifier  string                    `json:"-"`
	ExpiresAt          *time.Time                `json:"expires_at,omitempty"`
	NeverExpire        *bool                     `json:"never_expire,omitempty"`
	Services           map[string]*ServiceConfig `json:"services,omitempty"`
	BuildOverridesYAML map[string]any            `json:"build_overrides_yaml,omitempty"`
}

func (p PlaygroundUpdateParams) MarshalJSON() ([]byte, error) {
	type alias PlaygroundUpdateParams
	data, err := json.Marshal(alias(p))
	if err != nil {
		return nil, err
	}
	var body map[string]any
	if err := json.Unmarshal(data, &body); err != nil {
		return nil, err
	}
	if p.PlayspecIdentifier != "" {
		body["playspec_id"] = p.PlayspecIdentifier
	}
	if p.MarqueeIdentifier != "" {
		body["marquee_id"] = p.MarqueeIdentifier
	}
	return json.Marshal(body)
}

type PlaygroundStatus struct {
	ID                 int64          `json:"id"`
	Status             string         `json:"status"`
	CreationStep       *string        `json:"creation_step,omitempty"`
	CreationStepLabel  *string        `json:"creation_step_label,omitempty"`
	ErrorMessage       *string        `json:"error_message,omitempty"`
	ErrorStep          *string        `json:"error_step,omitempty"`
	ErrorStepLabel     *string        `json:"error_step_label,omitempty"`
	ErrorDetails       map[string]any `json:"error_details,omitempty"`
	FailureDiagnostics map[string]any `json:"failure_diagnostics,omitempty"`
	NeedsRecreation    *bool          `json:"needs_recreation,omitempty"`
	Services           []any          `json:"services,omitempty"`
	JobResult          *JobResult     `json:"job_result,omitempty"`
}

const (
	PlaygroundActionRollout      = "rollout"
	PlaygroundActionHardRestart  = "hard_restart"
	PlaygroundActionStop         = "stop"
	PlaygroundActionStart        = "start"
	PlaygroundActionRetryCompose = "retry_compose"
)

var ValidPlaygroundActions = []string{
	PlaygroundActionRollout,
	PlaygroundActionHardRestart,
	PlaygroundActionStop,
	PlaygroundActionStart,
	PlaygroundActionRetryCompose,
}

type PlaygroundActionParams struct {
	ActionType string `json:"action_type"`
	Force      *bool  `json:"force,omitempty"`
}

func (p *PlaygroundActionParams) Validate() error {
	v := &validator{}
	v.required("action_type", p.ActionType)
	v.oneOf("action_type", p.ActionType, ValidPlaygroundActions)
	return v.err()
}

type PlaygroundDebugParams struct {
	Mode     string `url:"mode,omitempty"`
	Service  string `url:"service,omitempty"`
	LogsTail int    `url:"logs_tail,omitempty"`
	Refresh  *bool  `url:"refresh,omitempty"`
}

type PlaygroundCompose struct {
	ComposeYAML    string `json:"compose_yaml"`
	ComposeProject string `json:"compose_project"`
}

type PlaygroundLogs struct {
	Service string   `json:"service"`
	Lines   []string `json:"lines"`
	Source  string   `json:"source"`
}

type PlaygroundEnvMetadata struct {
	Merged     map[string]string `json:"merged"`
	Metadata   map[string]any    `json:"metadata"`
	SystemKeys []string          `json:"system_keys"`
}

type PlaygroundExtendResult struct {
	ID            int64     `json:"id"`
	ExpiresAt     time.Time `json:"expires_at"`
	TimeRemaining float64   `json:"time_remaining"`
}

type TrickTriggerParams struct {
	PlayspecID         int64  `json:"playspec_id"`
	PlayspecIdentifier string `json:"-"`
	MarqueeID          *int64 `json:"marquee_id,omitempty"`
	MarqueeIdentifier  string `json:"-"`
	Name               string `json:"name,omitempty"` // auto-generated if empty
}

func (p *TrickTriggerParams) playspecIdentifier() string {
	if p.PlayspecIdentifier != "" {
		return p.PlayspecIdentifier
	}
	return int64Identifier(p.PlayspecID)
}
