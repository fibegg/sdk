package fibe

import "time"

const (
	ProviderGemini      = "gemini"
	ProviderClaudeCode  = "claude-code"
	ProviderOpenAICodex = "openai-codex"
	ProviderOpenCode    = "opencode"
)

var ValidProviders = []string{ProviderGemini, ProviderClaudeCode, ProviderOpenAICodex, ProviderOpenCode}

// Agent represents an AI agent configuration.
type Agent struct {
	ID              int64              `json:"id"`
	Name            string             `json:"name"`
	Description     *string            `json:"description"`
	Provider        string             `json:"provider"`
	Status          string             `json:"status"`
	SyncEnabled     bool               `json:"sync_enabled"`
	SyscheckEnabled bool               `json:"syscheck_enabled"`
	APIKeyID        *int64             `json:"api_key_id"`
	CreatedAt       *time.Time         `json:"created_at"`
	UpdatedAt       *time.Time         `json:"updated_at"`
	Authenticated   bool               `json:"authenticated"`
	ProviderLabel   string             `json:"provider_label"`
	MountedFiles    []MountedFileEntry `json:"mounted_files,omitempty"`
}

type MountedFileEntry struct {
	Name           string   `json:"name"`
	Path           string   `json:"path,omitempty"`
	ContentType    string   `json:"content_type,omitempty"`
	TargetServices []string `json:"target_services,omitempty"`
	ReadOnly       bool     `json:"readonly"`
}

type AgentCreateParams struct {
	Name               string  `json:"name"`
	Provider           string  `json:"provider"`
	APIKeyID           *int64  `json:"api_key_id,omitempty"`
	SyncEnabled        *bool   `json:"sync_enabled,omitempty"`
	SyscheckEnabled    *bool   `json:"syscheck_enabled,omitempty"`
	BuildInPublic      *bool   `json:"build_in_public,omitempty"`
	Description        *string `json:"description,omitempty"`
	ProviderAPIKeyMode *string `json:"provider_api_key_mode,omitempty"`
	MemoryLimit        *int    `json:"memory_limit,omitempty"`
	CpuLimit           *int    `json:"cpu_limit,omitempty"`
}

func (p *AgentCreateParams) Validate() error {
	v := &validator{}
	v.required("name", p.Name)
	v.required("provider", p.Provider)
	v.oneOf("provider", p.Provider, ValidProviders)
	return v.err()
}

type AgentUpdateParams struct {
	Name                      *string `json:"name,omitempty"`
	APIKeyID                  *int64  `json:"api_key_id,omitempty"`
	SyncEnabled               *bool   `json:"sync_enabled,omitempty"`
	SyscheckEnabled           *bool   `json:"syscheck_enabled,omitempty"`
	BuildInPublic             *bool   `json:"build_in_public,omitempty"`
	Description               *string `json:"description,omitempty"`
	MemoryLimit               *int    `json:"memory_limit,omitempty"`
	CpuLimit                  *int    `json:"cpu_limit,omitempty"`
	BuildInPublicPlaygroundID *int64  `json:"build_in_public_playground_id,omitempty"`
}

type AgentChatParams struct {
	Text                string   `json:"text"`
	Images              []string `json:"images,omitempty"`
	AttachmentFilenames []string `json:"attachment_filenames,omitempty"`
}

type AgentChatSession struct {
	ID             int64   `json:"id"`
	Status         string  `json:"status"`
	ChatURL        *string `json:"chat_url"`
	Subdomain      string  `json:"subdomain"`
	ComposeProject string  `json:"compose_project"`
}

type AgentSearchResult struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	Messages   []any  `json:"messages"`
	Activities []any  `json:"activities"`
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
