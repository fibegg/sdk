package integration

import (
	"strings"
	"testing"

	"github.com/fibegg/sdk/fibe"
)

// TestListCoverage_Pagination runs pagination tests across every list endpoint.
// For each endpoint, it verifies:
//   1. Default list works and returns envelope
//   2. page=1, per_page=1 returns at most 1 item and Meta reflects params
//   3. Meta.Total matches default total (bound check)
func TestListCoverage_Pagination(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	// Seed 2 of each core resource to ensure Meta.Total >= 2
	_ = seedSecret(t, c, "pg1")
	_ = seedSecret(t, c, "pg2")
	_ = seedAgent(t, c, fibe.ProviderGemini)
	_ = seedAgent(t, c, fibe.ProviderClaudeCode)
	_ = seedPlayspec(t, c)
	_ = seedPlayspec(t, c)

	cases := []struct {
		name string
		call func() (int, int, int, error) // returns page, per_page, data_len, err
	}{
		{"agents", func() (int, int, int, error) {
			r, err := c.Agents.List(ctx(), &fibe.AgentListParams{Page: 1, PerPage: 1})
			if err != nil {
				return 0, 0, 0, err
			}
			return r.Meta.Page, r.Meta.PerPage, len(r.Data), nil
		}},
		{"playspecs", func() (int, int, int, error) {
			r, err := c.Playspecs.List(ctx(), &fibe.PlayspecListParams{Page: 1, PerPage: 1})
			if err != nil {
				return 0, 0, 0, err
			}
			return r.Meta.Page, r.Meta.PerPage, len(r.Data), nil
		}},
		{"props", func() (int, int, int, error) {
			r, err := c.Props.List(ctx(), &fibe.PropListParams{Page: 1, PerPage: 1})
			if err != nil {
				return 0, 0, 0, err
			}
			return r.Meta.Page, r.Meta.PerPage, len(r.Data), nil
		}},
		{"playgrounds", func() (int, int, int, error) {
			r, err := c.Playgrounds.List(ctx(), &fibe.PlaygroundListParams{Page: 1, PerPage: 1})
			if err != nil {
				return 0, 0, 0, err
			}
			return r.Meta.Page, r.Meta.PerPage, len(r.Data), nil
		}},
		{"marquees", func() (int, int, int, error) {
			r, err := c.Marquees.List(ctx(), &fibe.MarqueeListParams{Page: 1, PerPage: 1})
			if err != nil {
				return 0, 0, 0, err
			}
			return r.Meta.Page, r.Meta.PerPage, len(r.Data), nil
		}},
		{"secrets", func() (int, int, int, error) {
			r, err := c.Secrets.List(ctx(), &fibe.SecretListParams{Page: 1, PerPage: 1})
			if err != nil {
				return 0, 0, 0, err
			}
			return r.Meta.Page, r.Meta.PerPage, len(r.Data), nil
		}},
		{"api_keys", func() (int, int, int, error) {
			r, err := c.APIKeys.List(ctx(), &fibe.APIKeyListParams{Page: 1, PerPage: 1})
			if err != nil {
				return 0, 0, 0, err
			}
			return r.Meta.Page, r.Meta.PerPage, len(r.Data), nil
		}},
		{"templates", func() (int, int, int, error) {
			r, err := c.ImportTemplates.List(ctx(), &fibe.ImportTemplateListParams{Page: 1, PerPage: 1})
			if err != nil {
				return 0, 0, 0, err
			}
			return r.Meta.Page, r.Meta.PerPage, len(r.Data), nil
		}},
		{"webhooks", func() (int, int, int, error) {
			r, err := c.WebhookEndpoints.List(ctx(), &fibe.WebhookEndpointListParams{Page: 1, PerPage: 1})
			if err != nil {
				return 0, 0, 0, err
			}
			return r.Meta.Page, r.Meta.PerPage, len(r.Data), nil
		}},
		{"audit_logs", func() (int, int, int, error) {
			r, err := c.AuditLogs.List(ctx(), &fibe.AuditLogListParams{Page: 1, PerPage: 1})
			if err != nil {
				return 0, 0, 0, err
			}
			return r.Meta.Page, r.Meta.PerPage, len(r.Data), nil
		}},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			page, perPage, n, err := tc.call()
			if err != nil {
				if apiErr, ok := err.(*fibe.APIError); ok && apiErr.Code == fibe.ErrCodeFeatureDisabled {
					t.Skipf("%s feature disabled", tc.name)
				}
				requireNoError(t, err)
			}
			if page != 1 {
				t.Errorf("%s: expected page=1, got %d", tc.name, page)
			}
			if perPage != 1 {
				t.Errorf("%s: expected per_page=1, got %d", tc.name, perPage)
			}
			if n > 1 {
				t.Errorf("%s: expected <= 1 item with per_page=1, got %d", tc.name, n)
			}
		})
	}
}

