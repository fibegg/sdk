package integration

import (
	"context"
	"errors"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/fibegg/sdk/fibe"
)

// Migrated from: 14-security-injection.spec.js
func TestSecurity_SQLInjection(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	injections := []string{
		"'; DROP TABLE players; --",
		"1 OR 1=1",
		"' UNION SELECT * FROM api_keys --",
		"1; DELETE FROM playgrounds",
	}

	t.Run("agent name SQL injection", func(t *testing.T) {
		t.Parallel()
		for _, payload := range injections {
			agent, err := c.Agents.Create(ctx(), &fibe.AgentCreateParams{
				Name:     payload,
				Provider: fibe.ProviderGemini,
			})
			if err != nil {
				continue // API rejected payload — acceptable
			}
			t.Cleanup(func() { c.Agents.Delete(ctx(), agent.ID) })

			// Verify payload was stored literally (parameterized), not interpreted
			if agent.Name != payload {
				t.Errorf("SQL injection payload was modified: sent %q, got %q", payload, agent.Name)
			}

			// Verify agent can be retrieved intact (injection didn't corrupt DB)
			fetched, err := c.Agents.Get(ctx(), agent.ID)
			if err != nil {
				t.Errorf("failed to re-fetch agent after SQL injection payload: %v", err)
			} else if fetched.Name != payload {
				t.Errorf("stored SQL payload was corrupted on re-fetch: expected %q, got %q", payload, fetched.Name)
			}
		}
	})

	t.Run("secret key SQL injection", func(t *testing.T) {
		t.Parallel()
		for _, payload := range injections {
			s, err := c.Secrets.Create(ctx(), &fibe.SecretCreateParams{
				Key:   uniqueName("SQL_TEST"),
				Value: payload,
			})
			if err != nil {
				continue // API rejected payload — acceptable
			}
			t.Cleanup(func() { c.Secrets.Delete(ctx(), *s.ID) })

			// Verify secret can be retrieved intact
			fetched, err := c.Secrets.Get(ctx(), *s.ID)
			if err != nil {
				t.Errorf("failed to re-fetch secret after SQL injection payload: %v", err)
			} else if fetched.Value != nil && *fetched.Value != payload {
				t.Errorf("stored SQL payload was corrupted: expected %q, got %q", payload, *fetched.Value)
			}
		}
	})

	t.Run("search param injection", func(t *testing.T) {
		t.Parallel()
		result, err := c.Monitor.List(ctx(), &fibe.MonitorListParams{Q: "' OR '1'='1"})
		if err != nil {
			return // API rejected — acceptable
		}
		// Injection should not return all records
		if len(result.Data) > 100 {
			t.Errorf("SQL injection in search may have bypassed filtering: got %d results", len(result.Data))
		}
	})
}

// Migrated from: 14-security-injection.spec.js
func TestSecurity_XSS(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	xssPayloads := []string{
		"<script>alert('xss')</script>",
		"<img src=x onerror=alert(1)>",
		"javascript:alert(1)",
	}

	t.Run("agent name XSS stored literally", func(t *testing.T) {
		t.Parallel()
		for _, payload := range xssPayloads {
			agent, err := c.Agents.Create(ctx(), &fibe.AgentCreateParams{
				Name:     payload,
				Provider: fibe.ProviderGemini,
			})
			if err != nil {
				continue // API rejected payload — acceptable
			}
			t.Cleanup(func() { c.Agents.Delete(ctx(), agent.ID) })

			// Verify payload is stored literally (not interpreted)
			if agent.Name != payload {
				t.Errorf("XSS payload was modified: sent %q, got %q", payload, agent.Name)
			}

			// Verify re-fetch returns it literally (not executed/transformed)
			fetched, err := c.Agents.Get(ctx(), agent.ID)
			if err != nil {
				t.Errorf("failed to re-fetch agent after XSS payload: %v", err)
			} else if fetched.Name != payload {
				t.Errorf("stored XSS payload was corrupted on re-fetch: expected %q, got %q", payload, fetched.Name)
			}
		}
	})
}

// Migrated from: 45-security-idor.spec.js
func TestSecurity_IDOR(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	resources := []struct {
		name string
		fn   func(int64) error
	}{
		{"playground", func(id int64) error { _, e := c.Playgrounds.Get(ctx(), id); return e }},
		{"agent", func(id int64) error { _, e := c.Agents.Get(ctx(), id); return e }},
		{"playspec", func(id int64) error { _, e := c.Playspecs.Get(ctx(), id); return e }},
		{"prop", func(id int64) error { _, e := c.Props.Get(ctx(), id); return e }},
		{"secret", func(id int64) error { _, e := c.Secrets.Get(ctx(), id); return e }},
		{"team", func(id int64) error { _, e := c.Teams.Get(ctx(), id); return e }},
		{"webhook", func(id int64) error { _, e := c.WebhookEndpoints.Get(ctx(), id); return e }},
	}

	for _, r := range resources {
		r := r
		t.Run(r.name+" returns 404 for nonexistent ID", func(t *testing.T) {
			t.Parallel()
			err := r.fn(999999999)
			requireAPIError(t, err, fibe.ErrCodeNotFound, 404)
		})
	}
}

