package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/fibegg/sdk/fibe"
	"github.com/fibegg/sdk/internal/localconversations"
	"github.com/fibegg/sdk/internal/resourceschema"
	"github.com/mark3labs/mcp-go/mcp"
)

func (s *Server) registerMemoryTools() {
	s.addTool(&toolImpl{
		name:        "fibe_memorize",
		description: "[MODE:SIDEEFFECTS] Create or update agent-generated memories grounded in one local source conversation.",
		tier:        tierBase,
		annotations: toolAnnotations{Destructive: false, Idempotent: true},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			if _, _, err := resourceschema.ValidatePayload("memory", "memorize", args); err != nil {
				return nil, err
			}
			memory := memoryPayloadFromArgs(args)
			if err := applyDefaultMemoryAgentID(memory); err != nil {
				return nil, err
			}

			conversationID := strings.TrimSpace(argString(args, "conversation_id"))
			detail, err := localconversations.Get(ctx, conversationID, localconversations.ListOptions{})
			if err != nil {
				return nil, err
			}

			payload := map[string]any{
				"conversation": localConversationArchivePayload(detail),
				"memory":       memory,
				"source":       "mcp",
			}
			return c.Memories.Memorize(ctx, payload)
		},
	}, mcp.NewTool("fibe_memorize",
		mcp.WithDescription(`[MODE:SIDEEFFECTS] Create or update one memory from a local source conversation.

Use after fibe_local_conversations_get. Pass conversation_id plus memory content, tags, confidence, and grounding proof. The SDK validates the local conversation exists, attaches the latest conversation snapshot internally, and Rails upserts that conversation archive before storing the memory. Use fibe_schema(resource:"memory", operation:"memorize") for the exact payload shape. Search, get, and delete memories through fibe_resource_list/get/delete with resource:"memory".`),
		withRawInputSchema(memoryMemorizeSchema()),
	))
}

func memoryPayloadFromArgs(args map[string]any) map[string]any {
	memory := map[string]any{}
	for _, key := range []string{"content", "agent_id", "tags", "confidence", "memory_key", "metadata", "groundings"} {
		if value, ok := args[key]; ok {
			memory[key] = value
		}
	}
	return memory
}

func applyDefaultMemoryAgentID(memory map[string]any) error {
	rawAgentID := os.Getenv("FIBE_AGENT_ID")
	if rawAgentID == "" {
		return nil
	}
	agentID, err := strconv.ParseInt(rawAgentID, 10, 64)
	if err != nil || agentID <= 0 {
		return fmt.Errorf("FIBE_AGENT_ID must be a positive integer")
	}

	if _, exists := memory["agent_id"]; !exists {
		memory["agent_id"] = agentID
	}
	return nil
}

func memoryMemorizeSchema() map[string]any {
	return resourceschema.MemoryMemorizeSchema()
}

func localConversationArchivePayload(detail *localconversations.ConversationDetail) map[string]any {
	if detail == nil {
		return nil
	}
	messages := make([]map[string]any, 0, len(detail.Messages))
	for _, message := range detail.Messages {
		messages = append(messages, localConversationMessagePayload(message))
	}
	metadata := map[string]any{}
	for key, value := range detail.Metadata {
		metadata[key] = value
	}
	if detail.Title != "" {
		metadata["title"] = detail.Title
	}
	payload := map[string]any{
		"provider":                    detail.Provider,
		"uuid":                        detail.UUID,
		"project":                     detail.Project,
		"path":                        detail.Path,
		"title":                       detail.Title,
		"last_message_at":             detail.LastMessageDate,
		"first_user_message_sentence": detail.FirstUserMessageSentence,
		"user_message_count":          detail.UserMessageCount,
		"total_token_count":           detail.TotalTokenCount,
		"message_count":               len(detail.Messages),
		"raw_event_count":             len(detail.RawEvents),
		"messages_complete":           true,
		"raw_events_complete":         true,
		"messages":                    messages,
		"raw_events":                  detail.RawEvents,
		"raw_content":                 localConversationRawContent(detail),
		"metadata":                    metadata,
	}
	return payload
}

func localConversationMessagePayload(message localconversations.ConversationMessage) map[string]any {
	return map[string]any{
		"position":              message.Position,
		"role":                  message.Role,
		"type":                  message.Type,
		"uuid":                  message.UUID,
		"provider_message_uuid": message.UUID,
		"timestamp":             message.Timestamp,
		"text":                  message.Text,
		"content":               message.Content,
		"token_count":           message.TokenCount,
		"metadata":              message.Metadata,
	}
}

func localConversationRawContent(detail *localconversations.ConversationDetail) string {
	data, err := json.Marshal(detail)
	if err != nil {
		return ""
	}
	return string(data)
}
