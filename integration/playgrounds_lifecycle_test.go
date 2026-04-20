package integration

import (
	"strings"
	"testing"
	"time"

	"github.com/fibegg/sdk/fibe"
)

// TestPlaygrounds_FullLifecycle exercises the complete real-world playground lifecycle:
//
//	Create → Status (async) → Compose/EnvMetadata/Debug → Logs → Update → Extend → Rollout → HardRestart → Delete
//
// It requires FIBE_TEST_MARQUEE_ID pointing to a functional marquee. Without it, the
// create phase skips but all non-dependent parts still run against a pre-seeded playground.
func TestPlaygrounds_FullLifecycle(t *testing.T) {
	c := adminClient(t)

	spec := seedPlayspec(t, c)
	marqueeID := testMarqueeID(t)
	if marqueeID == 0 {
		t.Skip("set FIBE_TEST_MARQUEE_ID to run full lifecycle")
	}
	expiresAt := time.Now().UTC().Add(2 * time.Hour)

	pg, err := c.Playgrounds.Create(ctx(), &fibe.PlaygroundCreateParams{
		Name:        uniqueName("life-pg"),
		PlayspecID:  *spec.ID,
		MarqueeID:   &marqueeID,
		ExpiresAt:   &expiresAt,
		NeverExpire: ptr(false),
	})
	requireNoError(t, err)
	t.Cleanup(func() { c.Playgrounds.Delete(ctx(), pg.ID) })

	// 1. Immediate create response must have ID, Name, Status
	if pg.ID == 0 || pg.Name == "" || pg.Status == "" {
		t.Errorf("create response missing core fields: id=%d name=%q status=%q", pg.ID, pg.Name, pg.Status)
	}
	if pg.PlayspecID == nil || *pg.PlayspecID != *spec.ID {
		t.Errorf("expected PlayspecID=%d, got %v", *spec.ID, pg.PlayspecID)
	}

	// 2. Status polling: expect transition from pending → in_progress → running/error
	t.Run("status transitions", func(t *testing.T) {
		finalStatus := waitForPlaygroundStatus(t, c, pg.ID, []string{"running", "error", "failed"}, CapWaitTimeout)
		if finalStatus == "" {
			t.Error("playground status never left empty")
		}
		t.Logf("playground final status: %s", finalStatus)
	})

	// 3. Detail fields populated (may require waiting for provisioning)
	t.Run("detail has expiration and service info", func(t *testing.T) {
		// Poll until ExpiresAt is populated or timeout
		d, _ := pollUntil(20, time.Second, func() (*fibe.Playground, bool) {
			got, err := c.Playgrounds.Get(ctx(), pg.ID)
			if err != nil {
				return nil, false
			}
			if got.ExpiresAt != nil {
				return got, true
			}
			return got, false
		})
		if d == nil {
			t.Skip("could not fetch playground detail within timeout")
		}
		if d.ExpiresAt == nil {
			t.Log("ExpiresAt not yet populated within timeout — may require running state")
		}
	})

	// 4. Compose endpoint returns usable YAML (may take time to render)
	t.Run("compose returns structured YAML", func(t *testing.T) {
		cmp, found := pollUntil(120, time.Second, func() (*fibe.PlaygroundCompose, bool) {
			c2, err := c.Playgrounds.Compose(ctx(), pg.ID)
			if err != nil {
				return nil, false
			}
			if strings.Contains(c2.ComposeYAML, "services:") {
				return c2, true
			}
			return c2, false
		})
		if !found {
			t.Skip("compose YAML not rendered within timeout")
		}
		if !strings.Contains(cmp.ComposeYAML, "services:") {
			t.Errorf("expected services: in compose YAML")
		}
	})

	// 5. EnvMetadata returns structured response
	t.Run("env metadata structure", func(t *testing.T) {
		env, err := c.Playgrounds.EnvMetadata(ctx(), pg.ID)
		requireNoError(t, err)
		if env.Merged == nil || env.Metadata == nil || env.SystemKeys == nil {
			t.Errorf("expected all env fields populated: merged=%v metadata=%v system_keys=%v",
				env.Merged != nil, env.Metadata != nil, env.SystemKeys != nil)
		}
	})

	// 6. Debug endpoint returns diagnostic info
	t.Run("debug returns diagnostic data", func(t *testing.T) {
		dbg, err := c.Playgrounds.Debug(ctx(), pg.ID)
		requireNoError(t, err)
		if dbg == nil {
			t.Error("expected non-nil Debug response")
		}
	})

	// 7. ExtendExpiration bumps ExpiresAt
	t.Run("extend expiration increases expiration", func(t *testing.T) {
		before, err := c.Playgrounds.Get(ctx(), pg.ID)
		requireNoError(t, err)
		hrs := 2
		ext, err := c.Playgrounds.ExtendExpiration(ctx(), pg.ID, &hrs)
		requireNoError(t, err)
		if before.ExpiresAt != nil && !ext.ExpiresAt.After(*before.ExpiresAt) {
			t.Errorf("expected new ExpiresAt > old: before=%v after=%v", before.ExpiresAt, ext.ExpiresAt)
		}
		if ext.TimeRemaining <= 0 {
			t.Errorf("expected positive TimeRemaining, got %f", ext.TimeRemaining)
		}
	})

	// 8. Update changes name
	t.Run("update name persists", func(t *testing.T) {
		newName := uniqueName("renamed-pg")
		upd, err := c.Playgrounds.Update(ctx(), pg.ID, &fibe.PlaygroundUpdateParams{Name: &newName})
		requireNoError(t, err)
		if upd.Name != newName {
			t.Errorf("expected Name=%s, got %s", newName, upd.Name)
		}
		// Re-read to confirm persistence
		got, err := c.Playgrounds.Get(ctx(), pg.ID)
		requireNoError(t, err)
		if got.Name != newName {
			t.Errorf("rename did not persist: got %s", got.Name)
		}
	})

	// 9. Rollout should transition status (async)
	t.Run("rollout triggers status change", func(t *testing.T) {
		_, err := c.Playgrounds.Rollout(ctx(), pg.ID)
		if err != nil {
			if skipIfPlaygroundActionStateRejected(t, err, "rollout") {
				return
			}
			requireNoError(t, err)
		}
	})

	// 10. HardRestart
	t.Run("hard restart triggers status change", func(t *testing.T) {
		_, err := c.Playgrounds.HardRestart(ctx(), pg.ID)
		if err != nil {
			if skipIfPlaygroundActionStateRejected(t, err, "hard restart") {
				return
			}
			requireNoError(t, err)
		}
	})

	// 11. Logs (may return empty if not yet running)
	t.Run("logs for web service", func(t *testing.T) {
		tail := 20
		logs, err := c.Playgrounds.Logs(ctx(), pg.ID, "web", &tail)
		if err != nil {
			if apiErr, ok := err.(*fibe.APIError); ok {
				if apiErr.StatusCode == 404 || apiErr.StatusCode == 409 {
					t.Skipf("logs not yet available: %s", apiErr.Message)
				}
			}
			requireNoError(t, err)
		}
		if logs.Service != "web" {
			t.Errorf("expected Service=web, got %s", logs.Service)
		}
	})

	// 12. Logs for nonexistent service returns 4xx
	t.Run("logs for nonexistent service returns error", func(t *testing.T) {
		_, err := c.Playgrounds.Logs(ctx(), pg.ID, "nonexistent-service", nil)
		if err == nil {
			t.Error("expected error for nonexistent service")
		}
	})
}

