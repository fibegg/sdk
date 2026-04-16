package mcpserver

import (
	"context"
	"errors"
	"sync"

	"github.com/fibegg/sdk/fibe"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

// sessionState is per-MCP-session state. Each connected client gets its own
// state instance so credentials and pipeline results never leak across
// tenants.
type sessionState struct {
	mu        sync.RWMutex
	apiKey    string
	domain    string
	client    *fibe.Client // lazily built, reused across calls
	sessionID string
}

// sessions maps mcp-go session ID -> sessionState.
type sessionRegistry struct {
	mu   sync.RWMutex
	byID map[string]*sessionState
}

func newSessionRegistry() *sessionRegistry {
	return &sessionRegistry{byID: map[string]*sessionState{}}
}

func (r *sessionRegistry) get(id string) *sessionState {
	r.mu.RLock()
	st := r.byID[id]
	r.mu.RUnlock()
	if st != nil {
		return st
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if st = r.byID[id]; st != nil {
		return st
	}
	st = &sessionState{sessionID: id}
	r.byID[id] = st
	return st
}

func (r *sessionRegistry) drop(id string) {
	r.mu.Lock()
	delete(r.byID, id)
	r.mu.Unlock()
}

// resolveClient returns the effective *fibe.Client for this call, honoring:
//  1. session override set via fibe_auth_set (or HTTP bearer header)
//  2. server-level default APIKey (FIBE_API_KEY / --api-key)
//
// If RequireAuth is set and neither source produced a key, returns an error.
// A per-session client is cached so circuit-breaker + rate-limit state stays
// isolated per tenant.
func (s *Server) resolveClient(ctx context.Context) (*fibe.Client, error) {
	st := s.sessionFor(ctx)

	st.mu.RLock()
	if st.client != nil {
		c := st.client
		st.mu.RUnlock()
		return c, nil
	}
	st.mu.RUnlock()

	st.mu.Lock()
	defer st.mu.Unlock()
	if st.client != nil {
		return st.client, nil
	}

	// Resolution order: fibe_auth_set > HTTP bearer > server default.
	apiKey := st.apiKey
	if apiKey == "" {
		if v := apiKeyFromContext(ctx); v != "" {
			apiKey = v
		}
	}
	if apiKey == "" {
		apiKey = s.cfg.APIKey
	}
	if apiKey == "" && s.cfg.RequireAuth {
		return nil, errors.New("no API key resolved; set Authorization bearer or call fibe_auth_set")
	}

	domain := st.domain
	if domain == "" {
		if v := domainFromContext(ctx); v != "" {
			domain = v
		}
	}
	if domain == "" {
		domain = s.cfg.Domain
	}

	// Fast path: if no per-session overrides, fork from the server base
	// client so we share HTTP transport + logger.
	if apiKey == s.cfg.APIKey && domain == s.cfg.Domain && s.baseCli != nil {
		st.client = s.baseCli
		return st.client, nil
	}

	// Build a fresh client with the resolved creds. WithKey reuses the base
	// client's transport config but gives us a separate circuit breaker +
	// rate-limit state. If domain is changed, we must rebuild from options below.
	if s.baseCli != nil && apiKey != "" && domain == s.cfg.Domain {
		st.client = s.baseCli.WithKey(apiKey)
		return st.client, nil
	}

	opts := []fibe.Option{
		fibe.WithCircuitBreaker(fibe.DefaultBreakerConfig),
		fibe.WithRateLimitAutoWait(),
	}
	if apiKey != "" {
		opts = append(opts, fibe.WithAPIKey(apiKey))
	}
	if domain != "" {
		opts = append(opts, fibe.WithDomain(domain))
	}
	if s.cfg.Debug {
		opts = append(opts, fibe.WithDebug())
	}
	st.client = fibe.NewClient(opts...)
	return st.client, nil
}

// sessionFor returns the sessionState associated with the current MCP call.
// Stdio transport uses a single shared session ID so there's always exactly
// one sessionState in that mode.
func (s *Server) sessionFor(ctx context.Context) *sessionState {
	id := "default"
	if cs := mcpserver.ClientSessionFromContext(ctx); cs != nil {
		id = cs.SessionID()
	}
	return s.sessions.get(id)
}

// setSessionAuth is invoked by the fibe_auth_set tool to inject per-session
// credentials. Nil-ing the existing client forces resolveClient to rebuild on
// the next call with the new values.
func (s *Server) setSessionAuth(ctx context.Context, apiKey, domain string) {
	st := s.sessionFor(ctx)
	st.mu.Lock()
	st.apiKey = apiKey
	st.domain = domain
	st.client = nil
	st.mu.Unlock()
}
