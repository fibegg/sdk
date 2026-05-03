package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"

	"github.com/fibegg/sdk/internal/mcpserver"
	"github.com/spf13/cobra"
)

func mcpCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "Run Fibe as a local Model Context Protocol server",
		Long: `Run the Fibe CLI as a local MCP server so LLM agents can drive Fibe
without paying the fork+exec cost of invoking the CLI per operation.

TRANSPORTS:
  stdio  (default) — one client per spawned process; single-tenant
  sse    --http :8080 — multiple clients, per-session auth required
  http   --streamable :8080 — same as SSE but with streamable-HTTP transport

SUBCOMMANDS:
  serve     Run the MCP server (default: stdio)
  install   Register the server in a client's MCP config
            (claude-code | claude-desktop | cursor | vscode | antigravity | codex)
  config    Print the client-specific config snippet needed to register the server manually

ENV VARS:
  FIBE_API_KEY              Default API key when per-session auth not set
  FIBE_DOMAIN               API domain override
  FIBE_MCP_YOLO=1           Skip confirm:true gate on destructive tools
  FIBE_MCP_TOOLS            Tool surface: full, core, or comma tiers (e.g. other,meta)
  FIBE_MCP_REQUIRE_AUTH=1   Refuse calls with no resolved API key (multi-tenant)

EXAMPLES:
  fibe mcp serve                                      # stdio, full toolset
  fibe mcp serve --tools core                         # meta+base+greenfield+brownfield
  fibe mcp serve --tools other,meta                   # selected named tiers
  fibe mcp serve --yolo                               # skip destructive confirm gate
  FIBE_MCP_TOOLS=full fibe mcp serve --http :8080     # multi-tenant SSE

  fibe mcp install --client claude-code               # wire into project .mcp.json
  fibe mcp install --client claude-code --user        # wire into ~/.claude.json
  fibe mcp install --client codex                     # wire into ~/.codex/config.toml
  fibe mcp config --client claude-desktop             # print config snippet`,
	}
	cmd.AddCommand(
		mcpServeCmd(),
		mcpInstallCmd(),
		mcpUninstallCmd(),
		mcpConfigCmd(),
		mcpDocsCmd(),
	)
	return cmd
}

func mcpServeCmd() *cobra.Command {
	var (
		httpAddr       string
		streamableHTTP bool
		toolSet        string
		yolo           bool
		requireAuth    bool
		cacheSize      int
		cacheEntryMax  int
		maxSteps       int
		maxIterations  int
	)

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Run the Fibe MCP server",
		Long: `Run the Fibe MCP server. Default transport is stdio (single-tenant,
spawned by MCP clients like Claude Code). Use --http to serve multiple
tenants over SSE.

TOOL SURFACE:
  --tools full             Complete registered tool surface
  --tools core             meta+base+greenfield+brownfield
  --tools other,meta       Comma-separated named tiers

SAFETY:
  --yolo         Skip the confirm:true gate on destructive tools
                 (equivalent to FIBE_MCP_YOLO=1)

MULTI-TENANT (HTTP only):
  --require-auth  Reject requests with no resolved API key

  Per-request auth: Authorization: Bearer <fibe-api-key>
  Per-session:      call the fibe_auth_set tool once per session

PIPELINE:
  --pipeline-cache-size N    Max cached pipeline results (default 256)
  --pipeline-max-steps N     Hard cap on steps per pipeline (default 25)

EXAMPLES:
  fibe mcp serve
  fibe mcp serve --tools full --yolo
  fibe mcp serve --http :8080 --require-auth`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := mcpserver.DefaultConfig()
			cfg.APIKey = resolveAPIKey()
			cfg.Domain = flagDomain
			cfg.Debug = flagDebug

			cfg.ToolSet = resolveEnv("FIBE_MCP_TOOLS", toolSet, cfg.ToolSet)
			cfg.Yolo = yolo || envBool("FIBE_MCP_YOLO")
			cfg.RequireAuth = requireAuth || envBool("FIBE_MCP_REQUIRE_AUTH")
			if cacheSize > 0 {
				cfg.PipelineCacheSize = cacheSize
			} else if v := envInt("FIBE_MCP_PIPELINE_CACHE_SIZE"); v > 0 {
				cfg.PipelineCacheSize = v
			}
			if cacheEntryMax > 0 {
				cfg.PipelineCacheEntryMax = cacheEntryMax
			} else if v := envInt("FIBE_MCP_PIPELINE_CACHE_ENTRY_MAX"); v > 0 {
				cfg.PipelineCacheEntryMax = v
			}
			if maxSteps > 0 {
				cfg.PipelineMaxSteps = maxSteps
			} else if v := envInt("FIBE_MCP_PIPELINE_MAX_STEPS"); v > 0 {
				cfg.PipelineMaxSteps = v
			}
			if maxIterations > 0 {
				cfg.PipelineMaxIterations = maxIterations
			} else if v := envInt("FIBE_MCP_PIPELINE_MAX_ITERATIONS"); v > 0 {
				cfg.PipelineMaxIterations = v
			}

			if cfg.Yolo {
				fmt.Fprintln(os.Stderr, "WARN: --yolo enabled; destructive tools do not require confirm:true")
			}

			// Wire the cobra root tree so fibe_help and fibe_run can introspect it.
			cfg.CobraRoot = cmd.Root()

			srv := mcpserver.New(cfg)
			if err := srv.RegisterAll(); err != nil {
				return fmt.Errorf("register: %w", err)
			}

			ctx := context.Background()
			if httpAddr != "" {
				return srv.ServeHTTP(ctx, httpAddr, streamableHTTP)
			}
			return srv.ServeStdio(ctx)
		},
	}
	cmd.Flags().StringVar(&httpAddr, "http", "", "Listen on host:port for SSE transport (default: stdio)")
	cmd.Flags().BoolVar(&streamableHTTP, "streamable", false, "Use streamable-HTTP transport instead of SSE (requires --http)")
	cmd.Flags().StringVar(&toolSet, "tools", "", "Tool surface: full, core, or comma tiers such as other,meta (env: FIBE_MCP_TOOLS, default: full)")
	cmd.Flags().BoolVar(&yolo, "yolo", false, "Skip confirm:true gate on destructive tools (env: FIBE_MCP_YOLO)")
	cmd.Flags().BoolVar(&requireAuth, "require-auth", false, "Reject requests with no resolved API key (multi-tenant)")
	cmd.Flags().IntVar(&cacheSize, "pipeline-cache-size", 0, "Max cached pipeline results (env: FIBE_MCP_PIPELINE_CACHE_SIZE)")
	cmd.Flags().IntVar(&cacheEntryMax, "pipeline-cache-entry-max", 0, "Max bytes per cached entry (env: FIBE_MCP_PIPELINE_CACHE_ENTRY_MAX)")
	cmd.Flags().IntVar(&maxSteps, "pipeline-max-steps", 0, "Max steps per pipeline (env: FIBE_MCP_PIPELINE_MAX_STEPS)")
	cmd.Flags().IntVar(&maxIterations, "pipeline-max-iterations", 0, "Max total for_each iterations (env: FIBE_MCP_PIPELINE_MAX_ITERATIONS)")
	return cmd
}