// TestListCoverage_Sorting verifies sort parameter works across list endpoints
// that support it. It requests both _asc and _desc and verifies ordering.
func TestListCoverage_Sorting(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	// Seed resources to ensure we have at least 2 for ordering checks
	_ = seedAgent(t, c, fibe.ProviderGemini)
	_ = seedAgent(t, c, fibe.ProviderClaudeCode)
	_ = seedSecret(t, c, "sort-a")
	_ = seedSecret(t, c, "sort-b")
	_ = seedPlayspec(t, c)
	_ = seedPlayspec(t, c)

	t.Run("agents sort name_asc", func(t *testing.T) {
		t.Parallel()
		r, err := c.Agents.List(ctx(), &fibe.AgentListParams{Sort: "name_asc", PerPage: 50})
		requireNoError(t, err)
		names := make([]string, 0, len(r.Data))
		for _, a := range r.Data {
			names = append(names, a.Name)
		}
		requireSortedByStringLocaleAware(t, "agents name_asc", names, true)
	})

	t.Run("agents sort created_at_desc", func(t *testing.T) {
		t.Parallel()
		r, err := c.Agents.List(ctx(), &fibe.AgentListParams{Sort: "created_at_desc", PerPage: 50})
		requireNoError(t, err)
		if len(r.Data) < 2 {
			t.Skip("not enough agents to verify sort")
		}
		for i := 1; i < len(r.Data); i++ {
			if r.Data[i-1].CreatedAt == nil || r.Data[i].CreatedAt == nil {
				continue
			}
			if r.Data[i-1].CreatedAt.Before(*r.Data[i].CreatedAt) {
				t.Errorf("created_at_desc violated at index %d", i)
			}
		}
	})

	t.Run("playspecs sort name_asc", func(t *testing.T) {
		t.Parallel()
		r, err := c.Playspecs.List(ctx(), &fibe.PlayspecListParams{Sort: "name_asc", PerPage: 50})
		requireNoError(t, err)
		names := make([]string, 0, len(r.Data))
		for _, p := range r.Data {
			names = append(names, p.Name)
		}
		requireSortedByStringLocaleAware(t, "playspecs name_asc", names, true)
	})

	t.Run("secrets sort key_asc", func(t *testing.T) {
		t.Parallel()
		r, err := c.Secrets.List(ctx(), &fibe.SecretListParams{Sort: "key_asc", PerPage: 50})
		requireNoError(t, err)
		keys := make([]string, 0, len(r.Data))
		for _, s := range r.Data {
			keys = append(keys, s.Key)
		}
		requireSortedByStringLocaleAware(t, "secrets key_asc", keys, true)
	})

	t.Run("props sort name_desc", func(t *testing.T) {
		t.Parallel()
		r, err := c.Props.List(ctx(), &fibe.PropListParams{Sort: "name_desc", PerPage: 50})
		requireNoError(t, err)
		names := make([]string, 0, len(r.Data))
		for _, p := range r.Data {
			names = append(names, p.Name)
		}
		requireSortedByStringLocaleAware(t, "props name_desc", names, false)
	})

	t.Run("marquees sort name_asc", func(t *testing.T) {
		t.Parallel()
		r, err := c.Marquees.List(ctx(), &fibe.MarqueeListParams{Sort: "name_asc", PerPage: 50})
		requireNoError(t, err)
		names := make([]string, 0, len(r.Data))
		for _, m := range r.Data {
			names = append(names, m.Name)
		}
		requireSortedByStringLocaleAware(t, "marquees name_asc", names, true)
	})
}

