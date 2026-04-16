package fibe

import "time"

type AuditLog struct {
	ID           int64          `json:"id"`
	ResourceType string         `json:"resource_type"`
	ResourceID   *int64         `json:"resource_id"`
	Action       string         `json:"action"`
	Channel      string         `json:"channel"`
	ActorType    string         `json:"actor_type"`
	ActorID      *int64         `json:"actor_id"`
	Metadata     map[string]any `json:"metadata"`
	CreatedAt    time.Time      `json:"created_at"`
}

type AuditLogListParams struct {
	Q             string `url:"q,omitempty"`
	ResourceType  string `url:"resource_type,omitempty"`
	Channel       string `url:"channel,omitempty"`
	ActionPrefix  string `url:"action_prefix,omitempty"`
	CreatedAfter  string `url:"created_after,omitempty"`
	CreatedBefore string `url:"created_before,omitempty"`
	Sort          string `url:"sort,omitempty"`
	Page          int    `url:"page,omitempty"`
	PerPage       int    `url:"per_page,omitempty"`
}

