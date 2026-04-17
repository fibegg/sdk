package mcpserver

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/PaesslerAG/jsonpath"
	"github.com/fibegg/sdk/fibe"
	"github.com/mark3labs/mcp-go/mcp"
)

// registerPipelineTools wires fibe_pipeline and fibe_pipeline_result.
//
// fibe_pipeline composes multiple tool calls into one round-trip, with
// JSONPath bindings ($.step_id.field) passing data between steps. The final
// result is cached per session for 5 minutes and addressable via
// fibe_pipeline_result.
func (s *Server) registerPipelineTools() {
	s.addTool(&toolImpl{
		name: "fibe_pipeline", description: "Execute multiple tool calls sequentially in a single round-trip using JSONPath bindings", tier: tierMeta,
		annotations: toolAnnotations{},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			return s.runPipeline(ctx, args)
		},
	}, mcp.NewTool("fibe_pipeline",
		mcp.WithDescription(`Compose multiple Fibe tool calls into a single MCP round-trip.

STEPS:
  Each step is {id, tool, args} or {parallel: [...]} or {for_each: "$.list", as, steps, collect}.
  Args may contain JSONPath expressions beginning with "$." — they resolve against a map of
  prior step outputs. Example: "$.pg.id" references the "pg" step's result field "id".

LIMITS:
  Default max 25 steps per pipeline (configurable via FIBE_MCP_PIPELINE_MAX_STEPS).
  Default max 50 total iterations inside for_each blocks.

RETURN:
  Default return is a map of every step's output. Set "return" to a JSONPath string or an
  object literal with JSONPath values to project only what you need.

CACHING:
  The full result tree is cached per session for 5 minutes under a returned pipeline_id —
  including partial results from pipelines that errored mid-run. Use fibe_pipeline_result
  to re-query specific fields later without rerunning. Successful runs have status:"completed";
  mid-run failures have status:"partial" with an "error" block identifying the failed step
  and a "completed_step_ids" array so you can garbage-collect created resources.

SAFETY:
  Destructive tools inside a pipeline still need confirm:true unless the server is in --yolo mode.
  fibe_pipeline cannot nest (no pipelines inside pipelines).
`),
		mcp.WithArray("steps", mcp.Required(),
			mcp.Description(`Ordered list of steps to execute.

Each step is one of:
  {"id": "<step_id>", "tool": "<tool_name>", "args": {...}}              (single tool call)
  {"parallel": [<step>, <step>, ...]}                                   (independent concurrent steps)
  {"id": "<step_id>", "for_each": "$.list", "as": "item",
    "steps": [<step>, ...], "collect": "$.something"}                   (fanout)

Args may contain JSONPath references starting with "$.", resolved against the map of prior step outputs.`),
			// Gemini's schema validator requires items to be declared on
			// every array; other MCP hosts are looser. We describe steps as
			// a permissive object — the runtime validates the polymorphic
			// shape (tool / parallel / for_each) itself.
			withObjectItems(stepSchema()),
		),
		mcp.WithString("return", mcp.Description("Optional JSONPath or object spec to project the final return")),
		mcp.WithBoolean("dry_run", mcp.Description("Validate refs + schemas without executing")),
		mcp.WithBoolean("cache", mcp.Description("Set to false to skip caching this pipeline's result (default: true)")),
		mcp.WithString("idempotency_key", mcp.Description("Optional key threaded through destructive steps for retry safety")),
	))

	s.addTool(&toolImpl{
		name: "fibe_pipeline_result", description: "Look up a cached result from a previous pipeline execution", tier: tierMeta,
		annotations: toolAnnotations{ReadOnly: true, Idempotent: true},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			pid := argString(args, "pipeline_id")
			if pid == "" {
				return nil, fmt.Errorf("required field 'pipeline_id' not set")
			}
			path := argString(args, "path")
			sess := s.sessionFor(ctx)
			payload, ok := s.cache.Get(sess.sessionID, pid)
			if !ok {
				return map[string]any{"expired": true, "pipeline_id": pid}, nil
			}
			if path == "" {
				var out any
				if err := json.Unmarshal(payload, &out); err != nil {
					return nil, err
				}
				return out, nil
			}
			return projectCached(payload, path)
		},
	}, mcp.NewTool("fibe_pipeline_result",
		mcp.WithDescription(`Look up a cached fibe_pipeline result. Results are retained per session for 5 minutes.

PATH RESOLUTION:
  path is a JSONPath expression rooted at the pipeline's step outputs (bindings).
  Example: path="$.create_team.id" returns the "create_team" step's "id" field.
  If the path doesn't resolve inside bindings, it's retried against the full cached
  response — so "$.status", "$.error", "$.completed_step_ids" also work.

Returns {expired: true} if the ID is unknown or the entry was evicted.`),
		mcp.WithString("pipeline_id", mcp.Required(), mcp.Description("ID returned by a prior fibe_pipeline call")),
		mcp.WithString("path", mcp.Description("Optional JSONPath. Rooted at step bindings; falls back to full response for \"$.status\" / \"$.error\" etc.")),
	))
}

