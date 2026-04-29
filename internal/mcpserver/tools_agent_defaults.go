package mcpserver

import (
	"context"
	"fmt"

	"github.com/fibegg/sdk/fibe"
	"github.com/mark3labs/mcp-go/mcp"
)

func (s *Server) registerAgentDefaultsTools() {
	s.addTool(&toolImpl{
		name:        "fibe_agent_defaults_get",
		description: "[MODE:DIALOG] Read the authenticated player's agent default overrides.",
		tier:        tierBase,
		annotations: toolAnnotations{ReadOnly: true, Idempotent: true},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			return c.AgentDefaults.Get(ctx)
		},
	}, mcp.NewTool("fibe_agent_defaults_get",
		mcp.WithDescription("[MODE:DIALOG] Read the authenticated player's agent default overrides."),
	))

	s.addTool(&toolImpl{
		name:        "fibe_agent_defaults_update",
		description: "[MODE:SIDEEFFECTS] Replace the authenticated player's agent default overrides. Use the same agent_defaults JSON shape as the profile UI.",
		tier:        tierBase,
		annotations: toolAnnotations{},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			raw, ok := args["agent_defaults"].(map[string]any)
			if !ok || raw == nil {
				return nil, fmt.Errorf("required field 'agent_defaults' must be an object")
			}
			return c.AgentDefaults.Update(ctx, fibe.AgentDefaults(raw))
		},
	}, mcp.NewTool("fibe_agent_defaults_update",
		mcp.WithDescription("[MODE:SIDEEFFECTS] Replace the authenticated player's agent default overrides. Use the same agent_defaults JSON shape as the profile UI."),
		mcp.WithObject("agent_defaults", mcp.Required(), mcp.Description("Player agent defaults object, including provider_overrides when needed.")),
	))

	s.addTool(&toolImpl{
		name:        "fibe_agent_defaults_reset",
		description: "[MODE:SIDEEFFECTS] Clear all player agent default overrides so admin defaults apply.",
		tier:        tierBase,
		annotations: toolAnnotations{},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			return c.AgentDefaults.Reset(ctx)
		},
	}, mcp.NewTool("fibe_agent_defaults_reset",
		mcp.WithDescription("[MODE:SIDEEFFECTS] Clear all player agent default overrides so admin defaults apply."),
	))
}
