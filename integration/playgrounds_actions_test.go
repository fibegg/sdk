package integration

import (
	"fmt"
	"testing"

	"github.com/fibegg/sdk/fibe"
)

func TestPlaygrounds_Actions(t *testing.T) {
	t.Parallel()
	c := adminClient(t)
	specID, marqueeID := setupPlaygroundDeps(t, c)

	if marqueeID == 0 {
		t.Skip("set FIBE_TEST_MARQUEE_ID to test playground actions")
	}

	pg, err := c.Playgrounds.Create(ctx(), &fibe.PlaygroundCreateParams{
		Name:       uniqueName("pg-actions"),
		PlayspecID: specID,
		MarqueeID:  &marqueeID,
	})
	requireNoError(t, err, "create playground for actions test")
	t.Cleanup(func() { c.Playgrounds.Delete(ctx(), pg.ID) })

	t.Run("status returns valid response", func(t *testing.T) {
		t.Parallel()
		status, err := c.Playgrounds.Status(ctx(), pg.ID)
		requireNoError(t, err)

		if status.Status == "" {
			t.Error("expected non-empty status string")
		}
		if status.ID != pg.ID {
			t.Errorf("expected ID %d, got %d", pg.ID, status.ID)
		}
	})

	t.Run("compose returns response", func(t *testing.T) {
		t.Parallel()
		compose, err := c.Playgrounds.Compose(ctx(), pg.ID)
		requireNoError(t, err)

		if compose == nil {
			t.Fatal("expected compose response to be non-nil")
		}
	})

	t.Run("env_metadata returns structure", func(t *testing.T) {
		t.Parallel()
		meta, err := c.Playgrounds.EnvMetadata(ctx(), pg.ID)
		requireNoError(t, err)

		if meta.Merged == nil {
			t.Error("expected merged map")
		}
	})

	t.Run("debug returns data", func(t *testing.T) {
		t.Parallel()
		debug, err := c.Playgrounds.Debug(ctx(), pg.ID)
		requireNoError(t, err)

		if debug == nil {
			t.Error("expected non-nil debug data")
		}
	})

	t.Run("logs for known service", func(t *testing.T) {
		t.Parallel()
		logs, err := c.Playgrounds.Logs(ctx(), pg.ID, "web", ptr(10))
		requireNoError(t, err)

		if logs.Service != "web" {
			t.Errorf("expected service 'web', got %q", logs.Service)
		}
	})

	t.Run("logs for unknown service returns error", func(t *testing.T) {
		t.Parallel()
		_, err := c.Playgrounds.Logs(ctx(), pg.ID, "nonexistent-service", nil)
		if err == nil {
			t.Error("expected error for unknown service")
		}
	})

	t.Run("extend_expiration extends time", func(t *testing.T) {
		t.Parallel()
		result, err := c.Playgrounds.ExtendExpiration(ctx(), pg.ID, nil)
		if err != nil && skipIfPlaygroundActionStateRejected(t, err, "extend expiration") {
			return
		}
		requireNoError(t, err)

		if result.ID != pg.ID {
			t.Errorf("expected ID %d, got %d", pg.ID, result.ID)
		}
		if result.ExpiresAt.IsZero() {
			t.Error("expected non-zero expires_at")
		}
	})

	t.Run("extend_expiration with custom duration", func(t *testing.T) {
		t.Parallel()
		result, err := c.Playgrounds.ExtendExpiration(ctx(), pg.ID, ptr(24))
		if err != nil && skipIfPlaygroundActionStateRejected(t, err, "extend expiration") {
			return
		}
		requireNoError(t, err)

		if result.TimeRemaining <= 0 {
			t.Error("expected positive time_remaining")
		}
	})

	t.Run("rollout triggers redeploy", func(t *testing.T) {
		t.Parallel()
		rolled, err := c.Playgrounds.Action(ctx(), pg.ID, &fibe.PlaygroundActionParams{ActionType: fibe.PlaygroundActionRollout})
		if err != nil && skipIfPlaygroundActionStateRejected(t, err, "rollout") {
			return
		}
		requireNoError(t, err)

		if rolled.ID != pg.ID {
			t.Errorf("expected ID %d, got %d", pg.ID, rolled.ID)
		}
	})

	t.Run("hard_restart triggers restart", func(t *testing.T) {
		t.Parallel()
		restarted, err := c.Playgrounds.Action(ctx(), pg.ID, &fibe.PlaygroundActionParams{ActionType: fibe.PlaygroundActionHardRestart})
		if err != nil && skipIfPlaygroundActionStateRejected(t, err, "hard restart") {
			return
		}
		requireNoError(t, err)

		if restarted.ID != pg.ID {
			t.Errorf("expected ID %d, got %d", pg.ID, restarted.ID)
		}
	})
}

