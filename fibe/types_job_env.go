package fibe

import (
	"encoding/json"
	"time"
)

type JobEnvEntry struct {
	ID          *int64     `json:"id"`
	PlayerID    *int64     `json:"player_id,omitempty"`
	PropID      *int64     `json:"prop_id,omitempty"`
	PropName    *string    `json:"prop_name,omitempty"`
	Key         string     `json:"key"`
	Value       *string    `json:"value,omitempty"`
	Secret      bool       `json:"secret"`
	Enabled     bool       `json:"enabled"`
	Description *string    `json:"description,omitempty"`
	CreatedAt   *time.Time `json:"created_at,omitempty"`
	UpdatedAt   *time.Time `json:"updated_at,omitempty"`
}

type JobEnvListParams struct {
	PropID         int64  `url:"prop_id,omitempty"`
	PropIdentifier string `url:"prop_id,omitempty"`
	Secret         *bool  `url:"secret,omitempty"`
	Enabled        *bool  `url:"enabled,omitempty"`
	Q              string `url:"q,omitempty"`
	Page           int    `url:"page,omitempty"`
	PerPage        int    `url:"per_page,omitempty"`
}

type JobEnvSetParams struct {
	PropID         *int64  `json:"prop_id,omitempty"`
	PropIdentifier string  `json:"-"`
	Key            string  `json:"key"`
	Value          string  `json:"value"`
	Secret         bool    `json:"secret,omitempty"`
	Enabled        *bool   `json:"enabled,omitempty"`
	Description    *string `json:"description,omitempty"`
}

func (p *JobEnvSetParams) Validate() error {
	v := &validator{}
	v.required("key", p.Key)
	v.required("value", p.Value)
	return v.err()
}

func (p JobEnvSetParams) MarshalJSON() ([]byte, error) {
	type alias JobEnvSetParams
	data, err := json.Marshal(alias(p))
	if err != nil {
		return nil, err
	}
	var body map[string]any
	if err := json.Unmarshal(data, &body); err != nil {
		return nil, err
	}
	if p.PropIdentifier != "" {
		body["prop_id"] = p.PropIdentifier
	}
	return json.Marshal(body)
}

type JobEnvUpdateParams struct {
	Value       *string `json:"value,omitempty"`
	Secret      *bool   `json:"secret,omitempty"`
	Enabled     *bool   `json:"enabled,omitempty"`
	Description *string `json:"description,omitempty"`
}
