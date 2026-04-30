package fibe

import "time"

type ConversationArchive struct {
	ID                       int64            `json:"id"`
	Provider                 string           `json:"provider"`
	UUID                     string           `json:"uuid"`
	Project                  string           `json:"project,omitempty"`
	Path                     string           `json:"path,omitempty"`
	SourcePath               string           `json:"source_path,omitempty"`
	RawContent               string           `json:"raw_content,omitempty"`
	LastMessageDate          *time.Time       `json:"last_message_date,omitempty"`
	FirstUserMessageSentence string           `json:"first_user_message_sentence,omitempty"`
	MessageCount             int              `json:"message_count"`
	UserMessageCount         int              `json:"user_message_count"`
	TotalTokenCount          int64            `json:"total_token_count"`
	RawEventCount            int              `json:"raw_event_count"`
	MessagesComplete         bool             `json:"messages_complete"`
	RawEventsComplete        bool             `json:"raw_events_complete"`
	MessagesDigest           string           `json:"messages_digest,omitempty"`
	RawEventsDigest          string           `json:"raw_events_digest,omitempty"`
	RawEventsAttached        bool             `json:"raw_events_attached"`
	Metadata                 map[string]any   `json:"metadata,omitempty"`
	ImportProvenance         map[string]any   `json:"import_provenance,omitempty"`
	ImportedAt               *time.Time       `json:"imported_at,omitempty"`
	CreatedAt                *time.Time       `json:"created_at,omitempty"`
	UpdatedAt                *time.Time       `json:"updated_at,omitempty"`
	Messages                 []map[string]any `json:"messages,omitempty"`
}

type MemoryGrounding struct {
	ID                    int64          `json:"id"`
	MemoryID              int64          `json:"memory_id"`
	ConversationMessageID *int64         `json:"conversation_message_id,omitempty"`
	MessagePosition       *int           `json:"message_position,omitempty"`
	ProviderMessageUUID   string         `json:"provider_message_uuid,omitempty"`
	Role                  string         `json:"role,omitempty"`
	RawEventIndex         *int           `json:"raw_event_index,omitempty"`
	StartCharacter        *int           `json:"start_character,omitempty"`
	EndCharacter          *int           `json:"end_character,omitempty"`
	RawStartCharacter     *int           `json:"raw_start_character,omitempty"`
	RawEndCharacter       *int           `json:"raw_end_character,omitempty"`
	Quote                 string         `json:"quote,omitempty"`
	Metadata              map[string]any `json:"metadata,omitempty"`
	CreatedAt             *time.Time     `json:"created_at,omitempty"`
	UpdatedAt             *time.Time     `json:"updated_at,omitempty"`
}

type Memory struct {
	ID             int64             `json:"id"`
	PlayerID       int64             `json:"player_id"`
	ConversationID string            `json:"conversation_id"`
	AgentID        *int64            `json:"agent_id,omitempty"`
	Provider       string            `json:"provider"`
	Project        string            `json:"project,omitempty"`
	MemoryKey      string            `json:"memory_key"`
	Content        string            `json:"content"`
	Tags           []string          `json:"tags"`
	Confidence     float64           `json:"confidence"`
	Metadata       map[string]any    `json:"metadata,omitempty"`
	Conversation   map[string]any    `json:"conversation,omitempty"`
	Groundings     []MemoryGrounding `json:"groundings,omitempty"`
	CreatedAt      *time.Time        `json:"created_at,omitempty"`
	UpdatedAt      *time.Time        `json:"updated_at,omitempty"`
}

type MemoryListParams struct {
	Query          string `url:"q,omitempty" json:"q,omitempty"`
	Tags           string `url:"tags,omitempty" json:"tags,omitempty"`
	Provider       string `url:"provider,omitempty" json:"provider,omitempty"`
	Project        string `url:"project,omitempty" json:"project,omitempty"`
	ConversationID string `url:"conversation_id,omitempty" json:"conversation_id,omitempty"`
	UpdatedAfter   string `url:"updated_after,omitempty" json:"updated_after,omitempty"`
	UpdatedBefore  string `url:"updated_before,omitempty" json:"updated_before,omitempty"`
	Sort           string `url:"sort,omitempty" json:"sort,omitempty"`
	Page           int    `url:"page,omitempty" json:"page,omitempty"`
	PerPage        int    `url:"per_page,omitempty" json:"per_page,omitempty"`
}

type MemoryMemorizeResult struct {
	Status       string              `json:"status"`
	Conversation ConversationArchive `json:"conversation"`
	Counts       map[string]int      `json:"counts"`
	Memories     []map[string]any    `json:"memories"`
	Errors       []map[string]any    `json:"errors"`
}
