package mcpserver

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/fibegg/sdk/fibe"
	"github.com/mark3labs/mcp-go/mcp"
)

func (s *Server) registerFeedbackMutationTools() {
	s.addTool(&toolImpl{
		name:        "fibe_feedbacks_list",
		description: "[MODE:OVERSEER] List all feedback entries associated with an agent.",
		tier:        tierBrownfield,
		annotations: toolAnnotations{ReadOnly: true, Idempotent: true},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			agentIDStr := os.Getenv("FIBE_AGENT_ID")
			agentID, err := strconv.ParseInt(agentIDStr, 10, 64)
			if err != nil || agentID <= 0 {
				return nil, fmt.Errorf("FIBE_AGENT_ID environment variable is missing or invalid")
			}
			var p fibe.FeedbackListParams
			if err := bindArgs(args, &p); err != nil {
				return nil, err
			}
			return c.Feedbacks.List(ctx, agentID, &p)
		},
	}, mcp.NewTool("fibe_feedbacks_list",
		mcp.WithDescription("[MODE:OVERSEER] List all feedback entries associated with an agent."),
		mcp.WithInputSchema[fibe.FeedbackListParams](),
	))

	s.addTool(&toolImpl{
		name:        "fibe_feedbacks_get",
		description: "[MODE:OVERSEER] Get one feedback entry for an agent, including player comments about artefacts or mutters.",
		tier:        tierBrownfield,
		annotations: toolAnnotations{ReadOnly: true, Idempotent: true},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			agentIDStr := os.Getenv("FIBE_AGENT_ID")
			agentID, err := strconv.ParseInt(agentIDStr, 10, 64)
			if err != nil || agentID <= 0 {
				return nil, fmt.Errorf("FIBE_AGENT_ID environment variable is missing or invalid")
			}
			id, ok := argInt64(args, "feedback_id")
			if !ok {
				return nil, fmt.Errorf("required field 'feedback_id' not set")
			}
			return c.Feedbacks.Get(ctx, agentID, id)
		},
	}, mcp.NewTool("fibe_feedbacks_get",
		mcp.WithDescription("[MODE:OVERSEER] Get one feedback entry for an agent, including player comments about artefacts or mutters."),
		mcp.WithNumber("feedback_id", mcp.Required(), mcp.Description(idDescription("feedback_id"))),
	))
}
