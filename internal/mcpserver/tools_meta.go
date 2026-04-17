package mcpserver

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/fibegg/sdk/fibe"
	"github.com/mark3labs/mcp-go/mcp"
)

// registerMetaTools wires meta/utility tools that don't map directly to an
// SDK call: fibe_me, fibe_status, fibe_doctor, fibe_help, fibe_run,
// fibe_auth_set, fibe_schema.
func (s *Server) registerMetaTools() {
	// ---------- fibe_me ----------
	s.addTool(&toolImpl{
		name: "fibe_me", description: "Display the currently authenticated user's profile information", tier: tierMeta,
		annotations: toolAnnotations{ReadOnly: true, Idempotent: true},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			return c.APIKeys.Me(ctx)
		},
	}, mcp.NewTool("fibe_me",
		mcp.WithDescription("Display the currently authenticated user's profile information"),
	))

	// ---------- fibe_status ----------
	s.addTool(&toolImpl{
		name: "fibe_status", description: "Display a comprehensive dashboard of resource counts across your account", tier: tierMeta,
		annotations: toolAnnotations{ReadOnly: true, Idempotent: true},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			return c.Status.Get(ctx)
		},
	}, mcp.NewTool("fibe_status",
		mcp.WithDescription("Display a comprehensive dashboard of resource counts across your account"),
	))

	// ---------- fibe_limits ----------
	s.addTool(&toolImpl{
		name: "fibe_limits", description: "Display current resource quotas, platform caps, and API rate-limit usage", tier: tierMeta,
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
		name: "fibe_doctor", description: "Run self-diagnostic checks to verify API validity and system connectivity", tier: tierMeta,
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
		name: "fibe_auth_set", description: "Configure session-scoped authentication credentials for multi-tenant setups", tier: tierMeta,
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
		name: "fibe_help", description: "Display detailed CLI help documentation for a specific Fibe command path", tier: tierMeta,
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
		name: "fibe_run", description: "Last-resort escape hatch: invoke an arbitrary Fibe CLI command when no dedicated MCP tool fits", tier: tierMeta,
		annotations: toolAnnotations{},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			return s.runCobra(ctx, args)
		},
	}, mcp.NewTool("fibe_run",
		mcp.WithDescription(`Last-resort escape hatch for arbitrary CLI commands.

Prefer dedicated MCP tools first (for example fibe_launch, fibe_playgrounds_*, fibe_props_*, etc.). If the target tool already exists but is not advertised in the current tier, prefer fibe_call over fibe_run.

Use timeout_ms to bound risky calls that might otherwise outlive the host's tool-call budget.`),
		mcp.WithArray("args", mcp.Required(),
			mcp.Description("Command args as if typed after `fibe`. Scalar items (string, number, boolean) are accepted and stringified into CLI tokens in-order.")),
		mcp.WithNumber("timeout_ms", mcp.Description("Optional per-call timeout in milliseconds. Recommended for risky escape-hatch calls.")),
	))

	// ---------- fibe_schema ----------
	// Returns hand-curated JSON-schema hints plus the list of available
	// resources. The per-tool input schemas registered on each MCP tool
	// are the machine-facing source of truth; fibe_schema is for agents
	// that want a consolidated overview.
	s.addTool(&toolImpl{
		name: "fibe_schema", description: "Return JSON Schema definitions for Fibe resource creation and updates", tier: tierMeta,
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
		return nil, fmt.Errorf("field 'args' must be a JSON array")
	}
	strs := make([]string, 0, len(rawSlice))
	for _, v := range rawSlice {
		token, err := stringifyCLIArg(v)
		if err != nil {
			return nil, err
		}
		strs = append(strs, token)
	}
	userArgs := append([]string(nil), strs...)

	var timeoutMs int64
	if v, ok := argInt64(args, "timeout_ms"); ok {
		if v <= 0 {
			return nil, fmt.Errorf("field 'timeout_ms' must be > 0")
		}
		timeoutMs = v
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(timeoutMs)*time.Millisecond)
		defer cancel()
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
	limit := fibeRunCaptureMaxBytes
	if limit <= 0 {
		limit = 1 << 20
	}
	var stdoutBuf truncatingBuffer
	stdoutBuf.limit = limit
	done := make(chan struct{})
	go func() {
		_, _ = io.Copy(&stdoutBuf, r)
		close(done)
	}()

	var stderrBuf truncatingBuffer
	stderrBuf.limit = limit
	root := s.cfg.CobraRoot
	root.SetArgs(strs)
	root.SetOut(&stdoutBuf) // for commands that DO honor cobra's writer
	root.SetErr(&stderrBuf)

	execErr := root.ExecuteContext(ctx)

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
	if timeoutMs > 0 {
		result["timeout_ms"] = timeoutMs
	}
	if recommended := s.recommendedToolForCLIArgs(userArgs); recommended != "" {
		result["recommended_tool"] = recommended
		result["warning"] = fmt.Sprintf("prefer %s over fibe_run when possible", recommended)
	}
	if stdoutBuf.truncated {
		result["stdout_truncated"] = true
		result["stdout_total_bytes"] = stdoutBuf.total
	}
	if stderrBuf.truncated {
		result["stderr_truncated"] = true
		result["stderr_total_bytes"] = stderrBuf.total
	}
	if stdoutBuf.truncated || stderrBuf.truncated {
		result["capture_limit_bytes"] = limit
	}
	if execErr != nil {
		result["error"] = execErr.Error()
		if errors.Is(execErr, context.DeadlineExceeded) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
			result["timed_out"] = true
		}
	}
	return result, nil
}

var fibeRunCaptureMaxBytes = 1 << 20

func stringifyCLIArg(v any) (string, error) {
	switch x := v.(type) {
	case string:
		return x, nil
	case bool:
		if x {
			return "true", nil
		}
		return "false", nil
	case float64, float32, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return fmt.Sprint(x), nil
	default:
		return "", fmt.Errorf("field 'args' may contain only scalars (string, number, boolean)")
	}
}

func (s *Server) recommendedToolForCLIArgs(args []string) string {
	if s.cfg.CobraRoot == nil || len(args) == 0 {
		return ""
	}

	current := s.cfg.CobraRoot
	var path []string
	for _, token := range args {
		if strings.HasPrefix(token, "-") {
			break
		}
		found := false
		for _, candidate := range current.Commands() {
			if !candidate.Hidden && candidate.Name() == token {
				path = append(path, token)
				current = candidate
				found = true
				break
			}
		}
		if !found {
			break
		}
	}
	if len(path) == 0 {
		return ""
	}
	name := "fibe_" + strings.Join(path, "_")
	if name == "fibe_run" {
		return ""
	}
	if _, ok := s.dispatcher.lookup(name); ok {
		return name
	}
	return ""
}

type truncatingBuffer struct {
	limit     int
	total     int
	truncated bool
	buf       bytes.Buffer
}

func (b *truncatingBuffer) Write(p []byte) (int, error) {
	b.total += len(p)
	if b.limit <= 0 {
		return b.buf.Write(p)
	}
	remaining := b.limit - b.buf.Len()
	if remaining <= 0 {
		b.truncated = true
		return len(p), nil
	}
	if len(p) > remaining {
		b.truncated = true
		_, _ = b.buf.Write(p[:remaining])
		return len(p), nil
	}
	return b.buf.Write(p)
}

func (b *truncatingBuffer) String() string {
	return b.buf.String()
}