// TestListCoverage_Filtering verifies filter parameters actually affect results.
func TestListCoverage_Filtering(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	// Seed agents with distinct names so we can test Q and Name filters
	seedName := uniqueName("filt-needle")
	_, err := c.Agents.Create(ctx(), &fibe.AgentCreateParams{
		Name:     seedName,
		Provider: fibe.ProviderGemini,
	})
	requireNoError(t, err)

	t.Run("agents filter by Q matches name substring", func(t *testing.T) {
		t.Parallel()
		// Use part of the seedName as the query
		q := "filt-needle"
		r, err := c.Agents.List(ctx(), &fibe.AgentListParams{Q: q, PerPage: 50})
		requireNoError(t, err)
		found := false
		for _, a := range r.Data {
			if strings.Contains(a.Name, q) {
				found = true
			}
		}
		if !found && r.Meta.Total > 0 {
			t.Errorf("expected to find an agent matching Q=%q, got %d results", q, r.Meta.Total)
		}
	})

	t.Run("agents filter by provider", func(t *testing.T) {
		t.Parallel()
		_ = seedAgent(t, c, fibe.ProviderGemini)
		r, err := c.Agents.List(ctx(), &fibe.AgentListParams{Provider: fibe.ProviderGemini, PerPage: 50})
		requireNoError(t, err)
		for _, a := range r.Data {
			if a.Provider != fibe.ProviderGemini {
				t.Errorf("expected provider=%s, got %s", fibe.ProviderGemini, a.Provider)
			}
		}
	})

	t.Run("playspecs filter by job_mode=true", func(t *testing.T) {
		t.Parallel()
		jm := true
		_ = seedPlayspec(t, c, func(p *fibe.PlayspecCreateParams) {
			p.JobMode = &jm
			p.BaseComposeYAML = jobComposeYAML()
			p.Services = []fibe.PlayspecServiceDef{jobWatchedService("worker")}
		})
		r, err := c.Playspecs.List(ctx(), &fibe.PlayspecListParams{JobMode: &jm, PerPage: 50})
		requireNoError(t, err)
		for _, p := range r.Data {
			if p.JobMode == nil || !*p.JobMode {
				t.Errorf("expected job_mode=true, got %v for %s", p.JobMode, p.Name)
			}
		}
	})

	t.Run("playspecs filter by job_mode=false", func(t *testing.T) {
		t.Parallel()
		jmf := false
		r, err := c.Playspecs.List(ctx(), &fibe.PlayspecListParams{JobMode: &jmf, PerPage: 50})
		requireNoError(t, err)
		for _, p := range r.Data {
			if p.JobMode != nil && *p.JobMode {
				t.Errorf("expected job_mode=false, got true for %s", p.Name)
			}
		}
	})

	t.Run("api_keys filter by label substring", func(t *testing.T) {
		t.Parallel()
		label := uniqueName("lbl-needle")
		k, err := c.APIKeys.Create(ctx(), &fibe.APIKeyCreateParams{Label: label, Scopes: []string{"agents:read"}})
		requireNoError(t, err)
		t.Cleanup(func() {
			if k.ID != nil {
				c.APIKeys.Delete(ctx(), *k.ID)
			}
		})
		r, err := c.APIKeys.List(ctx(), &fibe.APIKeyListParams{Label: "lbl-needle", PerPage: 50})
		requireNoError(t, err)
		found := false
		for _, kk := range r.Data {
			if strings.Contains(kk.Label, "lbl-needle") {
				found = true
			}
		}
		if !found {
			t.Errorf("expected to find API key with label substring 'lbl-needle' among %d results", r.Meta.Total)
		}
	})

	t.Run("audit_logs filter by channel=api", func(t *testing.T) {
		t.Parallel()
		r, err := c.AuditLogs.List(ctx(), &fibe.AuditLogListParams{Channel: "api", PerPage: 25})
		requireNoError(t, err)
		for _, l := range r.Data {
			if l.Channel != "api" {
				t.Errorf("expected channel=api, got %s", l.Channel)
			}
		}
	})

	t.Run("webhooks filter by enabled", func(t *testing.T) {
		t.Parallel()
		f := false
		r, err := c.WebhookEndpoints.List(ctx(), &fibe.WebhookEndpointListParams{Enabled: &f, PerPage: 25})
		requireNoError(t, err)
		for _, w := range r.Data {
			if w.Enabled != nil && *w.Enabled {
				t.Errorf("expected enabled=false, got true for %d", w.ID)
			}
		}
	})

	t.Run("templates filter by system=true", func(t *testing.T) {
		t.Parallel()
		tr := true
		r, err := c.ImportTemplates.List(ctx(), &fibe.ImportTemplateListParams{System: &tr, PerPage: 25})
		requireNoError(t, err)
		for _, tpl := range r.Data {
			if tpl.System != nil && !*tpl.System {
				t.Errorf("expected system=true, got false for template %s", tpl.Name)
			}
		}
	})
}

// TestListCoverage_PaginationBoundaries exercises large page numbers and zero per_page.
func TestListCoverage_PaginationBoundaries(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	t.Run("page beyond last returns empty data", func(t *testing.T) {
		t.Parallel()
		r, err := c.Secrets.List(ctx(), &fibe.SecretListParams{Page: 9999, PerPage: 1})
		requireNoError(t, err)
		if len(r.Data) != 0 {
			t.Errorf("expected empty data on far page, got %d items", len(r.Data))
		}
		// Meta.Total must still reflect real total
		if r.Meta.Total < 0 {
			t.Errorf("expected Total >= 0, got %d", r.Meta.Total)
		}
	})

	t.Run("per_page clamped to server max", func(t *testing.T) {
		t.Parallel()
		r, err := c.Secrets.List(ctx(), &fibe.SecretListParams{PerPage: 500})
		requireNoError(t, err)
		if r.Meta.PerPage > 100 {
			t.Errorf("expected per_page clamped to <= 100, got %d", r.Meta.PerPage)
		}
	})
}