// withObjectItems is a PropertyOption that sets the "items" schema on an
// array property. mcp-go ships WithStringItems/WithNumberItems/WithBooleanItems
// but nothing for object items — and Gemini's validator rejects arrays that
// have no items at all.
func withObjectItems(itemsSchema map[string]any) mcp.PropertyOption {
	return func(m map[string]any) {
		m["items"] = itemsSchema
	}
}

// stepSchema returns the recursive schema describing a single pipeline
// step. Gemini's validator walks every nested array and demands an items
// declaration, so we make sure every inner "parallel" and "steps" array
// carries one.
//
// The schema is intentionally permissive: the runtime validates the
// polymorphic tool / parallel / for_each shape itself. Describing it with a
// strict oneOf would bloat the schema without giving the agent better
// guidance than the free-form description does.
func stepSchema() map[string]any {
	// Inner object schema used as items for arrays of sub-steps. We stop
	// one level deep — enough for Gemini to accept the schema without
	// exploding into a fully recursive definition.
	innerStep := map[string]any{
		"type":        "object",
		"description": "A nested pipeline step.",
		"properties": map[string]any{
			"id":          map[string]any{"type": "string"},
			"tool":        map[string]any{"type": "string"},
			"args":        map[string]any{"type": "object"},
			"for_each":    map[string]any{"type": "string"},
			"as":          map[string]any{"type": "string"},
			"collect":     map[string]any{"type": "string"},
			"on_error":    map[string]any{"type": "string", "enum": []string{"abort", "continue"}},
			"input_path":  map[string]any{"type": "string"},
			"output_path": map[string]any{"type": "string"},
		},
	}

	return map[string]any{
		"type":        "object",
		"description": "A pipeline step: tool call, parallel block, or for_each fanout.",
		"properties": map[string]any{
			"id":   map[string]any{"type": "string"},
			"tool": map[string]any{"type": "string"},
			"args": map[string]any{"type": "object"},
			"parallel": map[string]any{
				"type":        "array",
				"description": "Sub-steps to execute concurrently.",
				"items":       innerStep,
			},
			"for_each": map[string]any{"type": "string"},
			"as":       map[string]any{"type": "string"},
			"steps": map[string]any{
				"type":        "array",
				"description": "Sub-steps executed per for_each iteration.",
				"items":       innerStep,
			},
			"collect":     map[string]any{"type": "string"},
			"on_error":    map[string]any{"type": "string", "enum": []string{"abort", "continue"}},
			"input_path":  map[string]any{"type": "string"},
			"output_path": map[string]any{"type": "string"},
		},
	}
}

// pipelineRequest is the DSL envelope.
type pipelineRequest struct {
	Steps          []pipelineStep `json:"steps"`
	Return         any            `json:"return,omitempty"` // string (JSONPath) or object literal
	DryRun         bool           `json:"dry_run,omitempty"`
	Cache          *bool          `json:"cache,omitempty"` // defaults to true
	IdempotencyKey string         `json:"idempotency_key,omitempty"`
}

