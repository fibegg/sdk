package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/fibegg/sdk/fibe"
	"github.com/google/jsonschema-go/jsonschema"
	"github.com/mark3labs/mcp-go/mcp"
)

// withSchemaAndExtras generates a JSON schema from struct P via the jsonschema
// package, then injects extra properties (e.g. "id") into the raw schema.
// This is necessary because mcp.WithInputSchema sets RawInputSchema which
// completely replaces any properties added via WithNumber/WithString.
func withSchemaAndExtras[P any](extras map[string]map[string]any, required ...string) mcp.ToolOption {
	return func(t *mcp.Tool) {
		schema, err := jsonschema.For[P](&jsonschema.ForOptions{IgnoreInvalidTypes: true})
		if err != nil {
			return
		}
		raw, err := json.Marshal(schema)
		if err != nil {
			return
		}
		var m map[string]any
		if err := json.Unmarshal(raw, &m); err != nil {
			return
		}
		props, _ := m["properties"].(map[string]any)
		if props == nil {
			props = map[string]any{}
			m["properties"] = props
		}
		for k, v := range extras {
			props[k] = v
		}
		if len(required) > 0 {
			existing, _ := m["required"].([]any)
			for _, r := range required {
				existing = append(existing, r)
			}
			m["required"] = existing
		}
		patched, err := json.Marshal(m)
		if err != nil {
			return
		}
		t.InputSchema.Type = ""
		t.RawInputSchema = json.RawMessage(patched)
	}
}

// withSchemaEnum injects an "enum" array into a specific property in the RawInputSchema.
// This allows strongly-typed structs to have enum constraints in the MCP schema.
func withSchemaEnum(property string, values []string) mcp.ToolOption {
	return func(t *mcp.Tool) {
		if t.RawInputSchema == nil {
			return
		}
		var m map[string]any
		if err := json.Unmarshal(t.RawInputSchema, &m); err != nil {
			return
		}
		props, ok := m["properties"].(map[string]any)
		if !ok || props == nil {
			return
		}
		prop, ok := props[property].(map[string]any)
		if !ok || prop == nil {
			return
		}

		enumList := make([]any, len(values))
		for i, v := range values {
			enumList[i] = v
		}
		prop["enum"] = enumList
		prop["type"] = "string" // Gemini requires exact "string" type for enums, no nulls allowed

		patched, err := json.Marshal(m)
		if err != nil {
			return
		}
		t.RawInputSchema = json.RawMessage(patched)
	}
}

func withRawInputSchema(schema map[string]any) mcp.ToolOption {
	return func(t *mcp.Tool) {
		raw, err := json.Marshal(schema)
		if err != nil {
			return
		}
		t.InputSchema.Type = ""
		t.RawInputSchema = json.RawMessage(raw)
	}
}

// This file holds the generic registration helpers. Each resource file
// (tools_playgrounds.go, tools_tricks.go, ...) calls into these helpers so
// the uniform "list/get/create/update/delete/action" shapes don't repeat.

// toolOpts bundles per-tool settings that vary across registrations.
type toolOpts struct {
	Tier        toolTier
	ReadOnly    bool
	Destructive bool
	Idempotent  bool
	Hidden      bool
	IDField     string
	// ExtraSchemaProps lets a caller add extra input properties on top of
	// the ones derived from the generic parameter type (e.g., confirm:true
	// for destructive ops).
	ExtraSchemaProps []mcp.ToolOption
	// Aliases maps a canonical arg name to the historical alternatives
	// clients may pass. Before the handler runs, any alternative that is
	// set and whose canonical counterpart is not set gets copied under the
	// canonical name. Prevents field-name drift between CLI flags / schema
	// docs / MCP input shapes from turning into hard errors.
	Aliases map[string][]string
}

// applyAliases runs the configured alias canonicalization on an args map.
// Safe to call with nil opts.Aliases.
func applyAliases(args map[string]any, aliases map[string][]string) {
	for canonical, alts := range aliases {
		aliasField(args, canonical, alts...)
	}
}