func mcpInstallCmd() *cobra.Command {
	var client, project string
	var dryRun, userScope bool
	var opts installOptions
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Register the Fibe MCP server in a client's config",
		Long: `Write the Fibe MCP server into an MCP client's configuration file.

SUPPORTED CLIENTS:
  claude-code      project .mcp.json by default (use --user for ~/.claude.json)
  claude-desktop   ~/Library/Application Support/Claude/claude_desktop_config.json
  cursor           ~/.cursor/mcp.json
  vscode           ~/.vscode/mcp.json (or workspace .vscode/mcp.json with --project .)
  antigravity      ~/.gemini/antigravity/mcp_config.json
  codex            ~/.codex/config.toml (or project .codex/config.toml with --project .)

The installer detects the absolute path of the current fibe executable and
emits an entry that launches "fibe mcp serve" on stdio by default.

For Antigravity, Cursor, VS Code, and Codex you can also emit a URL-backed
config entry that points at a separately managed streamable-HTTP server:

  fibe mcp install --client antigravity --transport streamable-http \
    --url https://fibe.example.com/mcp

  fibe mcp install --client codex --transport streamable-http \
    --url http://127.0.0.1:7797/mcp

Cursor and VS Code support the same transport:

  fibe mcp install --client cursor --transport streamable-http \
    --url https://fibe.example.com/mcp
  fibe mcp install --client vscode --transport streamable-http \
    --url https://fibe.example.com/mcp

URL-backed mode is useful when you already operate a managed remote MCP
endpoint. Stdio remains the default because the SDK does not supervise a
long-lived local HTTP daemon yet.

ENV VARS:
  By default, FIBE_API_KEY is written as "${FIBE_API_KEY}" so placeholder-
  expanding clients resolve it at runtime. Antigravity does NOT expand
  placeholders, so the installer auto-resolves FIBE_API_KEY from your shell
  at install time for that client. Codex uses env_vars forwarding in
  ~/.codex/config.toml instead of ${VAR} placeholders. Override with
  --api-key to inline any value explicitly.

TOOL TIERS:
  --tools full             Expose the complete registered tool surface up front (default).
  --tools core             Expose meta+base+greenfield+brownfield.
  --tools other,meta       Expose only the selected named tiers.

EXAMPLES:
  fibe mcp install --client claude-code
  fibe mcp install --client claude-code --user
  fibe mcp install --client antigravity --api-key pk_live_... --domain http://dev.local:3000
  fibe mcp install --client claude-desktop --tools full --yolo
  fibe mcp install --client cursor --env FOO=bar --env BAZ=qux
  fibe mcp install --client codex --project .
  fibe mcp install --client cursor --transport streamable-http --url https://fibe.example.com/mcp
  fibe mcp install --client vscode --transport streamable-http --url https://fibe.example.com/mcp
  fibe mcp install --client codex --transport streamable-http --url http://127.0.0.1:7797/mcp
  fibe mcp install --client antigravity --dry-run`,
		RunE: func(cmd *cobra.Command, args []string) error {
			resolvedProject, err := resolveMCPProjectScope(client, project, userScope)
			if err != nil {
				return err
			}
			return runMCPInstall(client, resolvedProject, dryRun, opts)
		},
	}
	cmd.Flags().StringVar(&client, "client", "claude-code", "Target client: "+mcpClientFlagHelp)
	cmd.Flags().StringVar(&project, "project", "", "Install into a project-scoped config (pass the project directory). Default for claude-code: current directory.")
	cmd.Flags().BoolVar(&userScope, "user", false, "Install into the user-scoped config when the client supports both user and project configs")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Print the resolved target path and proposed config without writing")
	cmd.Flags().StringVar(&opts.APIKey, "api-key", "", "Inline a literal FIBE_API_KEY value (skip ${VAR} placeholder)")
	cmd.Flags().StringVar(&opts.Domain, "domain", "", "Inline a literal FIBE_DOMAIN value")
	cmd.Flags().StringArrayVar(&opts.Env, "env", nil, "Additional env var (KEY=VALUE). Repeatable.")
	cmd.Flags().StringVar(&opts.ToolSet, "tools", "", "Tool surface: full, core, or comma tiers such as other,meta")
	cmd.Flags().BoolVar(&opts.Yolo, "yolo", false, "Pass FIBE_MCP_YOLO=1 so destructive tools skip the confirm:true gate")
	cmd.Flags().StringVar(&opts.AuditLog, "audit-log", "", "Write MCP tool-call audit log to this path (or 'stderr')")
	cmd.Flags().StringVar(&opts.Transport, "transport", "", "Install transport override. Supported: stdio (default) or streamable-http with --url (antigravity, cursor, vscode, codex)")
	cmd.Flags().StringVar(&opts.URL, "url", "", "URL for URL-backed MCP clients, e.g. http://127.0.0.1:7797/mcp or https://fibe.example.com/mcp")
	return cmd
}

