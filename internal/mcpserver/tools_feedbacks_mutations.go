package mcpserver

import (
	"context"
	"fmt"
	"os"

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
			agentIdentifier := os.Getenv("FIBE_AGENT_ID")
			if agentIdentifier == "" {
				return nil, fmt.Errorf("FIBE_AGENT_ID environment variable is missing")
			}
			var p fibe.FeedbackListParams
			backendArgs := map[string]any{}
			for k, v := range args {
				if k != "playground_id_or_name" {
					backendArgs[k] = v
				}
			}
			if playgroundIdentifier := argString(args, "playground_id_or_name"); playgroundIdentifier != "" {
				backendArgs["playground_id"] = playgroundIdentifier
			}
			if err := bindArgs(backendArgs, &p); err != nil {
				return nil, err
			}
			return c.Feedbacks.ListByAgentIdentifier(ctx, agentIdentifier, &p)
		},
	}, mcp.NewTool("fibe_feedbacks_list",
		mcp.WithDescription("[MODE:OVERSEER] List all feedback entries associated with an agent."),
		withRawInputSchema(feedbackListSchema()),
	))

	s.addTool(&toolImpl{
		name:        "fibe_feedbacks_get",
		description: "[MODE:OVERSEER] Get one feedback entry for an agent, including player comments about artefacts or mutters.",
		tier:        tierBrownfield,
		annotations: toolAnnotations{ReadOnly: true, Idempotent: true},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			agentIdentifier := os.Getenv("FIBE_AGENT_ID")
			if agentIdentifier == "" {
				return nil, fmt.Errorf("FIBE_AGENT_ID environment variable is missing")
			}
			id, ok := argInt64(args, "feedback_id")
			if !ok {
				return nil, fmt.Errorf("required field 'feedback_id' not set")
			}
			return c.Feedbacks.GetByAgentIdentifier(ctx, agentIdentifier, id)
		},
	}, mcp.NewTool("fibe_feedbacks_get",
		mcp.WithDescription("[MODE:OVERSEER] Get one feedback entry for an agent, including player comments about artefacts or mutters."),
		mcp.WithNumber("feedback_id", mcp.Required(), mcp.Description(idDescription("feedback_id"))),
	))
}

func feedbackListSchema() map[string]any {
	return map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"properties": map[string]any{
			"query":                 map[string]any{"type": "string", "description": "Search across comment, selected_text, and context."},
			"source_type":           map[string]any{"type": "string", "description": "Filter by feedback source type."},
			"source_id":             map[string]any{"type": "string", "description": "Filter by source identifier."},
			"playground_id_or_name": identifierInputProperty("Optional playground ID or slug-safe name filter."),
			"created_after":         map[string]any{"type": "string", "description": "Filter to feedback created at or after this timestamp."},
			"created_before":        map[string]any{"type": "string", "description": "Filter to feedback created at or before this timestamp."},
			"sort":                  map[string]any{"type": "string", "description": "Sort order, for example created_at_desc."},
			"page":                  map[string]any{"type": "integer", "minimum": 1},
			"per_page":              map[string]any{"type": "integer", "minimum": 1},
		},
	}
}
