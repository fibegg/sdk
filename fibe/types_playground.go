package fibe

import (
	"encoding/json"
	"time"
)

// Playground represents a running environment instance.
type Playground struct {
	EnvPackAttachments []EnvPackAttachment `json:"env_pack_attachments,omitempty"`
	EnvPacks           map[string]any      `json:"env_packs,omitempty"`
	OwnershipMetadata
	ID                 int64          `json:"id"`
	Name               string         `json:"name"`
	Status             string         `json:"status"`
	ResultStatus       *string        `json:"result_status,omitempty"`
	MaintenanceEnabled bool           `json:"maintenance_enabled"`
	JobMode            bool           `json:"job_mode"`
	StateReason        *string        `json:"state_reason,omitempty"`
	StateReasons       []string       `json:"state_reasons,omitempty"`
	SpecID             *int64         `json:"spec_id"`
	SpecName           *string        `json:"spec_name"`
	HostID             *int64         `json:"host_id,omitempty"`
	ServiceBranches    map[string]any `json:"service_branches"`
	ExpiresAt          *time.Time     `json:"expires_at"`
	CreatedAt          time.Time      `json:"created_at"`

	// Detail fields (only present on Get, not List)
	ComposeProject           *string                 `json:"compose_project,omitempty"`
	HostName                 *string                 `json:"host_name,omitempty"`
	RootDomain               *string                 `json:"root_domain,omitempty"`
	RoutingScheme            *string                 `json:"routing_scheme,omitempty"`
	InternalPassword         *string                 `json:"internal_password,omitempty"`
	EnvOverrides             map[string]string       `json:"env_overrides,omitempty"`
	LastAppliedAt            *time.Time              `json:"last_applied_at,omitempty"`
	ErrorMessage             *string                 `json:"error_message,omitempty"`
	PlayguardRepairReason    *string                 `json:"playguard_repair_reason,omitempty"`
	PlayguardRepairLockUntil *time.Time              `json:"playguard_repair_lock_until,omitempty"`
	NeedsRecreation          *bool                   `json:"needs_recreation,omitempty"`
	TimeRemaining            *float64                `json:"time_remaining,omitempty"`
	ExpirationPercentage     *float64                `json:"expiration_percentage,omitempty"`
	BuildWarnings            []string                `json:"build_warnings,omitempty"`
	BuildStatuses            []PlaygroundBuildStatus `json:"build_statuses,omitempty"`
	PersistentVolumePrefix   *string                 `json:"persistent_volume_prefix,omitempty"`
	ServiceURLs              []PlaygroundServiceURL  `json:"service_urls,omitempty"`
	Services                 []PlaygroundServiceInfo `json:"services,omitempty"`
	JobResult                *JobResult              `json:"job_result,omitempty"`
	RunOverrides             map[string]any          `json:"run_overrides,omitempty"`
	Teardown                 map[string]any          `json:"teardown,omitempty"`
	ServiceSources           []map[string]any        `json:"service_sources,omitempty"`
}

type PlaygroundServiceURL struct {
	Name         string `json:"name"`
	Type         string `json:"type,omitempty"`
	URL          string `json:"url"`
	Visibility   string `json:"visibility,omitempty"`
	AuthRequired bool   `json:"auth_required"`
	Status       string `json:"status,omitempty"`
	Health       string `json:"health,omitempty"`
	Running      *bool  `json:"running,omitempty"`
	ExitCode     *int   `json:"exit_code,omitempty"`
}

type PlaygroundServiceInfo struct {
	Name     string `json:"name"`
	Status   string `json:"status"`
	Image    string `json:"image,omitempty"`
	Health   string `json:"health,omitempty"`
	Running  bool   `json:"running,omitempty"`
	ExitCode *int   `json:"exit_code,omitempty"`
}

type PlaygroundBuildStatus struct {
	ServiceName    string                         `json:"service_name"`
	RepositoryID   int64                          `json:"repository_id,omitempty"`
	RepositoryName string                         `json:"repository_name,omitempty"`
	Branch         string                         `json:"branch,omitempty"`
	Running        *PlaygroundBuildRecordSnapshot `json:"running,omitempty"`
	Latest         *PlaygroundBuildRecordSnapshot `json:"latest,omitempty"`
	Active         *PlaygroundBuildRecordSnapshot `json:"active,omitempty"`
}

type PlaygroundBuildRecordSnapshot struct {
	ID             int64      `json:"id"`
	ServiceName    string     `json:"service_name,omitempty"`
	Status         string     `json:"status"`
	CommitSHA      string     `json:"commit_sha"`
	ShortCommitSHA string     `json:"short_commit_sha,omitempty"`
	ImageRef       string     `json:"image_ref,omitempty"`
	ErrorMessage   string     `json:"error_message,omitempty"`
	StartedAt      *time.Time `json:"started_at,omitempty"`
	CompletedAt    *time.Time `json:"completed_at,omitempty"`
	CreatedAt      *time.Time `json:"created_at,omitempty"`
}

