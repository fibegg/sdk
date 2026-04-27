package mcpserver

import (
	"context"

	"github.com/fibegg/sdk/fibe"
	"github.com/mark3labs/mcp-go/mcp"
)

func (s *Server) registerInstallationActionTools() {
	// fibe_find_github_repos — aggregated search across all installations
	s.addTool(&toolImpl{
		name: "fibe_find_github_repos", description: "[MODE:DIALOG] Search GitHub repositories across all connected installations. Returns deduplicated results.", tier: tierOther,
		annotations: toolAnnotations{ReadOnly: true, Idempotent: true},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			var p fibe.InstallationReposParams
			_ = bindArgs(args, &p)
			return c.Installations.FindGitHubRepos(ctx, &p)
		},
	}, mcp.NewTool("fibe_find_github_repos",
		mcp.WithDescription("[MODE:DIALOG] Search GitHub repositories across all connected installations. Returns deduplicated results. No installation ID needed."),
		mcp.WithString("q", mcp.Description("Search query (filters by repo name). Optional; omit to list all accessible repos.")),
		mcp.WithNumber("page", mcp.Description("Page number (default: 1)")),
		mcp.WithNumber("per_page", mcp.Description("Results per page (default: 30, max: 100)")),
	))
}

// ---------- Marquees ----------
