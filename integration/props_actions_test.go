package integration

import (
	"strings"
	"testing"

	"github.com/fibegg/sdk/fibe"
)

func propWithBranchFixture(t *testing.T, c *fibe.Client) (fibe.Prop, string) {
	t.Helper()

	usable := func(prop fibe.Prop, branch string) bool {
		if branch == "" {
			return false
		}
		result, err := c.Props.EnvDefaults(ctx(), prop.ID, branch, seededPropEnvFile)
		if err == nil {
			if strings.HasPrefix(prop.Name, seededPropNamePrefix) || strings.HasPrefix(prop.RepositoryURL, seededPropRepoPrefix) {
				if result.Defaults["FIBE_E2E"] != "1" {
					t.Logf("skipping prop %d/%s branch %q for env_defaults fixture: missing seeded defaults %#v", prop.ID, prop.Name, branch, result.Defaults)
					return false
				}
			}
			return true
		}
		t.Logf("skipping prop %d/%s branch %q for env_defaults fixture: %v", prop.ID, prop.Name, branch, err)
		return false
	}

	props, err := c.Props.List(ctx(), &fibe.PropListParams{PerPage: 50})
	requireNoError(t, err)

	for _, prop := range props.Data {
		if strings.HasPrefix(prop.Name, seededPropNamePrefix) || strings.HasPrefix(prop.RepositoryURL, seededPropRepoPrefix) {
			branch := prop.DefaultBranch
			if branch == "" {
				branch = "main"
			}
			if usable(prop, branch) {
				return prop, branch
			}
		}
	}

	for _, prop := range props.Data {
		branches, err := c.Props.Branches(ctx(), prop.ID, "", 20)
		if err != nil || len(branches.Branches) == 0 {
			continue
		}

		branch := prop.DefaultBranch
		if branch == "" {
			for _, candidate := range branches.Branches {
				if candidate.Default {
					branch = candidate.Name
					break
				}
			}
		}
		if branch == "" {
			branch = branches.Branches[0].Name
		}
		if branch != "" {
			if usable(prop, branch) {
				return prop, branch
			}
		}
	}

	t.Skip("no props with usable env defaults available for env_defaults test")
	return fibe.Prop{}, ""
}

func TestProps_EnvDefaults(t *testing.T) {
	t.Parallel()
	c := userClient(t)

	prop, branch := propWithBranchFixture(t, c)

	t.Run("returns defaults for valid branch", func(t *testing.T) {
		t.Parallel()
		result, err := c.Props.EnvDefaults(ctx(), prop.ID, branch, seededPropEnvFile)
		if apiErr, ok := err.(*fibe.APIError); ok && apiErr.StatusCode == 404 {
			t.Skipf("env defaults fixture prop became unavailable: %s", apiErr.Message)
		}
		requireNoError(t, err)

		if result.Defaults == nil {
			t.Error("expected non-nil defaults map")
		}
		if strings.HasPrefix(prop.Name, seededPropNamePrefix) && result.Defaults["FIBE_E2E"] != "1" {
			t.Errorf("expected seeded fixture env defaults, got %#v", result.Defaults)
		}
	})

	t.Run("returns empty for nonexistent branch", func(t *testing.T) {
		t.Parallel()
		result, err := c.Props.EnvDefaults(ctx(), prop.ID, "nonexistent-branch-xyz", seededPropEnvFile)
		if err != nil {
			// Error is expected for nonexistent branch
			return
		}
		if len(result.Defaults) != 0 {
			t.Errorf("expected empty defaults for nonexistent branch, got %d entries", len(result.Defaults))
		}
	})

	t.Run("returns error for missing branch param", func(t *testing.T) {
		t.Parallel()
		result, err := c.Props.EnvDefaults(ctx(), prop.ID, "", seededPropEnvFile)
		if err != nil {
			return // Error for empty branch is expected behavior
		}
		// If no error, verify we got empty defaults (not arbitrary data)
		if result.Defaults != nil && len(result.Defaults) > 0 {
			t.Error("expected empty defaults for missing branch param")
		}
	})
}

func TestProps_EnvDefaults_NonexistentProp(t *testing.T) {
	t.Parallel()
	c := userClient(t)

	t.Run("returns 404 for nonexistent prop", func(t *testing.T) {
		t.Parallel()
		_, err := c.Props.EnvDefaults(ctx(), 999999999, "main", "")
		requireAPIError(t, err, fibe.ErrCodeNotFound, 404)
	})
}

func TestProps_EnvDefaults_ScopeEnforcement(t *testing.T) {
	t.Parallel()
	c := userClient(t)

	prop, branch := propWithBranchFixture(t, c)

	t.Run("read scope can access env_defaults", func(t *testing.T) {
		t.Parallel()
		readOnly := createScopedKey(t, c, "prop-envdef-read", []string{"props:read"})
		_, err := readOnly.Props.EnvDefaults(ctx(), prop.ID, branch, seededPropEnvFile)
		if err != nil {
			apiErr, ok := err.(*fibe.APIError)
			if ok && apiErr.StatusCode == 403 {
				t.Error("read scope should allow env_defaults access")
			}
		}
	})

	t.Run("no props scope denied", func(t *testing.T) {
		t.Parallel()
		noScope := createScopedKey(t, c, "prop-envdef-noscope", []string{"agents:read"})
		_, err := noScope.Props.EnvDefaults(ctx(), prop.ID, branch, seededPropEnvFile)
		if err == nil {
			t.Error("expected error when accessing props without props scope")
			return
		}
		apiErr, ok := err.(*fibe.APIError)
		if !ok {
			t.Fatalf("expected API error, got: %v", err)
		}
		// API may return 403 (forbidden) or 404 (resource hidden from unauthorized scope)
		if apiErr.StatusCode != 403 && apiErr.StatusCode != 404 {
			t.Errorf("expected 403 or 404 for missing scope, got %d: %v", apiErr.StatusCode, apiErr)
		}
	})
}

func TestProps_EnvDefaults_IDOR(t *testing.T) {
	t.Parallel()
	c := userClient(t)
	userB := userBClient(t)

	prop, branch := propWithBranchFixture(t, c)

	t.Run("user B cannot access primary prop env_defaults", func(t *testing.T) {
		t.Parallel()
		_, err := userB.Props.EnvDefaults(ctx(), prop.ID, branch, seededPropEnvFile)
		requireAPIError(t, err, fibe.ErrCodeNotFound, 404)
	})
}
