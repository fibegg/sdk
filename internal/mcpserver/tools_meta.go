package mcpserver

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/fibegg/sdk/fibe"
	"github.com/mark3labs/mcp-go/mcp"
)

// registerMetaTools wires meta/utility tools that don't map directly to an
// SDK call: fibe_me, fibe_status, fibe_doctor, fibe_help, fibe_run,
// fibe_auth_set, fibe_schema.
func (s *Server) registerMetaTools() {
	// ---------- fibe_me ----------
	s.addTool(&toolImpl{
		name: "fibe_me", description: "Display the currently authenticated user's profile information", tier: tierCore,
		annotations: toolAnnotations{ReadOnly: true, Idempotent: true},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			return c.APIKeys.Me(ctx)
		},
	}, mcp.NewTool("fibe_me",
		mcp.WithDescription("Display the currently authenticated user's profile information"),
	))

	// ---------- fibe_status ----------
	s.addTool(&toolImpl{
		name: "fibe_status", description: "Display a comprehensive dashboard of resource counts across your account", tier: tierCore,
		annotations: toolAnnotations{ReadOnly: true, Idempotent: true},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			return c.Status.Get(ctx)
		},
	}, mcp.NewTool("fibe_status",
		mcp.WithDescription("Display a comprehensive dashboard of resource counts across your account"),
	))

	// ---------- fibe_limits ----------
	s.addTool(&toolImpl{
		name: "fibe_limits", description: "Display current resource quotas, platform caps, and API rate-limit usage", tier: tierCore,
		annotations: toolAnnotations{ReadOnly: true, Idempotent: true},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			status, err := c.Status.Get(ctx)
			if err != nil {
				return nil, err
			}
			if status.ResourceQuotas == nil && status.PerParentCaps == nil && status.RateLimits == nil {
				return nil, fmt.Errorf("no limits data returned — ensure the request is authenticated with an API key")
			}
			return map[string]any{
				"resource_quotas": status.ResourceQuotas,
				"per_parent_caps": status.PerParentCaps,
				"rate_limits":     status.RateLimits,
			}, nil
		},
	}, mcp.NewTool("fibe_limits",
		mcp.WithDescription("Display current resource quotas, platform caps, and API rate-limit usage"),
	))

	// ---------- fibe_server_info ----------
	s.addTool(&toolImpl{
		name: "fibe_server_info", description: "Display the Fibe server's system time, build version, and active commit SHA", tier: tierCore,
		annotations: toolAnnotations{ReadOnly: true, Idempotent: true},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			return c.ServerInfo.Get(ctx)
		},
	}, mcp.NewTool("fibe_server_info",
		mcp.WithDescription("Display the Fibe server's system time, build version, and active commit SHA"),
	))

	// ---------- fibe_doctor ----------
	s.addTool(&toolImpl{
		name: "fibe_doctor", description: "Run self-diagnostic checks to verify API validity and system connectivity", tier: tierCore,
		annotations: toolAnnotations{ReadOnly: true, Idempotent: true},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			me, err := c.APIKeys.Me(ctx)
			result := map[string]any{
				"domain":  c.BaseURL(),
				"version": Version,
			}
			if err != nil {
				result["authenticated"] = false
				result["error"] = err.Error()
				return result, nil
			}
			result["authenticated"] = true
			if me != nil {
				result["user_id"] = me.ID
				result["username"] = me.Username
				if len(me.APIKeyScopes) > 0 {
					result["api_key_scopes"] = me.APIKeyScopes
				}
			}
			return result, nil
		},
	}, mcp.NewTool("fibe_doctor",
		mcp.WithDescription("Run self-diagnostic checks to verify API validity and system connectivity"),
	))

	// ---------- fibe_auth_set ----------
	// Session-scoped credential override — useful for multi-tenant HTTP
	// deployments where different sessions need different API keys.
	//
	// By default fibe_auth_set runs a Ping() against /api/me with the new
	// creds before committing them to session state. If the ping fails the
	// old creds are left intact, preventing the "poisoned session" failure
	// mode where subsequent calls keep returning 401 because a typo'd key
	// was silently installed. Pass validate:false to skip the ping.
	s.addTool(&toolImpl{
		name: "fibe_auth_set", description: "Configure session-scoped authentication credentials for multi-tenant setups", tier: tierCore,
		annotations: toolAnnotations{},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			apiKey := argString(args, "api_key")
			domain := argString(args, "domain")
			if apiKey == "" && domain == "" {
				return nil, fmt.Errorf("at least one of 'api_key' or 'domain' must be provided")
			}
			validate := true
			if v, ok := args["validate"]; ok {
				if b, ok := v.(bool); ok {
					validate = b
				}
			}

			// Snapshot prior state so we can roll back on validation failure.
			prev := s.sessionFor(ctx)
			prev.mu.RLock()
			prevKey, prevDomain := prev.apiKey, prev.domain
			prev.mu.RUnlock()

			s.setSessionAuth(ctx, apiKey, domain)

			if validate {
				// Force a rebuild of the cached client so the ping uses the
				// new creds.
				newClient, err := s.resolveClient(ctx)
				if err == nil && newClient != nil {
					if pingErr := newClient.Ping(ctx); pingErr != nil {
						// Roll back — leave the session in its prior state.
						s.setSessionAuth(ctx, prevKey, prevDomain)
						return nil, fmt.Errorf("fibe_auth_set validation failed (credentials NOT saved): %w", pingErr)
					}
				}
			}

			return map[string]any{
				"ok":          true,
				"api_key_set": apiKey != "",
				"domain_set":  domain != "",
				"validated":   validate,
			}, nil
		},
	}, mcp.NewTool("fibe_auth_set",
		mcp.WithDescription(`Set session-scoped API key and/or domain override.

By default the server pings /api/me with the new credentials before committing
them to session state. If the ping fails, the previous credentials are kept
(prevents "poisoned session" failure modes). Pass validate:false to skip.

Stdio transport can usually rely on the FIBE_API_KEY env var. fibe_auth_set
is most useful in multi-tenant HTTP deployments.`),
		mcp.WithString("api_key", mcp.Description("Fibe API key (pk_live_... or pk_test_...)")),
		mcp.WithString("domain", mcp.Description("API domain override (default: fibe.gg)")),
		mcp.WithBoolean("validate", mcp.Description("Ping /api/me with the new creds before saving (default: true)")),
	))

	// ---------- fibe_help ----------
	// Returns cobra Long help for any fibe subcommand.
	s.addTool(&toolImpl{
		name: "fibe_help", description: "Display detailed CLI help documentation for a specific Fibe command path", tier: tierCore,
		annotations: toolAnnotations{ReadOnly: true, Idempotent: true},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			path := argString(args, "path")
			if s.cfg.CobraRoot == nil {
				return nil, fmt.Errorf("fibe_help not available: server was started without CobraRoot")
			}
			cmd, _, err := s.cfg.CobraRoot.Find(strings.Fields(path))
			if err != nil || cmd == nil {
				return nil, fmt.Errorf("unknown command %q", path)
			}
			var buf bytes.Buffer
			cmd.SetOut(&buf)
			_ = cmd.Help()
			return map[string]any{
				"path":  path,
				"short": cmd.Short,
				"long":  cmd.Long,
				"help":  buf.String(),
			}, nil
		},
	}, mcp.NewTool("fibe_help",
		mcp.WithDescription("Display detailed CLI help documentation for a specific Fibe command path"),
		mcp.WithString("path", mcp.Description("Space-separated command path, e.g. \"playgrounds create\". Empty = root help.")),
	))

	// ---------- fibe_run ----------
	// Escape hatch: invoke any fibe CLI command programmatically.
	s.addTool(&toolImpl{
		name: "fibe_run", description: "Programmatically invoke any arbitrary Fibe CLI command", tier: tierCore,
		annotations: toolAnnotations{},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			return s.runCobra(ctx, args)
		},
	}, mcp.NewTool("fibe_run",
		mcp.WithDescription("Programmatically invoke any arbitrary Fibe CLI command"),
		mcp.WithArray("args", mcp.Required(),
			mcp.Description("Command args as if typed after `fibe`, e.g. [\"playgrounds\", \"list\", \"--status\", \"running\"]"),
			mcp.WithStringItems()),
	))

	// ---------- fibe_schema ----------
	// Returns hand-curated JSON-schema hints plus the list of available
	// resources. The per-tool input schemas registered on each MCP tool
	// are the machine-facing source of truth; fibe_schema is for agents
	// that want a consolidated overview.
	s.addTool(&toolImpl{
		name: "fibe_schema", description: "Return JSON Schema definitions for Fibe resource creation and updates", tier: tierCore,
		annotations: toolAnnotations{ReadOnly: true, Idempotent: true},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			resource := argString(args, "resource")
			operation := argString(args, "operation")
			if resource == "" {
				out := map[string]any{}
				for k, v := range schemaRegistry {
					out[k] = v
				}
				return out, nil
			}
			ops, ok := schemaRegistry[resource]
			if !ok {
				return nil, fmt.Errorf("unknown resource %q", resource)
			}
			if operation == "" {
				return ops, nil
			}
			schema, ok := ops[operation]
			if !ok {
				return nil, fmt.Errorf("unknown operation %q for resource %q", operation, resource)
			}
			return schema, nil
		},
	}, mcp.NewTool("fibe_schema",
		mcp.WithDescription("Return JSON Schema definitions for Fibe resource creation and updates"),
		mcp.WithString("resource", mcp.Description("Resource name: playground, agent, playspec, prop, marquee, secret, team, webhook, api_key")),
		mcp.WithString("operation", mcp.Description("Operation name: create, update (resource-dependent)")),
	))
}