func mcpUninstallCmd() *cobra.Command {
	var client, project string
	var dryRun, userScope bool
	cmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Remove the Fibe MCP server from a client's config",
		Long: `Remove the "fibe" entry from an MCP client's configuration file without
touching other registered servers.

EXAMPLES:
  fibe mcp uninstall --client claude-code
  fibe mcp uninstall --client claude-code --user
  fibe mcp uninstall --client claude-desktop --dry-run
  fibe mcp uninstall --client codex --project .`,
		RunE: func(cmd *cobra.Command, args []string) error {
			resolvedProject, err := resolveMCPProjectScope(client, project, userScope)
			if err != nil {
				return err
			}
			return runMCPUninstall(client, resolvedProject, dryRun)
		},
	}
	cmd.Flags().StringVar(&client, "client", "claude-code", "Target client: "+mcpClientFlagHelp)
	cmd.Flags().StringVar(&project, "project", "", "Operate on a project-scoped config (pass the project directory). Default for claude-code: current directory.")
	cmd.Flags().BoolVar(&userScope, "user", false, "Operate on the user-scoped config when the client supports both user and project configs")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Print the resolved target path and proposed content without writing")
	return cmd
}

func mcpConfigCmd() *cobra.Command {
	var client string
	var opts installOptions
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Print the MCP config snippet for a client",
		Long: `Print the client-specific config snippet needed to register the Fibe
MCP server manually in an MCP client's configuration.

EXAMPLES:
  fibe mcp config --client claude-code
  fibe mcp config --client claude-desktop
  fibe mcp config --client antigravity --transport streamable-http --url https://fibe.example.com/mcp
  fibe mcp config --client cursor --transport streamable-http --url https://fibe.example.com/mcp
  fibe mcp config --client vscode --transport streamable-http --url https://fibe.example.com/mcp
  fibe mcp config --client codex
  fibe mcp config --client codex --transport streamable-http --url http://127.0.0.1:7797/mcp`,
		RunE: func(cmd *cobra.Command, args []string) error {
			bin, err := os.Executable()
			if err != nil {
				bin = "fibe"
			}
			entry, _, err := buildMCPInstallEntry(client, bin, opts)
			if err != nil {
				return err
			}
			switch client {
			case "claude-code":
				fmt.Println(`// Add under "mcpServers" in ~/.claude.json or project-root .mcp.json:`)
				snippet := map[string]any{mcpServerName: entry}
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(snippet)
			case "claude-desktop":
				fmt.Println(`// Add under "mcpServers" in ~/Library/Application Support/Claude/claude_desktop_config.json:`)
				snippet := map[string]any{mcpServerName: entry}
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(snippet)
			case "cursor":
				fmt.Println(`// Add under "mcpServers" in ~/.cursor/mcp.json:`)
				snippet := map[string]any{mcpServerName: entry}
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(snippet)
			case "vscode":
				fmt.Println(`// Add under "servers" in .vscode/mcp.json (schema differs slightly — see VS Code docs):`)
				snippet := map[string]any{mcpServerName: entry}
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(snippet)
			case "antigravity":
				fmt.Println(`// Add under "mcpServers" in ~/.gemini/antigravity/mcp_config.json:`)
				snippet := map[string]any{mcpServerName: entry}
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(snippet)
			case "codex":
				fmt.Println(`# Add to ~/.codex/config.toml or .codex/config.toml:`)
				snippet := map[string]any{
					"mcp_servers": map[string]any{
						mcpServerName: entry,
					},
				}
				out, err := marshalMCPConfig(snippet, mcpConfigTOML)
				if err != nil {
					return err
				}
				_, err = os.Stdout.Write(out)
				return err
			default:
				return fmt.Errorf("unknown client %q — valid: %s", client, mcpValidClientList)
			}
		},
	}
	cmd.Flags().StringVar(&client, "client", "claude-code", "Target client: "+mcpClientFlagHelp)
	cmd.Flags().StringVar(&opts.APIKey, "api-key", "", "Inline a literal FIBE_API_KEY value (skip ${VAR} placeholder)")
	cmd.Flags().StringVar(&opts.Domain, "domain", "", "Inline a literal FIBE_DOMAIN value")
	cmd.Flags().StringArrayVar(&opts.Env, "env", nil, "Additional env var (KEY=VALUE). Repeatable.")
	cmd.Flags().StringVar(&opts.ToolSet, "tools", "", "Tool surface: full, core, or comma tiers such as other,meta")
	cmd.Flags().BoolVar(&opts.Yolo, "yolo", false, "Pass FIBE_MCP_YOLO=1 so destructive tools skip the confirm:true gate")
	cmd.Flags().StringVar(&opts.AuditLog, "audit-log", "", "Write MCP tool-call audit log to this path (or 'stderr')")
	cmd.Flags().StringVar(&opts.Transport, "transport", "", "Snippet transport override. Supported: stdio (default) or streamable-http with --url (antigravity, cursor, vscode, codex)")
	cmd.Flags().StringVar(&opts.URL, "url", "", "URL for URL-backed MCP clients, e.g. http://127.0.0.1:7797/mcp or https://fibe.example.com/mcp")
	return cmd
}

