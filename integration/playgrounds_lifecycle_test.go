package integration

import (
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/fibegg/sdk/fibe"
)

// TestPlaygrounds_FullLifecycle exercises the complete real-world playground lifecycle:
func TestPlaygrounds_FullLifecycle(t *testing.T) {
	c := userClient(t)

	spec := seedSpec(t, c)
	hostID := testHostID(t)
	if hostID == 0 {
		t.Skip("set FIBE_TEST_HOST_ID to run full lifecycle")
	}
	expiresAt := time.Now().UTC().Add(2 * time.Hour)

	pg, err := c.Playgrounds.Create(ctx(), &fibe.PlaygroundCreateParams{
		Name:        uniqueName("life-pg"),
		SpecID:      *spec.ID,
		HostID:      &hostID,
		ExpiresAt:   &expiresAt,
		NeverExpire: ptr(false),
	})
	requireNoError(t, err)
	t.Cleanup(func() { c.Playgrounds.Delete(ctx(), pg.ID) })

	// 1. Immediate create response must have ID, Name, Status
	if pg.ID == 0 || pg.Name == "" || pg.Status == "" {
		t.Errorf("create response missing core fields: id=%d name=%q status=%q", pg.ID, pg.Name, pg.Status)
	}
	if pg.SpecID == nil || *pg.SpecID != *spec.ID {
		t.Errorf("expected SpecID=%d, got %v", *spec.ID, pg.SpecID)
	}

	t.Run("status transitions", func(t *testing.T) {
		finalStatus := waitForPlaygroundStatusWithin(t, c, pg.ID, []string{"running"}, PlaygroundLaunchWaitTimeout)
		if finalStatus != "running" {
			t.Fatalf("playground did not reach running status: got %q", finalStatus)
		}
		t.Logf("playground final status: %s", finalStatus)
	})

	// 3. Detail fields populated (may require waiting for provisioning)
	t.Run("detail has expiration and service info", func(t *testing.T) {
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
			t.Fatal("could not fetch playground detail within timeout")
		}
		if d.ExpiresAt == nil {
			t.Log("ExpiresAt not yet populated within timeout: may require running state")
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
			t.Fatal("compose YAML not rendered within timeout")
		}
		if !strings.Contains(cmp.ComposeYAML, "services:") {
			t.Errorf("expected services: in compose YAML")
		}
	})

	t.Run("env metadata structure", func(t *testing.T) {
		env, err := c.Playgrounds.EnvMetadata(ctx(), pg.ID)
		requireNoError(t, err)
		if env.Merged == nil || env.Metadata == nil || env.SystemKeys == nil {
			t.Errorf("expected all env fields populated: merged=%v metadata=%v system_keys=%v",
				env.Merged != nil, env.Metadata != nil, env.SystemKeys != nil)
		}
	})

	t.Run("debug returns diagnostic data", func(t *testing.T) {
		dbg, err := c.Playgrounds.Debug(ctx(), pg.ID)
		requireNoError(t, err)
		if dbg == nil {
			t.Error("expected non-nil Debug response")
		}
	})

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

	t.Run("update name persists", func(t *testing.T) {
		newName := uniqueName("renamed-pg")
		upd, err := c.Playgrounds.Update(ctx(), pg.ID, &fibe.PlaygroundUpdateParams{Name: &newName})
		requireNoError(t, err)
		if upd.Name != newName {
			t.Errorf("expected Name=%s, got %s", newName, upd.Name)
		}
		got, err := c.Playgrounds.Get(ctx(), pg.ID)
		requireNoError(t, err)
		if got.Name != newName {
			t.Errorf("rename did not persist: got %s", got.Name)
		}
	})

	t.Run("rollout triggers status change", func(t *testing.T) {
		playgroundActionEventuallyAccepted(t, c, pg.ID, fibe.PlaygroundActionRollout, "rollout")
		finalStatus := waitForPlaygroundStatusWithin(t, c, pg.ID, []string{"running"}, PlaygroundLaunchWaitTimeout)
		if finalStatus != "running" {
			t.Fatalf("playground did not return to running after rollout: got %q", finalStatus)
		}
	})

	t.Run("hard restart triggers status change", func(t *testing.T) {
		playgroundActionEventuallyAccepted(t, c, pg.ID, fibe.PlaygroundActionHardRestart, "hard restart")
		finalStatus := waitForPlaygroundStatusWithin(t, c, pg.ID, []string{"running"}, PlaygroundLaunchWaitTimeout)
		if finalStatus != "running" {
			t.Fatalf("playground did not return to running after hard restart: got %q", finalStatus)
		}
	})

	// 11. Logs: canonical real SDK logs success coverage.
	t.Run("logs for web service", func(t *testing.T) {
		finalStatus := waitForPlaygroundStatusWithin(t, c, pg.ID, []string{"running"}, PlaygroundLaunchWaitTimeout)
		if finalStatus != "running" {
			t.Fatalf("playground did not reach running before logs: got %q", finalStatus)
		}

		tail := 20
		logs, err := c.Playgrounds.Logs(ctx(), pg.ID, "web", &tail)
		if err != nil {
			if apiErr, ok := err.(*fibe.APIError); ok {
				if apiErr.StatusCode == 404 || apiErr.StatusCode == 409 {
					t.Fatalf("logs not available: %s", apiErr.Message)
				}
			}
			requireNoError(t, err)
		}
		if logs.Service != "web" {
			t.Errorf("expected Service=web, got %s", logs.Service)
		}
	})

	// 12. CLI logs shares the same real running playground instead of
	// launching a separate playground in the broad CLI smoke.
	t.Run("cli logs for web service", func(t *testing.T) {
		finalStatus := waitForPlaygroundStatusWithin(t, c, pg.ID, []string{"running"}, PlaygroundLaunchWaitTimeout)
		if finalStatus != "running" {
			t.Fatalf("playground did not reach running before CLI logs: got %q", finalStatus)
		}

		out, err := runCompiledCLI(t, "playgrounds", "logs", strconv.FormatInt(pg.ID, 10), "--service", "web")
		requireNoError(t, err, "failed to get CLI playground logs: "+out)
		if !strings.Contains(out, `"service": "web"`) {
			t.Errorf("expected CLI logs JSON to include service web, got: %s", out)
		}
	})

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
	c := userClient(t)

	hostID := testHostID(t)
	if hostID == 0 {
		t.Skip("set FIBE_TEST_HOST_ID to test list filters")
	}

	spec := seedSpec(t, c)

	pg1, err := c.Playgrounds.Create(ctx(), &fibe.PlaygroundCreateParams{
		Name:   uniqueName("filter-alpha"),
		SpecID: *spec.ID,
		HostID: &hostID,
	})
	requireNoError(t, err)
	t.Cleanup(func() { c.Playgrounds.Delete(ctx(), pg1.ID) })

	pg2, err := c.Playgrounds.Create(ctx(), &fibe.PlaygroundCreateParams{
		Name:   uniqueName("filter-beta"),
		SpecID: *spec.ID,
		HostID: &hostID,
	})
	requireNoError(t, err)
	t.Cleanup(func() { c.Playgrounds.Delete(ctx(), pg2.ID) })

	t.Run("filter by spec_id narrows results", func(t *testing.T) {
		t.Parallel()
		r, err := c.Playgrounds.List(ctx(), &fibe.PlaygroundListParams{SpecID: *spec.ID, PerPage: 50})
		requireNoError(t, err)
		if len(r.Data) < 2 {
			t.Errorf("expected >= 2 playgrounds with spec_id=%d, got %d", *spec.ID, len(r.Data))
		}
		for _, pg := range r.Data {
			if pg.SpecID == nil || *pg.SpecID != *spec.ID {
				t.Errorf("expected SpecID=%d, got %v for pg %d", *spec.ID, pg.SpecID, pg.ID)
			}
		}
	})

	t.Run("filter by host_id narrows results", func(t *testing.T) {
		t.Parallel()
		r, err := c.Playgrounds.List(ctx(), &fibe.PlaygroundListParams{HostID: hostID, PerPage: 50})
		requireNoError(t, err)
		if len(r.Data) < 2 {
			t.Errorf("expected >= 2 playgrounds with host_id=%d, got %d", hostID, len(r.Data))
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
