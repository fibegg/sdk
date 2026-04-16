package fibe

import "time"

const (
	MonitorTypeMessage  = "message"
	MonitorTypeActivity = "activity"
	MonitorTypeMutter   = "mutter"
	MonitorTypeArtefact = "artefact"
)

var MonitorValidTypes = []string{
	MonitorTypeMessage,
	MonitorTypeActivity,
	MonitorTypeMutter,
	MonitorTypeArtefact,
}

// MonitorEvent is a single normalized event across the monitorable types.
type MonitorEvent struct {
	Type       string         `json:"type"`
	AgentID    int64          `json:"agent_id"`
	OccurredAt string         `json:"occurred_at"`
	ItemID     string         `json:"item_id"`
	Payload    map[string]any `json:"payload"`
	Truncated  bool           `json:"truncated"`
}

// MonitorMeta is the standard page/per_page list envelope for monitor events.
type MonitorMeta struct {
	Page    int `json:"page"`
	PerPage int `json:"per_page"`
	Total   int `json:"total"`
}

type MonitorListResult struct {
	Data []MonitorEvent
	Meta MonitorMeta
}

type monitorListEnvelope struct {
	Data []MonitorEvent `json:"data"`
	Meta MonitorMeta    `json:"meta"`
}

// MonitorListParams are the filters accepted by GET /api/monitor.
//
// CSV fields are serialized as comma-separated values because the Rails API
// accepts both array-form and comma lists.
type MonitorListParams struct {
	AgentIDs     string `url:"agent_id,omitempty"`
	Types        string `url:"types,omitempty"`
	Since        string `url:"since,omitempty"`
	Q            string `url:"q,omitempty"`
	ContentLimit int    `url:"content_limit,omitempty"`
	Page         int    `url:"page,omitempty"`
	PerPage      int    `url:"per_page,omitempty"`
}

// MonitorFollowOptions controls the polling-based Follow loop.
type MonitorFollowOptions struct {
	PollInterval time.Duration
	Duration     time.Duration
	MaxEvents    int
}