func toolIDField(opts toolOpts) string {
	if opts.IDField != "" {
		return opts.IDField
	}
	return "id"
}

// addTool wires a toolImpl into both the dispatcher and the mcp-go server.
// All registration helpers funnel through here so the MCP ↔ dispatcher
// plumbing is in one place.
//
// Tools are always registered on the dispatcher so fibe_pipeline and
// fibe_call can reference any tool by name regardless of the configured
// advertised surface. The mcp-go tool registry is filtered by tier so the
// toolset advertised to clients respects FIBE_MCP_TOOLS.
func (s *Server) addTool(t *toolImpl, tool mcp.Tool) {
	s.dispatcher.register(t)

	// Apply every annotation explicitly. mcp-go defaults destructiveHint to
	// true, which would otherwise make read-only tools look destructive.
	mcp.WithReadOnlyHintAnnotation(t.annotations.ReadOnly)(&tool)
	mcp.WithDestructiveHintAnnotation(t.annotations.Destructive)(&tool)
	mcp.WithIdempotentHintAnnotation(t.annotations.Idempotent)(&tool)
	enrichToolInputSchema(t.name, &tool)
	if schema, ok := toolInputSchemaToMap(tool).(map[string]any); ok {
		s.toolSchemas[t.name] = schema
	}

	// Tier gating: hidden or non-selected tier tools remain dispatcher
	// reachable but are not advertised natively to the MCP client.
	if !s.includeTool(t) {
		return
	}

	s.mcp.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()
		if args == nil {
			args = map[string]any{}
		}

		start := time.Now()
		result, err := s.dispatcher.dispatch(ctx, t.name, args)
		s.auditLog(ctx, t.name, args, err, time.Since(start))
		if err != nil {
			return toolResultFromError(t.name, err), nil
		}
		if result == nil {
			return mcp.NewToolResultText("{}"), nil
		}
		data, err := json.Marshal(result)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("marshal result: %v", err)), nil
		}
		return mcp.NewToolResultText(string(data)), nil
	})
}

// toolResultFromError preserves the structured fields of *fibe.APIError
// (code, request_id, status, details) when surfacing errors to MCP hosts.
// Agents then have enough context to decide whether to retry, branch, or
// surface the failure to the user.
func toolResultFromError(toolName string, err error) *mcp.CallToolResult {
	payload := map[string]any{
		"tool":    toolName,
		"message": err.Error(),
	}

	if apiErr, ok := err.(*fibe.APIError); ok {
		payload["code"] = apiErr.Code
		payload["status"] = apiErr.StatusCode
		payload["message"] = apiErr.Message
		if apiErr.RequestID != "" {
			payload["request_id"] = apiErr.RequestID
		}
		if apiErr.Details != nil {
			payload["details"] = apiErr.Details
		}
		if apiErr.IdempotentReplayed {
			payload["idempotent_replayed"] = true
		}
		if apiErr.RetryAfter > 0 {
			payload["retry_after_seconds"] = apiErr.RetryAfter.Seconds()
		}
	} else if cbErr, ok := err.(*fibe.CircuitOpenError); ok {
		payload["code"] = "CIRCUIT_OPEN"
		payload["resource"] = cbErr.Resource
	} else if _, ok := err.(*confirmRequiredError); ok {
		payload["code"] = "CONFIRM_REQUIRED"
		payload["hint"] = "pass confirm:true or run server with --yolo"
	}

	body, mErr := json.Marshal(payload)
	if mErr != nil {
		return mcp.NewToolResultError(err.Error())
	}
	// Use NewToolResultError so hosts flag this as a tool failure. We embed
	// the structured payload as the error body so agents get everything in
	// one string to parse.
	return mcp.NewToolResultError(string(body))
}

