package integration

import (
	"errors"
	"testing"

	"github.com/fibegg/sdk/fibe"
)

func TestErrors_NotFound(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	t.Run("nonexistent playground", func(t *testing.T) {
		t.Parallel()
		_, err := c.Playgrounds.Get(ctx(), 999999999)
		apiErr := requireAPIError(t, err, fibe.ErrCodeNotFound, 404)
		if !apiErr.IsNotFound() {
			t.Error("IsNotFound() should be true")
		}
	})

	t.Run("nonexistent agent", func(t *testing.T) {
		t.Parallel()
		_, err := c.Agents.Get(ctx(), 999999999)
		requireAPIError(t, err, fibe.ErrCodeNotFound, 404)
	})

	t.Run("nonexistent playspec", func(t *testing.T) {
		t.Parallel()
		_, err := c.Playspecs.Get(ctx(), 999999999)
		requireAPIError(t, err, fibe.ErrCodeNotFound, 404)
	})

	t.Run("nonexistent prop", func(t *testing.T) {
		t.Parallel()
		_, err := c.Props.Get(ctx(), 999999999)
		requireAPIError(t, err, fibe.ErrCodeNotFound, 404)
	})

	t.Run("nonexistent secret", func(t *testing.T) {
		t.Parallel()
	_, err := c.Secrets.Get(ctx(), 999999999, false)
		requireAPIError(t, err, fibe.ErrCodeNotFound, 404)
	})

	t.Run("nonexistent team", func(t *testing.T) {
		t.Parallel()
		_, err := c.Teams.Get(ctx(), 999999999)
		requireAPIError(t, err, fibe.ErrCodeNotFound, 404)
	})

	t.Run("nonexistent webhook endpoint", func(t *testing.T) {
		t.Parallel()
		_, err := c.WebhookEndpoints.Get(ctx(), 999999999)
		requireAPIError(t, err, fibe.ErrCodeNotFound, 404)
	})
}

func TestErrors_Unauthorized(t *testing.T) {
	t.Parallel()
	c := adminClient(t)
	bad := c.WithKey("totally-invalid-key")

	t.Run("invalid token gets 401", func(t *testing.T) {
		t.Parallel()
		_, err := bad.APIKeys.Me(ctx())
		apiErr := requireAPIError(t, err, fibe.ErrCodeUnauthorized, 401)
		if !apiErr.IsUnauthorized() {
			t.Error("IsUnauthorized() should be true")
		}
	})

	t.Run("invalid token on list", func(t *testing.T) {
		t.Parallel()
		_, err := bad.Playgrounds.List(ctx(), nil)
		requireAPIError(t, err, fibe.ErrCodeUnauthorized, 401)
	})
}

func TestErrors_Forbidden(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	readOnly := createScopedKey(t, c, "forbidden-test", []string{"playgrounds:read"})

	t.Run("403 has correct code", func(t *testing.T) {
		t.Parallel()
		_, err := readOnly.Secrets.List(ctx(), nil)
		apiErr := requireAPIError(t, err, fibe.ErrCodeForbidden, 403)
		if !apiErr.IsForbidden() {
			t.Error("IsForbidden() should be true")
		}
	})
}

func TestErrors_ValidationFailed(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	t.Run("client-side validation", func(t *testing.T) {
		t.Parallel()
		_, err := c.Agents.Create(ctx(), &fibe.AgentCreateParams{
			Name:     "",
			Provider: fibe.ProviderGemini,
		})
		if err == nil {
			t.Fatal("expected validation error")
		}
		var ve fibe.ValidationErrors
		if !errors.As(err, &ve) {
			t.Fatalf("expected ValidationErrors, got %T", err)
		}
	})

	t.Run("server-side validation", func(t *testing.T) {
		t.Parallel()
		_, err := c.Playgrounds.Create(ctx(), &fibe.PlaygroundCreateParams{
			Name:       uniqueName("bad-pg"),
			PlayspecID: 999999999,
		})
		apiErr := requireAPIError(t, err, fibe.ErrCodeValidationFailed, 422)
		if !apiErr.IsValidation() {
			t.Error("IsValidation() should be true")
		}
	})
}

func TestErrors_ErrorsAs(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	_, err := c.Playgrounds.Get(ctx(), 999999999)
	if err == nil {
		t.Fatal("expected error")
	}

	var apiErr *fibe.APIError
	if !errors.As(err, &apiErr) {
		t.Fatal("errors.As should work with *fibe.APIError")
	}
	if apiErr.StatusCode != 404 {
		t.Errorf("expected 404, got %d", apiErr.StatusCode)
	}
}

func TestErrors_NotRetryable(t *testing.T) {
	t.Parallel()
	c := adminClient(t)
	bad := c.WithKey("bad-key")

	_, err := bad.APIKeys.Me(ctx())
	var apiErr *fibe.APIError
	if errors.As(err, &apiErr) {
		if apiErr.IsRetryable() {
			t.Error("401 should not be retryable")
		}
	}
}