func TestPlaygrounds_Actions_NonexistentID(t *testing.T) {
	t.Parallel()
	c := adminClient(t)
	badID := int64(999999999)

	t.Run("status returns 404", func(t *testing.T) {
		t.Parallel()
		_, err := c.Playgrounds.Status(ctx(), badID)
		requireAPIError(t, err, fibe.ErrCodeNotFound, 404)
	})

	t.Run("compose returns 404", func(t *testing.T) {
		t.Parallel()
		_, err := c.Playgrounds.Compose(ctx(), badID)
		requireAPIError(t, err, fibe.ErrCodeNotFound, 404)
	})

	t.Run("env_metadata returns 404", func(t *testing.T) {
		t.Parallel()
		_, err := c.Playgrounds.EnvMetadata(ctx(), badID)
		requireAPIError(t, err, fibe.ErrCodeNotFound, 404)
	})

	t.Run("debug returns 404", func(t *testing.T) {
		t.Parallel()
		_, err := c.Playgrounds.Debug(ctx(), badID)
		requireAPIError(t, err, fibe.ErrCodeNotFound, 404)
	})

	t.Run("logs returns 404", func(t *testing.T) {
		t.Parallel()
		_, err := c.Playgrounds.Logs(ctx(), badID, "web", nil)
		requireAPIError(t, err, fibe.ErrCodeNotFound, 404)
	})

	t.Run("rollout returns 404", func(t *testing.T) {
		t.Parallel()
		_, err := c.Playgrounds.Action(ctx(), badID, &fibe.PlaygroundActionParams{ActionType: fibe.PlaygroundActionRollout})
		requireAPIError(t, err, fibe.ErrCodeNotFound, 404)
	})

	t.Run("hard_restart returns 404", func(t *testing.T) {
		t.Parallel()
		_, err := c.Playgrounds.Action(ctx(), badID, &fibe.PlaygroundActionParams{ActionType: fibe.PlaygroundActionHardRestart})
		requireAPIError(t, err, fibe.ErrCodeNotFound, 404)
	})

	t.Run("extend_expiration returns 404", func(t *testing.T) {
		t.Parallel()
		_, err := c.Playgrounds.ExtendExpiration(ctx(), badID, nil)
		requireAPIError(t, err, fibe.ErrCodeNotFound, 404)
	})
}

