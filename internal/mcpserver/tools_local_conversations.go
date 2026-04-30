package mcpserver

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/fibegg/sdk/fibe"
	"github.com/fibegg/sdk/internal/localconversations"
	"github.com/mark3labs/mcp-go/mcp"
)

const defaultLocalConversationLimit = 25
const defaultUserMessageCharLimit = 5000
const defaultAssistantMessageCharLimit = 10000

var localConversationViews = []string{"messages", "chat", "user-messages", "full"}

type localConversationMCPConversation struct {
	Provider                 string         `json:"provider"`
	UUID                     string         `json:"uuid"`
	Title                    string         `json:"title,omitempty"`
	Project                  string         `json:"project,omitempty"`
	LastMessageDate          *time.Time     `json:"last_message_date,omitempty"`
	FirstUserMessageSentence string         `json:"first_user_message_sentence"`
	UserMessageCount         int            `json:"user_message_count"`
	TotalTokenCount          int64          `json:"total_token_count"`
	Metadata                 map[string]any `json:"metadata,omitempty"`
}

type localConversationsListResponse struct {
	Conversations []localConversationMCPConversation `json:"conversations"`
	Count         int                                `json:"count"`
	Limit         int                                `json:"limit,omitempty"`
	Query         string                             `json:"query,omitempty"`
	HasMore       bool                               `json:"has_more,omitempty"`
	NextCursor    string                             `json:"next_cursor,omitempty"`
	LimitReached  bool                               `json:"limit_reached,omitempty"`
}

type localConversationGetResponse struct {
	View          string                                   `json:"view"`
	Conversation  localConversationMCPConversation         `json:"conversation"`
	Messages      []localconversations.ConversationMessage `json:"messages,omitempty"`
	Chat          []localconversations.ChatTurn            `json:"chat,omitempty"`
	UserMessages  []localconversations.ConversationMessage `json:"user_messages,omitempty"`
	RawEvents     []map[string]any                         `json:"raw_events,omitempty"`
	MessageCount  int                                      `json:"message_count"`
	RawEventCount int                                      `json:"raw_event_count"`
}

type localConversationGetMessageResponse struct {
	ConversationUUID string `json:"conversation_uuid"`
	Role             string `json:"role,omitempty"`
	Type             string `json:"type,omitempty"`
	Content          string `json:"content,omitempty"`
}

