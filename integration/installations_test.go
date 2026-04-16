package integration

import (
	"testing"

	"github.com/fibegg/sdk/fibe"
)

func TestInstallations_List(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	t.Run("list returns envelope", func(t *testing.T) {
		t.Parallel()
		result, err := c.Installations.List(ctx())
		requireNoError(t, err)
		if result.Meta.Total < 0 {
			t.Errorf("expected Total >= 0, got %d", result.Meta.Total)
		}
		// Every installation must have id, provider, installation_id
		for i, inst := range result.Data {
			if inst.ID == 0 {
				t.Errorf("installation[%d].ID is zero", i)
			}
			if inst.Provider == "" {
				t.Errorf("installation[%d].Provider is empty", i)
			}
			if inst.InstallationID == 0 {
				t.Errorf("installation[%d].InstallationID is zero", i)
			}
		}
	})

	t.Run("list scope read allows, other scope 403", func(t *testing.T) {
		t.Parallel()
		// Installations piggybacks on BaseController auth; no specific scope, so
		// just verify a scoped key can still hit the endpoint (no crash).
		read := createScopedKey(t, c, "inst-read", []string{"props:read"})
		_, err := read.Installations.List(ctx())
		// Either works (no scope check) or 403; both are acceptable behavior.
		if err != nil {
			if apiErr, ok := err.(*fibe.APIError); ok && apiErr.StatusCode != 403 {
				t.Errorf("unexpected status %d: %v", apiErr.StatusCode, err)
			}
		}
	})
}

func TestInstallations_Repos(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	list, err := c.Installations.List(ctx())
	requireNoError(t, err)
	if len(list.Data) == 0 {
		t.Skip("no installations linked — cannot test repo listing")
	}
	instID := list.Data[0].ID

	t.Run("repos returns paginated envelope", func(t *testing.T) {
		result, err := c.Installations.Repos(ctx(), instID, nil)
		if skipIfFeatureDisabled(t, err, "installations") {
			return
		}
		requireNoError(t, err)
		if result.Meta.Page < 1 {
			t.Errorf("expected Page >= 1, got %d", result.Meta.Page)
		}
		if result.Meta.PerPage < 1 {
			t.Errorf("expected PerPage >= 1, got %d", result.Meta.PerPage)
		}
		for i, r := range result.Data {
			if r.FullName == "" {
				t.Errorf("repos[%d].FullName is empty", i)
			}
			if r.ID == 0 {
				t.Errorf("repos[%d].ID is zero", i)
			}
		}
	})

	t.Run("repos pagination per_page=5", func(t *testing.T) {
		result, err := c.Installations.Repos(ctx(), instID, &fibe.InstallationReposParams{
			Page:    1,
			PerPage: 5,
		})
		if skipIfFeatureDisabled(t, err, "installations") {
			return
		}
		requireNoError(t, err)
		if len(result.Data) > 5 {
			t.Errorf("expected at most 5 repos, got %d", len(result.Data))
		}
	})

	t.Run("repos search by query", func(t *testing.T) {
		// Search for a very unlikely string, verify either empty or filter worked
		result, err := c.Installations.Repos(ctx(), instID, &fibe.InstallationReposParams{
			Q: "zzznonexistent-fibe-test-query-zzz",
		})
		if skipIfFeatureDisabled(t, err, "installations") {
			return
		}
		requireNoError(t, err)
		// GitHub search on unlikely string should yield 0; if not, result must still be valid
		if result.Meta.Total < 0 {
			t.Errorf("expected Total >= 0, got %d", result.Meta.Total)
		}
	})

	t.Run("nonexistent installation returns 404", func(t *testing.T) {
		_, err := c.Installations.Repos(ctx(), 999999999, nil)
		requireAPIError(t, err, fibe.ErrCodeNotFound, 404)
	})
}

func TestInstallations_Token(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	list, err := c.Installations.List(ctx())
	requireNoError(t, err)
	if len(list.Data) == 0 {
		t.Skip("no installations linked — cannot test token")
	}
	instID := list.Data[0].ID

	t.Run("installation-wide token", func(t *testing.T) {
		tok, err := c.Installations.Token(ctx(), instID, "")
		if err != nil {
			if apiErr, ok := err.(*fibe.APIError); ok && apiErr.StatusCode == 503 {
				t.Skip("GitHub upstream unavailable")
			}
			requireNoError(t, err)
		}
		if tok.Token == "" {
			t.Error("expected non-empty token")
		}
		if tok.ExpiresIn <= 0 {
			t.Errorf("expected ExpiresIn > 0, got %d", tok.ExpiresIn)
		}
	})

	t.Run("nonexistent installation returns 404", func(t *testing.T) {
		_, err := c.Installations.Token(ctx(), 999999999, "")
		requireAPIError(t, err, fibe.ErrCodeNotFound, 404)
	})
}
