package mcpserver

import (
	"context"
	"fmt"

	"github.com/fibegg/sdk/fibe"
	"github.com/mark3labs/mcp-go/mcp"
)

// registerCustomTools wires tools that don't fit the uniform CRUD helpers in
// tools_generated.go — operations with odd signatures, extra required
// string parameters, or compose-YAML payloads.
func (s *Server) registerCustomTools() {
	// ---------- fibe_playgrounds_logs ----------
	// Needs: id (int64), service (string), tail (int, optional).
	s.addTool(&toolImpl{
		name: "fibe_playgrounds_logs", description: "[MODE:DIALOG] Retrieve the consolidated service logs from a playground. Use when troubleshooting startup errors.", tier: tierCore,
		annotations: toolAnnotations{ReadOnly: true, Idempotent: true},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			id, ok := argInt64(args, "id")
			if !ok {
				return nil, fmt.Errorf("required field 'id' not set")
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
		mcp.WithNumber("id", mcp.Required(), mcp.Description("Playground ID")),
		mcp.WithString("service", mcp.Required(), mcp.Description("Service name")),
		mcp.WithNumber("tail", mcp.Description("Number of log lines to return (default: 50)")),
	))

	// ---------- fibe_tricks_logs ----------
	s.addTool(&toolImpl{
		name: "fibe_tricks_logs", description: "[MODE:DIALOG] Retrieve the execution and background service logs from a trick.", tier: tierFull,
		annotations: toolAnnotations{ReadOnly: true, Idempotent: true},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			id, ok := argInt64(args, "id")
			if !ok {
				return nil, fmt.Errorf("required field 'id' not set")
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
			return c.Tricks.Logs(ctx, id, service, tail)
		},
	}, mcp.NewTool("fibe_tricks_logs",
		mcp.WithDescription("[MODE:DIALOG] Retrieve the execution and background service logs from a trick."),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("Trick ID")),
		mcp.WithString("service", mcp.Required(), mcp.Description("Service name")),
		mcp.WithNumber("tail", mcp.Description("Number of log lines to return (default: 50)")),
	))

	// ---------- fibe_playgrounds_extend ----------
	s.addTool(&toolImpl{
		name: "fibe_playgrounds_extend", description: "[MODE:SIDEEFFECTS] Extend the operational expiration time of an active playground.", tier: tierFull,
		annotations: toolAnnotations{Idempotent: true},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			id, ok := argInt64(args, "id")
			if !ok {
				return nil, fmt.Errorf("required field 'id' not set")
			}
			var hours *int
			if h, ok := argInt64(args, "hours"); ok && h > 0 {
				n := int(h)
				hours = &n
			}
			return c.Playgrounds.ExtendExpiration(ctx, id, hours)
		},
	}, mcp.NewTool("fibe_playgrounds_extend",
		mcp.WithDescription("[MODE:SIDEEFFECTS] Extend the operational expiration time of an active playground."),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("Playground ID")),
		mcp.WithNumber("hours", mcp.Description("Hours to extend (default: platform setting)")),
	))

	// ---------- fibe_launch ----------
	// Parses compose YAML → creates playspec → deploys playground on a marquee.
	s.addTool(&toolImpl{
		name: "fibe_launch", description: "[MODE:GREENFIELD] Create a playspec and optionally a playground from compose YAML or a local compose file path. Useful when there is no existing template matching Player's request.", tier: tierCore,
		annotations: toolAnnotations{Idempotent: true},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			aliasField(args, "compose_yaml", "compose")
			aliasField(args, "compose_path", "compose_file_path")

			var p fibe.LaunchParams
			if err := bindArgs(args, &p); err != nil {
				return nil, err
			}
			if p.ComposeYAML == "" {
				yaml, err := readInlineOrPathTextArg(args, "compose_yaml", "compose_path")
				if err == nil {
					p.ComposeYAML = yaml
				}
			}
			if p.Name == "" {
				return nil, fmt.Errorf("required field 'name' not set")
			}
			if p.ComposeYAML == "" {
				return nil, fmt.Errorf("required field 'compose_yaml' not set (or pass compose_path / compose_file_path)")
			}
			result, err := c.Launch.Create(ctx, &p)
			if err != nil {
				return nil, err
			}
			// Surface both the semantically-named fields (playspec_id,
			// playground_id) and the raw backend keys (playspecs_created)
			// so pipelines can reference whichever matches the agent's
			// mental model, and the original response shape is preserved
			// for downstream callers.
			out := map[string]any{
				"playspec_id":       result.PlayspecID,
				"playspecs_created": result.PlayspecID,
				"playground_id":     result.PlaygroundID,
				"props_created":     result.PropsCreated,
			}
			return out, nil
		},
	}, mcp.NewTool("fibe_launch",
		mcp.WithDescription(`Preferred one-shot deployment tool. Use this instead of fibe_run(["launch", ...]) whenever possible.

Pass either compose_yaml inline or compose_path/compose_file_path pointing at an absolute local file. The file-path form is preferred for larger compose documents in local MCP sessions.`),
		mcp.WithString("name", mcp.Required(), mcp.Description("Playground or trick name")),
		mcp.WithString("compose_yaml", mcp.Description("Docker Compose YAML content. Use compose_path for larger files.")),
		mcp.WithString("compose_path", mcp.Description("Absolute local path to a compose YAML file (local MCP only). Alias: compose_file_path.")),
		mcp.WithNumber("marquee_id", mcp.Description("Target marquee ID")),
		mcp.WithBoolean("job_mode", mcp.Description("Create as a trick/job instead of a playground")),
		mcp.WithBoolean("create_playground", mcp.Description("Override automatic playground creation behavior")),
		mcp.WithObject("variables", mcp.Description("Template variables map, e.g. {\"subdomain\":\"unstable\",\"production\":\"false\"}")),
		mcp.WithObject("prop_mappings", mcp.Description("Dictionary mapping repository URL to Prop ID, e.g. {\"https://github.com/viktorvsk/fibe\":4}")),
	))

	// ---------- fibe_repo_status ----------
	s.addTool(&toolImpl{
		name: "fibe_repo_status_check", description: "[MODE:DIALOG] Verify the system's access and view of multiple GitHub repository URLs.", tier: tierFull,
		annotations: toolAnnotations{ReadOnly: true, Idempotent: true},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			aliasField(args, "github_urls", "urls")
			type input struct {
				GithubURLs []string `json:"github_urls"`
			}
			var in input
			if err := bindArgs(args, &in); err != nil {
				return nil, err
			}
			if len(in.GithubURLs) == 0 {
				return nil, fmt.Errorf("required field 'github_urls' not set (also accepts 'urls' as alias)")
			}
			return c.RepoStatus.Check(ctx, in.GithubURLs)
		},
	}, mcp.NewTool("fibe_repo_status_check",
		mcp.WithDescription("[MODE:GREENFIELD] Create a playspec and optionally a playground from compose YAML or a local compose file path. Useful when there is no existing template matching Player's request."),
		mcp.WithArray("github_urls", mcp.Required(),
			mcp.Description("List of GitHub repo URLs to check (alias: 'urls')"),
			mcp.WithStringItems()),
	))

	// ---------- fibe_props_branches ----------
	s.addTool(&toolImpl{
		name: "fibe_props_branches", description: "[MODE:DIALOG] List all git branches for a prop.", tier: tierFull,
		annotations: toolAnnotations{ReadOnly: true, Idempotent: true},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			id, ok := argInt64(args, "id")
			if !ok {
				return nil, fmt.Errorf("required field 'id' not set")
			}
			query := argString(args, "query")
			limit := 0
			if n, ok := argInt64(args, "limit"); ok {
				limit = int(n)
			}
			return c.Props.Branches(ctx, id, query, limit)
		},
	}, mcp.NewTool("fibe_props_branches",
		mcp.WithDescription("[MODE:DIALOG] List all git branches for a prop."),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("Prop ID")),
		mcp.WithString("query", mcp.Description("Optional branch name filter (substring match)")),
		mcp.WithNumber("limit", mcp.Description("Max branches to return")),
	))

	// ---------- fibe_props_attach ----------
	s.addTool(&toolImpl{
		name: "fibe_props_attach", description: "[MODE:GREENFIELD] Attach and map an existing GitHub repository to your account as a prop", tier: tierCore,
		annotations: toolAnnotations{Idempotent: true},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			aliasField(args, "repo_full_name", "repository_url", "repo", "url")
			name := argString(args, "repo_full_name")
			if name == "" {
				return nil, fmt.Errorf("required field 'repo_full_name' not set (also accepts 'repository_url' as alias). Pass an owner/repo string, not a full git URL.")
			}
			// If a callers passes a full GitHub URL, trim it down to owner/repo
			// so the backend can look it up. Rails' attach action expects
			// the short form.
			if u := parseRepoFullName(name); u != "" {
				name = u
			}
			return c.Props.Attach(ctx, name)
		},
	}, mcp.NewTool("fibe_props_attach",
		mcp.WithDescription("[MODE:GREENFIELD] Attach and map an existing GitHub repository to your account as a prop"),
		mcp.WithString("repo_full_name", mcp.Required(), mcp.Description("Full repo name, e.g. owner/repo (alias: 'repository_url' accepts https URLs and gets trimmed)")),
	))

	// ---------- fibe_playspecs_validate ----------
	s.addTool(&toolImpl{
		name: "fibe_playspecs_validate_compose", description: "[MODE:DIALOG] Validate a docker-compose YAML file against the playspec schema.", tier: tierFull,
		annotations: toolAnnotations{ReadOnly: true, Idempotent: true},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			aliasField(args, "compose_path", "compose_file_path")
			yaml, err := readInlineOrPathTextArg(args, "compose_yaml", "compose_path")
			if err != nil {
				return nil, fmt.Errorf("required field 'compose_yaml' not set (or pass compose_path / compose_file_path)")
			}
			return c.Playspecs.ValidateCompose(ctx, yaml)
		},
	}, mcp.NewTool("fibe_playspecs_validate_compose",
		mcp.WithDescription("[MODE:DIALOG] Validate a docker-compose YAML file against the playspec schema."),
		mcp.WithString("compose_yaml", mcp.Description("Docker-compose YAML content")),
		mcp.WithString("compose_path", mcp.Description("Absolute local path to a compose YAML file (local MCP only). Alias: compose_file_path.")),
	))

	// ---------- fibe_installations_list ----------
	s.addTool(&toolImpl{
		name: "fibe_installations_list", description: "[MODE:DIALOG] List all GitHub App installations connected to your account", tier: tierCore,
		annotations: toolAnnotations{ReadOnly: true, Idempotent: true},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			return c.Installations.List(ctx)
		},
	}, mcp.NewTool("fibe_installations_list",
		mcp.WithDescription("[MODE:DIALOG] List all GitHub App installations connected to your account"),
	))

	// ---------- fibe_installations_token ----------
	s.addTool(&toolImpl{
		name: "fibe_installations_token", description: "[MODE:SIDEEFFECTS] Generate a scoped GitHub access token for an installation. Useful for bypassing Github limits.", tier: tierCore,
		annotations: toolAnnotations{Idempotent: true},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			id, ok := argInt64(args, "id")
			if !ok {
				return nil, fmt.Errorf("required field 'id' not set")
			}
			repo := argString(args, "repo")
			if repo == "" {
				return nil, fmt.Errorf("required field 'repo' not set")
			}
			return c.Installations.Token(ctx, id, repo)
		},
	}, mcp.NewTool("fibe_installations_token",
		mcp.WithDescription("[MODE:SIDEEFFECTS] Generate a scoped GitHub access token for an installation. Useful for bypassing Github limits."),
		mcp.WithNumber("id", mcp.Required(), mcp.Description("Installation ID")),
		mcp.WithString("repo", mcp.Required(), mcp.Description("Full repo name, e.g. owner/repo")),
	))
}
