package mcpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
)

// registerStaticResources wires the fixed URIs whose content doesn't depend
// on path parameters: fibe://me, fibe://status, fibe://pipeline/schema.
func (s *Server) registerStaticResources() {
	// fibe://me — authenticated user snapshot
	s.mcp.AddResource(mcp.NewResource(
		"fibe://me",
		"Authenticated user",
		mcp.WithResourceDescription("Profile of the API key currently in use. Reads /api/me once and returns the JSON."),
		mcp.WithMIMEType("application/json"),
	), func(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		c, err := s.resolveClient(ctx)
		if err != nil {
			return nil, err
		}
		me, err := c.APIKeys.Me(ctx)
		if err != nil {
			return nil, err
		}
		return jsonResource(req.Params.URI, me), nil
	})

	// fibe://status — account status dashboard
	s.mcp.AddResource(mcp.NewResource(
		"fibe://status",
		"Account status dashboard",
		mcp.WithResourceDescription("Counts across all resources (playgrounds, agents, props, ...). Single request, full context."),
		mcp.WithMIMEType("application/json"),
	), func(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		c, err := s.resolveClient(ctx)
		if err != nil {
			return nil, err
		}
		st, err := c.Status.Get(ctx)
		if err != nil {
			return nil, err
		}
		return jsonResource(req.Params.URI, st), nil
	})

	// fibe://schema — all schemas at once
	s.mcp.AddResource(mcp.NewResource(
		"fibe://schema",
		"All JSON Schema hints",
		mcp.WithResourceDescription("The full Fibe resource schema registry (playground, agent, playspec, prop, marquee, secret, team, webhook, api_key)."),
		mcp.WithMIMEType("application/json"),
	), func(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		return jsonResource(req.Params.URI, schemaRegistry), nil
	})

	// fibe://pipeline/schema — how to write a fibe_pipeline request
	s.mcp.AddResource(mcp.NewResource(
		"fibe://pipeline/schema",
		"fibe_pipeline DSL reference",
		mcp.WithResourceDescription("Envelope format for fibe_pipeline: steps, parallel, for_each, return projection, JSONPath bindings."),
		mcp.WithMIMEType("application/json"),
	), func(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		return jsonResource(req.Params.URI, pipelineDSLSchema), nil
	})
}

// registerResourceTemplates wires the parameterized URIs:
//
//	fibe://schema/{resource}          (playground, agent, ...)
//	fibe://schema/{resource}/{op}     (create, update)
//	fibe://help/{path}                 (e.g., fibe://help/playgrounds/create)
//	fibe://pipelines/{pipeline_id}     (cached pipeline result)
func (s *Server) registerResourceTemplates() {
	// Schema per resource.
	s.mcp.AddResourceTemplate(mcp.NewResourceTemplate(
		"fibe://schema/{resource}",
		"Fibe schema per resource",
		mcp.WithTemplateDescription("JSON Schema for create/update params of a specific Fibe resource."),
		mcp.WithTemplateMIMEType("application/json"),
	), func(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		resource, _ := parseURIPath(req.Params.URI, "fibe://schema/")
		if resource == "" {
			return nil, fmt.Errorf("missing resource in URI %q", req.Params.URI)
		}
		// Allow nested op: fibe://schema/<resource>/<op>
		op := ""
		if slash := strings.Index(resource, "/"); slash != -1 {
			op = resource[slash+1:]
			resource = resource[:slash]
		}
		schemas, ok := schemaRegistry[resource]
		if !ok {
			return nil, fmt.Errorf("unknown resource %q", resource)
		}
		if op != "" {
			entry, ok := schemas[op]
			if !ok {
				return nil, fmt.Errorf("unknown operation %q for resource %q", op, resource)
			}
			return jsonResource(req.Params.URI, entry), nil
		}
		return jsonResource(req.Params.URI, schemas), nil
	})

	// Help per command path.
	s.mcp.AddResourceTemplate(mcp.NewResourceTemplate(
		"fibe://help/{path}",
		"Fibe CLI extended help",
		mcp.WithTemplateDescription("cobra Long help for any fibe subcommand. Use for flag/usage details."),
		mcp.WithTemplateMIMEType("text/plain"),
	), func(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		path, _ := parseURIPath(req.Params.URI, "fibe://help/")
		if s.cfg.CobraRoot == nil {
			return nil, fmt.Errorf("help unavailable: server started without CobraRoot")
		}
		// URI path uses slashes as separators; cobra wants space-separated.
		parts := strings.FieldsFunc(path, func(r rune) bool { return r == '/' })
		cmd, _, err := s.cfg.CobraRoot.Find(parts)
		if err != nil || cmd == nil {
			return nil, fmt.Errorf("unknown command %q", path)
		}
		var buf bytes.Buffer
		cmd.SetOut(&buf)
		_ = cmd.Help()
		return []mcp.ResourceContents{mcp.TextResourceContents{
			URI:      req.Params.URI,
			MIMEType: "text/plain",
			Text:     buf.String(),
		}}, nil
	})

	// Cached pipeline results.
	s.mcp.AddResourceTemplate(mcp.NewResourceTemplate(
		"fibe://pipelines/{pipeline_id}",
		"Cached pipeline result",
		mcp.WithTemplateDescription("Cached fibe_pipeline result for this session (5-minute TTL). Same backing store as fibe_pipeline_result."),
		mcp.WithTemplateMIMEType("application/json"),
	), func(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		id, _ := parseURIPath(req.Params.URI, "fibe://pipelines/")
		if id == "" {
			return nil, fmt.Errorf("missing pipeline_id in URI %q", req.Params.URI)
		}
		sess := s.sessionFor(ctx)
		payload, ok := s.cache.Get(sess.sessionID, id)
		if !ok {
			return jsonResource(req.Params.URI, map[string]any{"expired": true, "pipeline_id": id}), nil
		}
		return []mcp.ResourceContents{mcp.TextResourceContents{
			URI:      req.Params.URI,
			MIMEType: "application/json",
			Text:     string(payload),
		}}, nil
	})
}