func TestPlaygrounds_Actions_ScopeEnforcement(t *testing.T) {
	t.Parallel()
	c := adminClient(t)
	specID, marqueeID := setupPlaygroundDeps(t, c)

	if marqueeID == 0 {
		t.Skip("set FIBE_TEST_MARQUEE_ID to test playground action scopes")
	}

	pg, err := c.Playgrounds.Create(ctx(), &fibe.PlaygroundCreateParams{
		Name:       uniqueName("pg-scope-actions"),
		PlayspecID: specID,
		MarqueeID:  &marqueeID,
	})
	requireNoError(t, err)
	t.Cleanup(func() { c.Playgrounds.Delete(ctx(), pg.ID) })

	t.Run("read-only key can get status", func(t *testing.T) {
		t.Parallel()
		readOnly := createScopedKey(t, c, "pg-action-read", []string{"playgrounds:read"})
		_, err := readOnly.Playgrounds.Status(ctx(), pg.ID)
		requireNoError(t, err)
	})

	t.Run("read-only key can get compose", func(t *testing.T) {
		t.Parallel()
		readOnly := createScopedKey(t, c, "pg-action-compose", []string{"playgrounds:read"})
		_, err := readOnly.Playgrounds.Compose(ctx(), pg.ID)
		requireNoError(t, err)
	})

	t.Run("read-only key can get env_metadata", func(t *testing.T) {
		t.Parallel()
		readOnly := createScopedKey(t, c, "pg-action-envmeta", []string{"playgrounds:read"})
		_, err := readOnly.Playgrounds.EnvMetadata(ctx(), pg.ID)
		requireNoError(t, err)
	})

	t.Run("read-only key can get debug", func(t *testing.T) {
		t.Parallel()
		readOnly := createScopedKey(t, c, "pg-action-debug", []string{"playgrounds:read"})
		_, err := readOnly.Playgrounds.Debug(ctx(), pg.ID)
		requireNoError(t, err)
	})

	t.Run("read-only key can get logs", func(t *testing.T) {
		t.Parallel()
		readOnly := createScopedKey(t, c, "pg-action-logs", []string{"playgrounds:read"})
		_, err := readOnly.Playgrounds.Logs(ctx(), pg.ID, "web", nil)
		requireNoError(t, err)
	})

	t.Run("read-only key cannot rollout", func(t *testing.T) {
		t.Parallel()
		readOnly := createScopedKey(t, c, "pg-action-rollout", []string{"playgrounds:read"})
		_, err := readOnly.Playgrounds.Action(ctx(), pg.ID, &fibe.PlaygroundActionParams{ActionType: fibe.PlaygroundActionRollout})
		requireAPIError(t, err, fibe.ErrCodeForbidden, 403)
	})

	t.Run("read-only key cannot hard_restart", func(t *testing.T) {
		t.Parallel()
		readOnly := createScopedKey(t, c, "pg-action-restart", []string{"playgrounds:read"})
		_, err := readOnly.Playgrounds.Action(ctx(), pg.ID, &fibe.PlaygroundActionParams{ActionType: fibe.PlaygroundActionHardRestart})
		requireAPIError(t, err, fibe.ErrCodeForbidden, 403)
	})

	t.Run("read-only key cannot extend_expiration", func(t *testing.T) {
		t.Parallel()
		readOnly := createScopedKey(t, c, "pg-action-extend", []string{"playgrounds:read"})
		_, err := readOnly.Playgrounds.ExtendExpiration(ctx(), pg.ID, nil)
		requireAPIError(t, err, fibe.ErrCodeForbidden, 403)
	})

	t.Run("no playground scope denied for all actions", func(t *testing.T) {
		t.Parallel()
		noScope := createScopedKey(t, c, "pg-no-scope", []string{"agents:read"})

		_, err := noScope.Playgrounds.Status(ctx(), pg.ID)
		requireAPIError(t, err, fibe.ErrCodeForbidden, 403)

		_, err = noScope.Playgrounds.Compose(ctx(), pg.ID)
		requireAPIError(t, err, fibe.ErrCodeForbidden, 403)

		_, err = noScope.Playgrounds.Action(ctx(), pg.ID, &fibe.PlaygroundActionParams{ActionType: fibe.PlaygroundActionRollout})
		requireAPIError(t, err, fibe.ErrCodeForbidden, 403)
	})
}