// registerLocalConversationTools exposes local AI transcript discovery as
// native MCP tools. These are intentionally local-tier tools: they read local
// filesystem state and never call the Fibe API.
func (s *Server) registerLocalConversationTools() {
	s.addTool(&toolImpl{
		name:        "fibe_local_conversations_list",
		description: "[MODE:DIALOG] List local Codex, Claude Code, and Claude Desktop conversations from this machine.",
		tier:        tierLocal,
		annotations: toolAnnotations{ReadOnly: true, Idempotent: true},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			opts, _, err := localConversationMCPListOptions(args, defaultLocalConversationLimit)
			if err != nil {
				return nil, err
			}

			page, err := localconversations.ListPage(ctx, opts)
			if err != nil {
				return nil, err
			}

			return localConversationsListResponse{
				Conversations: mcpConversations(page.Conversations),
				Count:         page.Count,
				Limit:         page.Limit,
				Query:         page.Query,
				HasMore:       page.HasMore,
				NextCursor:    page.NextCursor,
				LimitReached:  page.HasMore,
			}, nil
		},
	}, mcp.NewTool("fibe_local_conversations_list",
		mcp.WithDescription(`List local AI conversations found on this machine across Codex, Claude Code, and Claude Desktop.

The default limit is 25 for MCP responsiveness. Use limit:1 for the fastest latest-conversation lookup. Use query to search raw local transcript files before parsing.
Pass cursor from a previous response to keep going from next_cursor. If query is omitted with cursor, the original query is reused.

Use only with conversation fields: provider, uuid, title, project, last_message_date, first_user_message_sentence, user_message_count, total_token_count, metadata.`),
		mcp.WithNumber("limit", mcp.Description("Maximum conversations to return. Default 25; pass 0 for no limit.")),
		mcp.WithString("query", mcp.Description("Case-insensitive substring search across local transcript file content and conversation UUIDs.")),
		mcp.WithString("cursor", mcp.Description("Opaque cursor returned as next_cursor by a previous list call. Use it to continue to the next page.")),
	))

	s.addTool(&toolImpl{
		name:        "fibe_local_conversations_get",
		description: "[MODE:DIALOG] View one local Codex or Claude conversation by UUID or UUID prefix.",
		tier:        tierLocal,
		annotations: toolAnnotations{ReadOnly: true, Idempotent: true},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			uuid := strings.TrimSpace(argString(args, "uuid"))
			if uuid == "" {
				return nil, fmt.Errorf("required field 'uuid' not set")
			}

			opts, _, err := localConversationMCPListOptions(args, 0)
			if err != nil {
				return nil, err
			}
			detail, err := localconversations.Get(ctx, uuid, opts)
			if err != nil {
				return nil, err
			}

			view := strings.ToLower(strings.TrimSpace(argString(args, "view")))
			if view == "" {
				view = "messages"
			}
			if !isLocalConversationView(view) {
				return nil, fmt.Errorf("unknown local conversation view %q (valid: %s)", view, strings.Join(localConversationViews, ", "))
			}

			userLimit, assistantLimit, err := localConversationMessageLimits(args)
			if err != nil {
				return nil, err
			}

			return buildLocalConversationGetResponse(detail, view, userLimit, assistantLimit), nil
		},
	}, mcp.NewTool("fibe_local_conversations_get",
		mcp.WithDescription(`View one local AI conversation by UUID or UUID prefix.

Views:
  messages       normalized messages with trimmed text and metadata, without raw provider events (default)
  chat      compact human-readable user/assistant turns
  user-messages  user messages only, with stable message ids
  full           trimmed messages plus raw provider events for debugging parser/provider behavior

Conversation fields: provider, uuid, title, project, last_message_date, first_user_message_sentence, user_message_count, total_token_count, metadata.
Message fields for only: id, position, role, type, uuid, timestamp, text, content_omitted, text_truncated, full_text_char_count, token_count, metadata. Use fibe_local_conversations_get_message with message_id for full content.`),
		mcp.WithString("uuid", mcp.Required(), mcp.Description("Conversation UUID or unique UUID prefix from fibe_local_conversations_list.")),
		mcp.WithString("view", mcp.Enum(localConversationViews...), mcp.Description("Output view: messages (default), chat, user-messages, or full.")),
		mcp.WithNumber("user_message_limit", mcp.Description("Maximum characters per user message preview. Default 5000; pass 0 for no limit.")),
		mcp.WithNumber("assistant_message_limit", mcp.Description("Maximum characters per assistant message preview. Default 10000; pass 0 for no limit.")),
	))

	s.addTool(&toolImpl{
		name:        "fibe_local_conversations_get_message",
		description: "[MODE:DIALOG] View one full local conversation message by conversation UUID and message ID.",
		tier:        tierLocal,
		annotations: toolAnnotations{ReadOnly: true, Idempotent: true},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			uuid := strings.TrimSpace(argString(args, "uuid"))
			if uuid == "" {
				return nil, fmt.Errorf("required field 'uuid' not set")
			}
			messageID := strings.TrimSpace(argString(args, "message_id"))
			if messageID == "" {
				return nil, fmt.Errorf("required field 'message_id' not set")
			}

			detail, err := localconversations.Get(ctx, uuid, localconversations.ListOptions{})
			if err != nil {
				return nil, err
			}
			message, ok := localconversations.MessageByID(detail, messageID)
			if !ok {
				return nil, fmt.Errorf("no message %q found in local conversation %q", messageID, uuid)
			}

			return localConversationGetMessageResponse{
				ConversationUUID: detail.UUID,
				Role:             message.Role,
				Type:             message.Type,
				Content:          message.Text,
			}, nil
		},
	}, mcp.NewTool("fibe_local_conversations_get_message",
		mcp.WithDescription(`View one full local conversation message. Returns only conversation_uuid, role, type, and content.

Message IDs are stable within a conversation. Providers that expose message UUIDs use those UUIDs; otherwise Fibe uses deterministic position IDs such as position:1.
The content field is the normalized full message text.`),
		mcp.WithString("uuid", mcp.Required(), mcp.Description("Conversation UUID or unique UUID prefix from fibe_local_conversations_list.")),
		mcp.WithString("message_id", mcp.Required(), mcp.Description("Message id from fibe_local_conversations_get. Numeric positions such as 1 are also accepted.")),
	))
}

type localConversationToolInput struct {
	query  string
	limit  int
	cursor string
}

func localConversationMCPListOptions(args map[string]any, defaultLimit int) (localconversations.ListOptions, localConversationToolInput, error) {
	limit, err := optionalNonNegativeInt(args, "limit")
	if err != nil {
		return localconversations.ListOptions{}, localConversationToolInput{}, err
	}
	if limit == -1 {
		limit = defaultLimit
	}

	input := localConversationToolInput{
		query:  strings.TrimSpace(argString(args, "query")),
		limit:  limit,
		cursor: strings.TrimSpace(argString(args, "cursor")),
	}

	opts := localconversations.ListOptions{
		Limit:  limit,
		Query:  input.query,
		Cursor: input.cursor,
	}
	return opts, input, nil
}

