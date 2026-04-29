package fibe

import "time"

const (
	ProviderGemini      = "gemini"
	ProviderClaudeCode  = "claude-code"
	ProviderOpenAICodex = "openai-codex"
	ProviderOpenCode    = "opencode"
	ProviderCursor      = "cursor"
)

var ValidProviders = []string{ProviderGemini, ProviderClaudeCode, ProviderOpenAICodex, ProviderOpenCode, ProviderCursor}

// Agent represents an AI agent configuration.
type Agent struct {
	ID                        int64              `json:"id"`
	Name                      string             `json:"name"`
	Description               *string            `json:"description"`
	Provider                  string             `json:"provider"`
	Status                    string             `json:"status"`
	SyncEnabled               bool               `json:"sync_enabled"`
	SyncSkillsEnabled         bool               `json:"sync_skills_enabled"`
	SyscheckEnabled           bool               `json:"syscheck_enabled"`
	BuildInPublic             bool               `json:"build_in_public"`
	BuildInPublicPlaygroundID *int64             `json:"build_in_public_playground_id"`
	ProviderAPIKeyMode        bool               `json:"provider_api_key_mode"`
	Mode                      string             `json:"mode"`
	Settings                  map[string]any     `json:"settings,omitempty"`
	Prompt                    *string            `json:"prompt"`
	MCPJSON                   *string            `json:"mcp_json"`
	PostInitScript            *string            `json:"post_init_script"`
	CustomEnv                 *string            `json:"custom_env"`
	CLIVersion                *string            `json:"cli_version"`
	ProviderArgs              map[string]any     `json:"provider_args,omitempty"`
	SkillToggles              map[string]any     `json:"skill_toggles,omitempty"`
	ModelOptions              *string            `json:"model_options"`
	MemoryLimit               *string            `json:"memory_limit"`
	CpuLimit                  *string            `json:"cpu_limit"`
	EffectivePrompt           *string            `json:"effective_prompt"`
	EffectiveModelOptions     *string            `json:"effective_model_options"`
	EffectiveMemoryLimit      *string            `json:"effective_memory_limit"`
	EffectiveCpuLimit         *string            `json:"effective_cpu_limit"`
	EffectivePostInitScript   *string            `json:"effective_post_init_script"`
	EffectiveCLIVersion       *string            `json:"effective_cli_version"`
	EffectiveProviderArgs     map[string]any     `json:"effective_provider_args,omitempty"`
	EffectiveSkillToggles     map[string]any     `json:"effective_skill_toggles,omitempty"`
	APIKeyID                  *int64             `json:"api_key_id"`
	CreatedAt                 *time.Time         `json:"created_at"`
	UpdatedAt                 *time.Time         `json:"updated_at"`
	Authenticated             bool               `json:"authenticated"`
	ProviderLabel             string             `json:"provider_label"`
	MountedFiles              []MountedFileEntry `json:"mounted_files,omitempty"`
}

type MountedFileEntry struct {
	Name           string   `json:"name"`
	Path           string   `json:"path,omitempty"`
	ContentType    string   `json:"content_type,omitempty"`
	TargetServices []string `json:"target_services,omitempty"`
	ReadOnly       bool     `json:"readonly"`
}

type AgentCreateParams struct {
	Name                      string           `json:"name"`
	Provider                  string           `json:"provider"`
	APIKeyID                  *int64           `json:"api_key_id,omitempty"`
	SyncEnabled               *bool            `json:"sync_enabled,omitempty"`
	SyncSkillsEnabled         *bool            `json:"sync_skills_enabled,omitempty"`
	SyscheckEnabled           *bool            `json:"syscheck_enabled,omitempty"`
	BuildInPublic             *bool            `json:"build_in_public,omitempty"`
	BuildInPublicPlaygroundID *int64           `json:"build_in_public_playground_id,omitempty"`
	Description               *string          `json:"description,omitempty"`
	ProviderAPIKeyMode        *bool            `json:"provider_api_key_mode,omitempty"`
	Mode                      *string          `json:"mode,omitempty"`
	ModelOptions              *string          `json:"model_options,omitempty"`
	MemoryLimit               *string          `json:"memory_limit,omitempty"`
	CpuLimit                  *string          `json:"cpu_limit,omitempty"`
	Settings                  map[string]any   `json:"settings,omitempty"`
	Prompt                    *string          `json:"prompt,omitempty"`
	MCPJSON                   *string          `json:"mcp_json,omitempty"`
	PostInitScript            *string          `json:"post_init_script,omitempty"`
	CustomEnv                 *string          `json:"custom_env,omitempty"`
	CLIVersion                *string          `json:"cli_version,omitempty"`
	ProviderArgs              map[string]any   `json:"provider_args,omitempty"`
	ProviderArgsCLI           *string          `json:"provider_args_cli,omitempty"`
	SkillToggles              map[string]bool  `json:"skill_toggles,omitempty"`
	Mounts                    []AgentMountSpec `json:"mounts,omitempty"`
}

