package mcpserver

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/fibegg/sdk/fibe"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

// ServeHTTP runs the MCP server over SSE (default) or streamable-HTTP.
//
// Per-request credentials are extracted from these headers (checked in order):
//
//	Authorization: Bearer <api_key>
//	X-Fibe-API-Key: <api_key>
//	X-Fibe-Domain:  <domain override>
//
// The credentials are injected into the request context; resolveClient()
// picks them up when the tool handler asks for an SDK client, giving us
// per-request auth isolation across tenants.
func (s *Server) ServeHTTP(ctx context.Context, addr string, streamable bool) error {
	if addr == "" {
		return errors.New("HTTP listen address is required")
	}
	loopback, err := isLoopbackListenAddress(addr)
	if err != nil {
		return err
	}
	if !loopback && !s.cfg.RequireAuth {
		return errors.New("non-loopback MCP HTTP listeners require --require-auth")
	}
	if s.cfg.AllowLocalAccess && !loopback {
		return errors.New("local MCP capabilities may only be enabled on a loopback listener")
	}
	for _, domain := range append([]string{s.cfg.Domain}, s.cfg.AllowedDomains...) {
		if _, err := normalizeMCPOrigin(domain); err != nil {
			return fmt.Errorf("invalid allowed MCP domain %q: %w", domain, err)
		}
	}

	s.httpMode = true
	s.baseCli = s.buildBaseClient()
	if s.audit != nil {
		defer s.audit.Close()
	}

	var handler http.Handler
	var shutdown func(context.Context) error
	if streamable {
		transport := mcpserver.NewStreamableHTTPServer(
			s.mcp,
			mcpserver.WithHTTPContextFunc(s.injectAuthFromRequest),
		)
		handler = transport
		shutdown = transport.Shutdown
	} else {
		transport := mcpserver.NewSSEServer(
			s.mcp,
			mcpserver.WithSSEContextFunc(s.injectAuthFromRequest),
		)
		handler = transport
		shutdown = transport.Shutdown
	}

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	httpServer := &http.Server{
		Addr:              addr,
		Handler:           s.secureHTTPHandler(handler),
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    1 << 20,
		BaseContext: func(net.Listener) context.Context {
			return ctx
		},
	}
	errCh := make(chan error, 1)
	go func() { errCh <- httpServer.Serve(listener) }()
	select {
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = shutdown(shutdownCtx)
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			return err
		}
		return ctx.Err()
	}
}

func (s *Server) secureHTTPHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if origin := r.Header.Get("Origin"); origin != "" {
			if !s.browserOriginAllowed(origin, r.Host) {
				http.Error(w, "cross-origin MCP requests are not allowed", http.StatusForbidden)
				return
			}
		}
		if r.Body != nil {
			r.Body = http.MaxBytesReader(w, r.Body, 64*1024*1024)
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) browserOriginAllowed(origin, requestHost string) bool {
	u, err := url.Parse(origin)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Path != "" && u.Path != "/") {
		return false
	}
	if strings.EqualFold(u.Host, requestHost) {
		return true
	}
	requestedOrigin := strings.ToLower(u.Scheme + "://" + u.Host)
	for _, candidate := range s.cfg.AllowedDomains {
		allowed, err := normalizeMCPOrigin(candidate)
		if err == nil && requestedOrigin == allowed {
			return true
		}
	}
	return false
}

func isLoopbackListenAddress(addr string) (bool, error) {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return false, fmt.Errorf("invalid MCP HTTP listen address %q: %w", addr, err)
	}
	if strings.EqualFold(host, "localhost") {
		return true, nil
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback(), nil
}

func normalizeMCPOrigin(raw string) (string, error) {
	if strings.TrimSpace(raw) == "" {
		raw = "fibe.gg"
	}
	resolved := fibe.NewClient(fibe.WithDomain(raw), fibe.WithDisableAutoConfig()).BaseURL()
	u, err := url.Parse(resolved)
	if err != nil || u.Host == "" {
		return "", errors.New("domain must include a valid host")
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", errors.New("domain must use http or https")
	}
	if u.User != nil || u.Fragment != "" || u.RawQuery != "" {
		return "", errors.New("domain must not include credentials, query, or fragment")
	}
	if u.Path != "" && u.Path != "/" {
		return "", errors.New("domain override must be an origin without a path")
	}
	return strings.ToLower(u.Scheme + "://" + u.Host), nil
}

func sameMCPOrigin(a, b string) bool {
	aOrigin, aErr := normalizeMCPOrigin(a)
	bOrigin, bErr := normalizeMCPOrigin(b)
	return aErr == nil && bErr == nil && aOrigin == bOrigin
}

func (s *Server) domainAllowed(domain string) bool {
	requested, err := normalizeMCPOrigin(domain)
	if err != nil {
		return false
	}
	for _, candidate := range append([]string{s.cfg.Domain}, s.cfg.AllowedDomains...) {
		allowed, err := normalizeMCPOrigin(candidate)
		if err == nil && requested == allowed {
			return true
		}
	}
	return false
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

	if apiKey != "" {
		ctx = context.WithValue(ctx, ctxKeyAPIKey{}, apiKey)
	}
	if domain != "" {
		ctx = context.WithValue(ctx, ctxKeyDomain{}, domain)
	}
	return ctx
}

type ctxKeyAPIKey struct{}
type ctxKeyDomain struct{}

func bearerFromRequest(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if h == "" {
		return ""
	}
	parts := strings.Fields(h)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return parts[1]
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

func yoloFromContext(ctx context.Context) bool { return false }