// Migrated from: 21-owasp-security.spec.js
func TestSecurity_OWASP_BrokenAccessControl(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	t.Run("different scoped keys cannot cross-access", func(t *testing.T) {
		t.Parallel()
		secretsOnly := createScopedKey(t, c, "secrets-only", []string{"secrets:read"})
		agentsOnly := createScopedKey(t, c, "agents-only", []string{"agents:read"})

		_, err := secretsOnly.Agents.List(ctx(), nil)
		requireAPIError(t, err, fibe.ErrCodeForbidden, 403)

		_, err = agentsOnly.Secrets.List(ctx(), nil)
		requireAPIError(t, err, fibe.ErrCodeForbidden, 403)
	})
}

// Migrated from: 21-owasp-security.spec.js
func TestSecurity_OWASP_CryptographicFailures(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	t.Run("API key token not exposed in list", func(t *testing.T) {
		t.Parallel()
		keys, err := c.APIKeys.List(ctx(), nil)
		requireNoError(t, err)

		for _, k := range keys.Data {
			if k.Token != nil {
				t.Error("token should NEVER be exposed in list response")
			}
		}
	})

	t.Run("secret value not in list", func(t *testing.T) {
		t.Parallel()
		s, err := c.Secrets.Create(ctx(), &fibe.SecretCreateParams{
			Key: uniqueName("CRYPTO_TEST"), Value: "super-secret",
		})
		requireNoError(t, err)
		t.Cleanup(func() { c.Secrets.Delete(ctx(), *s.ID) })

		list, err := c.Secrets.List(ctx(), nil)
		requireNoError(t, err)

		for _, item := range list.Data {
			if item.Value != nil {
				t.Error("value should NOT be in list response")
			}
		}
	})
}

// Migrated from: 47-auth-hardening.spec.js
func TestSecurity_AuthHardening(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	t.Run("invalid token returns 401", func(t *testing.T) {
		t.Parallel()
		bad := c.WithKey("completely-invalid-token")
		_, err := bad.APIKeys.Me(ctx())
		requireAPIError(t, err, fibe.ErrCodeUnauthorized, 401)
	})

	t.Run("empty token returns 401", func(t *testing.T) {
		t.Parallel()
		bad := c.WithKey("")
		_, err := bad.APIKeys.Me(ctx())
		requireAPIError(t, err, fibe.ErrCodeUnauthorized, 401)
	})

	t.Run("malformed bearer token", func(t *testing.T) {
		t.Parallel()
		apiKey := os.Getenv("FIBE_API_KEY")
		if apiKey == "" {
			t.Skip("FIBE_API_KEY not set")
		}
		base := c.BaseURL()

		tokens := []string{
			"Basic " + apiKey,
			"Token " + apiKey,
			apiKey,
		}

		for _, tok := range tokens {
			req, _ := http.NewRequestWithContext(context.Background(), "GET", base+"/api/me", nil)
			req.Header.Set("Authorization", tok)
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				continue
			}
			resp.Body.Close()
			if resp.StatusCode == 200 {
				t.Errorf("token format %q should not authenticate", tok[:min(20, len(tok))])
			}
		}
	})

	t.Run("no error details leak in 401", func(t *testing.T) {
		bad := c.WithKey("invalid")
		_, err := bad.APIKeys.Me(ctx())
		var apiErr *fibe.APIError
		if errors.As(err, &apiErr) {
			msg := strings.ToLower(apiErr.Message)
			if strings.Contains(msg, "stack") || strings.Contains(msg, "trace") || strings.Contains(msg, "bcrypt") {
				t.Errorf("error message should not leak internals: %q", apiErr.Message)
			}
		}
	})
}

// Migrated from: 26-rate-limit-headers.spec.js
func TestSecurity_RateLimitHeaders(t *testing.T) {
	c := adminClient(t)
	apiKey := os.Getenv("FIBE_API_KEY")
	if apiKey == "" {
		t.Skip("FIBE_API_KEY not set")
	}

	base := c.BaseURL()
	req, _ := http.NewRequestWithContext(ctx(), "GET", base+"/api/me", nil)
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := http.DefaultClient.Do(req)
	requireNoError(t, err)
	resp.Body.Close()

	t.Run("X-RateLimit-Limit present", func(t *testing.T) {
		if resp.Header.Get("X-Ratelimit-Limit") == "" {
			t.Error("expected X-RateLimit-Limit header")
		}
	})

	t.Run("X-RateLimit-Remaining present", func(t *testing.T) {
		if resp.Header.Get("X-Ratelimit-Remaining") == "" {
			t.Error("expected X-RateLimit-Remaining header")
		}
	})

	t.Run("X-RateLimit-Reset present", func(t *testing.T) {
		if resp.Header.Get("X-Ratelimit-Reset") == "" {
			t.Error("expected X-RateLimit-Reset header")
		}
	})

	t.Run("SDK tracks rate limits", func(t *testing.T) {
		// Because the adminClient() harness explicitly bypasses rate limit headers,
		// we must spin up a raw client to verify SDK tracking logic parses HTTP 429 schemas!
		cNoBypass := fibe.NewClient(fibe.WithAPIKey(apiKey), fibe.WithBaseURL(base))
		cNoBypass.APIKeys.Me(ctx())
		rl := cNoBypass.RateLimit()
		if rl.Limit == 0 {
			t.Error("expected non-zero rate limit after request")
		}
	})
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
