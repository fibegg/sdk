package mcpserver

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/fibegg/sdk/fibe"
	"github.com/fibegg/sdk/internal/localplaygrounds"
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
			profile, authSource, domainSource := s.doctorAuthMetadata(ctx)
			result := map[string]any{
				"profile":       profile,
				"domain":        c.BaseURL(),
				"version":       Version,
				"auth_source":   authSource,
				"domain_source": domainSource,
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
		handler: func(ctx context.Context, _ *fibe.Client, args map[string]any) (any, error) {
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

Prefer fibe_auth_use for local profile switching because it never reveals API
keys to the agent. fibe_auth_set remains useful for HTTP deployments or
one-off credentials supplied explicitly by the user.`),
		mcp.WithString("api_key", mcp.Description("Fibe API key, for example fibe_live_... or fibe_test_...")),
		mcp.WithString("domain", mcp.Description("API domain override (default: fibe.gg)")),
		mcp.WithBoolean("validate", mcp.Description("Ping /api/me with the new creds before saving (default: true)")),
	))

	// ---------- fibe_auth_list ----------
	s.addTool(&toolImpl{
		name: "fibe_auth_list", description: "[MODE:DIALOG] List local Fibe auth profiles available to this MCP server without revealing API keys.", tier: tierMeta,
		annotations: toolAnnotations{ReadOnly: true, Idempotent: true},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			profiles, err := listMCPAuthProfiles()
			if err != nil {
				return nil, err
			}
			return map[string]any{
				"profiles": profiles,
			}, nil
		},
	}, mcp.NewTool("fibe_auth_list",
		mcp.WithDescription("List local Fibe auth profiles available to this MCP server. API keys are masked and never returned in full."),
	))

	// ---------- fibe_auth_status ----------
	s.addTool(&toolImpl{
		name: "fibe_auth_status", description: "[MODE:DIALOG] Show the current MCP session auth target and selected profile, if any.", tier: tierMeta,
		annotations: toolAnnotations{ReadOnly: true, Idempotent: true},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			st := s.sessionFor(ctx)
			st.mu.RLock()
			profile, sessionProfile, domain, apiKeySet := st.profile, st.profile, st.domain, st.apiKey != ""
			st.mu.RUnlock()
			if profile == "" {
				profile = s.cfg.Profile
			}
			if domain == "" {
				domain = s.cfg.Domain
			}
			return map[string]any{
				"profile":          profile,
				"session_profile":  sessionProfile,
				"server_profile":   s.cfg.Profile,
				"base_url":         c.BaseURL(),
				"session_domain":   normalizeMCPDomain(domain),
				"session_key_set":  apiKeySet,
				"server_key_set":   s.cfg.APIKey != "",
				"require_auth":     s.cfg.RequireAuth,
				"configured_tools": s.cfg.ToolSet,
			}, nil
		},
	}, mcp.NewTool("fibe_auth_status",
		mcp.WithDescription("Show the current MCP session auth target and selected profile, if any."),
	))

	// ---------- fibe_auth_use ----------
	s.addTool(&toolImpl{
		name: "fibe_auth_use", description: "[MODE:SIDEEFFECTS] Switch this MCP session to a local Fibe auth profile by name, rebuilding the session client immediately.", tier: tierMeta,
		annotations: toolAnnotations{},
		handler: func(ctx context.Context, _ *fibe.Client, args map[string]any) (any, error) {
			profile := strings.TrimSpace(argString(args, "profile"))
			domain, apiKey, apiKeyID, ok, err := resolveMCPAuthProfile(profile)
			if err != nil {
				return nil, err
			}
			if !ok {
				return nil, fmt.Errorf("auth profile %q does not exist", profile)
			}
			validate := true
			if v, ok := args["validate"]; ok {
				if b, ok := v.(bool); ok {
					validate = b
				}
			}

			prev := s.sessionFor(ctx)
			prev.mu.RLock()
			prevProfile, prevKey, prevDomain := prev.profile, prev.apiKey, prev.domain
			prev.mu.RUnlock()

			s.setSessionProfile(ctx, profile, apiKey, domain)
			if validate {
				newClient, err := s.resolveClient(ctx)
				if err == nil && newClient != nil {
					if pingErr := newClient.Ping(ctx); pingErr != nil {
						s.setSessionProfile(ctx, prevProfile, prevKey, prevDomain)
						return nil, fmt.Errorf("fibe_auth_use validation failed (profile NOT selected): %w", pingErr)
					}
				}
			}
			return map[string]any{
				"ok":         true,
				"profile":    profile,
				"domain":     domain,
				"base_url":   mcpBaseURL(domain),
				"api_key_id": apiKeyID,
				"key_set":    apiKey != "",
				"validated":  validate,
			}, nil
		},
	}, mcp.NewTool("fibe_auth_use",
		mcp.WithDescription(`Switch this MCP session to a local Fibe auth profile by name.

This does not reveal API keys to the agent. By default it validates the
selected profile with /api/me before keeping the new session client.`),
		mcp.WithString("profile", mcp.Required(), mcp.Description("Local Fibe auth profile name, for example default, staging, local, or a feature-env profile.")),
		mcp.WithBoolean("validate", mcp.Description("Ping /api/me with the selected profile before saving (default: true).")),
	))

	// ---------- local playground helpers ----------
	s.addTool(&toolImpl{
		name: "fibe_local_playgrounds_info", description: "[MODE:BROWNFIELD] Inspect local playground names, URLs, mounts, or details from /opt/fibe/playgrounds or MARQUEE_ROOT.", tier: tierBrownfield,
		annotations: toolAnnotations{ReadOnly: true, Idempotent: true},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			view := strings.ToLower(strings.TrimSpace(argString(args, "view")))
			if view == "" {
				return nil, fmt.Errorf("required field 'view' not set")
			}
			selector, err := localPlaygroundSelectorFromArgs(args, view)
			if err != nil {
				return nil, err
			}
			playgrounds, err := localplaygrounds.Scan(localplaygrounds.BaseDir())
			if err != nil {
				return nil, err
			}
			return localplaygrounds.View(playgrounds, view, selector, localplaygrounds.RootDomain())
		},
	}, mcp.NewTool("fibe_local_playgrounds_info",
		mcp.WithDescription(`Inspect local playgrounds from the Marquee filesystem without calling the Fibe API.

Views:
  names    list selector-visible local playground names, playspecs, IDs, and paths; omit id_or_name
  urls     exposed service URLs for one playground
  mounts   source-code mount locations for one playground
  details  full local metadata for one playground

Selectors accept local numeric playground ID, compose project/name, playspec, or unique playspec prefix.`),
		mcp.WithString("view", mcp.Required(), mcp.Enum(localplaygrounds.Views...), mcp.Description("Output view: names, urls, mounts, or details.")),
		mcp.WithString("id_or_name", mcp.Description("Local playground ID, name, compose project, playspec, or unique playspec prefix. Omit for view=names.")),
	))

	s.addTool(&toolImpl{
		name: "fibe_local_playgrounds_link", description: "[MODE:BROWNFIELD] Link local playground mounts into a working directory.", tier: tierBrownfield,
		annotations: toolAnnotations{Idempotent: true},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			selector, err := localPlaygroundTargetFromArgs(args)
			if err != nil {
				return nil, err
			}
			cliArgs := []any{"local", "playgrounds", "link", selector}
			if linkDir := argString(args, "link_dir"); linkDir != "" {
				cliArgs = append(cliArgs, "--link-dir", linkDir)
			}
			return s.runCobra(ctx, map[string]any{"args": cliArgs})
		},
	}, mcp.NewTool("fibe_local_playgrounds_link",
		mcp.WithDescription("[MODE:BROWNFIELD] Link local playground mounts into a working directory."),
		mcp.WithString("id_or_name", mcp.Description("Local playground ID, name, compose project, playspec, or unique playspec prefix")),
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
			if fibeRunRequiresConfirm(args) && !s.cfg.Yolo && !yoloFromContext(ctx) && !argBool(args, "confirm") {
				return nil, &confirmRequiredError{tool: "fibe_run"}
			}
			return s.runCobra(ctx, args)
		},
	}, mcp.NewTool("fibe_run",
		mcp.WithDescription(`Last-resort escape hatch for arbitrary CLI commands.

Prefer dedicated MCP tools first (for example fibe_launch, fibe_greenfield_create, fibe_playgrounds_*, fibe_props_*, etc.). If the target tool already exists but is not advertised in the current tier, prefer fibe_call over fibe_run.

Use timeout_ms to bound risky calls that might otherwise outlive the host's tool-call budget.`),
		mcp.WithArray("args", mcp.Required(), mcp.WithStringItems(),
			mcp.Description("Command args as if typed after `fibe`. Scalar items (string, number, boolean) are accepted and stringified into CLI tokens in-order.")),
		mcp.WithNumber("timeout_ms", mcp.Description("Optional per-call timeout in milliseconds. Recommended for risky escape-hatch calls.")),
		mcp.WithBoolean("confirm", mcp.Description("Required for delete/destroy/remove CLI paths unless server runs with --yolo.")),
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

// runCobra implements the fibe_run escape hatch. In real MCP sessions it runs
// the configured fibe executable as a subprocess so behavior matches the
// installed CLI. Embedded tests and hosts without CobraExecutable use the
// in-process cobra fallback below.
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
	if legacy := s.legacyFibeRunSyntaxResult(userArgs); legacy != nil {
		return legacy, nil
	}

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
	if s.cfg.CobraExecutable != "" {
		return s.runCobraSubprocess(ctx, strs, userArgs, timeoutMs)
	}

	if s.cfg.CobraRoot == nil {
		return nil, fmt.Errorf("fibe_run not available: server was started without CobraRoot or CobraExecutable")
	}
	return s.runCobraInProcess(ctx, strs, userArgs, timeoutMs)
}

func (s *Server) legacyFibeRunSyntaxResult(userArgs []string) map[string]any {
	if len(userArgs) == 0 {
		return nil
	}
	if resourceResult := legacyResourceCommandResult(userArgs); resourceResult != nil {
		return resourceResult
	}
	if containsCLIFlag(userArgs, "--format") {
		result := map[string]any{
			"args":              userArgs,
			"ok":                false,
			"legacy_cli_syntax": true,
			"unsupported_flag":  "--format",
			"error":             "fibe_run received obsolete CLI syntax: --format is not a Fibe CLI flag",
			"guidance":          "Use --output for raw CLI commands, and prefer dedicated MCP tools over fibe_run. For resource reads, call fibe_resource_list or fibe_resource_get directly.",
		}
		if recommended := s.recommendedToolForCLIArgs(userArgs); recommended != "" {
			result["recommended_tool"] = recommended
			result["warning"] = fmt.Sprintf("prefer %s over fibe_run when possible", recommended)
		}
		return result
	}
	return nil
}

func legacyResourceCommandResult(userArgs []string) map[string]any {
	command := strings.ToLower(strings.TrimSpace(userArgs[0]))
	if command != "resource" && command != "resources" {
		return nil
	}
	result := map[string]any{
		"args":              userArgs,
		"ok":                false,
		"legacy_cli_syntax": true,
		"error":             "fibe_run received obsolete CLI syntax: fibe resource/resources is not a Fibe CLI command",
		"guidance":          "Use the dedicated MCP resource tools instead: fibe_resource_list for list operations and fibe_resource_get for get/inspect operations. The raw CLI uses plural top-level commands and --output, not fibe resource ... --format.",
	}
	if len(userArgs) < 2 {
		result["recommended_tool"] = "fibe_tools_catalog"
		result["recommended_args"] = map[string]any{"name_pattern": "resource"}
		return result
	}

	operation := strings.ToLower(strings.TrimSpace(userArgs[1]))
	var recommendedTool string
	switch operation {
	case "list", "ls":
		recommendedTool = "fibe_resource_list"
	case "get", "show", "inspect":
		recommendedTool = "fibe_resource_get"
	default:
		result["recommended_tool"] = "fibe_tools_catalog"
		result["recommended_args"] = map[string]any{"name_pattern": "resource"}
		return result
	}
	result["recommended_tool"] = recommendedTool

	if len(userArgs) >= 3 {
		if resource, ok := resourceschema.CanonicalResource(userArgs[2]); ok {
			recommendedArgs := map[string]any{"resource": resource}
			if recommendedTool == "fibe_resource_get" && len(userArgs) >= 4 && !strings.HasPrefix(userArgs[3], "-") {
				recommendedArgs["id_or_name"] = userArgs[3]
			}
			result["recommended_args"] = recommendedArgs
		}
	}
	return result
}

func containsCLIFlag(args []string, flag string) bool {
	for _, arg := range args {
		if arg == flag || strings.HasPrefix(arg, flag+"=") {
			return true
		}
	}
	return false
}

func fibeRunRequiresConfirm(args map[string]any) bool {
	rawSlice, ok := args["args"].([]any)
	if !ok {
		return false
	}
	for _, raw := range rawSlice {
		token, err := stringifyCLIArg(raw)
		if err != nil {
			continue
		}
		token = strings.ToLower(strings.TrimSpace(token))
		switch token {
		case "delete", "destroy", "destroy-version", "remove", "remove-mounted-file", "remove-registry-credential":
			return true
		}
	}
	return false
}

func (s *Server) runCobraSubprocess(ctx context.Context, strs []string, userArgs []string, timeoutMs int64) (any, error) {
	s.runMu.Lock()
	defer s.runMu.Unlock()

	limit := fibeRunCaptureLimit()
	var stdoutBuf truncatingBuffer
	stdoutBuf.limit = limit
	var stderrBuf truncatingBuffer
	stderrBuf.limit = limit

	cmd := exec.CommandContext(ctx, s.cfg.CobraExecutable, strs...)
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf
	cmd.Env = os.Environ()
	execErr := cmd.Run()

	result := fibeRunResult(strs, userArgs, timeoutMs, stdoutBuf, stderrBuf, execErr, ctx, s.recommendedToolForCLIArgs(userArgs))
	result["execution_mode"] = "subprocess"
	result["executable"] = s.cfg.CobraExecutable
	return result, nil
}

func (s *Server) runCobraInProcess(ctx context.Context, strs []string, userArgs []string, timeoutMs int64) (any, error) {

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
	limit := fibeRunCaptureLimit()
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

	result := fibeRunResult(strs, userArgs, timeoutMs, stdoutBuf, stderrBuf, execErr, ctx, s.recommendedToolForCLIArgs(userArgs))
	result["execution_mode"] = "embedded"
	return result, nil
}

func fibeRunResult(strs []string, userArgs []string, timeoutMs int64, stdoutBuf truncatingBuffer, stderrBuf truncatingBuffer, execErr error, ctx context.Context, recommended string) map[string]any {
	result := map[string]any{
		"args":   strs,
		"stdout": stdoutBuf.String(),
		"stderr": stderrBuf.String(),
	}
	if timeoutMs > 0 {
		result["timeout_ms"] = timeoutMs
	}
	if recommended != "" {
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
		result["capture_limit_bytes"] = stdoutBuf.limit
	}
	if execErr != nil {
		result["error"] = execErr.Error()
		if errors.Is(execErr, context.DeadlineExceeded) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
			result["timed_out"] = true
		}
	}
	return result
}