type JobResult struct {
	ID             *int64            `json:"id"`
	Success        *bool             `json:"success"`
	CompletedAt    *time.Time        `json:"completed_at"`
	ServiceResults map[string]any    `json:"service_results"`
	Summary        *JobResultSummary `json:"summary,omitempty"`
}

type JobResultSummary struct {
	ServicesTotal int                   `json:"services_total"`
	WatchedTotal  int                   `json:"watched_total"`
	Failed        []string              `json:"failed"`
	Succeeded     []string              `json:"succeeded"`
	Rows          []JobResultSummaryRow `json:"rows"`
}

type JobResultSummaryRow struct {
	Name              string `json:"name"`
	Watched           bool   `json:"watched"`
	Status            string `json:"status,omitempty"`
	ExitCode          *int   `json:"exit_code,omitempty"`
	Result            string `json:"result,omitempty"`
	FinishedAt        string `json:"finished_at,omitempty"`
	LogLines          int    `json:"log_lines,omitempty"`
	LogBytes          int    `json:"log_bytes,omitempty"`
	StructuredSummary any    `json:"structured_summary,omitempty"`
}

type PlaygroundCreateParams struct {
	EnvPackAttachments *[]EnvPackAttachmentInput `json:"env_pack_attachments,omitempty"`
	Name               string                    `json:"name"`
	SpecID             int64                     `json:"spec_id"`
	SpecIdentifier     string                    `json:"-"`
	HostID             *int64                    `json:"host_id,omitempty"`
	HostIdentifier     string                    `json:"-"`
	ExpiresAt          *time.Time                `json:"expires_at,omitempty"`
	NeverExpire        *bool                     `json:"never_expire,omitempty"`
	Services           map[string]*ServiceConfig `json:"services,omitempty"`
	BuildOverridesYAML map[string]any            `json:"build_overrides_yaml,omitempty"`
	EnvOverrides       map[string]string         `json:"env_overrides,omitempty"`
	OnlyServices       []string                  `json:"only_services,omitempty"`
	ExceptServices     []string                  `json:"except_services,omitempty"`
}

func (p *PlaygroundCreateParams) Validate() error {
	v := &validator{}
	v.required("name", p.Name)
	v.requiredIDOrIdentifier("spec_id", p.SpecID, p.SpecIdentifier)
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
	if p.SpecIdentifier != "" {
		body["spec_id"] = p.SpecIdentifier
	}
	if p.HostIdentifier != "" {
		body["host_id"] = p.HostIdentifier
	}
	return json.Marshal(body)
}

// ServiceConfig configures a single service within a playground.
type ServiceConfig struct {
	Subdomain          string            `json:"subdomain,omitempty"`
	ExposureVisibility string            `json:"exposure_visibility,omitempty"`
	PathRule           string            `json:"path_rule,omitempty"`
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
	Q              string `url:"q,omitempty"`
	Status         string `url:"status,omitempty"`
	JobMode        *bool  `url:"job_mode,omitempty"`
	SpecID         int64  `url:"spec_id,omitempty"`
	SpecIdentifier string `url:"spec_id,omitempty"`
	HostID         int64  `url:"host_id,omitempty"`
	HostIdentifier string `url:"host_id,omitempty"`
	Name           string `url:"name,omitempty"`
	ResultStatus   string `url:"result_status,omitempty"`
	CreatedAfter   string `url:"created_after,omitempty"`
	CreatedBefore  string `url:"created_before,omitempty"`
	Sort           string `url:"sort,omitempty"`
	Page           int    `url:"page,omitempty"`
	PerPage        int    `url:"per_page,omitempty"`
}

type PlaygroundUpdateParams struct {
	EnvPackAttachments *[]EnvPackAttachmentInput `json:"env_pack_attachments,omitempty"`
	Name               *string                   `json:"name,omitempty"`
	SpecID             *int64                    `json:"spec_id,omitempty"`
	SpecIdentifier     string                    `json:"-"`
	HostID             *int64                    `json:"host_id,omitempty"`
	HostIdentifier     string                    `json:"-"`
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
	if p.SpecIdentifier != "" {
		body["spec_id"] = p.SpecIdentifier
	}
	if p.HostIdentifier != "" {
		body["host_id"] = p.HostIdentifier
	}
	return json.Marshal(body)
}