// pipelineStep is a single step. Exactly one of Tool / Parallel / ForEach
// must be set.
type pipelineStep struct {
	ID       string         `json:"id,omitempty"`
	Tool     string         `json:"tool,omitempty"`
	Args     map[string]any `json:"args,omitempty"`
	Parallel []pipelineStep `json:"parallel,omitempty"`

	// for_each semantics
	ForEach string         `json:"for_each,omitempty"`
	As      string         `json:"as,omitempty"`
	Steps   []pipelineStep `json:"steps,omitempty"`
	Collect string         `json:"collect,omitempty"` // optional JSONPath on each iteration's final output

	// Error handling
	OnError string `json:"on_error,omitempty"` // abort (default) | continue

	// input_path/output_path filtering (Tool Chainer compatibility)
	InputPath  string `json:"input_path,omitempty"`
	OutputPath string `json:"output_path,omitempty"`
}

func (s *Server) runPipeline(ctx context.Context, raw map[string]any) (any, error) {
	// Refuse nested pipelines at the top level — a fibe_pipeline step that
	// invokes fibe_pipeline is caught by dispatcher lookup later. The
	// dispatcher will return "unknown tool" if nesting is disallowed in
	// dispatch; we handle that at step-invocation time too.

	var req pipelineRequest
	if err := bindArgs(raw, &req); err != nil {
		return nil, fmt.Errorf("invalid pipeline envelope: %w", err)
	}
	if len(req.Steps) == 0 {
		return nil, errors.New("pipeline has no steps")
	}
	if len(req.Steps) > s.cfg.PipelineMaxSteps && s.cfg.PipelineMaxSteps > 0 {
		return nil, fmt.Errorf("pipeline exceeds max steps (%d)", s.cfg.PipelineMaxSteps)
	}

	runner := &pipelineRunner{
		srv:            s,
		ctx:            ctx,
		bindings:       map[string]any{},
		maxSteps:       s.cfg.PipelineMaxSteps,
		maxIter:        s.cfg.PipelineMaxIterations,
		dryRun:         req.DryRun,
		idempotencyKey: req.IdempotencyKey,
	}

	// Execute each step. On failure, we STOP the pipeline but still return
	// whatever bindings accumulated so far so the agent can reuse the prior
	// step outputs (e.g., freshly-created IDs) for manual cleanup. Without
	// this, a create→update→delete pipeline that fails at "update" loses
	// the "create" IDs entirely and forces agents to garbage-collect blind.
	var failure *pipelineFailure
	for i, step := range req.Steps {
		if err := runner.execStep(i, step, runner.bindings); err != nil {
			failure = newPipelineFailure(i, step, err)
			break
		}
	}

	var final any
	if failure == nil {
		projected, err := projectReturn(runner.bindings, req.Return)
		if err != nil {
			return nil, err
		}
		final = projected
	}

	// Cache if enabled. We cache both successful and partial runs so the
	// agent can re-query prior outputs via fibe_pipeline_result regardless
	// of whether the pipeline ran to completion.
	cacheOn := true
	if req.Cache != nil {
		cacheOn = *req.Cache
	}
	resp := map[string]any{
		"steps": runner.bindings,
	}
	if failure == nil {
		resp["result"] = final
		resp["status"] = "completed"
	} else {
		resp["result"] = nil
		resp["status"] = "partial"
		resp["error"] = failure.toMap()
		resp["completed_step_ids"] = collectStepIDs(req.Steps[:failure.index])
	}

	if cacheOn && !req.DryRun {
		sess := s.sessionFor(ctx)
		pid, truncated, err := s.cache.Put(sess.sessionID, resp)
		if err == nil && pid != "" {
			resp["pipeline_id"] = pid
			if truncated {
				resp["truncated"] = true
			}
		}
	}
	if req.DryRun {
		resp["dry_run"] = true
	}

	return resp, nil
}