func fibeRunCaptureLimit() int {
	limit := fibeRunCaptureMaxBytes
	if limit <= 0 {
		return 1 << 20
	}
	return limit
}

func (s *Server) runCobraArgs(ctx context.Context, args ...string) (any, error) {
	raw := make([]any, len(args))
	for i, arg := range args {
		raw[i] = arg
	}
	return s.runCobra(ctx, map[string]any{"args": raw})
}

func localPlaygroundSelectorFromArgs(args map[string]any, view string) (string, error) {
	selector, err := localPlaygroundOptionalSelectorFromArgs(args)
	if err != nil {
		return "", err
	}
	if view == "names" {
		if selector != "" {
			return "", fmt.Errorf("view 'names' does not accept id_or_name")
		}
		return "", nil
	}
	if selector == "" {
		return "", fmt.Errorf("view '%s' requires id_or_name", view)
	}
	return selector, nil
}

func localPlaygroundTargetFromArgs(args map[string]any) (string, error) {
	selector, err := localPlaygroundOptionalSelectorFromArgs(args)
	if err != nil {
		return "", err
	}
	if selector == "" {
		return "", fmt.Errorf("required field 'id_or_name' not set")
	}
	return selector, nil
}

func localPlaygroundOptionalSelectorFromArgs(args map[string]any) (string, error) {
	return strings.TrimSpace(argString(args, "id_or_name")), nil
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
	if strings.Join(path, " ") == "artefacts create" {
		if _, ok := s.dispatcher.lookup("fibe_artefact_upload"); ok {
			return "fibe_artefact_upload"
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

func (s *Server) doctorAuthMetadata(ctx context.Context) (profile, authSource, domainSource string) {
	st := s.sessionFor(ctx)
	st.mu.RLock()
	sessionProfile, sessionKey, sessionDomain := st.profile, st.apiKey, st.domain
	st.mu.RUnlock()

	profile = sessionProfile
	if profile == "" {
		profile = s.cfg.Profile
	}
	if profile == "" {
		profile = fibe.DefaultProfileName
	}

	authSource = s.cfg.AuthSource
	domainSource = s.cfg.DomainSource
	if sessionProfile != "" {
		source := "profile " + sessionProfile
		authSource = source
		domainSource = source
	} else {
		if sessionKey != "" {
			authSource = "session"
		}
		if sessionDomain != "" {
			domainSource = "session"
		}
	}

	if authSource == "" {
		authSource = inferMCPAuthSource(s.cfg.APIKey, s.cfg.Profile)
	}
	if domainSource == "" {
		domainSource = inferMCPDomainSource(s.cfg.Domain, s.cfg.Profile)
	}
	return profile, authSource, domainSource
}

func inferMCPAuthSource(apiKey, profile string) string {
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return "none"
	}
	if envKey := strings.TrimSpace(os.Getenv("FIBE_API_KEY")); envKey != "" && envKey == apiKey {
		return "FIBE_API_KEY env"
	}
	if profile = strings.TrimSpace(profile); profile != "" {
		return "profile " + profile
	}
	return "server config"
}

func inferMCPDomainSource(domain, profile string) string {
	domain = strings.TrimSpace(domain)
	if domain == "" {
		return "default"
	}
	if envDomain := strings.TrimSpace(os.Getenv("FIBE_DOMAIN")); envDomain != "" && normalizeMCPDomain(envDomain) == normalizeMCPDomain(domain) {
		return "FIBE_DOMAIN env"
	}
	if profile = strings.TrimSpace(profile); profile != "" {
		return "profile " + profile
	}
	return "server config"
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
