package fibe

import "time"

type Mutter struct {
	ID           *int64         `json:"id"`
	AgentID      int64          `json:"agent_id"`
	PlaygroundID *int64         `json:"playground_id"`
	Data         []any          `json:"data,omitempty"`
	Meta         *ListMeta      `json:"meta,omitempty"`
	Content      map[string]any `json:"content,omitempty"`
	CreatedAt    *time.Time     `json:"created_at"`
	UpdatedAt    *time.Time     `json:"updated_at"`
}

type MutterListParams struct {
	PlaygroundID string `url:"playground_id,omitempty"`
	Query        string `url:"q,omitempty"`
	Status       string `url:"status,omitempty"`
	Severity     string `url:"severity,omitempty"`
	Page         int    `url:"page,omitempty"`
	PerPage      int    `url:"per_page,omitempty"`
}

type MutterItemParams struct {
	Type         string `json:"type"`
	Body         string `json:"body"`
	PlaygroundID *int64 `json:"playground_id,omitempty"`
}

