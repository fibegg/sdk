package mcpserver

import (
	"context"
	"errors"
	"net/http"
	"strings"

	mcpserver "github.com/mark3labs/mcp-go/server"
)

// ServeHTTP runs the MCP server over SSE (default) or streamable-HTTP.
//
// Per-request credentials are extracted from these headers (checked in order):
//
//	Authorization: Bearer <api_key>
//	X-Fibe-API-Key: <api_key>
//	X-Fibe-Domain:  <domain override>
//	X-Fibe-Yolo:    1  (per-request destructive bypass)
//
// The credentials are injected into the request context; resolveClient()
// picks them up when the tool handler asks for an SDK client, giving us
// per-request auth isolation across tenants.
func (s *Server) ServeHTTP(ctx context.Context, addr string, streamable bool) error {
	if addr == "" {
		return errors.New("HTTP listen address is required")
	}
	s.baseCli = s.buildBaseClient()

	if streamable {
		srv := mcpserver.NewStreamableHTTPServer(
			s.mcp,
			mcpserver.WithHTTPContextFunc(s.injectAuthFromRequest),
		)
		return srv.Start(addr)
	}

	srv := mcpserver.NewSSEServer(
		s.mcp,
		mcpserver.WithSSEContextFunc(s.injectAuthFromRequest),
	)
	return srv.Start(addr)
}

// injectAuthFromRequest reads per-request headers and stores them on the
// session state keyed by the MCP session ID. Because mcp-go assigns a
// stable session ID per HTTP connection, the state carries across all calls
// made on that connection.
//
// Note: the session ID isn't available in this function yet (the session is
// created *after* the context func runs). We stash the raw values on the
// context itself using unexported keys; resolveClient reads them later.
func (s *Server) injectAuthFromRequest(ctx context.Context, r *http.Request) context.Context {
	apiKey := bearerFromRequest(r)
	if apiKey == "" {
		apiKey = r.Header.Get("X-Fibe-API-Key")
	}
	domain := r.Header.Get("X-Fibe-Domain")
	yolo := r.Header.Get("X-Fibe-Yolo")

	if apiKey != "" {
		ctx = context.WithValue(ctx, ctxKeyAPIKey{}, apiKey)
	}
	if domain != "" {
		ctx = context.WithValue(ctx, ctxKeyDomain{}, domain)
	}
	if yolo != "" {
		ctx = context.WithValue(ctx, ctxKeyYolo{}, yolo)
	}
	return ctx
}

type ctxKeyAPIKey struct{}
type ctxKeyDomain struct{}
type ctxKeyYolo struct{}

func bearerFromRequest(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if h == "" {
		return ""
	}
	if !strings.HasPrefix(h, "Bearer ") {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(h, "Bearer "))
}

// apiKeyFromContext is consulted by resolveClient when the session hasn't
// been populated yet (i.e. on the first tool call). After the first call
// the sessionState caches the key so subsequent lookups skip this path.
func apiKeyFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(ctxKeyAPIKey{}).(string); ok {
		return v
	}
	return ""
}

func domainFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(ctxKeyDomain{}).(string); ok {
		return v
	}
	return ""
}

func yoloFromContext(ctx context.Context) bool {
	v, ok := ctx.Value(ctxKeyYolo{}).(string)
	if !ok {
		return false
	}
	return v == "1" || v == "true" || v == "yes"
}
