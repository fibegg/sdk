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
			templateIdentifier := argString(args, "template_id_or_name")
			if templateIdentifier == "" {
				if v, ok := argInt64(args, "template_id_or_name"); ok {
					formatted := fmt.Sprintf("%d", v)
					templateIdentifier = formatted
					tmplID = &v
				}
			}
			return c.ImportTemplates.SearchWithParams(ctx, &fibe.ImportTemplateSearchParams{
				Query:              q,
				TemplateID:         tmplID,
				TemplateIdentifier: templateIdentifier,
				Regex:              argBool(args, "regex"),
			})
		},
	}, mcp.NewTool("fibe_templates_search",
		mcp.WithDescription("[MODE:GREENFIELD] Search the import-template catalog by text or PostgreSQL regex. Regex mode requires a 3+ character literal token for indexed prefiltering."),
		mcp.WithString("query", mcp.Description("Search query. In regex mode, this is a PostgreSQL regex pattern.")),
		mcp.WithString("template_id_or_name", mcp.Description("Optional template ID or name filter")),
		mcp.WithBoolean("regex", mcp.Description("Treat query as PostgreSQL regex. Requires a 3+ character literal token so the server can prefilter with indexed text search.")),
	))

	s.addTool(&toolImpl{
		name: "fibe_templates_launch", description: "[MODE:GREENFIELD] Bootstrap and launch a new playground directly from an import template. Target Marquee must be funded or the server returns MARQUEE_NOT_FUNDED.", tier: tierGreenfield,
		annotations: toolAnnotations{Idempotent: true},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			identifier, err := requiredIdentifier(args, "template_id_or_name", "")
			if err != nil {
				return nil, err
			}
			var p fibe.ImportTemplateLaunchParams
			if err := bindArgs(args, &p); err != nil {
				return nil, err
			}
			if marqueeIdentifier := argString(args, "marquee_id_or_name"); marqueeIdentifier != "" {
				p.MarqueeIdentifier = marqueeIdentifier
			}
			if p.MarqueeID == 0 && p.MarqueeIdentifier == "" {
				envID, err := parseMarqueeIDEnv()
				if err != nil {
					return nil, fmt.Errorf("marquee_id is required either in payload or via FIBE_MARQUEE_ID env var: %w", err)
				}
				p.MarqueeID = envID
			}
			return c.ImportTemplates.LaunchWithParamsByIdentifier(ctx, identifier, &p)
		},
	}, mcp.NewTool("fibe_templates_launch",
		mcp.WithDescription("[MODE:GREENFIELD] Bootstrap and launch a new playground directly from an import template. Target Marquee must be funded or the server returns MARQUEE_NOT_FUNDED."),
		mcp.WithString("template_id_or_name", mcp.Required(), mcp.Description("Template ID or name")),
		mcp.WithString("marquee_id_or_name", mcp.Description("Target marquee ID or name. Optional; defaults to FIBE_MARQUEE_ID.")),
		mcp.WithString("name", mcp.Description("Optional playground name override")),
		mcp.WithNumber("version", mcp.Description("Optional template version to launch")),
		mcp.WithObject("variables", mcp.Description("Template variables used while rendering the selected template version.")),
		mcp.WithObject("env_overrides", mcp.Description("Environment variable overrides for the launched playground.")),
		mcp.WithObject("service_subdomains", mcp.Description("Per-service subdomain overrides for exposed services.")),
		mcp.WithObject("services", mcp.Description("Per-service launch configuration overrides.")),
	))

}

// ---------- Job ENV ----------
