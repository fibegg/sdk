package mcpserver

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/fibegg/sdk/fibe"
	"github.com/mark3labs/mcp-go/mcp"
)

func (s *Server) registerAgentActionTools() {
	s.addTool(&toolImpl{
		name: "fibe_agents_send_message", description: "[MODE:OVERSEER] Send one text message to an agent chat. Fails with HOST_NOT_FUNDED when the chat Host is unpaid.", tier: tierOverseer,
		annotations: toolAnnotations{},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			identifier, err := requiredIdentifier(args, "id_or_name", "")
			if err != nil {
				return nil, err
			}
			var p fibe.AgentChatParams
			if err := bindArgs(args, &p); err != nil {
				return nil, err
			}
			if p.Text == "" {
				return nil, fmt.Errorf("required field 'text' not set")
			}
			for _, path := range argStringList(args, "attachment_paths") {
				upload, err := c.Agents.UploadByIdentifier(ctx, identifier, &fibe.AgentUploadParams{
					FilePath:       path,
					ConversationID: p.ConversationID,
				})
				if err != nil {
					return nil, err
				}
				if upload.Filename != "" {
					p.AttachmentFilenames = append(p.AttachmentFilenames, upload.Filename)
				}
			}
			return c.Agents.ChatByIdentifier(ctx, identifier, &p)
		},
	}, mcp.NewTool("fibe_agents_send_message",
		mcp.WithDescription("[MODE:OVERSEER] Send one text message to an agent chat, optionally uploading local attachments first. Fails with HOST_NOT_FUNDED when the chat Host is unpaid."),
		withRawInputSchema(agentSendMessageInputSchema()),
	))

	s.addTool(&toolImpl{
		name: "fibe_agents_start_chat", description: "[MODE:SIDEEFFECTS] Start or reconnect an agent chat on the current Host. Requires a funded Host; unpaid Hosts fail with HOST_NOT_FUNDED.", tier: tierOverseer,
		annotations: toolAnnotations{},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			identifier, err := requiredIdentifier(args, "id_or_name", "")
			if err != nil {
				return nil, err
			}
			hostID, err := parseHostIDEnv()
			if err != nil {
				return nil, fmt.Errorf("fibe_agents_start_chat requires an implicit Host: %w", err)
			}
			return c.Agents.StartChatByAgentIdentifier(ctx, identifier, strconv.FormatInt(hostID, 10))
		},
	}, mcp.NewTool("fibe_agents_start_chat",
		mcp.WithDescription("[MODE:SIDEEFFECTS] Start or reconnect an agent chat on the current Host from FIBE_HOST_ID. Requires a funded Host; unpaid Hosts fail with HOST_NOT_FUNDED."),
		withRawInputSchema(agentIdentifierOnlyInputSchema("Agent ID or name.")),
	))

	s.addTool(&toolImpl{
		name: "fibe_agents_runtime_status", description: "[MODE:OVERSEER] Check agent reachability, authentication, queue, and processing state. Live checks fail with HOST_NOT_FUNDED when unpaid.", tier: tierOverseer,
		annotations: toolAnnotations{ReadOnly: true, Idempotent: true},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			identifier, err := requiredIdentifier(args, "id_or_name", "")
			if err != nil {
				return nil, err
			}
			return c.Agents.RuntimeStatusByIdentifier(ctx, identifier)
		},
	}, mcp.NewTool("fibe_agents_runtime_status",
		mcp.WithDescription("[MODE:OVERSEER] Check agent reachability, authentication, queue, and processing state. Live checks fail with HOST_NOT_FUNDED when unpaid."),
		withRawInputSchema(agentIdentifierOnlyInputSchema("Agent ID or name.")),
	))

	s.addTool(&toolImpl{
		name: "fibe_agents_live_state", description: "[MODE:OVERSEER] Check conversation-scoped agent live stream state.", tier: tierOverseer,
		annotations: toolAnnotations{ReadOnly: true, Idempotent: true},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			identifier, err := requiredIdentifier(args, "id_or_name", "")
			if err != nil {
				return nil, err
			}
			return c.Agents.LiveStateByIdentifier(ctx, identifier, &fibe.AgentDataParams{ConversationID: argString(args, "conversation_id")})
		},
	}, mcp.NewTool("fibe_agents_live_state",
		mcp.WithDescription("[MODE:OVERSEER] Check conversation-scoped agent live stream state."),
		withRawInputSchema(agentConversationInputSchema("Agent ID or name.", false)),
	))

	s.addTool(&toolImpl{
		name: "fibe_agents_messages", description: "[MODE:OVERSEER] Read agent messages, optionally scoped to a conversation.", tier: tierOverseer,
		annotations: toolAnnotations{ReadOnly: true, Idempotent: true},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			identifier, err := requiredIdentifier(args, "id_or_name", "")
			if err != nil {
				return nil, err
			}
			return c.Agents.GetMessagesByIdentifierWithParams(ctx, identifier, &fibe.AgentDataParams{ConversationID: argString(args, "conversation_id")})
		},
	}, mcp.NewTool("fibe_agents_messages",
		mcp.WithDescription("[MODE:OVERSEER] Read agent messages, optionally scoped to a conversation."),
		withRawInputSchema(agentConversationInputSchema("Agent ID or name.", false)),
	))

	s.addTool(&toolImpl{
		name: "fibe_agents_activity", description: "[MODE:OVERSEER] Read agent activity, optionally scoped to a conversation.", tier: tierOverseer,
		annotations: toolAnnotations{ReadOnly: true, Idempotent: true},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			identifier, err := requiredIdentifier(args, "id_or_name", "")
			if err != nil {
				return nil, err
			}
			return c.Agents.GetActivityByIdentifierWithParams(ctx, identifier, &fibe.AgentDataParams{ConversationID: argString(args, "conversation_id")})
		},
	}, mcp.NewTool("fibe_agents_activity",
		mcp.WithDescription("[MODE:OVERSEER] Read agent activity, optionally scoped to a conversation."),
		withRawInputSchema(agentConversationInputSchema("Agent ID or name.", false)),
	))

	s.addTool(&toolImpl{
		name: "fibe_agents_create_conversation", description: "[MODE:SIDEEFFECTS] Create or upsert an agent conversation.", tier: tierOverseer,
		annotations: toolAnnotations{},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			identifier, err := requiredIdentifier(args, "id_or_name", "")
			if err != nil {
				return nil, err
			}
			return c.Agents.CreateConversationByIdentifier(ctx, identifier, &fibe.AgentConversationParams{
				ConversationID: argString(args, "conversation_id"),
				Title:          argString(args, "title"),
			})
		},
	}, mcp.NewTool("fibe_agents_create_conversation",
		mcp.WithDescription("[MODE:SIDEEFFECTS] Create or upsert an agent conversation."),
		withRawInputSchema(agentCreateConversationInputSchema()),
	))

	s.addTool(&toolImpl{
		name: "fibe_agents_delete_conversation", description: "[MODE:SIDEEFFECTS] Delete an agent conversation.", tier: tierOverseer,
		annotations: toolAnnotations{Destructive: true},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			identifier, err := requiredIdentifier(args, "id_or_name", "")
			if err != nil {
				return nil, err
			}
			if err := c.Agents.DeleteConversationByIdentifier(ctx, identifier, argString(args, "conversation_id")); err != nil {
				return nil, err
			}
			return map[string]any{"deleted": true, "conversation_id": argString(args, "conversation_id")}, nil
		},
	}, mcp.NewTool("fibe_agents_delete_conversation",
		mcp.WithDescription("[MODE:SIDEEFFECTS] Delete an agent conversation."),
		withRawInputSchema(agentConversationInputSchema("Agent ID or name.", true)),
	))

	s.addTool(&toolImpl{
		name: "fibe_agents_interrupt", description: "[MODE:SIDEEFFECTS] Interrupt a running agent turn.", tier: tierOverseer,
		annotations: toolAnnotations{},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			identifier, err := requiredIdentifier(args, "id_or_name", "")
			if err != nil {
				return nil, err
			}
			return c.Agents.InterruptByIdentifier(ctx, identifier, &fibe.AgentConversationParams{ConversationID: argString(args, "conversation_id")})
		},
	}, mcp.NewTool("fibe_agents_interrupt",
		mcp.WithDescription("[MODE:SIDEEFFECTS] Interrupt a running agent turn."),
		withRawInputSchema(agentConversationInputSchema("Agent ID or name.", false)),
	))

	s.addTool(&toolImpl{
		name: "fibe_update_name", description: "[MODE:DIALOG] Update your own agent name.", tier: tierBase,
		annotations: toolAnnotations{Idempotent: false},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			agentIDStr := os.Getenv("FIBE_AGENT_ID")
			agentID, err := strconv.ParseInt(agentIDStr, 10, 64)
			if err != nil || agentID <= 0 {
				return nil, fmt.Errorf("FIBE_AGENT_ID environment variable is missing or invalid")
			}
			name := argString(args, "name")
			if name == "" {
				return nil, fmt.Errorf("required field 'name' not set or empty")
			}
			params := &fibe.AgentUpdateParams{
				Name: &name,
			}
			if conversationID := strings.TrimSpace(os.Getenv("CONVERSATION_ID")); conversationID != "" {
				params.RenameContext = &fibe.AgentRenameContext{
					ConversationClientID: conversationID,
				}
			}
			return c.Agents.Update(ctx, agentID, params)
		},
	}, mcp.NewTool("fibe_update_name",
		mcp.WithDescription("[MODE:DIALOG] Update your own agent name. Use this to update your name when conversation topic changes significantly."),
		mcp.WithString("name", mcp.Required(), mcp.Description("Your new name")),
	))
}

