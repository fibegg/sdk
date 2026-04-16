package integration

import (
	"testing"
)

func TestClient_Ping(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	t.Run("ping succeeds with valid key", func(t *testing.T) {
		t.Parallel()
		err := c.Ping(ctx())
		requireNoError(t, err)
	})

	t.Run("ping fails with invalid key", func(t *testing.T) {
		t.Parallel()
		bad := c.WithKey("fibe_invalid_key_" + uniqueName(""))
		err := bad.Ping(ctx())
		if err == nil {
			t.Error("expected error with invalid key")
		}
	})
}

func TestStatus_Get(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	t.Run("status returns resource summary", func(t *testing.T) {
		t.Parallel()
		s, err := c.Status.Get(ctx())
		requireNoError(t, err)

		// Counts must be non-negative
		if s.Playgrounds.Total < 0 {
			t.Errorf("expected Playgrounds.Total >= 0, got %d", s.Playgrounds.Total)
		}
		if s.Playgrounds.Active < 0 {
			t.Errorf("expected Playgrounds.Active >= 0, got %d", s.Playgrounds.Active)
		}
		if s.Agents.Total < 0 {
			t.Errorf("expected Agents.Total >= 0, got %d", s.Agents.Total)
		}
		if s.Props < 0 {
			t.Errorf("expected Props >= 0, got %d", s.Props)
		}
		if s.Playspecs < 0 {
			t.Errorf("expected Playspecs >= 0, got %d", s.Playspecs)
		}
		if s.APIKeys < 0 {
			t.Errorf("expected APIKeys >= 0, got %d", s.APIKeys)
		}
		if s.Subscription.Plan == "" {
			t.Error("expected non-empty Subscription.Plan")
		}
	})

	t.Run("status counts update after creating a secret", func(t *testing.T) {
		t.Parallel()
		s := seedSecret(t, c, "status-count")
		after, err := c.Status.Get(ctx())
		requireNoError(t, err)
		if s.ID == nil {
			t.Skip("seed secret returned no ID")
		}
		if after.Secrets < 1 {
			t.Errorf("expected at least 1 secret after seeding, got %d", after.Secrets)
		}
	})
}

func TestClient_LastRequestID(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	_, err := c.APIKeys.Me(ctx())
	requireNoError(t, err)
	rid := c.LastRequestID()
	if rid == "" {
		t.Error("expected LastRequestID to be set after successful request")
	}
}
