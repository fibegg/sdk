package mcpserver

import (
	"context"
	"fmt"

	"github.com/fibegg/sdk/fibe"
	"github.com/mark3labs/mcp-go/mcp"
)

func (s *Server) registerImportTemplateActionTools() {
	s.addTool(&toolImpl{
		name: "fibe_templates_search", description: "[MODE:GREENFIELD] Search the import-template catalog by text or PostgreSQL regex. Regex mode requires a 3+ character literal token for indexed prefiltering.", tier: tierGreenfield,
		annotations: toolAnnotations{ReadOnly: true, Idempotent: true},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			q := argString(args, "query")
			var tmplID *int64
			if v, ok := argInt64(args, "template_id"); ok {
				tmplID = &v
			}
			return c.ImportTemplates.SearchWithParams(ctx, &fibe.ImportTemplateSearchParams{
				Query:      q,
				TemplateID: tmplID,
				Regex:      argBool(args, "regex"),
			})
		},
	}, mcp.NewTool("fibe_templates_search",
		mcp.WithDescription("[MODE:GREENFIELD] Search the import-template catalog by text or PostgreSQL regex. Regex mode requires a 3+ character literal token for indexed prefiltering."),
		mcp.WithString("query", mcp.Description("Search query. In regex mode, this is a PostgreSQL regex pattern.")),
		mcp.WithNumber("template_id", mcp.Description("Optional template ID filter")),
		mcp.WithBoolean("regex", mcp.Description("Treat query as PostgreSQL regex. Requires a 3+ character literal token so the server can prefilter with indexed text search.")),
	))

	s.addTool(&toolImpl{
		name: "fibe_templates_launch", description: "[MODE:GREENFIELD] Bootstrap and launch a new playground directly from an import template.", tier: tierGreenfield,
		annotations: toolAnnotations{Idempotent: true},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			id, ok := argInt64(args, "template_id")
			if !ok {
				return nil, fmt.Errorf("required field 'template_id' not set")
			}
			var p fibe.ImportTemplateLaunchParams
			if err := bindArgs(args, &p); err != nil {
				return nil, err
			}
			if p.MarqueeID == 0 {
				envID, err := parseMarqueeIDEnv()
				if err != nil {
					return nil, fmt.Errorf("marquee_id is required either in payload or via FIBE_MARQUEE_ID env var: %w", err)
				}
				p.MarqueeID = envID
			}
			return c.ImportTemplates.LaunchWithParams(ctx, id, &p)
		},
	}, mcp.NewTool("fibe_templates_launch",
		mcp.WithDescription("[MODE:GREENFIELD] Bootstrap and launch a new playground directly from an import template."),
		mcp.WithNumber("template_id", mcp.Required(), mcp.Description("Template ID")),
		mcp.WithNumber("marquee_id", mcp.Description("Target marquee ID. Optional; defaults to FIBE_MARQUEE_ID.")),
		mcp.WithString("name", mcp.Description("Optional playground name override")),
		mcp.WithNumber("version", mcp.Description("Optional template version to launch")),
		mcp.WithObject("variables", mcp.Description("Template variables used while rendering the selected template version.")),
		mcp.WithObject("env_overrides", mcp.Description("Environment variable overrides for the launched playground.")),
		mcp.WithObject("service_subdomains", mcp.Description("Per-service subdomain overrides for exposed services.")),
		mcp.WithObject("services", mcp.Description("Per-service launch configuration overrides.")),
	))

}

// ---------- Job ENV ----------
