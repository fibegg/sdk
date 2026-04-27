// Package mcpserver implements the local MCP server for the Fibe SDK.
//
// The server is launched via `fibe mcp serve` and exposes the Fibe API as a
// Model Context Protocol server so LLM agents can drive Fibe resources
// without paying the fork+exec cost of invoking the CLI per operation.
//
// Design:
//
//   - Generic flat resource reads/deletes are exposed through
//     fibe_resource_list/get/delete. Create/update and custom domain actions
//     remain concrete MCP tools, matching the CLI where those workflows need
//     richer input shapes.
//
//   - All tool invocations route through a single dispatcher that enforces
//     destructive-op gating, --yolo bypass, per-session authentication, and
//     idempotency. Both direct tool calls and fibe_pipeline steps go through
//     the same dispatcher, so safety is checked in one place.
//
//   - Streaming tools (wait, logs --follow) emit MCP progress notifications
//     rather than blocking until completion.
//
//   - fibe_pipeline composes multiple tool calls in one round-trip using
//     JSONPath bindings; pipeline results are cached per session for five
//     minutes and addressable via fibe_pipeline_result.
package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/fibegg/sdk/fibe"
	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
	"github.com/spf13/cobra"
)

// serverAlias is an indirection so we can keep the existing `server` local
// variable name from the pre-refactor code while importing mcp-go's server
// package under a distinct alias.
var _ = mcpserver.NewMCPServer

// Version is stamped at build time via ldflags; defaults to "dev".
var Version = "dev"

// Config bundles runtime configuration for the MCP server.
type Config struct {
	// APIKey is the default Fibe API key (env or --api-key passthrough).
	// Per-session auth takes precedence over this when set.
	APIKey string

	// Domain overrides the default Fibe API domain.
	Domain string

	// Debug turns on verbose logging to stderr.
	Debug bool

	// Yolo skips the confirm:true gate on destructive tools. Equivalent to
	// setting FIBE_MCP_YOLO=1. Use in non-interactive environments (CI).
	Yolo bool

	// ToolSet selects which tools are advertised. "full" exposes every tier;
	// "core" expands to meta+base+greenfield+brownfield; comma-separated
	// named tiers such as "other,meta" are also accepted. Default: full.
	ToolSet string

	// RequireAuth refuses tool calls that could not resolve an API key from
	// any source (bearer header, fibe_auth_set, env). Recommended for
	// multi-tenant HTTP/SSE deployments.
	RequireAuth bool

	// PipelineCacheSize caps the per-server LRU. 0 disables pipeline caching.
	PipelineCacheSize int

	// PipelineCacheEntryMax caps the size of a single cached pipeline result
	// in bytes. Larger results get truncated with a truncated:true marker.
	PipelineCacheEntryMax int

	// PipelineMaxSteps and PipelineMaxIterations bound pipeline execution.
	PipelineMaxSteps      int
	PipelineMaxIterations int

	// CobraRoot is the root of the fibe cobra command tree. Used by the
	// fibe_help tool to surface cobra Long help text and by the fibe_run
	// escape hatch to dispatch any CLI command as an MCP tool call.
	CobraRoot *cobra.Command
}

// DefaultConfig returns a Config populated with sensible defaults.
func DefaultConfig() Config {
	return Config{
		ToolSet:               "full",
		PipelineCacheSize:     256,
		PipelineCacheEntryMax: 1 << 20, // 1 MiB
		PipelineMaxSteps:      25,
		PipelineMaxIterations: 50,
	}
}

// Server holds the MCP server instance and shared state.
type Server struct {
	cfg     Config
	mcp     *mcpserver.MCPServer
	baseCli *fibe.Client // lazily built at Serve() time

	// cache is the per-server pipeline result store (multi-tenant keyed).
	cache *pipelineCache

	// sessions holds per-MCP-session state (auth overrides, cached clients).
	sessions *sessionRegistry

	// dispatcher is the central handler that wraps every tool call with
	// safety/auth/idempotency checks. Populated during RegisterAll.
	dispatcher *dispatcher

	// toolSchemas stores enriched input schemas for every registered tool,
	// including full/hidden tools that may not be advertised by mcp-go in
	// the current tier.
	toolSchemas map[string]map[string]any

	// runMu serializes fibe_run calls so concurrent cobra-tree edits
	// (flag state) don't race. Stdio transport is effectively single-tenant
	// but HTTP/SSE may see concurrent fibe_run invocations.
	runMu sync.Mutex

	// audit writes one JSON line per tool invocation when
	// FIBE_MCP_AUDIT_LOG is set. nil when disabled.
	audit *AuditLog

	// realStdout is the os.Stdout we captured at ServeStdio startup,
	// before redirecting the global to stderr. fibe_run uses this when it
	// needs to restore the real stdout for a child cobra command and then
	// re-hijack it without affecting the MCP pipe.
	realStdout *os.File
}

// New constructs a Server with the given configuration. The returned server
// has not yet registered any tools or resources; call RegisterAll before
// Serve.
func New(cfg Config) *Server {
	s := &Server{cfg: cfg}
	s.cache = newPipelineCache(cfg.PipelineCacheSize, cfg.PipelineCacheEntryMax)
	s.sessions = newSessionRegistry()
	s.dispatcher = newDispatcher(s)
	s.toolSchemas = map[string]map[string]any{}
	s.audit = newAuditLog()

	opts := []mcpserver.ServerOption{
		mcpserver.WithToolCapabilities(true),
		mcpserver.WithResourceCapabilities(false, true),
	}
	if cfg.Debug {
		opts = append(opts, mcpserver.WithLogging())
	}

	s.mcp = mcpserver.NewMCPServer(
		"fibe-mcp",
		Version,
		opts...,
	)
	return s
}

