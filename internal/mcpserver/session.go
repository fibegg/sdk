package mcpserver

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/fibegg/sdk/fibe"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

// sessionState is per-MCP-session state. Each connected client gets its own
// state instance so credentials and pipeline results never leak across
// tenants.
type sessionState struct {
	mu              sync.RWMutex
	apiKey          string
	domain          string
	profile         string
	client          *fibe.Client // lazily built, reused across calls
	sessionID       string
	effectiveAPIKey string
	effectiveDomain string
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
//  1. session override set via fibe_auth_use/fibe_auth_set
//  2. HTTP bearer/domain headers
//  3. server-level default profile/API key
//
// If RequireAuth is set and neither source produced a key, returns an error.
// A per-session client is cached so circuit-breaker + rate-limit state stays
// isolated per tenant.
func (s *Server) resolveClient(ctx context.Context) (*fibe.Client, error) {
	st := s.sessionFor(ctx)
	requestKey := apiKeyFromContext(ctx)
	requestDomain := domainFromContext(ctx)

	st.mu.RLock()
	if st.client != nil {
		c := st.client
		effectiveKey := st.effectiveAPIKey
		effectiveDomain := st.effectiveDomain
		st.mu.RUnlock()
		if err := validatePinnedRequest(effectiveKey, effectiveDomain, requestKey, requestDomain); err != nil {
			return nil, err
		}
		return c, nil
	}
	st.mu.RUnlock()

	st.mu.Lock()
	defer st.mu.Unlock()
	if st.client != nil {
		if err := validatePinnedRequest(st.effectiveAPIKey, st.effectiveDomain, requestKey, requestDomain); err != nil {
			return nil, err
		}
		return st.client, nil
	}
	if st.apiKey != "" && requestKey != "" && st.apiKey != requestKey {
		return nil, errors.New("authentication headers do not match the credentials pinned to this MCP session")
	}
	if st.domain != "" && requestDomain != "" {
		if err := validatePinnedRequest(st.apiKey, st.domain, requestKey, requestDomain); err != nil {
			return nil, err
		}
	}

	// Resolution order: fibe_auth_set > HTTP bearer > server default.
	apiKey := st.apiKey
	if apiKey == "" {
		apiKey = requestKey
	}
	explicitKey := apiKey != ""
	if s.cfg.RequireAuth && !explicitKey {
		return nil, errors.New("no API key resolved; set Authorization bearer or call fibe_auth_set")
	}
	if apiKey == "" {
		apiKey = s.cfg.APIKey
	}

	domain := st.domain
	if domain == "" {
		domain = requestDomain
	}
	if domain == "" {
		domain = s.cfg.Domain
	}

	if s.httpMode {
		if !s.domainAllowed(domain) {
			return nil, fmt.Errorf("domain %q is not allowed for this MCP server", domain)
		}
		if requestDomain != "" && !explicitKey {
			return nil, errors.New("X-Fibe-Domain requires request or session credentials")
		}
	}

	// Always fork, even for the default credentials, so rate-limit and circuit
	// breaker state remain isolated between MCP sessions.
	if s.baseCli != nil && sameMCPOrigin(domain, s.cfg.Domain) {
		st.client = s.baseCli.WithKey(apiKey)
		st.effectiveAPIKey = apiKey
		st.effectiveDomain = domain
		return st.client, nil
	}

	opts := []fibe.Option{
		fibe.WithDisableAutoConfig(),
		fibe.WithCircuitBreaker(fibe.DefaultBreakerConfig),
		fibe.WithRateLimitAutoWait(),
		fibe.WithProgress(s.sendClientProgress),
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
	st.effectiveAPIKey = apiKey
	st.effectiveDomain = domain
	return st.client, nil
}

func validatePinnedRequest(effectiveKey, effectiveDomain, requestKey, requestDomain string) error {
	if requestKey != "" && requestKey != effectiveKey {
		return errors.New("authentication headers do not match the credentials pinned to this MCP session")
	}
	if requestDomain == "" {
		return nil
	}
	requestOrigin, err := normalizeMCPOrigin(requestDomain)
	if err != nil {
		return err
	}
	effectiveOrigin, err := normalizeMCPOrigin(effectiveDomain)
	if err != nil || requestOrigin != effectiveOrigin {
		return errors.New("domain header does not match the domain pinned to this MCP session")
	}
	return nil
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
	st.profile = ""
	st.client = nil
	st.effectiveAPIKey = ""
	st.effectiveDomain = ""
	st.mu.Unlock()
}

func (s *Server) setSessionProfile(ctx context.Context, profile, apiKey, domain string) {
	st := s.sessionFor(ctx)
	st.mu.Lock()
	st.profile = profile
	st.apiKey = apiKey
	st.domain = domain
	st.client = nil
	st.effectiveAPIKey = ""
	st.effectiveDomain = ""
	st.mu.Unlock()
}
