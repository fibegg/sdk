package fibe

import "time"

type APIKey struct {
	ID                   *int64     `json:"id"`
	Label                string     `json:"label"`
	Scopes               []string   `json:"scopes"`
	ExpiresAt            *time.Time `json:"expires_at"`
	ClientID             string     `json:"client_id"`
	CreatedAt            *time.Time `json:"created_at"`
	RateLimitRPHOverride *int64     `json:"rate_limit_rph_override"`
	MaskedToken          string     `json:"masked_token"`
	AgentAccessible      bool       `json:"agent_accessible"`
	Source               string     `json:"source"`
	Token                *string    `json:"token,omitempty"`
}

type APIKeyCreateParams struct {
	Label           string             `json:"label"`
	ExpiresAt       *time.Time         `json:"expires_at,omitempty"`
	AgentAccessible *bool              `json:"agent_accessible,omitempty"`
	Scopes          []string           `json:"scopes,omitempty"`
	GranularScopes  map[string][]int64 `json:"granular_scopes,omitempty"`
}

type Player struct {
	ID           int64    `json:"id"`
	Username     string   `json:"username"`
	GithubHandle *string  `json:"github_handle"`
	Email        *string  `json:"email"`
	AvatarURL    *string  `json:"avatar_url"`
	APIKeyScopes []string `json:"api_key_scopes,omitempty"`
}

type APIKeyListParams struct {
	Q       string `url:"q,omitempty"`
	Label   string `url:"label,omitempty"`
	Sort    string `url:"sort,omitempty"`
	Page    int    `url:"page,omitempty"`
	PerPage int    `url:"per_page,omitempty"`
}
