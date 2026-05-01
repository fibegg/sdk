package mcpserver

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/fibegg/sdk/fibe"
	"github.com/fibegg/sdk/internal/resourceschema"
	"github.com/mark3labs/mcp-go/mcp"
)

func (s *Server) registerMutterActionTools() {
	mutterCreateSchema, _, _, _ := resourceschema.SchemaFor("mutter", "create")
	mutterCreateInputSchema := cloneSchemaAndRemoveAgentID(mutterCreateSchema)

	s.addTool(&toolImpl{
		name:        "fibe_mutter",
		description: "[MODE:SIDEEFFECTS] Create one short mutter for an agent: a visible internal note used for progress, proof, blocker, or problem updates.",
		tier:        tierBase,
		annotations: toolAnnotations{},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			agentIDStr := os.Getenv("FIBE_AGENT_ID")
			agentID, err := strconv.ParseInt(agentIDStr, 10, 64)
			if err != nil || agentID <= 0 {
				return nil, fmt.Errorf("FIBE_AGENT_ID environment variable is missing or invalid")
			}

			// Inject ambient agent_id into payload before schema validation
			if args == nil {
				args = make(map[string]any)
			}
			args["agent_id"] = agentID

			if _, _, err := resourceschema.ValidatePayload("mutter", "create", args); err != nil {
				return nil, err
			}
			var p fibe.MutterItemParams
			if err := bindArgs(args, &p); err != nil {
				return nil, err
			}
			return c.Mutters.CreateItem(ctx, agentID, &p)
		},
	}, mcp.NewTool("fibe_mutter",
		mcp.WithDescription(`Create one short mutter for an agent.

Use this as the dedicated agent progress channel described in public/prompts/main.md. Prefer concise body text. Use type=proof for verified progress, type=problem for unexpected issues, and type=blocker when work cannot continue without Player input.`),
		withRawInputSchema(mutterCreateInputSchema),
	))

	s.addTool(&toolImpl{
		name:        "fibe_mutters_get",
		description: "[MODE:OVERSEER] Retrieve an agent's mutter stream by agent_id, with optional query/status/severity/playground filters.",
		tier:        tierOverseer,
		annotations: toolAnnotations{ReadOnly: true, Idempotent: true},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			agentIdentifier, err := requiredIdentifier(args, "agent_id", "")
			if err != nil {
				return nil, err
			}
			var p fibe.MutterListParams
			_ = bindArgs(args, &p)
			return c.Mutters.GetByAgentIdentifier(ctx, agentIdentifier, &p)
		},
	}, mcp.NewTool("fibe_mutters_get",
		mcp.WithDescription(`Retrieve an agent's mutter stream.

Pass agent_id for the agent whose mutters you want to inspect. Optional filters narrow the stream: playground_id, query, status, severity, page, and per_page. This is a transcript-style read, not a get-by-mutter-id call.`),
		withRawInputSchema(mutterGetInputSchema()),
	))
}

func mutterGetInputSchema() map[string]any {
	return map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"required":             []string{"agent_id"},
		"properties": map[string]any{
			"agent_id": map[string]any{
				"oneOf": []any{
					map[string]any{"type": "integer", "minimum": 1},
					map[string]any{"type": "string", "minLength": 1},
				},
				"description": "Agent ID or name whose mutter stream should be retrieved.",
			},
			"playground_id": map[string]any{
				"type":        "integer",
				"description": "Optional playground ID used to filter the mutter stream.",
				"minimum":     1,
			},
			"query": map[string]any{
				"type":        "string",
				"description": "Optional substring search across mutter values.",
			},
			"status": map[string]any{
				"type":        "string",
				"description": "Optional status filter.",
			},
			"severity": map[string]any{
				"type":        "string",
				"description": "Optional severity filter.",
			},
			"page": map[string]any{
				"type":        "integer",
				"description": "Page number for paginated mutter results.",
				"minimum":     1,
			},
			"per_page": map[string]any{
				"type":        "integer",
				"description": "Number of mutter results per page.",
				"minimum":     1,
			},
		},
	}
}

func cloneSchemaAndRemoveAgentID(schema any) map[string]any {
	m, ok := schema.(map[string]any)
	if !ok {
		return map[string]any{}
	}

	// Create a shallow copy of the top-level map
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = v
	}

	// Copy properties and remove agent_id
	if props, ok := out["properties"].(map[string]any); ok {
		newProps := make(map[string]any, len(props))
		for k, v := range props {
			if k != "agent_id" {
				newProps[k] = v
			}
		}
		out["properties"] = newProps
	}

	// Remove agent_id from required list
	if reqs, ok := out["required"].([]string); ok {
		newReqs := make([]string, 0, len(reqs))
		for _, r := range reqs {
			if r != "agent_id" {
				newReqs = append(newReqs, r)
			}
		}
		out["required"] = newReqs
	}

	return out
}
