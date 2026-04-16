package mcpserver

import (
	"context"
	"fmt"
	"strings"

	"github.com/fibegg/sdk/fibe"
	"github.com/mark3labs/mcp-go/mcp"
)

// registerDiscoveryTools wires the two meta-tools that make the full tool
// surface reachable from core mode without loading all 159 descriptions
// into the agent's context:
//
//   fibe_tools_catalog  — list every registered tool (name, description,
//                         annotations, optional input schema), filterable
//                         by tier/pattern
//   fibe_call           — invoke any registered tool by name through the
//                         same dispatcher path direct MCP calls use, so
//                         destructive gating, auth, and idempotency still
//                         apply
//
// Both live in the meta tier (always advertised) so agents can rely on
// them regardless of FIBE_MCP_TOOLS=core|full.
func (s *Server) registerDiscoveryTools() {
	s.addTool(&toolImpl{
		name:        "fibe_tools_catalog",
		description: "List every tool registered on the Fibe MCP server (including tools not advertised in the current tier)",
		tier:        tierMeta,
		annotations: toolAnnotations{ReadOnly: true, Idempotent: true},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			tierFilter := strings.ToLower(argString(args, "tier"))
			pattern := strings.ToLower(argString(args, "name_pattern"))
			includeSchema := argBool(args, "include_schema")

			advertisedSet := advertisedToolNames(s)

			type entry struct {
				Name         string         `json:"name"`
				Description  string         `json:"description"`
				Tier         string         `json:"tier"`
				Advertised   bool           `json:"advertised"`
				ReadOnly     bool           `json:"read_only,omitempty"`
				Destructive  bool           `json:"destructive,omitempty"`
				Idempotent   bool           `json:"idempotent,omitempty"`
				InputSchema  map[string]any `json:"input_schema,omitempty"`
			}

			var out []entry
			for _, name := range s.dispatcher.names() {
				t, ok := s.dispatcher.lookup(name)
				if !ok {
					continue
				}
				tierName := tierToString(t.tier)
				if tierFilter != "" && tierFilter != "all" && tierFilter != tierName {
					continue
				}
				if pattern != "" && !strings.Contains(strings.ToLower(name), pattern) {
					continue
				}
				e := entry{
					Name:        name,
					Description: t.description,
					Tier:        tierName,
					Advertised:  advertisedSet[name],
					ReadOnly:    t.annotations.ReadOnly,
					Destructive: t.annotations.Destructive,
					Idempotent:  t.annotations.Idempotent,
				}
				if includeSchema {
					if schema := schemaForTool(s, name); schema != nil {
						e.InputSchema = schema
					}
				}
				out = append(out, e)
			}

			return map[string]any{
				"count":      len(out),
				"tool_set":   s.cfg.ToolSet,
				"tools":      out,
			}, nil
		},
	}, mcp.NewTool("fibe_tools_catalog",
		mcp.WithDescription(`List every tool registered on the Fibe MCP server, including tools not advertised in the current tier.

Use this to discover the full capability surface when the server is running in core mode (the default). Call fibe_call(tool=<name>, args=...) to invoke any tool found here, whether or not it's in the advertised list.

FILTERS:
  tier           "core" | "full" | "meta" | "all" (default: all)
  name_pattern   Case-insensitive substring match on tool name
  include_schema Include each tool's input JSON schema (bloats output; default: false)`),
		mcp.WithString("tier", mcp.Description("Filter by tier: core, full, meta, all")),
		mcp.WithString("name_pattern", mcp.Description("Substring to match in tool name")),
		mcp.WithBoolean("include_schema", mcp.Description("Include input schemas (larger response)")),
	))

	s.addTool(&toolImpl{
		name:        "fibe_call",
		description: "Invoke any registered tool by name (including tools not advertised in the current tier)",
		tier:        tierMeta,
		annotations: toolAnnotations{},
		handler: func(ctx context.Context, c *fibe.Client, args map[string]any) (any, error) {
			name := argString(args, "tool")
			if name == "" {
				return nil, fmt.Errorf("required field 'tool' not set")
			}
			if name == "fibe_call" {
				return nil, fmt.Errorf("fibe_call cannot invoke itself")
			}
			if name == "fibe_pipeline" {
				return nil, fmt.Errorf("use fibe_pipeline directly for multi-step calls, not via fibe_call")
			}

			var subArgs map[string]any
			if raw, ok := args["args"]; ok && raw != nil {
				if m, ok := raw.(map[string]any); ok {
					subArgs = m
				} else {
					return nil, fmt.Errorf("field 'args' must be an object")
				}
			}
			if subArgs == nil {
				subArgs = map[string]any{}
			}
			// Forward the top-level confirm if set (so agents don't have to
			// nest confirm inside args for destructive calls).
			if v, ok := args["confirm"]; ok {
				subArgs["confirm"] = v
			}
			return s.dispatcher.dispatch(ctx, name, subArgs)
		},
	}, mcp.NewTool("fibe_call",
		mcp.WithDescription(`Invoke any registered Fibe tool by name. Useful in core mode to reach tools not in the advertised list.

Safety, auth, and idempotency gates apply exactly as they would for a direct call to the named tool. Destructive tools still require confirm:true unless the server is running with --yolo.

Discover available tool names and their input schemas via fibe_tools_catalog.`),
		mcp.WithString("tool", mcp.Required(), mcp.Description("The target tool name, e.g. fibe_playgrounds_debug")),
		mcp.WithObject("args", mcp.Description("The target tool's args object")),
		mcp.WithBoolean("confirm", mcp.Description("Forwarded as args.confirm for destructive tools")),
	))
}

// advertisedToolNames returns the set of tool names that are actually
// registered on the mcp-go server (i.e., visible to clients under the
// current FIBE_MCP_TOOLS tier).
func advertisedToolNames(s *Server) map[string]bool {
	out := map[string]bool{}
	for _, name := range s.dispatcher.names() {
		if t, ok := s.dispatcher.lookup(name); ok {
			if s.includeTool(t) {
				out[name] = true
			}
		}
	}
	return out
}

func tierToString(t toolTier) string {
	switch t {
	case tierCore:
		return "core"
	case tierFull:
		return "full"
	case tierMeta:
		return "meta"
	default:
		return "unknown"
	}
}

// schemaForTool reaches into the tool registry via a round-trip through the
// MCP server. The mcp-go server doesn't expose a direct accessor, so we
// fall back to nil for now — the catalog's include_schema flag will be a
// no-op until we wire a cleaner path (tracked as a follow-up).
func schemaForTool(s *Server, name string) map[string]any {
	_ = s
	_ = name
	return nil
}
