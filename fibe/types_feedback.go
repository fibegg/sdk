package fibe

import (
	"encoding/json"
	"time"
)

type Feedback struct {
	ID             *int64     `json:"id"`
	AgentID        int64      `json:"agent_id"`
	PlayerID       int64      `json:"player_id"`
	PlaygroundID   *int64     `json:"playground_id"`
	SourceType     string     `json:"source_type"`
	SourceID       int64      `json:"source_id"`
	SelectionStart int        `json:"selection_start"`
	SelectionEnd   int        `json:"selection_end"`
	SelectedText   string     `json:"selected_text"`
	Comment        string     `json:"comment"`
	LineText       *string    `json:"line_text"`
	Context        *string    `json:"context"`
	CreatedAt      *time.Time `json:"created_at"`
	UpdatedAt      *time.Time `json:"updated_at"`
}

type FeedbackCreateParams struct {
	SourceType           string  `json:"source_type"`
	SourceID             *int64  `json:"source_id,omitempty"`
	SelectionStart       *int    `json:"selection_start,omitempty"`
	SelectionEnd         *int    `json:"selection_end,omitempty"`
	SelectedText         *string `json:"selected_text,omitempty"`
	Comment              *string `json:"comment,omitempty"`
	LineText             *string `json:"line_text,omitempty"`
	Context              *string `json:"context,omitempty"`
	PlaygroundID         *int64  `json:"playground_id,omitempty"`
	PlaygroundIdentifier string  `json:"-"`
}

func (p FeedbackCreateParams) MarshalJSON() ([]byte, error) {
	type alias FeedbackCreateParams
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

type FeedbackUpdateParams struct {
	Comment string `json:"comment"`
}

type FeedbackListParams struct {
	SourceType    string `url:"source_type,omitempty"`
	SourceID      string `url:"source_id,omitempty"`
	PlaygroundID  string `url:"playground_id,omitempty"`
	Query         string `url:"q,omitempty"`
	CreatedAfter  string `url:"created_after,omitempty"`
	CreatedBefore string `url:"created_before,omitempty"`
	Sort          string `url:"sort,omitempty"`
	Page          int    `url:"page,omitempty"`
	PerPage       int    `url:"per_page,omitempty"`
}
