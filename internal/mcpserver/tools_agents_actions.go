package mcpserver

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/fibegg/sdk/fibe"
	"github.com/mark3labs/mcp-go/mcp"
)

func (s *Server) registerAgentActionTools() {
	// chat
	s.addTool(&toolImpl{
		name: "fibe_agents_send_message", description: "[MODE:OVERSEER] Send one text message to an agent runtime chat.", tier: tierOverseer,
		annotations: toolAnnotations{},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			identifier, err := requiredIdentifier(args, "agent_id", "")
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
			return c.Agents.ChatByIdentifier(ctx, identifier, &p)
		},
	}, mcp.NewTool("fibe_agents_send_message",
		mcp.WithDescription("[MODE:OVERSEER] Send one text message to an agent runtime chat."),
		withRawInputSchema(agentSendMessageInputSchema()),
	))

	s.addTool(&toolImpl{
		name: "fibe_agents_start_chat", description: "[MODE:SIDEEFFECTS] Start or reconnect an agent runtime chat on the current Marquee.", tier: tierOverseer,
		annotations: toolAnnotations{},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			identifier, err := requiredIdentifier(args, "agent_id", "")
			if err != nil {
				return nil, err
			}
			marqueeID, err := parseMarqueeIDEnv()
			if err != nil {
				return nil, fmt.Errorf("fibe_agents_start_chat requires an implicit Marquee: %w", err)
			}
			return c.Agents.StartChatByAgentIdentifier(ctx, identifier, strconv.FormatInt(marqueeID, 10))
		},
	}, mcp.NewTool("fibe_agents_start_chat",
		mcp.WithDescription("[MODE:SIDEEFFECTS] Start or reconnect an agent runtime chat on the current Marquee from FIBE_MARQUEE_ID."),
		withRawInputSchema(agentIdentifierOnlyInputSchema("Agent ID or name.")),
	))

	s.addTool(&toolImpl{
		name: "fibe_agents_runtime_status", description: "[MODE:OVERSEER] Check agent runtime reachability, authentication, queue, and processing state.", tier: tierOverseer,
		annotations: toolAnnotations{ReadOnly: true, Idempotent: true},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			identifier, err := requiredIdentifier(args, "agent_id", "")
			if err != nil {
				return nil, err
			}
			return c.Agents.RuntimeStatusByIdentifier(ctx, identifier)
		},
	}, mcp.NewTool("fibe_agents_runtime_status",
		mcp.WithDescription("[MODE:OVERSEER] Check agent runtime reachability, authentication, queue, and processing state."),
		withRawInputSchema(agentIdentifierOnlyInputSchema("Agent ID or name.")),
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
			return c.Agents.Update(ctx, agentID, &fibe.AgentUpdateParams{
				Name: &name,
			})
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
		"required":             []string{"agent_id"},
		"properties": map[string]any{
			"agent_id": identifierInputProperty(description),
		},
	}
}

func agentSendMessageInputSchema() map[string]any {
	schema := agentIdentifierOnlyInputSchema("Agent ID or name.")
	props := schema["properties"].(map[string]any)
	props["text"] = map[string]any{
		"type":        "string",
		"minLength":   1,
		"description": "Text to send to the agent.",
	}
	schema["required"] = []string{"agent_id", "text"}
	return schema
}

// ---------- Artefacts ----------
