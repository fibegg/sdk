package integration

import (
	"testing"
	"time"

	"github.com/fibegg/sdk/fibe"
)

// TestSortColumns_AllSupportedColumns verifies every documented sort column
// produces results sorted correctly for both asc and desc directions.
func TestSortColumns_AllSupportedColumns(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	// Seed fresh resources so we have data with known ordering
	_ = seedAgent(t, c, fibe.ProviderGemini)
	_ = seedAgent(t, c, fibe.ProviderClaudeCode)
	_ = seedSecret(t, c, "sortA")
	_ = seedSecret(t, c, "sortB")

	t.Run("agents created_at_asc", func(t *testing.T) {
		t.Parallel()
		r, err := c.Agents.List(ctx(), &fibe.AgentListParams{Sort: "created_at_asc", PerPage: 50})
		requireNoError(t, err)
		times := []time.Time{}
		for _, a := range r.Data {
			if a.CreatedAt != nil {
				times = append(times, *a.CreatedAt)
			}
		}
		requireSortedByTime(t, "agents created_at_asc", times, true)
	})

	t.Run("agents created_at_desc", func(t *testing.T) {
		t.Parallel()
		r, err := c.Agents.List(ctx(), &fibe.AgentListParams{Sort: "created_at_desc", PerPage: 50})
		requireNoError(t, err)
		times := []time.Time{}
		for _, a := range r.Data {
			if a.CreatedAt != nil {
				times = append(times, *a.CreatedAt)
			}
		}
		requireSortedByTime(t, "agents created_at_desc", times, false)
	})

	t.Run("playspecs created_at_asc", func(t *testing.T) {
		t.Parallel()
		r, err := c.Playspecs.List(ctx(), &fibe.PlayspecListParams{Sort: "created_at_asc", PerPage: 50})
		requireNoError(t, err)
		times := []time.Time{}
		for _, p := range r.Data {
			if p.CreatedAt != nil {
				times = append(times, *p.CreatedAt)
			}
		}
		requireSortedByTime(t, "playspecs created_at_asc", times, true)
	})

	t.Run("props created_at_desc", func(t *testing.T) {
		t.Parallel()
		r, err := c.Props.List(ctx(), &fibe.PropListParams{Sort: "created_at_desc", PerPage: 50})
		requireNoError(t, err)
		times := []time.Time{}
		for _, p := range r.Data {
			times = append(times, p.CreatedAt)
		}
		requireSortedByTime(t, "props created_at_desc", times, false)
	})

	t.Run("marquees created_at_asc", func(t *testing.T) {
		t.Parallel()
		r, err := c.Marquees.List(ctx(), &fibe.MarqueeListParams{Sort: "created_at_asc", PerPage: 50})
		requireNoError(t, err)
		times := []time.Time{}
		for _, m := range r.Data {
			times = append(times, m.CreatedAt)
		}
		requireSortedByTime(t, "marquees created_at_asc", times, true)
	})

	t.Run("secrets created_at_desc", func(t *testing.T) {
		t.Parallel()
		r, err := c.Secrets.List(ctx(), &fibe.SecretListParams{Sort: "created_at_desc", PerPage: 50})
		requireNoError(t, err)
		times := []time.Time{}
		for _, s := range r.Data {
			if s.CreatedAt != nil {
				times = append(times, *s.CreatedAt)
			}
		}
		requireSortedByTime(t, "secrets created_at_desc", times, false)
	})

	t.Run("api_keys created_at_asc", func(t *testing.T) {
		t.Parallel()
		r, err := c.APIKeys.List(ctx(), &fibe.APIKeyListParams{Sort: "created_at_asc", PerPage: 50})
		requireNoError(t, err)
		times := []time.Time{}
		for _, k := range r.Data {
			if k.CreatedAt != nil {
				times = append(times, *k.CreatedAt)
			}
		}
		requireSortedByTime(t, "api_keys created_at_asc", times, true)
	})

	t.Run("webhooks created_at_asc", func(t *testing.T) {
		t.Parallel()
		r, err := c.WebhookEndpoints.List(ctx(), &fibe.WebhookEndpointListParams{Sort: "created_at_asc", PerPage: 50})
		requireNoError(t, err)
		times := []time.Time{}
		for _, w := range r.Data {
			if w.CreatedAt != nil {
				times = append(times, *w.CreatedAt)
			}
		}
		requireSortedByTime(t, "webhooks created_at_asc", times, true)
	})

	t.Run("audit_logs created_at_desc is default", func(t *testing.T) {
		t.Parallel()
		r, err := c.AuditLogs.List(ctx(), &fibe.AuditLogListParams{Sort: "created_at_desc", PerPage: 25})
		requireNoError(t, err)
		times := []time.Time{}
		for _, l := range r.Data {
			times = append(times, l.CreatedAt)
		}
		requireSortedByTime(t, "audit_logs created_at_desc", times, false)
	})
}

// TestSortColumns_InvalidSortIsRejectedOrIgnored ensures the backend either
// rejects invalid sort values or ignores them — it should NOT crash.
func TestSortColumns_InvalidSortIsRejectedOrIgnored(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	_, err := c.Agents.List(ctx(), &fibe.AgentListParams{Sort: "this_is_not_a_real_column_xyz"})
	// Either 400/422 OR falls back to default sort — both are acceptable.
	if err != nil {
		if apiErr, ok := err.(*fibe.APIError); ok {
			if apiErr.StatusCode >= 500 {
				t.Errorf("expected 2xx/4xx for invalid sort, got 5xx: %v", err)
			}
		}
	}
}
