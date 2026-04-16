package integration

import (
	"context"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/fibegg/sdk/fibe"
)

// Migrated from: 08-error-handling.spec.js + 20-edge-cases.spec.js
func TestEdgeCases_ErrorHandling(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	t.Run("nonexistent resource returns structured error", func(t *testing.T) {
		t.Parallel()
		_, err := c.Marquees.Get(ctx(), 999999)
		apiErr := requireAPIError(t, err, fibe.ErrCodeNotFound, 404)
		if apiErr.Message == "" {
			t.Error("expected error message")
		}
	})

	t.Run("delete nonexistent playground returns 404", func(t *testing.T) {
		t.Parallel()
		err := c.Playgrounds.Delete(ctx(), 999999)
		requireAPIError(t, err, fibe.ErrCodeNotFound, 404)
	})

	t.Run("delete nonexistent prop returns 404", func(t *testing.T) {
		t.Parallel()
		err := c.Props.Delete(ctx(), 999999)
		requireAPIError(t, err, fibe.ErrCodeNotFound, 404)
	})

	t.Run("delete nonexistent agent returns 404", func(t *testing.T) {
		t.Parallel()
		err := c.Agents.Delete(ctx(), 999999)
		requireAPIError(t, err, fibe.ErrCodeNotFound, 404)
	})
}

// Migrated from: 20-edge-cases.spec.js
func TestEdgeCases_OversizedInputs(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	t.Run("oversized agent name", func(t *testing.T) {
		t.Parallel()
		longName := strings.Repeat("a", 1000)
		agent, err := c.Agents.Create(ctx(), &fibe.AgentCreateParams{
			Name:     longName,
			Provider: fibe.ProviderGemini,
		})
		if err == nil {
			t.Cleanup(func() { c.Agents.Delete(ctx(), agent.ID) })
			t.Error("server accepted oversized agent name (1000 chars) — should reject with 422")
		}
	})

	t.Run("oversized secret value", func(t *testing.T) {
		t.Parallel()
		bigValue := strings.Repeat("x", 100_000)
		s, err := c.Secrets.Create(ctx(), &fibe.SecretCreateParams{
			Key:   uniqueName("BIG_VAL"),
			Value: bigValue,
		})
		if err == nil {
			t.Cleanup(func() { c.Secrets.Delete(ctx(), *s.ID) })
			t.Error("server accepted oversized secret value (100KB) — should reject with 422")
		}
	})
}

// Migrated from: 17-api-contract.spec.js
func TestEdgeCases_APIContract(t *testing.T) {
	c := adminClient(t)

	t.Run("list endpoints return data+meta envelope", func(t *testing.T) {
		result, err := c.Agents.List(ctx(), nil)
		requireNoError(t, err)

		if result.Meta.Page == 0 {
			t.Error("expected meta.page > 0")
		}
		if result.Meta.PerPage == 0 {
			t.Error("expected meta.per_page > 0")
		}
	})

	t.Run("GET with wrong method returns error", func(t *testing.T) {
		apiKey := os.Getenv("FIBE_API_KEY")
		if apiKey == "" {
			t.Skip("FIBE_API_KEY not set")
		}

		base := c.BaseURL()
		req, _ := http.NewRequestWithContext(context.Background(), "PATCH", base+"/api/me", nil)
		req.Header.Set("Authorization", "Bearer "+apiKey)
		resp, err := http.DefaultClient.Do(req)
		requireNoError(t, err)
		resp.Body.Close()

		if resp.StatusCode == 200 {
			t.Error("PATCH /api/me should not return 200")
		}
	})

	t.Run("X-Request-Id returned on all responses", func(t *testing.T) {
		apiKey := os.Getenv("FIBE_API_KEY")
		if apiKey == "" {
			t.Skip("FIBE_API_KEY not set")
		}

		paths := []string{"/api/me", "/api/agents", "/api/playspecs"}
		for _, path := range paths {
			req, _ := http.NewRequestWithContext(context.Background(), "GET", c.BaseURL()+path, nil)
			req.Header.Set("Authorization", "Bearer "+apiKey)
			resp, err := http.DefaultClient.Do(req)
			requireNoError(t, err)
			resp.Body.Close()

			if resp.Header.Get("X-Request-Id") == "" {
				t.Errorf("missing X-Request-Id on %s", path)
			}
		}
	})
}

// Migrated from: 20-edge-cases.spec.js
func TestEdgeCases_ContentType(t *testing.T) {
	c := adminClient(t)
	apiKey := os.Getenv("FIBE_API_KEY")
	if apiKey == "" {
		t.Skip("FIBE_API_KEY not set")
	}

	t.Run("wrong content-type still processed", func(t *testing.T) {
		base := c.BaseURL()
		body := strings.NewReader(`{"agent":{"name":"ct-test","provider":"gemini"}}`)
		req, _ := http.NewRequestWithContext(ctx(), "POST", base+"/api/agents", body)
		req.Header.Set("Authorization", "Bearer "+apiKey)
		req.Header.Set("Content-Type", "text/plain")

		resp, err := http.DefaultClient.Do(req)
		requireNoError(t, err)
		resp.Body.Close()
		if resp.StatusCode != 201 && resp.StatusCode != 400 && resp.StatusCode != 422 && resp.StatusCode != 415 {
			t.Errorf("expected 201, 400, 422, or 415 for wrong content-type, got %d", resp.StatusCode)
		}
	})
}
