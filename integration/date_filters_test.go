package integration

import (
	"testing"
	"time"

	"github.com/fibegg/sdk/fibe"
)

// TestDateFilters_CoverageMatrix exercises created_after/created_before on every
// list endpoint that supports them.
func TestDateFilters_CoverageMatrix(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	// Use "way in the past" as after-bound (should include all results) and
	// "way in the future" as before-bound (should exclude all).
	past := "2000-01-01T00:00:00Z"
	future := time.Now().Add(100 * 365 * 24 * time.Hour).Format(time.RFC3339)

	cases := []struct {
		name string
		run  func(t *testing.T, past, future string)
	}{
		{
			name: "agents",
			run: func(t *testing.T, past, future string) {
				r1, err := c.Agents.List(ctx(), &fibe.AgentListParams{CreatedAfter: past, PerPage: 100})
				requireNoError(t, err)
				r2, err := c.Agents.List(ctx(), &fibe.AgentListParams{CreatedBefore: past, PerPage: 100})
				requireNoError(t, err)
				if r2.Meta.Total > r1.Meta.Total {
					t.Errorf("before=%s should return <= after=%s (%d > %d)", past, past, r2.Meta.Total, r1.Meta.Total)
				}
				// Future-after should return nothing created after far future
				r3, err := c.Agents.List(ctx(), &fibe.AgentListParams{CreatedAfter: future, PerPage: 100})
				requireNoError(t, err)
				if r3.Meta.Total != 0 {
					t.Errorf("expected 0 agents created after %s, got %d", future, r3.Meta.Total)
				}
			},
		},
		{
			name: "playspecs",
			run: func(t *testing.T, past, future string) {
				r, err := c.Playspecs.List(ctx(), &fibe.PlayspecListParams{CreatedBefore: past, PerPage: 100})
				requireNoError(t, err)
				if r.Meta.Total != 0 {
					t.Errorf("expected 0 playspecs before %s, got %d", past, r.Meta.Total)
				}
			},
		},
		{
			name: "props",
			run: func(t *testing.T, past, future string) {
				r, err := c.Props.List(ctx(), &fibe.PropListParams{CreatedAfter: future, PerPage: 100})
				requireNoError(t, err)
				if r.Meta.Total != 0 {
					t.Errorf("expected 0 props created after far future, got %d", r.Meta.Total)
				}
			},
		},
		{
			name: "playgrounds",
			run: func(t *testing.T, past, future string) {
				r, err := c.Playgrounds.List(ctx(), &fibe.PlaygroundListParams{CreatedBefore: past, PerPage: 100})
				requireNoError(t, err)
				if r.Meta.Total != 0 {
					t.Errorf("expected 0 playgrounds before %s, got %d", past, r.Meta.Total)
				}
			},
		},
		{
			name: "marquees",
			run: func(t *testing.T, past, future string) {
				r, err := c.Marquees.List(ctx(), &fibe.MarqueeListParams{CreatedBefore: past, PerPage: 100})
				requireNoError(t, err)
				if r.Meta.Total != 0 {
					t.Errorf("expected 0 marquees before %s, got %d", past, r.Meta.Total)
				}
			},
		},
		{
			name: "secrets",
			run: func(t *testing.T, past, future string) {
				r, err := c.Secrets.List(ctx(), &fibe.SecretListParams{CreatedBefore: past, PerPage: 100})
				requireNoError(t, err)
				if r.Meta.Total != 0 {
					t.Errorf("expected 0 secrets before %s, got %d", past, r.Meta.Total)
				}
			},
		},
		{
			name: "audit_logs",
			run: func(t *testing.T, past, future string) {
				r, err := c.AuditLogs.List(ctx(), &fibe.AuditLogListParams{CreatedAfter: future, PerPage: 100})
				requireNoError(t, err)
				if r.Meta.Total != 0 {
					t.Errorf("expected 0 audit logs after far future, got %d", r.Meta.Total)
				}
			},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.run(t, past, future)
		})
	}
}

// TestDateFilters_RangeBehavior verifies the after/before range intersects correctly.
func TestDateFilters_RangeBehavior(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	// Seed a secret so we have at least one entry in the window
	s := seedSecret(t, c, "date-range")

	// Window: 1 hour before now to 1 hour after now — should include our secret
	after := time.Now().Add(-1 * time.Hour).Format(time.RFC3339)
	before := time.Now().Add(1 * time.Hour).Format(time.RFC3339)

	r, err := c.Secrets.List(ctx(), &fibe.SecretListParams{
		CreatedAfter:  after,
		CreatedBefore: before,
		PerPage:       100,
	})
	requireNoError(t, err)
	found := false
	for _, sec := range r.Data {
		if sec.ID != nil && s.ID != nil && *sec.ID == *s.ID {
			found = true
		}
	}
	if !found {
		t.Errorf("expected to find just-created secret %v in range %s..%s, got %d results", s.ID, after, before, r.Meta.Total)
	}
}
