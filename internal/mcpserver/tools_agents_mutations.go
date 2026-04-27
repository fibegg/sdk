package mcpserver

import (
	"context"

	"github.com/fibegg/sdk/fibe"
)

func (s *Server) registerAgentMutationTools() {
	registerIDAction(s, "fibe_agents_duplicate", "[MODE:OVERSEER] Duplicate an agent configuration.", toolOpts{Tier: tierOverseer, IDField: "agent_id"},
		func(ctx context.Context, c *fibe.Client, id int64) (*fibe.Agent, error) {
			return c.Agents.Duplicate(ctx, id)
		})
}
