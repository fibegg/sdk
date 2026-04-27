package mcpserver

import (
	"context"
	"fmt"

	"github.com/fibegg/sdk/fibe"
	"github.com/mark3labs/mcp-go/mcp"
)

// registerCustomTools wires tools that don't fit the uniform CRUD helpers in
// tools_resource_mutations.go — operations with odd signatures, extra required
// string parameters, or compose-YAML payloads.
func (s *Server) registerCustomTools() {
	// ---------- fibe_playgrounds_logs ----------
	// Needs: id (int64), service (string), tail (int, optional).
	s.addTool(&toolImpl{
		name: "fibe_playgrounds_logs", description: "[MODE:DIALOG] Retrieve the consolidated service logs from a playground. Use when troubleshooting startup errors.", tier: tierBrownfield,
		annotations: toolAnnotations{ReadOnly: true, Idempotent: true},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			id, ok := argInt64(args, "playground_id")
			if !ok {
				return nil, fmt.Errorf("required field 'playground_id' not set")
			}
			service := argString(args, "service")
			if service == "" {
				return nil, fmt.Errorf("required field 'service' not set")
			}
			var tail *int
			if t, ok := argInt64(args, "tail"); ok && t > 0 {
				n := int(t)
				tail = &n
			}
			return c.Playgrounds.Logs(ctx, id, service, tail)
		},
	}, mcp.NewTool("fibe_playgrounds_logs",
		mcp.WithDescription("[MODE:DIALOG] Retrieve the consolidated service logs from a playground. Use when troubleshooting startup errors."),
		mcp.WithNumber("playground_id", mcp.Required(), mcp.Description("Playground ID")),
		mcp.WithString("service", mcp.Required(), mcp.Description("Compose service name, for example web or worker.")),
		mcp.WithNumber("tail", mcp.Description("Number of log lines to return (default: 50)")),
	))

	// ---------- fibe_repo_status ----------
	s.addTool(&toolImpl{
		name: "fibe_repo_status_check", description: "[MODE:DIALOG] Verify the system's access and view of multiple GitHub repository URLs.", tier: tierOther,
		annotations: toolAnnotations{ReadOnly: true, Idempotent: true},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			type input struct {
				GithubURLs []string `json:"github_urls"`
			}
			var in input
			if err := bindArgs(args, &in); err != nil {
				return nil, err
			}
			if len(in.GithubURLs) == 0 {
				return nil, fmt.Errorf("required field 'github_urls' not set")
			}
			return c.RepoStatus.Check(ctx, in.GithubURLs)
		},
	}, mcp.NewTool("fibe_repo_status_check",
		mcp.WithDescription("[MODE:DIALOG] Verify the system's access and view of multiple GitHub repository URLs."),
		mcp.WithArray("github_urls", mcp.Required(),
			mcp.Description("GitHub repository URLs to check."),
			mcp.WithStringItems()),
	))

	// ---------- fibe_get_github_token ----------
	s.addTool(&toolImpl{
		name: "fibe_get_github_token", description: "[MODE:SIDEEFFECTS] Get a GitHub access token for a repository. Auto-resolves the correct installation.", tier: tierOther,
		annotations: toolAnnotations{Idempotent: true},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			repo := argString(args, "repo")
			if repo == "" {
				return nil, fmt.Errorf("required field 'repo' not set")
			}
			return c.Installations.GetGitHubToken(ctx, repo)
		},
	}, mcp.NewTool("fibe_get_github_token",
		mcp.WithDescription("[MODE:SIDEEFFECTS] Get a GitHub access token for a repository. Auto-resolves the correct GitHub App installation. No installation ID needed."),
		mcp.WithString("repo", mcp.Required(), mcp.Description("Full repo name, e.g. owner/repo")),
	))
}