// registerList registers a list endpoint. The generic type P is the list
// params struct; R is the element type inside ListResult.
func registerList[P any, R any](s *Server, name, desc string, opts toolOpts,
	fn func(ctx context.Context, c *fibe.Client, params *P) (*fibe.ListResult[R], error)) {

	t := &toolImpl{
		name:        name,
		description: desc,
		tier:        opts.Tier,
		hidden:      opts.Hidden,
		annotations: toolAnnotations{ReadOnly: true, Idempotent: true},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			applyAliases(args, opts.Aliases)
			var p P
			if err := bindArgs(args, &p); err != nil {
				return nil, err
			}
			return fn(ctx, c, &p)
		},
	}

	mcpOpts := []mcp.ToolOption{
		mcp.WithDescription(desc),
		mcp.WithInputSchema[P](),
	}
	mcpOpts = append(mcpOpts, opts.ExtraSchemaProps...)
	tool := mcp.NewTool(name, mcpOpts...)
	s.addTool(t, tool)
}


// registerCreate registers a create endpoint that takes only the params.
func registerCreate[P any, R any](s *Server, name, desc string, opts toolOpts,
	fn func(ctx context.Context, c *fibe.Client, params *P) (*R, error)) {

	t := &toolImpl{
		name:        name,
		description: desc,
		tier:        opts.Tier,
		hidden:      opts.Hidden,
		annotations: toolAnnotations{Idempotent: opts.Idempotent},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			applyAliases(args, opts.Aliases)
			var p P
			if err := bindArgs(args, &p); err != nil {
				return nil, err
			}
			return fn(ctx, c, &p)
		},
	}

	mcpOpts := []mcp.ToolOption{
		mcp.WithDescription(desc),
		mcp.WithInputSchema[P](),
	}
	mcpOpts = append(mcpOpts, opts.ExtraSchemaProps...)
	tool := mcp.NewTool(name, mcpOpts...)
	s.addTool(t, tool)
}

// registerCreateNested registers a create endpoint scoped by a parent ID.
func registerCreateNested[P any, R any](s *Server, name, desc, parentIDField string, opts toolOpts,
	fn func(ctx context.Context, c *fibe.Client, parentID int64, params *P) (*R, error)) {

	t := &toolImpl{
		name:        name,
		description: desc,
		tier:        opts.Tier,
		hidden:      opts.Hidden,
		annotations: toolAnnotations{Idempotent: opts.Idempotent},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			applyAliases(args, opts.Aliases)
			pid, ok := argInt64(args, parentIDField)
			if !ok {
				return nil, fmt.Errorf("required field %q not set", parentIDField)
			}
			var p P
			if err := bindArgs(args, &p); err != nil {
				return nil, err
			}
			return fn(ctx, c, pid, &p)
		},
	}

	tool := mcp.NewTool(name,
		mcp.WithDescription(desc),
		mcp.WithNumber(parentIDField, mcp.Required(), mcp.Description(idDescription(parentIDField))),
	)
	s.addTool(t, tool)
}

// registerUpdate registers an update-by-id endpoint.
func registerUpdate[P any, R any](s *Server, name, desc string, opts toolOpts,
	fn func(ctx context.Context, c *fibe.Client, id int64, params *P) (*R, error)) {

	t := &toolImpl{
		name:        name,
		description: desc,
		tier:        opts.Tier,
		hidden:      opts.Hidden,
		annotations: toolAnnotations{Idempotent: true},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			applyAliases(args, opts.Aliases)
			id, ok := argInt64(args, "id")
			if !ok {
				return nil, fmt.Errorf("required field 'id' not set")
			}
			if !hasUpdateFields(args, "id") {
				// Reject "only id" updates locally so the agent gets a clean
				// "nothing to update" error instead of a server-side 400
				// from Rails' require(:resource) on an empty param wrapper.
				return nil, fmt.Errorf("%s: pass at least one field to update (update tools reject empty payloads to avoid 400 responses)", name)
			}
			var p P
			if err := bindArgs(args, &p); err != nil {
				return nil, err
			}
			return fn(ctx, c, id, &p)
		},
	}

	mcpOpts := []mcp.ToolOption{
		mcp.WithDescription(desc),
		withSchemaAndExtras[P](map[string]map[string]any{
			"id": {"type": "number", "description": idDescription("id")},
		}, "id"),
	}
	mcpOpts = append(mcpOpts, opts.ExtraSchemaProps...)
	tool := mcp.NewTool(name, mcpOpts...)
	s.addTool(t, tool)
}