// pipelineFailure describes the step that halted a pipeline. It surfaces
// the failed step's index, ID, tool name, and the full structured error so
// agents can decide whether to retry, clean up, or escalate.
type pipelineFailure struct {
	index  int
	stepID string
	tool   string
	err    error
}

func newPipelineFailure(index int, step pipelineStep, err error) *pipelineFailure {
	return &pipelineFailure{
		index:  index,
		stepID: step.ID,
		tool:   step.Tool,
		err:    err,
	}
}

func (f *pipelineFailure) toMap() map[string]any {
	out := map[string]any{
		"step_index": f.index,
		"step_id":    f.stepID,
		"tool":       f.tool,
		"message":    f.err.Error(),
	}
	// Unwrap *fibe.APIError so agents can branch on code / status without
	// string-matching. Mirrors the toolResultFromError surface.
	var apiErr *fibe.APIError
	if errors.As(f.err, &apiErr) {
		out["code"] = apiErr.Code
		out["status"] = apiErr.StatusCode
		if apiErr.RequestID != "" {
			out["request_id"] = apiErr.RequestID
		}
		if apiErr.Details != nil {
			out["details"] = apiErr.Details
		}
	}
	return out
}

// collectStepIDs returns the IDs of the steps that ran successfully
// (ignoring parallel/for_each blocks that don't have IDs).
func collectStepIDs(steps []pipelineStep) []string {
	ids := make([]string, 0, len(steps))
	for _, s := range steps {
		if s.ID != "" {
			ids = append(ids, s.ID)
		}
	}
	return ids
}

// pipelineRunner holds per-invocation state.
type pipelineRunner struct {
	srv            *Server
	ctx            context.Context
	mu             sync.Mutex
	bindings       map[string]any
	maxSteps       int
	maxIter        int
	iterUsed       int
	dryRun         bool
	idempotencyKey string
}

func (r *pipelineRunner) execStep(idx int, step pipelineStep, scope map[string]any) error {
	switch {
	case len(step.Parallel) > 0:
		return r.execParallel(step.Parallel, scope)
	case step.ForEach != "":
		return r.execForEach(step, scope)
	case step.Tool != "":
		return r.execTool(step, scope)
	}
	return fmt.Errorf("malformed step (expected tool, parallel, or for_each)")
}

func (r *pipelineRunner) execTool(step pipelineStep, scope map[string]any) error {
	if step.Tool == "fibe_pipeline" {
		return errors.New("nested fibe_pipeline is not allowed")
	}

	resolved, err := resolveArgs(step.Args, scope)
	if err != nil {
		return fmt.Errorf("resolve args for %q: %w", step.ID, err)
	}

	// input_path filters the resolved args down to the subtree named by the path.
	if step.InputPath != "" {
		filtered, err := projectOnMap(resolved, step.InputPath)
		if err == nil {
			if m, ok := filtered.(map[string]any); ok {
				resolved = m
			}
		}
	}

	if r.dryRun {
		r.mu.Lock()
		if step.ID != "" {
			r.bindings[step.ID] = map[string]any{"dry_run": true, "tool": step.Tool, "args": resolved}
		}
		r.mu.Unlock()
		return nil
	}

	// If the pipeline carries an idempotency_key, derive a stable per-step
	// key and attach it to the ctx. The SDK picks this up via
	// fibe.WithIdempotencyKey; the server caches the response for 24h so a
	// pipeline retry won't re-create/rollout/hard-restart the same resource.
	stepCtx := r.ctx
	if r.idempotencyKey != "" && step.ID != "" {
		sum := sha256.Sum256([]byte(r.idempotencyKey + ":" + step.ID))
		stepCtx = fibe.WithIdempotencyKey(r.ctx, hex.EncodeToString(sum[:16]))
	}

	out, err := r.srv.dispatcher.dispatch(stepCtx, step.Tool, resolved)
	if err != nil {
		if step.OnError == "continue" {
			r.mu.Lock()
			if step.ID != "" {
				r.bindings[step.ID] = map[string]any{"error": err.Error()}
			}
			r.mu.Unlock()
			return nil
		}
		return fmt.Errorf("%s: %w", step.Tool, err)
	}

	// Normalize the tool result into plain JSON values (map[string]any,
	// []any, primitives) so the PaesslerAG/jsonpath engine can walk it in
	// subsequent steps. Without this, a tool returning a typed struct
	// pointer (e.g. *fibe.Team) crashes later steps with
	//   "unsupported value type *fibe.Team for select,
	//    expected map[string]interface{} or []interface{}"
	// because jsonpath only understands the generic JSON shape.
	normalized, nerr := normalizeForJSONPath(out)
	if nerr != nil {
		return fmt.Errorf("%s: normalize result for pipeline bindings: %w", step.Tool, nerr)
	}

	// output_path projects the normalized result before storing.
	if step.OutputPath != "" {
		normalized = mustProject(normalized, step.OutputPath)
	}

	r.mu.Lock()
	if step.ID != "" {
		r.bindings[step.ID] = normalized
	}
	r.mu.Unlock()
	return nil
}