// optionalNonNegativeInt returns -1 when the field was not provided. Zero is
// a valid explicit value for "unlimited" on local conversation MCP tools.
func optionalNonNegativeInt(args map[string]any, key string) (int, error) {
	n, ok := argInt64(args, key)
	if !ok {
		return -1, nil
	}
	if n < 0 {
		return 0, fmt.Errorf("field %q must be >= 0", key)
	}
	if n > int64(math.MaxInt) {
		return 0, fmt.Errorf("field %q is too large", key)
	}
	return int(n), nil
}

func isLocalConversationView(view string) bool {
	for _, candidate := range localConversationViews {
		if view == candidate {
			return true
		}
	}
	return false
}

func mcpConversations(conversations []localconversations.Conversation) []localConversationMCPConversation {
	out := make([]localConversationMCPConversation, len(conversations))
	for i, conversation := range conversations {
		out[i] = mcpConversation(conversation)
	}
	return out
}

func mcpConversation(conversation localconversations.Conversation) localConversationMCPConversation {
	return localConversationMCPConversation{
		Provider:                 conversation.Provider,
		UUID:                     conversation.UUID,
		Title:                    conversation.Title,
		Project:                  conversation.Project,
		LastMessageDate:          conversation.LastMessageDate,
		FirstUserMessageSentence: conversation.FirstUserMessageSentence,
		UserMessageCount:         conversation.UserMessageCount,
		TotalTokenCount:          conversation.TotalTokenCount,
		Metadata:                 sanitizeLocalConversationMetadata(conversation.Metadata),
	}
}

func sanitizeLocalConversationMetadata(metadata map[string]any) map[string]any {
	if len(metadata) == 0 {
		return nil
	}
	out := make(map[string]any, len(metadata))
	for key, value := range metadata {
		switch key {
		case "cwd", "origin_cwd", "project", "title":
			continue
		default:
			out[key] = value
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func localConversationMessageLimits(args map[string]any) (int, int, error) {
	userLimit, err := optionalNonNegativeIntDefault(args, defaultUserMessageCharLimit, "user_message_limit", "user-message-limit")
	if err != nil {
		return 0, 0, err
	}
	assistantLimit, err := optionalNonNegativeIntDefault(args, defaultAssistantMessageCharLimit, "assistant_message_limit", "assistant-message-limit")
	if err != nil {
		return 0, 0, err
	}
	return userLimit, assistantLimit, nil
}

func optionalNonNegativeIntDefault(args map[string]any, defaultValue int, keys ...string) (int, error) {
	for _, key := range keys {
		if _, ok := args[key]; !ok {
			continue
		}
		value, err := optionalNonNegativeInt(args, key)
		if err != nil {
			return 0, err
		}
		if value == -1 {
			return defaultValue, nil
		}
		return value, nil
	}
	return defaultValue, nil
}

func buildLocalConversationGetResponse(detail *localconversations.ConversationDetail, view string, userLimit, assistantLimit int) localConversationGetResponse {
	response := localConversationGetResponse{
		View:          view,
		Conversation:  mcpConversation(detail.Conversation),
		MessageCount:  len(detail.Messages),
		RawEventCount: len(detail.RawEvents),
	}
	previewMessages := previewConversationMessages(detail.Messages, userLimit, assistantLimit)
	previewDetail := *detail
	previewDetail.Messages = previewMessages

	switch view {
	case "messages":
		response.Messages = previewMessages
	case "chat":
		response.Chat = localconversations.ChatTranscript(&previewDetail)
	case "user-messages":
		response.UserMessages = localconversations.UserMessages(&previewDetail)
	case "full":
		response.Messages = previewMessages
		response.RawEvents = detail.RawEvents
	}
	return response
}

func previewConversationMessages(messages []localconversations.ConversationMessage, userLimit, assistantLimit int) []localconversations.ConversationMessage {
	out := make([]localconversations.ConversationMessage, len(messages))
	for i, message := range messages {
		out[i] = previewConversationMessage(message, userLimit, assistantLimit)
	}
	return out
}

func previewConversationMessage(message localconversations.ConversationMessage, userLimit, assistantLimit int) localconversations.ConversationMessage {
	if message.Content != nil {
		message.Content = nil
		message.ContentOmitted = true
	}
	message.Metadata = sanitizeLocalConversationMetadata(message.Metadata)

	limit := assistantLimit
	if message.Role == "user" {
		limit = userLimit
	}
	text, truncated, fullCount := truncateMessageText(message.Text, limit)
	message.Text = text
	if truncated {
		message.TextTruncated = true
		message.FullTextCharCount = fullCount
	}
	return message
}

func truncateMessageText(text string, limit int) (string, bool, int) {
	runes := []rune(text)
	if limit == 0 || len(runes) <= limit {
		return text, false, len(runes)
	}
	return string(runes[:limit]), true, len(runes)
}