// registerUpdateNested registers an update endpoint scoped by a parent ID.
func registerUpdateNested[P any, R any](s *Server, name, desc, parentIDField string, opts toolOpts,
	fn func(ctx context.Context, c *fibe.Client, parentID, id int64, params *P) (*R, error)) {

	t := &toolImpl{
		name:        name,
		description: desc,
		tier:        opts.Tier,
		hidden:      opts.Hidden,
		annotations: toolAnnotations{Idempotent: true},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			applyAliases(args, opts.Aliases)
			pid, ok := argInt64(args, parentIDField)
			if !ok {
				return nil, fmt.Errorf("required field %q not set", parentIDField)
			}
			id, ok := argInt64(args, "id")
			if !ok {
				return nil, fmt.Errorf("required field 'id' not set")
			}
			if !hasUpdateFields(args, parentIDField, "id") {
				return nil, fmt.Errorf("%s: pass at least one field to update (update tools reject empty payloads to avoid 400 responses)", name)
			}
			var p P
			if err := bindArgs(args, &p); err != nil {
				return nil, err
			}
			return fn(ctx, c, pid, id, &p)
		},
	}

	tool := mcp.NewTool(name,
		mcp.WithDescription(desc),
		withSchemaAndExtras[P](map[string]map[string]any{
			parentIDField: {"type": "number", "description": idDescription(parentIDField)},
			"id":          {"type": "number", "description": idDescription("id")},
		}, parentIDField, "id"),
	)
	s.addTool(t, tool)
}

// registerDelete registers a destructive delete-by-id endpoint.
// confirm:true is required at call time unless --yolo is set.
func registerDelete(s *Server, name, desc string, opts toolOpts,
	fn func(ctx context.Context, c *fibe.Client, id int64) error) {

	t := &toolImpl{
		name:        name,
		description: desc,
		tier:        opts.Tier,
		hidden:      opts.Hidden,
		annotations: toolAnnotations{Destructive: true, Idempotent: true},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			applyAliases(args, opts.Aliases)
			id, ok := argInt64(args, "id")
			if !ok {
				return nil, fmt.Errorf("required field 'id' not set")
			}
			if err := fn(ctx, c, id); err != nil {
				return nil, err
			}
			return map[string]any{"id": id, "deleted": true}, nil
		},
	}

	tool := mcp.NewTool(name,
		mcp.WithDescription(desc),
		mcp.WithNumber("id", mcp.Required(), mcp.Description(idDescription("id"))),
		mcp.WithBoolean("confirm", mcp.Description("Must be true unless server is running with --yolo")),
	)
	s.addTool(t, tool)
}

// registerDeleteNested registers a destructive delete under a parent.
func registerDeleteNested(s *Server, name, desc, parentIDField string, opts toolOpts,
	fn func(ctx context.Context, c *fibe.Client, parentID, id int64) error) {

	t := &toolImpl{
		name:        name,
		description: desc,
		tier:        opts.Tier,
		hidden:      opts.Hidden,
		annotations: toolAnnotations{Destructive: true, Idempotent: true},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			applyAliases(args, opts.Aliases)
			pid, ok := argInt64(args, parentIDField)
			if !ok {
				return nil, fmt.Errorf("required field %q not set", parentIDField)
			}
			idField := toolIDField(opts)
			id, ok := argInt64(args, idField)
			if !ok {
				return nil, fmt.Errorf("required field %q not set", idField)
			}
			if err := fn(ctx, c, pid, id); err != nil {
				return nil, err
			}
			return map[string]any{parentIDField: pid, idField: id, "deleted": true}, nil
		},
	}
	idField := toolIDField(opts)

	tool := mcp.NewTool(name,
		mcp.WithDescription(desc),
		mcp.WithNumber(parentIDField, mcp.Required(), mcp.Description(idDescription(parentIDField))),
		mcp.WithNumber(idField, mcp.Required(), mcp.Description(idDescription(idField))),
		mcp.WithBoolean("confirm", mcp.Description("Must be true unless server is running with --yolo")),
	)
	s.addTool(t, tool)
}