func agentIdentifierOnlyInputSchema(description string) map[string]any {
	return map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"required":             []string{"id_or_name"},
		"properties": map[string]any{
			"id_or_name": identifierInputProperty(description),
		},
	}
}

func agentConversationInputSchema(description string, requireConversation bool) map[string]any {
	schema := agentIdentifierOnlyInputSchema(description)
	repositories := schema["properties"].(map[string]any)
	repositories["conversation_id"] = map[string]any{
		"type":        "string",
		"description": "Specific conversation/thread ID.",
	}
	if requireConversation {
		schema["required"] = []string{"id_or_name", "conversation_id"}
	}
	return schema
}

func agentCreateConversationInputSchema() map[string]any {
	schema := agentConversationInputSchema("Agent ID or name.", true)
	repositories := schema["properties"].(map[string]any)
	repositories["title"] = map[string]any{
		"type":        "string",
		"description": "Human-readable conversation title. Optional.",
	}
	return schema
}

func agentSendMessageInputSchema() map[string]any {
	schema := agentIdentifierOnlyInputSchema("Agent ID or name.")
	repositories := schema["properties"].(map[string]any)
	repositories["text"] = map[string]any{
		"type":        "string",
		"minLength":   1,
		"description": "Text to send to the agent.",
	}
	repositories["conversation_id"] = map[string]any{
		"type":        "string",
		"description": "Specific conversation/thread ID. Optional.",
	}
	repositories["busy_policy"] = map[string]any{
		"type":        "string",
		"description": "Runtime busy behavior, e.g. queue. Optional.",
	}
	repositories["images"] = map[string]any{
		"type":        "array",
		"items":       map[string]any{"type": "string"},
		"description": "Image payloads to send to the runtime, such as data URLs. Optional.",
	}
	repositories["attachment_paths"] = map[string]any{
		"type":        "array",
		"items":       map[string]any{"type": "string"},
		"description": "Local file paths to upload before sending. Optional.",
	}
	repositories["attachment_filenames"] = map[string]any{
		"type":        "array",
		"items":       map[string]any{"type": "string"},
		"description": "Runtime attachment filenames returned by a previous upload. Optional.",
	}
	schema["required"] = []string{"id_or_name", "text"}
	return schema
}

func argStringList(args map[string]any, key string) []string {
	v, ok := args[key]
	if !ok {
		return nil
	}
	switch x := v.(type) {
	case []string:
		return x
	case []any:
		out := make([]string, 0, len(x))
		for _, item := range x {
			if s, ok := item.(string); ok && strings.TrimSpace(s) != "" {
				out = append(out, s)
			}
		}
		return out
	case string:
		if strings.TrimSpace(x) == "" {
			return nil
		}
		return []string{x}
	default:
		return nil
	}
}
