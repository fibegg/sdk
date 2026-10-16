package mcpserver

import (
	"context"

	"github.com/fibegg/sdk/fibe"
	"github.com/mark3labs/mcp-go/mcp"
)

func (s *Server) registerInstallationActionTools() {
	s.addTool(&toolImpl{
		name: "fibe_find_github_repos", description: "[MODE:DIALOG] Search Personal installations or a selected verified Team installation.", tier: tierOther,
		annotations: toolAnnotations{ReadOnly: true, Idempotent: true},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			var p fibe.InstallationReposParams
			_ = bindArgs(args, &p)
			return c.Installations.FindGitHubRepos(ctx, &p)
		},
	}, mcp.NewTool("fibe_find_github_repos",
		mcp.WithDescription("[MODE:DIALOG] Search Personal installations or a selected verified Team installation. Select an installation when the Team has multiple."),
		mcp.WithString("q", mcp.Description("Search query (filters by repo name). Optional; omit to list all accessible repos.")),
		mcp.WithNumber("github_installation_id", mcp.Description("Verified Team GitHub installation; required when multiple are connected.")),
		mcp.WithNumber("page", mcp.Description("Page number (default: 1)")),
		mcp.WithNumber("per_page", mcp.Description("Results per page (default: 30, max: 100)")),
	))
}