func (r *pipelineRunner) execParallel(steps []pipelineStep, scope map[string]any) error {
	var wg sync.WaitGroup
	errCh := make(chan error, len(steps))
	for i, st := range steps {
		i, st := i, st
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := r.execStep(i, st, scope); err != nil {
				errCh <- err
			}
		}()
	}
	wg.Wait()
	close(errCh)
	for e := range errCh {
		if e != nil {
			return e
		}
	}
	return nil
}

func (r *pipelineRunner) execForEach(step pipelineStep, scope map[string]any) error {
	if step.As == "" {
		return errors.New("for_each requires 'as'")
	}
	if len(step.Steps) == 0 {
		return errors.New("for_each requires 'steps'")
	}

	listVal, err := projectOnMap(r.bindings, step.ForEach)
	if err != nil {
		return fmt.Errorf("for_each resolve %q: %w", step.ForEach, err)
	}
	items, ok := listVal.([]any)
	if !ok {
		return fmt.Errorf("for_each value is not an array: %q", step.ForEach)
	}

	results := make([]any, 0, len(items))
	for _, item := range items {
		if r.maxIter > 0 && r.iterUsed >= r.maxIter {
			return fmt.Errorf("for_each exceeded max iterations (%d)", r.maxIter)
		}
		r.iterUsed++

		iterScope := map[string]any{}
		r.mu.Lock()
		for k, v := range r.bindings {
			iterScope[k] = v
		}
		r.mu.Unlock()
		iterScope[step.As] = item

		inner := &pipelineRunner{
			srv:      r.srv,
			ctx:      r.ctx,
			bindings: iterScope,
			maxSteps: r.maxSteps,
			maxIter:  r.maxIter,
			iterUsed: r.iterUsed,
			dryRun:   r.dryRun,
		}
		for i, s := range step.Steps {
			if err := inner.execStep(i, s, iterScope); err != nil {
				return fmt.Errorf("for_each[%v]: %w", item, err)
			}
		}
		r.iterUsed = inner.iterUsed

		// Collect projection
		var collected any = iterScope
		if step.Collect != "" {
			projected, err := projectOnMap(iterScope, step.Collect)
			if err == nil {
				collected = projected
			}
		}
		results = append(results, collected)
	}

	r.mu.Lock()
	if step.ID != "" {
		r.bindings[step.ID] = results
	}
	r.mu.Unlock()
	return nil
}

// resolveArgs walks an args map and replaces every string value beginning
// with "$." with the JSONPath lookup result from scope. "$$." escapes to a
// literal "$." string.
func resolveArgs(args map[string]any, scope map[string]any) (map[string]any, error) {
	if args == nil {
		return map[string]any{}, nil
	}
	out := make(map[string]any, len(args))
	for k, v := range args {
		resolved, err := resolveValue(v, scope)
		if err != nil {
			return nil, err
		}
		out[k] = resolved
	}
	return out, nil
}