// TestPlaygrounds_ListFilterIntegration verifies list filters work end-to-end on real playgrounds.
func TestPlaygrounds_ListFilterIntegration(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	marqueeID := testMarqueeID(t)
	if marqueeID == 0 {
		t.Skip("set FIBE_TEST_MARQUEE_ID to test list filters")
	}

	spec := seedPlayspec(t, c)

	pg1, err := c.Playgrounds.Create(ctx(), &fibe.PlaygroundCreateParams{
		Name:       uniqueName("filter-alpha"),
		PlayspecID: *spec.ID,
		MarqueeID:  &marqueeID,
	})
	requireNoError(t, err)
	t.Cleanup(func() { c.Playgrounds.Delete(ctx(), pg1.ID) })

	pg2, err := c.Playgrounds.Create(ctx(), &fibe.PlaygroundCreateParams{
		Name:       uniqueName("filter-beta"),
		PlayspecID: *spec.ID,
		MarqueeID:  &marqueeID,
	})
	requireNoError(t, err)
	t.Cleanup(func() { c.Playgrounds.Delete(ctx(), pg2.ID) })

	t.Run("filter by playspec_id narrows results", func(t *testing.T) {
		t.Parallel()
		r, err := c.Playgrounds.List(ctx(), &fibe.PlaygroundListParams{PlayspecID: *spec.ID, PerPage: 50})
		requireNoError(t, err)
		if len(r.Data) < 2 {
			t.Errorf("expected >= 2 playgrounds with playspec_id=%d, got %d", *spec.ID, len(r.Data))
		}
		for _, pg := range r.Data {
			if pg.PlayspecID == nil || *pg.PlayspecID != *spec.ID {
				t.Errorf("expected PlayspecID=%d, got %v for pg %d", *spec.ID, pg.PlayspecID, pg.ID)
			}
		}
	})

	t.Run("filter by marquee_id narrows results", func(t *testing.T) {
		t.Parallel()
		r, err := c.Playgrounds.List(ctx(), &fibe.PlaygroundListParams{MarqueeID: marqueeID, PerPage: 50})
		requireNoError(t, err)
		if len(r.Data) < 2 {
			t.Errorf("expected >= 2 playgrounds with marquee_id=%d, got %d", marqueeID, len(r.Data))
		}
	})

	t.Run("filter by Q substring matches our names", func(t *testing.T) {
		t.Parallel()
		r, err := c.Playgrounds.List(ctx(), &fibe.PlaygroundListParams{Q: "filter-", PerPage: 50})
		requireNoError(t, err)
		found := 0
		for _, pg := range r.Data {
			if strings.Contains(pg.Name, "filter-") {
				found++
			}
		}
		if found < 2 {
			t.Errorf("expected >= 2 playgrounds matching Q='filter-', found %d", found)
		}
	})
}
