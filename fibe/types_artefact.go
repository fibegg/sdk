package fibe

import "time"

// Artefact represents a file produced by an agent.
type Artefact struct {
	ID           int64     `json:"id"`
	AgentID      int64     `json:"agent_id"`
	PlayerID     *int64    `json:"player_id"`
	PlaygroundID *int64    `json:"playground_id"`
	Name         string    `json:"name"`
	Description  *string   `json:"description"`
	Body         *string   `json:"body"`
	PlainText    *bool     `json:"plain_text"`
	Filename     *string   `json:"filename"`
	ContentType  *string   `json:"content_type"`
	ByteSize     *int64    `json:"byte_size"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type ArtefactListParams struct {
	Query         string `url:"q,omitempty"`
	Name          string `url:"name,omitempty"`
	AgentID       string `url:"agent_id,omitempty"`
	PlaygroundID  string `url:"playground_id,omitempty"`
	ContentType   string `url:"content_type,omitempty"`
	CreatedAfter  string `url:"created_after,omitempty"`
	CreatedBefore string `url:"created_before,omitempty"`
	Sort          string `url:"sort,omitempty"`
	Page          int    `url:"page,omitempty"`
	PerPage       int    `url:"per_page,omitempty"`
}

type ArtefactCreateParams struct {
	Name         string `json:"name"`
	Description  string `json:"description,omitempty"`
	PlaygroundID *int64 `json:"playground_id,omitempty"`
}
