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
			id, ok := argInt64(args, "agent_id")
			if !ok {
				return nil, fmt.Errorf("required field 'agent_id' not set")
			}
			var p fibe.AgentChatParams
			if err := bindArgs(args, &p); err != nil {
				return nil, err
			}
			if p.Text == "" {
				return nil, fmt.Errorf("required field 'text' not set")
			}
			return c.Agents.Chat(ctx, id, &p)
		},
	}, mcp.NewTool("fibe_agents_send_message",
		mcp.WithDescription("[MODE:OVERSEER] Send one text message to an agent runtime chat."),
		mcp.WithNumber("agent_id", mcp.Required(), mcp.Description("Agent ID")),
		mcp.WithString("text", mcp.Required(), mcp.Description("Text to send to the agent.")),
	))

	s.addTool(&toolImpl{
		name: "fibe_agents_start_chat", description: "[MODE:SIDEEFFECTS] Start or reconnect an agent runtime chat on the current Marquee.", tier: tierOverseer,
		annotations: toolAnnotations{},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			id, ok := argInt64(args, "agent_id")
			if !ok {
				return nil, fmt.Errorf("required field 'agent_id' not set")
			}
			marqueeID, err := parseMarqueeIDEnv()
			if err != nil {
				return nil, fmt.Errorf("fibe_agents_start_chat requires an implicit Marquee: %w", err)
			}
			return c.Agents.StartChat(ctx, id, marqueeID)
		},
	}, mcp.NewTool("fibe_agents_start_chat",
		mcp.WithDescription("[MODE:SIDEEFFECTS] Start or reconnect an agent runtime chat on the current Marquee from FIBE_MARQUEE_ID."),
		mcp.WithNumber("agent_id", mcp.Required(), mcp.Description("Agent ID")),
	))

	s.addTool(&toolImpl{
		name: "fibe_agents_runtime_status", description: "[MODE:OVERSEER] Check agent runtime reachability, authentication, queue, and processing state.", tier: tierOverseer,
		annotations: toolAnnotations{ReadOnly: true, Idempotent: true},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			id, ok := argInt64(args, "agent_id")
			if !ok {
				return nil, fmt.Errorf("required field 'agent_id' not set")
			}
			return c.Agents.RuntimeStatus(ctx, id)
		},
	}, mcp.NewTool("fibe_agents_runtime_status",
		mcp.WithDescription("[MODE:OVERSEER] Check agent runtime reachability, authentication, queue, and processing state."),
		mcp.WithNumber("agent_id", mcp.Required(), mcp.Description("Agent ID")),
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

// ---------- Artefacts ----------