// resolveAPIKey resolves the effective API key for the MCP server base
// client: first --api-key, then FIBE_API_KEY. Per-session overrides are
// resolved later, inside the dispatcher.
func resolveAPIKey() string {
	if flagAPIKey != "" {
		return flagAPIKey
	}
	return os.Getenv("FIBE_API_KEY")
}

func resolveEnv(envKey, flagVal, def string) string {
	if flagVal != "" {
		return flagVal
	}
	if v := os.Getenv(envKey); v != "" {
		return v
	}
	return def
}

func envBool(key string) bool {
	v := os.Getenv(key)
	if v == "" {
		return false
	}
	switch v {
	case "1", "true", "TRUE", "yes", "on":
		return true
	}
	return false
}

func envInt(key string) int {
	v := os.Getenv(key)
	if v == "" {
		return 0
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0
	}
	return n
}

func mcpDocsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "docs",
		Short: "Print full MCP tool catalog as JSON",
		Long: `Print metadata for every registered MCP tool as a single JSON document.

Useful for generating documentation, feeding into search indexes, or
inspecting the full tool surface without starting an MCP server.

The output includes all tools regardless of tier (meta, base, greenfield,
brownfield, overseer, local, other).`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := mcpserver.DefaultConfig()
			cfg.CobraRoot = cmd.Root()
			srv := mcpserver.New(cfg)
			if err := srv.RegisterAll(); err != nil {
				return fmt.Errorf("register: %w", err)
			}

			tools := srv.AllTools()
			sort.Slice(tools, func(i, j int) bool {
				return tools[i].Name < tools[j].Name
			})

			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(map[string]any{
				"count": len(tools),
				"tools": tools,
			})
		},
	}
}