func TestPlaygrounds_CreateWithServiceConfig(t *testing.T) {
	t.Parallel()
	c := adminClient(t)
	specID, marqueeID := setupPlaygroundDeps(t, c)

	if marqueeID == 0 {
		t.Skip("set FIBE_TEST_MARQUEE_ID to test playground creation with service config")
	}

	t.Run("create with _global env_vars", func(t *testing.T) {
		t.Parallel()
		pg, err := c.Playgrounds.Create(ctx(), &fibe.PlaygroundCreateParams{
			Name:       uniqueName("pg-global-env"),
			PlayspecID: specID,
			MarqueeID:  &marqueeID,
			Services: map[string]*fibe.ServiceConfig{
				"_global": {
					EnvVars: map[string]string{
						"GLOBAL_VAR": "test_value",
					},
				},
			},
		})
		requireNoError(t, err)
		t.Cleanup(func() { c.Playgrounds.Delete(ctx(), pg.ID) })

		if pg.ID == 0 {
			t.Error("expected valid playground ID")
		}
	})

	t.Run("create with per-service subdomain", func(t *testing.T) {
		t.Parallel()
		sub := fmt.Sprintf("sub-%d", nameCounter.Add(1))
		pg, err := c.Playgrounds.Create(ctx(), &fibe.PlaygroundCreateParams{
			Name:       uniqueName("pg-subdomain"),
			PlayspecID: specID,
			MarqueeID:  &marqueeID,
			Services: map[string]*fibe.ServiceConfig{
				"web": {
					Subdomain: sub,
				},
			},
		})
		requireNoError(t, err)
		t.Cleanup(func() { c.Playgrounds.Delete(ctx(), pg.ID) })

		if pg.ID == 0 {
			t.Error("expected valid playground ID")
		}
	})

	t.Run("create with env_vars per service", func(t *testing.T) {
		t.Parallel()
		pg, err := c.Playgrounds.Create(ctx(), &fibe.PlaygroundCreateParams{
			Name:       uniqueName("pg-svc-env"),
			PlayspecID: specID,
			MarqueeID:  &marqueeID,
			Services: map[string]*fibe.ServiceConfig{
				"web": {
					EnvVars: map[string]string{
						"SERVICE_VAR": "web_value",
					},
				},
			},
		})
		requireNoError(t, err)
		t.Cleanup(func() { c.Playgrounds.Delete(ctx(), pg.ID) })

		if pg.ID == 0 {
			t.Error("expected valid playground ID")
		}
	})
}

func TestPlaygrounds_IDOR(t *testing.T) {
	t.Parallel()
	c := adminClient(t)
	userB := userBClient(t)
	specID, marqueeID := setupPlaygroundDeps(t, c)

	if marqueeID == 0 {
		t.Skip("set FIBE_TEST_MARQUEE_ID to test playground IDOR")
	}

	pg, err := c.Playgrounds.Create(ctx(), &fibe.PlaygroundCreateParams{
		Name:       uniqueName("pg-idor"),
		PlayspecID: specID,
		MarqueeID:  &marqueeID,
	})
	requireNoError(t, err)
	t.Cleanup(func() { c.Playgrounds.Delete(ctx(), pg.ID) })

	t.Run("user B cannot get admin playground", func(t *testing.T) {
		t.Parallel()
		_, err := userB.Playgrounds.Get(ctx(), pg.ID)
		requireAPIError(t, err, fibe.ErrCodeNotFound, 404)
	})

	t.Run("user B cannot update admin playground", func(t *testing.T) {
		t.Parallel()
		newName := "hacked"
		_, err := userB.Playgrounds.Update(ctx(), pg.ID, &fibe.PlaygroundUpdateParams{
			Name: &newName,
		})
		requireAPIError(t, err, fibe.ErrCodeNotFound, 404)
	})

	t.Run("user B cannot delete admin playground", func(t *testing.T) {
		t.Parallel()
		err := userB.Playgrounds.Delete(ctx(), pg.ID)
		requireAPIError(t, err, fibe.ErrCodeNotFound, 404)
	})

	t.Run("user B cannot rollout admin playground", func(t *testing.T) {
		t.Parallel()
		_, err := userB.Playgrounds.Action(ctx(), pg.ID, &fibe.PlaygroundActionParams{ActionType: fibe.PlaygroundActionRollout})
		requireAPIError(t, err, fibe.ErrCodeNotFound, 404)
	})

	t.Run("user B cannot get status of admin playground", func(t *testing.T) {
		t.Parallel()
		_, err := userB.Playgrounds.Status(ctx(), pg.ID)
		requireAPIError(t, err, fibe.ErrCodeNotFound, 404)
	})

	t.Run("user B cannot get compose of admin playground", func(t *testing.T) {
		t.Parallel()
		_, err := userB.Playgrounds.Compose(ctx(), pg.ID)
		requireAPIError(t, err, fibe.ErrCodeNotFound, 404)
	})

	t.Run("user B cannot extend admin playground", func(t *testing.T) {
		t.Parallel()
		_, err := userB.Playgrounds.ExtendExpiration(ctx(), pg.ID, nil)
		requireAPIError(t, err, fibe.ErrCodeNotFound, 404)
	})
}
