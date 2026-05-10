package fibe

import (
	"encoding/json"
	"time"
)

// Artefact represents a file produced by an agent.
type Artefact struct {
	ID           int64     `json:"id"`
	AgentID      *int64    `json:"agent_id"`
	PlayerID     *int64    `json:"player_id"`
	PlaygroundID *int64    `json:"playground_id"`
	Name         string    `json:"name"`
	Description  *string   `json:"description"`
	Body         *string   `json:"body"`
	PlainText    *bool     `json:"plain_text"`
	Skill        bool      `json:"skill"`
	SkillEnabled bool      `json:"skill_enabled"`
	SkillPath    string    `json:"skill_path"`
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
	Name                 string `json:"name"`
	Description          string `json:"description,omitempty"`
	AgentID              *int64 `json:"agent_id,omitempty"`
	AgentIdentifier      string `json:"-"`
	PlaygroundID         *int64 `json:"playground_id,omitempty"`
	PlaygroundIdentifier string `json:"-"`
	Body                 string `json:"body,omitempty"`
	PlainText            *bool  `json:"plain_text,omitempty"`
	Skill                *bool  `json:"skill,omitempty"`
	SkillEnabled         *bool  `json:"skill_enabled,omitempty"`
}

func (p ArtefactCreateParams) MarshalJSON() ([]byte, error) {
	type alias ArtefactCreateParams
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
	if p.AgentIdentifier != "" {
		body["agent_id"] = p.AgentIdentifier
	}
	return json.Marshal(body)
}

type ArtefactUpdateParams struct {
	Name         *string `json:"name,omitempty"`
	Description  *string `json:"description,omitempty"`
	Body         *string `json:"body,omitempty"`
	PlainText    *bool   `json:"plain_text,omitempty"`
	Skill        *bool   `json:"skill,omitempty"`
	SkillEnabled *bool   `json:"skill_enabled,omitempty"`
}