// RegisterAll wires up tools, resources, and prompts. Called after New and
// before Serve.
func (s *Server) RegisterAll() error {
	// Populated in later phases: registerTools, registerResources, registerPrompts.
	if err := s.registerTools(); err != nil {
		return fmt.Errorf("register tools: %w", err)
	}
	if err := s.registerResources(); err != nil {
		return fmt.Errorf("register resources: %w", err)
	}
	return nil
}

// ServeStdio runs the MCP server against stdin/stdout.
//
// CRITICAL: The stdio transport piggybacks JSON-RPC on os.Stdout. Any stray
// byte written to os.Stdout by any goroutine anywhere in the process (a
// misbehaving log line, a cobra handler using fmt.Println, a panic trace)
// will corrupt the MCP pipe with an unrecoverable parse error like:
//
//	calling "tools/call": invalid message version tag ""; expected "2.0"
//
// To prevent that, we:
//  1. Capture the real os.Stdout into a private variable BEFORE mcp-go
//     starts using it.
//  2. Replace os.Stdout with os.Stderr so every fmt.Println / log.Println
//     in the rest of the process writes to stderr instead of the pipe.
//  3. Hand the captured real stdout to mcp-go's StdioServer.Listen so only
//     properly-framed JSON-RPC messages reach the client.
//  4. Redirect the stdlib "log" package to stderr too (its default writer
//     is the same os.Stderr, but be explicit because log.SetOutput(nil)
//     would target os.Stdout post-redirect).
func (s *Server) ServeStdio(ctx context.Context) error {
	s.baseCli = s.buildBaseClient()

	// Capture & redirect. If anything later in this function calls
	// fmt.Println or similar it will now go to stderr, which mcp-go and
	// most MCP hosts treat as a debug channel, not as JSON-RPC.
	realStdout := os.Stdout
	os.Stdout = os.Stderr
	log.SetOutput(os.Stderr)
	s.realStdout = realStdout

	defer func() {
		os.Stdout = realStdout
	}()

	stdio := mcpserver.NewStdioServer(s.mcp)
	stdio.SetContextFunc(func(base context.Context) context.Context {
		return base
	})
	stdio.SetErrorLogger(log.New(os.Stderr, "[fibe-mcp] ", log.LstdFlags))

	// Wire signals the same way mcp-go's ServeStdio helper does.
	srvCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	return stdio.Listen(srvCtx, os.Stdin, realStdout)
}

// buildBaseClient constructs the base Fibe client from the server config.
// Per-session clients are forked from this via client.WithKey.
func (s *Server) buildBaseClient() *fibe.Client {
	opts := []fibe.Option{
		fibe.WithCircuitBreaker(fibe.DefaultBreakerConfig),
		fibe.WithRateLimitAutoWait(),
	}
	if s.cfg.APIKey != "" {
		opts = append(opts, fibe.WithAPIKey(s.cfg.APIKey))
	}
	if s.cfg.Domain != "" {
		opts = append(opts, fibe.WithDomain(s.cfg.Domain))
	}
	if s.cfg.Debug {
		opts = append(opts, fibe.WithDebug())
	}
	return fibe.NewClient(opts...)
}

// ToolInfo is the public view of a registered MCP tool.
type ToolInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Tier        string `json:"tier"`
	ReadOnly    bool   `json:"read_only,omitempty"`
	Destructive bool   `json:"destructive,omitempty"`
	Idempotent  bool   `json:"idempotent,omitempty"`
	InputSchema any    `json:"input_schema,omitempty"`
}

// AllTools returns metadata for every registered tool regardless of tier.
// Used by `fibe mcp docs` to dump the full catalog without starting a server.
//
// It merges dispatcher metadata (tier, annotations) with the mcp-go tool
// registry (input schema with property names, types, descriptions, and
// required fields) so the output is complete enough for an LLM to know
// exactly what tool to call and how.
func (s *Server) AllTools() []ToolInfo {
	var out []ToolInfo
	for _, name := range s.dispatcher.names() {
		t, ok := s.dispatcher.lookup(name)
		if !ok {
			continue
		}
		info := ToolInfo{
			Name:        t.name,
			Description: t.description,
			Tier:        tierToString(t.tier),
			ReadOnly:    t.annotations.ReadOnly,
			Destructive: t.annotations.Destructive,
			Idempotent:  t.annotations.Idempotent,
		}
		if schema := schemaForTool(s, name); schema != nil {
			info.InputSchema = schema
		}
		out = append(out, info)
	}
	return out
}

// toolInputSchemaToMap extracts the input schema from an mcp.Tool as a
// plain map[string]any suitable for JSON marshaling. It handles both the
// structured InputSchema path and the RawInputSchema path.
func toolInputSchemaToMap(tool mcp.Tool) any {
	// The mcp.Tool has a custom MarshalJSON that resolves InputSchema vs
	// RawInputSchema. We marshal the whole tool, then pull out inputSchema.
	data, err := json.Marshal(tool)
	if err != nil {
		return nil
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return nil
	}
	schema, ok := m["inputSchema"]
	if !ok {
		return nil
	}
	// Skip empty schemas (just {"type":"object"} or {"type":"object","properties":{}}).
	if sm, ok := schema.(map[string]any); ok {
		props, hasProps := sm["properties"]
		if !hasProps {
			return nil
		}
		if pm, ok := props.(map[string]any); ok && len(pm) == 0 {
			return nil
		}
	}
	return schema
}
