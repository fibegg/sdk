package integration

import (
	"testing"

	"github.com/fibegg/sdk/fibe"
)

func TestAPIKeys_CRUD(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	t.Run("create key with specific scopes", func(t *testing.T) {
		t.Parallel()
		key, err := c.APIKeys.Create(ctx(), &fibe.APIKeyCreateParams{
			Label:  uniqueName("test-key"),
			Scopes: []string{"playgrounds:read", "agents:read"},
		})
		requireNoError(t, err)

		if key.ID == nil {
			t.Fatal("expected key ID")
		}
		if key.Token == nil {
			t.Fatal("expected token (only shown on creation)")
		}
		if key.Label == "" {
			t.Error("expected label")
		}

		t.Cleanup(func() { c.APIKeys.Delete(ctx(), *key.ID) })
	})

	t.Run("list keys", func(t *testing.T) {
		t.Parallel()
		result, err := c.APIKeys.List(ctx(), nil)
		requireNoError(t, err)

		if len(result.Data) == 0 {
			t.Error("expected at least one key (the admin key)")
		}
		if result.Meta.Total == 0 {
			t.Error("expected total > 0")
		}
		for _, k := range result.Data {
			if k.Token != nil {
				t.Error("token should NOT be exposed in list response")
			}
			if k.MaskedToken == "" {
				t.Error("expected masked_token in list response")
			}
		}
	})

	t.Run("delete key", func(t *testing.T) {
		t.Parallel()
		key, err := c.APIKeys.Create(ctx(), &fibe.APIKeyCreateParams{
			Label:  uniqueName("delete-me"),
			Scopes: []string{"agents:read"},
		})
		requireNoError(t, err)

		err = c.APIKeys.Delete(ctx(), *key.ID)
		requireNoError(t, err)

		deleted := c.WithKey(*key.Token)
		_, err = deleted.APIKeys.Me(ctx())
		requireAPIError(t, err, fibe.ErrCodeUnauthorized, 401)
	})
}

func TestAPIKeys_ScopeEnforcement(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	t.Run("read-only key cannot create", func(t *testing.T) {
		t.Parallel()
		readOnly := createScopedKey(t, c, "read-only", []string{"playgrounds:read"})

		_, err := readOnly.Playgrounds.Create(ctx(), &fibe.PlaygroundCreateParams{
			Name:       uniqueName("should-fail"),
			PlayspecID: 99999,
		})
		requireAPIError(t, err, fibe.ErrCodeForbidden, 403)
	})

	t.Run("wrong-scope key gets 403", func(t *testing.T) {
		t.Parallel()
		agentsOnly := createScopedKey(t, c, "agents-only", []string{"agents:read"})

		_, err := agentsOnly.Playgrounds.List(ctx(), nil)
		requireAPIError(t, err, fibe.ErrCodeForbidden, 403)
	})

	t.Run("keys:manage required to list keys", func(t *testing.T) {
		t.Parallel()
		noKeys := createScopedKey(t, c, "no-keys", []string{"playgrounds:read"})

		_, err := noKeys.APIKeys.List(ctx(), nil)
		requireAPIError(t, err, fibe.ErrCodeForbidden, 403)
	})
}
