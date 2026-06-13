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

}

// ---------- Job ENV ----------
