package fibe

import "time"

type WebhookEndpoint struct {
	ID              *int64              `json:"id"`
	URL             string              `json:"url"`
	Events          []string            `json:"events"`
	EventFilters    any                 `json:"event_filters"`
	ToolFilters     map[string][]string `json:"tool_filters"`
	Description     *string             `json:"description"`
	Enabled         *bool               `json:"enabled"`
	FailureCount    *int64              `json:"failure_count"`
	LastTriggeredAt *time.Time          `json:"last_triggered_at"`
	CreatedAt       *time.Time          `json:"created_at"`
	UpdatedAt       *time.Time          `json:"updated_at"`
	Secret          *string             `json:"secret,omitempty"`
}

type WebhookDelivery struct {
	ID           *int64     `json:"id"`
	EventType    string     `json:"event_type"`
	Status       string     `json:"status"`
	ResponseCode *int       `json:"response_code"`
	Attempt      *int       `json:"attempt"`
	DeliveredAt  *time.Time `json:"delivered_at"`
	CreatedAt    *time.Time `json:"created_at"`
}

type WebhookEndpointCreateParams struct {
	URL          string              `json:"url"`
	Secret       string              `json:"secret,omitempty"`
	Events       []string            `json:"events"`
	Description  *string             `json:"description,omitempty"`
	EventFilters map[string]any      `json:"event_filters,omitempty"`
	ToolFilters  map[string][]string `json:"tool_filters,omitempty"`
}

func (p *WebhookEndpointCreateParams) Validate() error {
	v := &validator{}
	v.required("url", p.URL)
	if len(p.Events) == 0 {
		v.errors = append(v.errors, ValidationError{Field: "events", Message: "at least one event is required"})
	}
	return v.err()
}

type WebhookEndpointUpdateParams struct {
	URL          *string             `json:"url,omitempty"`
	Secret       *string             `json:"secret,omitempty"`
	Events       []string            `json:"events,omitempty"`
	Description  *string             `json:"description,omitempty"`
	Enabled      *bool               `json:"enabled,omitempty"`
	EventFilters map[string]any      `json:"event_filters,omitempty"`
	ToolFilters  map[string][]string `json:"tool_filters,omitempty"`
}

type WebhookEndpointListParams struct {
	Q       string `url:"q,omitempty"`
	Enabled *bool  `url:"enabled,omitempty"`
	URL     string `url:"url,omitempty"`
	Sort    string `url:"sort,omitempty"`
	Page    int    `url:"page,omitempty"`
	PerPage int    `url:"per_page,omitempty"`
}
