package integration

import (
	"testing"

	"github.com/fibegg/sdk/fibe"
)

func TestCrossKey_Isolation(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	adminSecret, err := c.Secrets.Create(ctx(), &fibe.SecretCreateParams{
		Key:   uniqueName("ADMIN_SECRET"),
		Value: "admin-value",
	})
	requireNoError(t, err)
	t.Cleanup(func() { c.Secrets.Delete(ctx(), *adminSecret.ID) })

	t.Run("different key same player sees same resources", func(t *testing.T) {
		t.Parallel()
		otherKey := createScopedKey(t, c, "same-player", []string{"secrets:read"})

		s, err := otherKey.Secrets.Get(ctx(), *adminSecret.ID)
		requireNoError(t, err)
		if s.Key != adminSecret.Key {
			t.Error("expected same secret visible from different key")
		}
	})

	t.Run("key with wrong scope cannot access", func(t *testing.T) {
		t.Parallel()
		wrongScope := createScopedKey(t, c, "wrong-scope", []string{"agents:read"})

		_, err := wrongScope.Secrets.Get(ctx(), *adminSecret.ID)
		requireAPIError(t, err, fibe.ErrCodeForbidden, 403)
	})

	t.Run("deleted key immediately stops working", func(t *testing.T) {
		t.Parallel()
		key, err := c.APIKeys.Create(ctx(), &fibe.APIKeyCreateParams{
			Label:  uniqueName("ephemeral"),
			Scopes: []string{"secrets:read"},
		})
		requireNoError(t, err)

		ephemeral := c.WithKey(*key.Token)

		_, err = ephemeral.Secrets.List(ctx(), nil)
		requireNoError(t, err, "should work before deletion")

		err = c.APIKeys.Delete(ctx(), *key.ID)
		requireNoError(t, err)

		_, err = ephemeral.Secrets.List(ctx(), nil)
		requireAPIError(t, err, fibe.ErrCodeUnauthorized, 401)
	})
}

func TestCrossKey_WithKeyConvenience(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	key1, err := c.APIKeys.Create(ctx(), &fibe.APIKeyCreateParams{
		Label:  uniqueName("key-1"),
		Scopes: []string{"secrets:read"},
	})
	requireNoError(t, err)
	t.Cleanup(func() { c.APIKeys.Delete(ctx(), *key1.ID) })

	key2, err := c.APIKeys.Create(ctx(), &fibe.APIKeyCreateParams{
		Label:  uniqueName("key-2"),
		Scopes: []string{"agents:read"},
	})
	requireNoError(t, err)
	t.Cleanup(func() { c.APIKeys.Delete(ctx(), *key2.ID) })

	client1 := c.WithKey(*key1.Token)
	client2 := c.WithKey(*key2.Token)

	t.Run("client1 can access secrets but not agents", func(t *testing.T) {
		t.Parallel()
		_, err := client1.Secrets.List(ctx(), nil)
		requireNoError(t, err)

		_, err = client1.Agents.List(ctx(), nil)
		requireAPIError(t, err, fibe.ErrCodeForbidden, 403)
	})

	t.Run("client2 can access agents but not secrets", func(t *testing.T) {
		t.Parallel()
		_, err := client2.Agents.List(ctx(), nil)
		requireNoError(t, err)

		_, err = client2.Secrets.List(ctx(), nil)
		requireAPIError(t, err, fibe.ErrCodeForbidden, 403)
	})

	t.Run("admin can access everything", func(t *testing.T) {
		t.Parallel()
		_, err := c.Secrets.List(ctx(), nil)
		requireNoError(t, err)
		_, err = c.Agents.List(ctx(), nil)
		requireNoError(t, err)
	})
}

func TestCrossKey_BaseURLShared(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	child := c.WithKey("different-key")

	if c.BaseURL() != child.BaseURL() {
		t.Errorf("WithKey should share base URL: parent=%q child=%q", c.BaseURL(), child.BaseURL())
	}
}