// runCobra implements the fibe_run escape hatch. It invokes a sub-command of
// the cobra root with args from the MCP request and captures output.
//
// This is particularly sensitive: every cmd_*.go handler uses fmt.Println /
// fmt.Printf which writes directly to os.Stdout. Under the MCP stdio
// transport, os.Stdout IS the JSON-RPC pipe, and a stray byte there
// permanently corrupts the connection. ServeStdio already redirects the
// global os.Stdout to os.Stderr (see server.go), so cobra-originated output
// normally lands on stderr — safe but invisible to the caller.
//
// For fibe_run specifically we want to RETURN the command output to the
// agent. So we additionally swap os.Stdout to a pipe for the duration of
// Execute() and drain the pipe into a buffer. When Execute() returns we
// restore the previous os.Stdout (still stderr, not the MCP pipe) so later
// tool calls remain isolated.
func (s *Server) runCobra(ctx context.Context, args map[string]any) (any, error) {
	if s.cfg.CobraRoot == nil {
		return nil, fmt.Errorf("fibe_run not available: server was started without CobraRoot")
	}
	raw, ok := args["args"]
	if !ok {
		return nil, fmt.Errorf("required field 'args' not set")
	}
	rawSlice, ok := raw.([]any)
	if !ok {
		return nil, fmt.Errorf("field 'args' must be a JSON array of strings")
	}
	strs := make([]string, 0, len(rawSlice))
	for _, v := range rawSlice {
		if s, ok := v.(string); ok {
			strs = append(strs, s)
		} else {
			return nil, fmt.Errorf("field 'args' must contain strings only")
		}
	}

	// Force JSON output for predictable downstream parsing.
	strs = append([]string{"--output", "json"}, strs...)

	// Serialize fibe_run across concurrent MCP calls. Cobra mutates flag
	// state on the shared root so this also prevents flag races. Note that
	// hijacking os.Stdout is a process-wide operation; it MUST happen under
	// this lock.
	s.runMu.Lock()
	defer s.runMu.Unlock()

	// Hijack os.Stdout so fmt.Println calls inside cobra handlers are
	// captured instead of escaping to stderr (or, worse, to the real
	// MCP pipe).
	prevStdout := os.Stdout
	r, w, pipeErr := os.Pipe()
	if pipeErr != nil {
		return nil, fmt.Errorf("os.Pipe: %w", pipeErr)
	}
	os.Stdout = w

	// Drain the pipe into a buffer. If this goroutine leaks we don't care
	// because each fibe_run call creates a fresh pipe.
	var stdoutBuf bytes.Buffer
	done := make(chan struct{})
	go func() {
		_, _ = io.Copy(&stdoutBuf, r)
		close(done)
	}()

	var stderrBuf bytes.Buffer
	root := s.cfg.CobraRoot
	root.SetArgs(strs)
	root.SetOut(&stdoutBuf) // for commands that DO honor cobra's writer
	root.SetErr(&stderrBuf)

	execErr := root.Execute()

	// Close the write side and wait for the copy goroutine to drain before
	// restoring, so we don't lose trailing bytes.
	_ = w.Close()
	<-done
	_ = r.Close()
	os.Stdout = prevStdout

	result := map[string]any{
		"args":   strs,
		"stdout": stdoutBuf.String(),
		"stderr": stderrBuf.String(),
	}
	if execErr != nil {
		result["error"] = execErr.Error()
	}
	return result, nil
}

