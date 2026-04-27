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
	"github.com/fibegg/sdk/internal/resourceschema"
	"github.com/mark3labs/mcp-go/mcp"
)

// registerMetaTools wires utility tools that don't map directly to an SDK
// call. Most are meta tools, with auth/local helpers assigned to their
// explicit advertised tiers.
func (s *Server) registerMetaTools() {
	// ---------- fibe_status ----------
	s.addTool(&toolImpl{
		name: "fibe_status", description: "[MODE:DIALOG] Display a comprehensive dashboard of resource counts, quotas, and rate limits across your account.", tier: tierMeta,
		annotations: toolAnnotations{ReadOnly: true, Idempotent: true},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			return c.Status.Get(ctx)
		},
	}, mcp.NewTool("fibe_status",
		mcp.WithDescription("[MODE:DIALOG] Display a comprehensive dashboard of resource counts, quotas, rate limits, and subscription info."),
	))

	// ---------- fibe_doctor ----------
	s.addTool(&toolImpl{
		name: "fibe_doctor", description: "[MODE:DIALOG] Run self-diagnostic checks: verify API key, connectivity, and display user profile", tier: tierMeta,
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
				result["github_handle"] = me.GithubHandle
				result["email"] = me.Email
				result["avatar_url"] = me.AvatarURL
				if len(me.APIKeyScopes) > 0 {
					result["api_key_scopes"] = me.APIKeyScopes
				}
			}

			return result, nil
		},
	}, mcp.NewTool("fibe_doctor",
		mcp.WithDescription("[MODE:DIALOG] Run self-diagnostic checks: verify API key validity, server connectivity, SDK version, and display user profile (ID, username, GitHub handle, email, avatar, scopes)."),
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
		name: "fibe_auth_set", description: "[MODE:SIDEEFFECTS] Configure session-scoped authentication credentials for multi-tenant setups in case you have to work with multiple FIBE_API_KEY+FIBE_DOMAIN combinations", tier: tierOther,
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
		mcp.WithString("api_key", mcp.Description("Fibe API key, for example fibe_live_... or fibe_test_...")),
		mcp.WithString("domain", mcp.Description("API domain override (default: fibe.gg)")),
		mcp.WithBoolean("validate", mcp.Description("Ping /api/me with the new creds before saving (default: true)")),
	))

	// ---------- local playground helpers ----------
	s.addTool(&toolImpl{
		name: "fibe_local_playgrounds_list", description: "[MODE:BROWNFIELD] List playgrounds available locally at /opt/fibe/playgrounds or PLAYROOMS_ROOT.", tier: tierLocal,
		annotations: toolAnnotations{ReadOnly: true, Idempotent: true},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			return s.runCobraArgs(ctx, "local-playgrounds", "list")
		},
	}, mcp.NewTool("fibe_local_playgrounds_list",
		mcp.WithDescription("[MODE:BROWNFIELD] List playgrounds available locally at /opt/fibe/playgrounds or PLAYROOMS_ROOT."),
	))

	s.addTool(&toolImpl{
		name: "fibe_local_playgrounds_info", description: "[MODE:BROWNFIELD] Get info about a local playground.", tier: tierLocal,
		annotations: toolAnnotations{ReadOnly: true, Idempotent: true},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			name := argString(args, "playground")
			if name == "" {
				return nil, fmt.Errorf("required field 'playground' not set")
			}
			return s.runCobraArgs(ctx, "local-playgrounds", "info", name)
		},
	}, mcp.NewTool("fibe_local_playgrounds_info",
		mcp.WithDescription("[MODE:BROWNFIELD] Get info about a local playground."),
		mcp.WithString("playground", mcp.Required(), mcp.Description("Local playground name, directory, or playspec prefix")),
	))

	s.addTool(&toolImpl{
		name: "fibe_local_playgrounds_urls", description: "[MODE:BROWNFIELD] Get URLs of a local playground.", tier: tierLocal,
		annotations: toolAnnotations{ReadOnly: true, Idempotent: true},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			name := argString(args, "playground")
			if name == "" {
				return nil, fmt.Errorf("required field 'playground' not set")
			}
			return s.runCobraArgs(ctx, "local-playgrounds", "urls", name)
		},
	}, mcp.NewTool("fibe_local_playgrounds_urls",
		mcp.WithDescription("[MODE:BROWNFIELD] Get URLs of a local playground."),
		mcp.WithString("playground", mcp.Required(), mcp.Description("Local playground name, directory, or playspec prefix")),
	))

	s.addTool(&toolImpl{
		name: "fibe_local_playgrounds_link", description: "[MODE:BROWNFIELD] Link local playground mounts into a working directory.", tier: tierBrownfield,
		annotations: toolAnnotations{Idempotent: true},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			name := argString(args, "playground")
			if name == "" {
				return nil, fmt.Errorf("required field 'playground' not set")
			}
			cliArgs := []any{"local-playgrounds", "link", name}
			if linkDir := argString(args, "link_dir"); linkDir != "" {
				cliArgs = append(cliArgs, "--link-dir", linkDir)
			}
			return s.runCobra(ctx, map[string]any{"args": cliArgs})
		},
	}, mcp.NewTool("fibe_local_playgrounds_link",
		mcp.WithDescription("[MODE:BROWNFIELD] Link local playground mounts into a working directory."),
		mcp.WithString("playground", mcp.Required(), mcp.Description("Local playground name, directory, or playspec prefix")),
		mcp.WithString("link_dir", mcp.Description("Target directory for symlinks (default: /app/playground)")),
	))

	// ---------- fibe_help ----------
	// Returns cobra Long help for any fibe subcommand.
	s.addTool(&toolImpl{
		name: "fibe_help", description: "[MODE:DIALOG] Display detailed CLI help documentation for a specific Fibe command path. Extremely useful to look up flag descriptions or expected payload shapes.", tier: tierMeta,
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
		mcp.WithDescription("[MODE:DIALOG] Display detailed CLI help documentation for a specific Fibe command path. Extremely useful to look up flag descriptions or expected payload shapes."),
		mcp.WithString("path", mcp.Description("Space-separated command path, e.g. \"playgrounds create\". Empty = root help.")),
	))

	// ---------- fibe_run ----------
	// Escape hatch: invoke any fibe CLI command programmatically.
	s.addTool(&toolImpl{
		name: "fibe_run", description: "[MODE:SIDEEFFECTS] Last-resort escape hatch: invoke an arbitrary Fibe CLI command when no dedicated MCP tool fits. Use sparingly.", tier: tierMeta,
		annotations: toolAnnotations{},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			return s.runCobra(ctx, args)
		},
	}, mcp.NewTool("fibe_run",
		mcp.WithDescription(`Last-resort escape hatch for arbitrary CLI commands.

Prefer dedicated MCP tools first (for example fibe_greenfield_create, fibe_templates_launch, fibe_playgrounds_*, fibe_props_*, etc.). If the target tool already exists but is not advertised in the current tier, prefer fibe_call over fibe_run.

Use timeout_ms to bound risky calls that might otherwise outlive the host's tool-call budget.`),
		mcp.WithArray("args", mcp.Required(), mcp.WithStringItems(),
			mcp.Description("Command args as if typed after `fibe`. Scalar items (string, number, boolean) are accepted and stringified into CLI tokens in-order.")),
		mcp.WithNumber("timeout_ms", mcp.Description("Optional per-call timeout in milliseconds. Recommended for risky escape-hatch calls.")),
	))

	// ---------- fibe_schema ----------
	// Returns the shared resource schema registry. Generic resource tools
	// validate against this registry before dispatch, so fibe_schema is the
	// authoritative source for resource-operation payload shapes.
	schemaResourceSelectors := resourceschema.SchemaResourceSelectors()
	schemaOperations := resourceschema.OperationNames()
	s.addTool(&toolImpl{
		name: "fibe_schema", description: "[MODE:DIALOG] Return JSON Schema definitions and the schema resource catalog.", tier: tierMeta,
		annotations: toolAnnotations{ReadOnly: true, Idempotent: true},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			resource := argString(args, "resource")
			operation := argString(args, "operation")
			if resource == "" {
				return resourceschema.Registry(), nil
			}
			if resourceschema.NormalizeResource(resource) == "list" {
				return resourceschema.CatalogResponse(), nil
			}
			ops, canonical, ok := resourceschema.SchemasFor(resource)
			if !ok {
				return nil, fmt.Errorf("unknown resource %q; supported resources: %s", resource, resourceschema.ResourceNamesString())
			}
			if operation == "" {
				return ops, nil
			}
			if canonical == "compose" && resourceschema.NormalizeResource(operation) == "validate" {
				if payload, ok := args["payload"].(map[string]any); ok && payload != nil {
					if _, _, err := resourceschema.ValidatePayload("compose", "validate", payload); err != nil {
						return nil, err
					}
					yaml, err := readInlineOrPathTextArg(payload, "compose_yaml", "compose_path")
					if err != nil {
						return nil, fmt.Errorf("required field 'payload.compose_yaml' not set (or pass payload.compose_path)")
					}
					params := &fibe.ComposeValidateParams{
						ComposeYAML: yaml,
						TargetType:  argString(payload, "target_type"),
					}
					if _, exists := payload["job_mode"]; exists {
						jobMode := argBool(payload, "job_mode")
						params.JobMode = &jobMode
					}
					return c.Playspecs.ValidateComposeWithParams(ctx, params)
				}
			}
			schema, _, op, ok := resourceschema.SchemaFor(canonical, operation)
			if !ok {
				return nil, fmt.Errorf("unknown operation %q for resource %q", op, canonical)
			}
			return schema, nil
		},
	}, mcp.NewTool("fibe_schema",
		mcp.WithDescription("[MODE:DIALOG] Return JSON Schema definitions and the schema resource catalog. Pass resource=list to discover resources, aliases, and supported operations."),
		mcp.WithString("resource", mcp.Enum(schemaResourceSelectors...), mcp.Description("Resource name, alias, or 'list' for the schema resource catalog.")),
		mcp.WithString("operation", mcp.Enum(schemaOperations...), mcp.Description("Operation name. Supported combinations are resource-dependent; pass resource=list for the catalog.")),
		mcp.WithObject("payload", mcp.Description("Optional operation payload for side-effect-free schema-backed validations such as compose.validate.")),
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

func (s *Server) runCobraArgs(ctx context.Context, args ...string) (any, error) {
	raw := make([]any, len(args))
	for i, arg := range args {
		raw[i] = arg
	}
	return s.runCobra(ctx, map[string]any{"args": raw})
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
	if strings.Join(path, " ") == "mutters create" {
		if _, ok := s.dispatcher.lookup("fibe_mutter"); ok {
			return "fibe_mutter"
		}
	}
	if isCLIResourceMutationPath(path) {
		if _, ok := s.dispatcher.lookup("fibe_resource_mutate"); ok {
			return "fibe_resource_mutate"
		}
	}
	return ""
}

func isCLIResourceMutationPath(path []string) bool {
	if len(path) < 2 {
		return false
	}
	key := strings.Join(path, " ")
	known := map[string]bool{
		"agents create":                   true,
		"agents update":                   true,
		"api-keys create":                 true,
		"artefacts create":                true,
		"marquees create":                 true,
		"marquees update":                 true,
		"playgrounds create":              true,
		"playgrounds update":              true,
		"playspecs create":                true,
		"playspecs update":                true,
		"props create":                    true,
		"props update":                    true,
		"secrets create":                  true,
		"secrets update":                  true,
		"templates create":                true,
		"templates update":                true,
		"templates versions create":       true,
		"templates versions patch-create": true,
		"webhooks create":                 true,
		"webhooks update":                 true,
		"job-env set":                     true,
		"job-env update":                  true,
	}
	return known[key]
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