func (p *AgentCreateParams) Validate() error {
	v := &validator{}
	v.required("name", p.Name)
	v.required("provider", p.Provider)
	v.oneOf("provider", p.Provider, ValidProviders)
	return v.err()
}

type AgentUpdateParams struct {
	Name                      *string         `json:"name,omitempty"`
	APIKeyID                  *int64          `json:"api_key_id,omitempty"`
	SyncEnabled               *bool           `json:"sync_enabled,omitempty"`
	SyncSkillsEnabled         *bool           `json:"sync_skills_enabled,omitempty"`
	SyscheckEnabled           *bool           `json:"syscheck_enabled,omitempty"`
	BuildInPublic             *bool           `json:"build_in_public,omitempty"`
	Description               *string         `json:"description,omitempty"`
	ProviderAPIKeyMode        *bool           `json:"provider_api_key_mode,omitempty"`
	Mode                      *string         `json:"mode,omitempty"`
	ModelOptions              *string         `json:"model_options,omitempty"`
	MemoryLimit               *string         `json:"memory_limit,omitempty"`
	CpuLimit                  *string         `json:"cpu_limit,omitempty"`
	BuildInPublicPlaygroundID *int64          `json:"build_in_public_playground_id,omitempty"`
	Settings                  map[string]any  `json:"settings,omitempty"`
	Prompt                    *string         `json:"prompt,omitempty"`
	MCPJSON                   *string         `json:"mcp_json,omitempty"`
	PostInitScript            *string         `json:"post_init_script,omitempty"`
	CustomEnv                 *string         `json:"custom_env,omitempty"`
	CLIVersion                *string         `json:"cli_version,omitempty"`
	ProviderArgs              map[string]any  `json:"provider_args,omitempty"`
	ProviderArgsCLI           *string         `json:"provider_args_cli,omitempty"`
	SkillToggles              map[string]bool `json:"skill_toggles,omitempty"`
}

type AgentMountSpec struct {
	SourceType     string   `json:"source_type,omitempty"`
	Filename       string   `json:"filename,omitempty"`
	ContentBase64  string   `json:"content_base64,omitempty"`
	ContentPath    string   `json:"content_path,omitempty"`
	ContentType    string   `json:"content_type,omitempty"`
	ArtefactID     *int64   `json:"artefact_id,omitempty"`
	MountPath      string   `json:"mount_path,omitempty"`
	TargetServices []string `json:"target_services,omitempty"`
	ReadOnly       *bool    `json:"readonly,omitempty"`
}

type AgentChatParams struct {
	Text string `json:"text"`
}

type AgentChatSession struct {
	ID             int64   `json:"id"`
	Status         string  `json:"status"`
	ChatURL        *string `json:"chat_url"`
	Subdomain      string  `json:"subdomain"`
	ComposeProject string  `json:"compose_project"`
}

type AgentRuntimeStatus struct {
	ID               int64   `json:"id"`
	Status           string  `json:"status"`
	ChatURL          *string `json:"chat_url"`
	Subdomain        string  `json:"subdomain"`
	ComposeProject   string  `json:"compose_project"`
	RuntimeReachable bool    `json:"runtime_reachable"`
	Authenticated    bool    `json:"authenticated"`
	IsProcessing     bool    `json:"is_processing"`
	QueueCount       int     `json:"queue_count"`
}

type AgentData struct {
	Content any `json:"content"`
}

type GitHubToken struct {
	Token     string `json:"token"`
	ExpiresIn int64  `json:"expires_in"`
}

type GiteaToken struct {
	Token     string `json:"token"`
	GiteaHost string `json:"gitea_host"`
	Username  string `json:"username"`
}

type AgentListParams struct {
	Q             string `url:"q,omitempty"`
	Provider      string `url:"provider,omitempty"`
	Status        string `url:"status,omitempty"`
	Name          string `url:"name,omitempty"`
	CreatedAfter  string `url:"created_after,omitempty"`
	CreatedBefore string `url:"created_before,omitempty"`
	Sort          string `url:"sort,omitempty"`
	Page          int    `url:"page,omitempty"`
	PerPage       int    `url:"per_page,omitempty"`
}
