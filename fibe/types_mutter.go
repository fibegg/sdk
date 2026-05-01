package fibe

import (
	"encoding/json"
	"time"
)

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
	Type                 string `json:"type"`
	Body                 string `json:"body"`
	PlaygroundID         *int64 `json:"playground_id,omitempty"`
	PlaygroundIdentifier string `json:"-"`
}

func (p MutterItemParams) MarshalJSON() ([]byte, error) {
	type alias MutterItemParams
	data, err := json.Marshal(alias(p))
	if err != nil {
		return nil, err
	}
	var body map[string]any
	if err := json.Unmarshal(data, &body); err != nil {
		return nil, err
	}
	if p.PlaygroundIdentifier != "" {
		body["playground_id"] = p.PlaygroundIdentifier
	}
	return json.Marshal(body)
}