type PlaygroundStatus struct {
	ID                       int64                         `json:"id"`
	Status                   string                        `json:"status"`
	ResultStatus             *string                       `json:"result_status,omitempty"`
	MaintenanceEnabled       bool                          `json:"maintenance_enabled"`
	JobMode                  bool                          `json:"job_mode,omitempty"`
	Startup                  *PlaygroundStartupDiagnostics `json:"startup,omitempty"`
	StateReason              *string                       `json:"state_reason,omitempty"`
	StateReasons             []string                      `json:"state_reasons,omitempty"`
	CreationStep             *string                       `json:"creation_step,omitempty"`
	CreationStepLabel        *string                       `json:"creation_step_label,omitempty"`
	ErrorMessage             *string                       `json:"error_message,omitempty"`
	ErrorStep                *string                       `json:"error_step,omitempty"`
	ErrorStepLabel           *string                       `json:"error_step_label,omitempty"`
	ErrorDetails             map[string]any                `json:"error_details,omitempty"`
	FailureDiagnostics       map[string]any                `json:"failure_diagnostics,omitempty"`
	PlayguardRepairReason    *string                       `json:"playguard_repair_reason,omitempty"`
	PlayguardRepairLockUntil *time.Time                    `json:"playguard_repair_lock_until,omitempty"`
	NeedsRecreation          *bool                         `json:"needs_recreation,omitempty"`
	BuildStatuses            []PlaygroundBuildStatus       `json:"build_statuses,omitempty"`
	Services                 []PlaygroundServiceInfo       `json:"services,omitempty"`
	JobResult                *JobResult                    `json:"job_result,omitempty"`
	RunOverrides             map[string]any                `json:"run_overrides,omitempty"`
	Teardown                 map[string]any                `json:"teardown,omitempty"`
}

const (
	PlaygroundActionRollout            = "rollout"
	PlaygroundActionHardRestart        = "hard_restart"
	PlaygroundActionStop               = "stop"
	PlaygroundActionStart              = "start"
	PlaygroundActionRetryCompose       = "retry_compose"
	PlaygroundActionEnableMaintenance  = "enable_maintenance"
	PlaygroundActionDisableMaintenance = "disable_maintenance"
)

var ValidPlaygroundActions = []string{
	PlaygroundActionRollout,
	PlaygroundActionHardRestart,
	PlaygroundActionStop,
	PlaygroundActionStart,
	PlaygroundActionRetryCompose,
	PlaygroundActionEnableMaintenance,
	PlaygroundActionDisableMaintenance,
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
	Service     string                        `json:"service"`
	Lines       []string                      `json:"lines"`
	Source      string                        `json:"source"`
	Entries     []LogEntry                    `json:"entries,omitempty"`
	Startup     *PlaygroundStartupDiagnostics `json:"startup,omitempty"`
	Diagnostics map[string]any                `json:"diagnostics,omitempty"`
}

type LogEntry struct {
	Service string `json:"service,omitempty"`
	Line    string `json:"line"`
	Source  string `json:"source,omitempty"`
}

type PlaygroundStartupDiagnostics struct {
	ComposeProject       string                               `json:"compose_project,omitempty"`
	RemotePlaygroundPath string                               `json:"remote_playground_path,omitempty"`
	ComposePath          string                               `json:"compose_path,omitempty"`
	Available            bool                                 `json:"available"`
	State                string                               `json:"state,omitempty"`
	ExitCode             *int                                 `json:"exit_code,omitempty"`
	Artifacts            map[string]PlaygroundStartupArtifact `json:"artifacts,omitempty"`
	LogTail              []string                             `json:"log_tail,omitempty"`
	MissingArtifacts     []string                             `json:"missing_artifacts,omitempty"`
	Error                string                               `json:"error,omitempty"`
}

type PlaygroundStartupArtifact struct {
	Path   string  `json:"path,omitempty"`
	Exists bool    `json:"exists"`
	PID    *string `json:"pid,omitempty"`
	Alive  *bool   `json:"alive,omitempty"`
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

type TaskTriggerParams struct {
	EnvPackAttachments *[]EnvPackAttachmentInput `json:"env_pack_attachments,omitempty"`
	SpecID             int64                     `json:"spec_id"`
	SpecIdentifier     string                    `json:"-"`
	HostID             *int64                    `json:"host_id,omitempty"`
	HostIdentifier     string                    `json:"-"`
	Name               string                    `json:"name,omitempty"` // auto-generated if empty
	EnvOverrides       map[string]string         `json:"env_overrides,omitempty"`
	OnlyServices       []string                  `json:"only_services,omitempty"`
	ExceptServices     []string                  `json:"except_services,omitempty"`
}

func (p *TaskTriggerParams) specIdentifier() string {
	if p.SpecIdentifier != "" {
		return p.SpecIdentifier
	}
	return int64Identifier(p.SpecID)
}