func parseURIPath(uri, prefix string) (string, bool) {
	if !strings.HasPrefix(uri, prefix) {
		return "", false
	}
	return strings.TrimPrefix(uri, prefix), true
}

func jsonResource(uri string, v any) []mcp.ResourceContents {
	data, err := json.Marshal(v)
	if err != nil {
		data = []byte(fmt.Sprintf(`{"error": %q}`, err.Error()))
	}
	return []mcp.ResourceContents{mcp.TextResourceContents{
		URI:      uri,
		MIMEType: "application/json",
		Text:     string(data),
	}}
}

// pipelineDSLSchema is the human-readable DSL reference exposed via
// fibe://pipeline/schema. It intentionally documents the envelope rather
// than imposing a strict JSON Schema, since the envelope is polymorphic
// (tool / parallel / for_each shapes).
var pipelineDSLSchema = map[string]any{
	"$description": "fibe_pipeline DSL. Compose multiple Fibe tool calls in one round-trip.",
	"properties": map[string]any{
		"steps": map[string]any{
			"type":        "array",
			"description": "Ordered list of steps. Each step is one of: tool-call | parallel-block | for-each-block.",
			"items": map[string]any{
				"oneOf": []any{
					map[string]any{
						"title":       "ToolCall",
						"required":    []string{"tool"},
						"properties": map[string]any{
							"id":          map[string]any{"type": "string", "description": "Step ID used as reference prefix in later JSONPath expressions ($.<id>.<field>)."},
							"tool":        map[string]any{"type": "string", "description": "A registered tool name, e.g. fibe_playgrounds_get. fibe_pipeline is not callable here."},
							"args":        map[string]any{"type": "object", "description": "Tool args. String values starting with \"$.\" are JSONPath expressions."},
							"on_error":    map[string]any{"type": "string", "enum": []string{"abort", "continue"}, "description": "Default: abort."},
							"input_path":  map[string]any{"type": "string", "description": "Optional JSONPath to narrow args before dispatch."},
							"output_path": map[string]any{"type": "string", "description": "Optional JSONPath to project tool output before storing."},
						},
					},
					map[string]any{
						"title":    "ParallelBlock",
						"required": []string{"parallel"},
						"properties": map[string]any{
							"parallel": map[string]any{"type": "array", "description": "Sub-steps executed concurrently."},
						},
					},
					map[string]any{
						"title":    "ForEachBlock",
						"required": []string{"for_each", "as", "steps"},
						"properties": map[string]any{
							"id":       map[string]any{"type": "string"},
							"for_each": map[string]any{"type": "string", "description": "JSONPath expression producing an array."},
							"as":       map[string]any{"type": "string", "description": "Iteration variable name (referenced as $.<as>)."},
							"steps":    map[string]any{"type": "array", "description": "Sub-steps executed for each item."},
							"collect":  map[string]any{"type": "string", "description": "Optional JSONPath projection applied to each iteration's final scope."},
						},
					},
				},
			},
		},
		"return":          map[string]any{"description": "Projection applied to the final bindings map. Either a JSONPath string or an object literal with JSONPath values."},
		"dry_run":         map[string]any{"type": "boolean", "description": "Validate refs/schemas without executing."},
		"cache":           map[string]any{"type": "boolean", "description": "Default true. Set false to skip caching this pipeline result."},
		"idempotency_key": map[string]any{"type": "string", "description": "Threaded through destructive steps for retry safety."},
	},
	"examples": []any{
		map[string]any{
			"description": "Create, wait, fetch logs — one round-trip.",
			"steps": []map[string]any{
				{"id": "pg", "tool": "fibe_playgrounds_create", "args": map[string]any{"name": "ci-test", "playspec_id": 5}},
				{"id": "wait", "tool": "fibe_playgrounds_wait", "args": map[string]any{"id": "$.pg.id", "status": "running"}},
				{"id": "logs", "tool": "fibe_playgrounds_logs", "args": map[string]any{"id": "$.pg.id", "service": "web", "tail": 100}},
			},
			"return": "$.logs.lines",
		},
	},
}
