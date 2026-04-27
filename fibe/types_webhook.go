package fibe

import "time"

var WebhookKnownEvents = []string{
	"playground.created",
	"playground.updated",
	"playground.destroyed",
	"playground.player.created",
	"playground.player.deleted",
	"playground.player.recreated",
	"playground.player.extended",
	"playground.player.retried",
	"playground.player.committed",
	"playground.status.changed",
	"playground.error",
	"playground.creation.started",
	"playground.creation.completed",
	"playground.creation.failed",
	"playground.recreated",
	"playground.recovered",
	"playground.completed",
	"playground.drift.detected",
	"playground.drift.recreated",
	"playground.drift.skipped_dirty",
	"playground.playguard.recovered",
	"playground.playguard.deleted",
	"playground.expired",
	"playground.expiration.extended",
	"playground.expiration.skipped_dirty",
	"playground.build_drift.detected",
	"marquee.created",
	"marquee.updated",
	"marquee.destroyed",
	"prop.created",
	"prop.updated",
	"prop.destroyed",
	"playspec.created",
	"playspec.updated",
	"playspec.destroyed",
	"agent.created",
	"agent.updated",
	"agent.destroyed",
	"agent.authenticated",
	"agent.revoked",
	"template.created",
	"template.updated",
	"template.destroyed",
	"artefact.created",
	"artefact.destroyed",
	"feedback.created",
	"feedback.updated",
	"feedback.destroyed",
	"mutter.created",
	"mutter.updated",
	"api_key.created",
	"api_key.destroyed",
	"secret.created",
	"secret.updated",
	"secret.destroyed",
	"webhook_endpoint.created",
	"webhook_endpoint.updated",
	"webhook_endpoint.destroyed",
	"webhook.test",
}

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
