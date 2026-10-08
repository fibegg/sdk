package mcpserver

import (
	"context"
	"github.com/fibegg/sdk/fibe"
	"github.com/fibegg/sdk/internal/resourceschema"
	"github.com/mark3labs/mcp-go/mcp"
)

func (s *Server) registerEnvPackTools() {
	schema, _, _, _ := resourceschema.SchemaFor("env_pack", "attachments_get")
	s.addTool(&toolImpl{name: "fibe_env_pack_attachments_get", description: "[MODE:DIALOG] Read ordered ENV-pack references, grant status, and desired/applied provenance for a Playground or Spec.", tier: tierBase, annotations: toolAnnotations{ReadOnly: true, Idempotent: true}, handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
		if _, _, err := resourceschema.ValidatePayload("env_pack", "attachments_get", args); err != nil {
			return nil, err
		}
		id, err := requiredPositiveID(args, "target_id")
		if err != nil {
			return nil, err
		}
		return c.EnvPacks.Attachments(ctx, argString(args, "target"), id)
	}}, mcp.NewTool("fibe_env_pack_attachments_get", withRawInputSchema(schema.(map[string]any))))
}