// registerIDAction registers a POST action on a resource identified by ID
// (e.g., Rollout, HardRestart). The action takes only the ID; pass
// destructive:true in opts for operations that mutate in place.
func registerIDAction[R any](s *Server, name, desc string, opts toolOpts,
	fn func(ctx context.Context, c *fibe.Client, id int64) (*R, error)) {

	t := &toolImpl{
		name:        name,
		description: desc,
		tier:        opts.Tier,
		hidden:      opts.Hidden,
		annotations: toolAnnotations{Destructive: opts.Destructive, Idempotent: true},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			applyAliases(args, opts.Aliases)
			idField := toolIDField(opts)
			id, ok := argInt64(args, idField)
			if !ok {
				return nil, fmt.Errorf("required field %q not set", idField)
			}
			return fn(ctx, c, id)
		},
	}
	idField := toolIDField(opts)

	toolOpts := []mcp.ToolOption{
		mcp.WithDescription(desc),
		mcp.WithNumber(idField, mcp.Required(), mcp.Description(idDescription(idField))),
	}
	if opts.Destructive {
		toolOpts = append(toolOpts, mcp.WithBoolean("confirm",
			mcp.Description("Must be true unless server is running with --yolo")))
	}
	tool := mcp.NewTool(name, toolOpts...)
	s.addTool(t, tool)
}

// registerIDActionNoReturn is like registerIDAction for operations that return
// only error (the resource is affected but no body is returned).
func registerIDActionNoReturn(s *Server, name, desc string, opts toolOpts,
	fn func(ctx context.Context, c *fibe.Client, id int64) error) {

	t := &toolImpl{
		name:        name,
		description: desc,
		tier:        opts.Tier,
		hidden:      opts.Hidden,
		annotations: toolAnnotations{Destructive: opts.Destructive, Idempotent: true},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			applyAliases(args, opts.Aliases)
			idField := toolIDField(opts)
			id, ok := argInt64(args, idField)
			if !ok {
				return nil, fmt.Errorf("required field %q not set", idField)
			}
			if err := fn(ctx, c, id); err != nil {
				return nil, err
			}
			return map[string]any{idField: id, "ok": true}, nil
		},
	}
	idField := toolIDField(opts)

	toolOpts := []mcp.ToolOption{
		mcp.WithDescription(desc),
		mcp.WithNumber(idField, mcp.Required(), mcp.Description(idDescription(idField))),
	}
	if opts.Destructive {
		toolOpts = append(toolOpts, mcp.WithBoolean("confirm",
			mcp.Description("Must be true unless server is running with --yolo")))
	}
	tool := mcp.NewTool(name, toolOpts...)
	s.addTool(t, tool)
}

// registerCustom registers a fully custom tool with a hand-written handler
// and schema. Use this for tools that don't fit the CRUD molds
// (fibe_playgrounds_wait, fibe_pipeline, etc.).
func registerCustom(s *Server, t *toolImpl, tool mcp.Tool) {
	s.addTool(t, tool)
}

// hasUpdateFields reports whether the caller provided at least one field
// besides the routing keys (parent ID, id) and the transport-level confirm
// flag. Rails' params.require(:resource).permit(...) raises 400 when the
// permitted sub-hash is empty, so we catch empty updates locally to surface
// a cleaner error.
func hasUpdateFields(args map[string]any, routingKeys ...string) bool {
	skip := map[string]bool{"confirm": true}
	for _, k := range routingKeys {
		skip[k] = true
	}
	for k, v := range args {
		if skip[k] {
			continue
		}
		// Treat explicit null / empty-string / empty-array as "not set"
		// because some clients serialize optional fields that way.
		if v == nil {
			continue
		}
		if s, ok := v.(string); ok && s == "" {
			continue
		}
		return true
	}
	return false
}