func resolveValue(v any, scope map[string]any) (any, error) {
	switch x := v.(type) {
	case string:
		if strings.HasPrefix(x, "$$.") {
			return "$" + x[2:], nil
		}
		if strings.HasPrefix(x, "$.") {
			return projectOnMap(scope, x)
		}
		return x, nil
	case map[string]any:
		out := make(map[string]any, len(x))
		for k, vv := range x {
			rv, err := resolveValue(vv, scope)
			if err != nil {
				return nil, err
			}
			out[k] = rv
		}
		return out, nil
	case []any:
		out := make([]any, len(x))
		for i, vv := range x {
			rv, err := resolveValue(vv, scope)
			if err != nil {
				return nil, err
			}
			out[i] = rv
		}
		return out, nil
	}
	return v, nil
}

// projectOnMap evaluates a JSONPath against an in-memory map.
// PaesslerAG/jsonpath supports v1 expressions.
func projectOnMap(data any, path string) (any, error) {
	return jsonpath.Get(path, data)
}

// normalizeForJSONPath converts an arbitrary Go value (typically a struct
// pointer returned by the SDK) into the subset jsonpath can walk:
// map[string]any, []any, and JSON primitives. Implemented as a JSON
// round-trip so we honor the same field projection and omitempty behavior
// the rest of the server already exposes to clients.
//
// This is the single point that prevents the "pipeline step 2 can't read
// step 1's fields" class of bug. It runs on every step output; keep it
// cheap — the SDK responses are already JSON-shaped so the marshal side is
// fast.
func normalizeForJSONPath(v any) (any, error) {
	if v == nil {
		return nil, nil
	}
	// Fast path: scalar primitives only. We deliberately do NOT fast-path
	// maps / slices even if they're already map[string]any / []any,
	// because their element types might still be struct pointers (e.g.,
	// the SDK's Debug endpoint returns map[string]any whose values could
	// be any type). A JSON round-trip guarantees all numeric values
	// surface as float64 — which the JSONPath library expects when
	// comparing ordered values.
	switch v.(type) {
	case string, bool, float64, int, int64, nil:
		return v, nil
	}
	data, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var out any
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// projectJSONPath evaluates a JSONPath against a raw JSON payload.
func projectJSONPath(raw json.RawMessage, path string) (any, error) {
	var data any
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, err
	}
	return jsonpath.Get(path, data)
}

// projectCached is the smart projection fibe_pipeline_result uses. It
// tries the path against the pipeline's step bindings first (so agents
// write $.step_id.field without having to remember the outer "steps"
// wrapper), then falls back to the whole cached response (so $.status,
// $.error, $.completed_step_ids remain reachable).
func projectCached(raw json.RawMessage, path string) (any, error) {
	var whole any
	if err := json.Unmarshal(raw, &whole); err != nil {
		return nil, err
	}
	if m, ok := whole.(map[string]any); ok {
		if bindings, ok := m["steps"].(map[string]any); ok {
			if v, err := jsonpath.Get(path, bindings); err == nil {
				return v, nil
			}
		}
	}
	return jsonpath.Get(path, whole)
}

// mustProject projects and returns the result, falling back to the original
// on error. Used inside step output projection (output_path) where a failed
// projection shouldn't abort the pipeline.
func mustProject(data any, path string) any {
	v, err := projectOnMap(data, path)
	if err != nil {
		return data
	}
	return v
}

// projectReturn projects req.Return against the step bindings. Return can
// be nil (→ full bindings), a string JSONPath, or an object literal with
// JSONPath values.
func projectReturn(bindings map[string]any, ret any) (any, error) {
	if ret == nil {
		return bindings, nil
	}
	switch x := ret.(type) {
	case string:
		if strings.HasPrefix(x, "$.") || x == "$" {
			return projectOnMap(bindings, x)
		}
		return x, nil
	case map[string]any:
		out := make(map[string]any, len(x))
		for k, v := range x {
			rv, err := resolveValue(v, bindings)
			if err != nil {
				return nil, err
			}
			out[k] = rv
		}
		return out, nil
	}
	return ret, nil
}
