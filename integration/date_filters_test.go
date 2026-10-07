package integration

import (
	"testing"
	"time"

	"github.com/fibegg/sdk/fibe"
)

// TestDateFilters_CoverageMatrix exercises date filters across list endpoints.
func TestDateFilters_CoverageMatrix(t *testing.T) {
	t.Parallel()
	c := userClient(t)

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
				r3, err := c.Agents.List(ctx(), &fibe.AgentListParams{CreatedAfter: future, PerPage: 100})
				requireNoError(t, err)
				if r3.Meta.Total != 0 {
					t.Errorf("expected 0 agents created after %s, got %d", future, r3.Meta.Total)
				}
			},
		},
		{
			name: "specs",
			run: func(t *testing.T, past, future string) {
				r, err := c.Specs.List(ctx(), &fibe.SpecListParams{CreatedBefore: past, PerPage: 100})
				requireNoError(t, err)
				if r.Meta.Total != 0 {
					t.Errorf("expected 0 specs before %s, got %d", past, r.Meta.Total)
				}
			},
		},
		{
			name: "repositories",
			run: func(t *testing.T, past, future string) {
				r, err := c.Repositories.List(ctx(), &fibe.RepositoryListParams{CreatedAfter: future, PerPage: 100})
				requireNoError(t, err)
				if r.Meta.Total != 0 {
					t.Errorf("expected 0 repositories created after far future, got %d", r.Meta.Total)
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
			name: "hosts",
			run: func(t *testing.T, past, future string) {
				r, err := c.Hosts.List(ctx(), &fibe.HostListParams{CreatedBefore: past, PerPage: 100})
				requireNoError(t, err)
				if r.Meta.Total != 0 {
					t.Errorf("expected 0 hosts before %s, got %d", past, r.Meta.Total)
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
	c := userClient(t)

	s := seedSecret(t, c, "date-range")

	// Window: 1 hour before now to 1 hour after now: should include our secret
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
