package integration

import (
	"testing"

	"github.com/fibegg/sdk/fibe"
)

func TestSecrets_CRUD(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	var secretID int64

	t.Run("create secret", func(t *testing.T) {
		// Parallel disabled: dependent sequence
		s, err := c.Secrets.Create(ctx(), &fibe.SecretCreateParams{
			Key:         uniqueName("TEST_SECRET"),
			Value:       "super-secret-value-123",
			Description: ptr("integration test secret"),
		})
		requireNoError(t, err)

		if s.ID == nil {
			t.Fatal("expected secret ID")
		}
		secretID = *s.ID
		if s.Key == "" {
			t.Error("expected key name")
		}
	})
	t.Cleanup(func() {
		if secretID > 0 {
			c.Secrets.Delete(ctx(), secretID)
		}
	})

	t.Run("list secrets returns data+meta", func(t *testing.T) {
		t.Parallel()
		result, err := c.Secrets.List(ctx(), nil)
		requireNoError(t, err)

		if result.Meta.Total == 0 {
			t.Error("expected total > 0")
		}
		if result.Meta.Page != 1 {
			t.Errorf("expected page 1, got %d", result.Meta.Page)
		}
	})

	t.Run("get secret reveals value", func(t *testing.T) {
		// Parallel disabled: dependent block
		if secretID == 0 {
			t.Skip("no secret created")
		}
		s, err := c.Secrets.Get(ctx(), secretID)
		requireNoError(t, err)

		if s.Value == nil || *s.Value != "super-secret-value-123" {
			t.Errorf("expected value 'super-secret-value-123', got %v", s.Value)
		}
	})

	t.Run("update secret value", func(t *testing.T) {
		// Parallel disabled: dependent block
		if secretID == 0 {
			t.Skip("no secret created")
		}
		s, err := c.Secrets.Update(ctx(), secretID, &fibe.SecretUpdateParams{
			Value:       ptr("updated-value"),
			Description: ptr("updated desc"),
		})
		requireNoError(t, err)

		if s.Description == nil || *s.Description != "updated desc" {
			t.Error("expected updated description")
		}

		got, err := c.Secrets.Get(ctx(), secretID)
		requireNoError(t, err)
		if got.Value == nil || *got.Value != "updated-value" {
			t.Error("expected updated value")
		}
	})

	t.Run("delete secret", func(t *testing.T) {
		t.Parallel()
		s, err := c.Secrets.Create(ctx(), &fibe.SecretCreateParams{
			Key:   uniqueName("DELETE_ME"),
			Value: "temp",
		})
		requireNoError(t, err)

		err = c.Secrets.Delete(ctx(), *s.ID)
		requireNoError(t, err)

		_, err = c.Secrets.Get(ctx(), *s.ID)
		requireAPIError(t, err, fibe.ErrCodeNotFound, 404)
	})
}

func TestSecrets_ScopeIsolation(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	s, err := c.Secrets.Create(ctx(), &fibe.SecretCreateParams{
		Key:   uniqueName("SCOPE_TEST"),
		Value: "hidden",
	})
	requireNoError(t, err)
	t.Cleanup(func() { c.Secrets.Delete(ctx(), *s.ID) })

	t.Run("read-only key cannot create", func(t *testing.T) {
		t.Parallel()
		readOnly := createScopedKey(t, c, "secrets-read", []string{"secrets:read"})
		_, err := readOnly.Secrets.Create(ctx(), &fibe.SecretCreateParams{
			Key:   "SHOULD_FAIL",
			Value: "nope",
		})
		requireAPIError(t, err, fibe.ErrCodeForbidden, 403)
	})

	t.Run("read-only key can list and get", func(t *testing.T) {
		t.Parallel()
		readOnly := createScopedKey(t, c, "secrets-read2", []string{"secrets:read"})

		result, err := readOnly.Secrets.List(ctx(), nil)
		requireNoError(t, err)
		if result.Meta.Total == 0 {
			t.Error("expected to see secrets")
		}

		got, err := readOnly.Secrets.Get(ctx(), *s.ID)
		requireNoError(t, err)
		if got.Key == "" {
			t.Error("expected key name")
		}
	})

	t.Run("no-scope key cannot access secrets", func(t *testing.T) {
		t.Parallel()
		noScope := createScopedKey(t, c, "no-secrets", []string{"agents:read"})
		_, err := noScope.Secrets.List(ctx(), nil)
		requireAPIError(t, err, fibe.ErrCodeForbidden, 403)
	})
}

func TestSecrets_ValidationErrors(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	t.Run("duplicate key name", func(t *testing.T) {
		t.Parallel()
		name := uniqueName("DUPE_KEY")
		s, err := c.Secrets.Create(ctx(), &fibe.SecretCreateParams{
			Key:   name,
			Value: "first",
		})
		requireNoError(t, err)
		t.Cleanup(func() { c.Secrets.Delete(ctx(), *s.ID) })

		_, err = c.Secrets.Create(ctx(), &fibe.SecretCreateParams{
			Key:   name,
			Value: "second",
		})
		requireAPIError(t, err, fibe.ErrCodeValidationFailed, 422)
	})
}
