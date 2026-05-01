package mcpserver

import (
	"context"

	"github.com/fibegg/sdk/fibe"
	"github.com/mark3labs/mcp-go/mcp"
)

func (s *Server) registerAgentMutationTools() {
	s.addTool(&toolImpl{
		name:        "fibe_agents_duplicate",
		description: "[MODE:OVERSEER] Duplicate an agent configuration.",
		tier:        tierOverseer,
		annotations: toolAnnotations{Idempotent: true},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			identifier, err := requiredIdentifier(args, "agent_id", "")
			if err != nil {
				return nil, err
			}
			return c.Agents.DuplicateByIdentifier(ctx, identifier)
		},
	}, mcp.NewTool("fibe_agents_duplicate",
		mcp.WithDescription("[MODE:OVERSEER] Duplicate an agent configuration."),
		withRawInputSchema(agentIdentifierOnlyInputSchema("Agent ID or name.")),
	))
}
