package integration

import (
	"os"
	"strings"
	"testing"

	"github.com/fibegg/sdk/fibe"
)

func repositoryWithBranchFixture(t *testing.T, c *fibe.Client) (fibe.Repository, string) {
	t.Helper()
	if os.Getenv("GITEA_HOST") != "" && os.Getenv("GITEA_ADMIN_TOKEN_FILE") != "" {
		return ownedEnvDefaultsFixture(t, c)
	}

	usable := func(repository fibe.Repository, branch string) bool {
		if branch == "" {
			return false
		}
		result, err := c.Repositories.EnvDefaults(ctx(), repository.ID, branch, seededRepositoryEnvFile)
		if err == nil {
			if strings.HasPrefix(repository.Name, seededRepositoryNamePrefix) || strings.HasPrefix(repository.RepositoryURL, seededRepositoryRepoPrefix) {
				if result.Defaults["FIBE_E2E"] != "1" {
					t.Logf("skipping repository %d/%s branch %q for env_defaults fixture: missing seeded defaults %#v", repository.ID, repository.Name, branch, result.Defaults)
					return false
				}
			}
			return true
		}
		t.Logf("skipping repository %d/%s branch %q for env_defaults fixture: %v", repository.ID, repository.Name, branch, err)
		return false
	}

	repositories, err := c.Repositories.List(ctx(), &fibe.RepositoryListParams{PerPage: 50})
	requireNoError(t, err)

	for _, repository := range repositories.Data {
		if strings.HasPrefix(repository.Name, seededRepositoryNamePrefix) || strings.HasPrefix(repository.RepositoryURL, seededRepositoryRepoPrefix) {
			branch := repository.DefaultBranch
			if branch == "" {
				branch = "main"
			}
			if usable(repository, branch) {
				return repository, branch
			}
		}
	}

	for _, repository := range repositories.Data {
		branches, err := c.Repositories.Branches(ctx(), repository.ID, "", 20)
		if err != nil || len(branches.Branches) == 0 {
			continue
		}

		branch := repository.DefaultBranch
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
			if usable(repository, branch) {
				return repository, branch
			}
		}
	}

	t.Skip("no repositories with usable env defaults available for env_defaults test")
	return fibe.Repository{}, ""
}

func TestProps_EnvDefaults(t *testing.T) {
	t.Parallel()
	c := userClient(t)

	repository, branch := repositoryWithBranchFixture(t, c)

	t.Run("returns defaults for valid branch", func(t *testing.T) {
		t.Parallel()
		result, err := c.Repositories.EnvDefaults(ctx(), repository.ID, branch, seededRepositoryEnvFile)
		requireNoError(t, err)

		if result.Defaults == nil {
			t.Error("expected non-nil defaults map")
		}
		if strings.HasPrefix(repository.Name, seededRepositoryNamePrefix) && result.Defaults["FIBE_E2E"] != "1" {
			t.Errorf("expected seeded fixture env defaults, got %#v", result.Defaults)
		}
	})

	t.Run("returns empty for nonexistent branch", func(t *testing.T) {
		t.Parallel()
		result, err := c.Repositories.EnvDefaults(ctx(), repository.ID, "nonexistent-branch-xyz", seededRepositoryEnvFile)
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
		result, err := c.Repositories.EnvDefaults(ctx(), repository.ID, "", seededRepositoryEnvFile)
		if err != nil {
			return // Error for empty branch is expected behavior
		}
		// If no error, verify we got empty defaults (not arbitrary data)
		if len(result.Defaults) > 0 {
			t.Error("expected empty defaults for missing branch param")
		}
	})
}

func TestProps_EnvDefaults_NonexistentRepository(t *testing.T) {
	t.Parallel()
	c := userClient(t)

	t.Run("returns 404 for nonexistent repository", func(t *testing.T) {
		t.Parallel()
		_, err := c.Repositories.EnvDefaults(ctx(), 999999999, "main", "")
		requireAPIError(t, err, fibe.ErrCodeNotFound, 404)
	})
}

func TestProps_EnvDefaults_ScopeEnforcement(t *testing.T) {
	t.Parallel()
	c := userClient(t)

	repository, branch := repositoryWithBranchFixture(t, c)

	t.Run("read scope can access env_defaults", func(t *testing.T) {
		t.Parallel()
		readOnly := createScopedKey(t, c, "repository-envdef-read", []string{"repositories:read"})
		_, err := readOnly.Repositories.EnvDefaults(ctx(), repository.ID, branch, seededRepositoryEnvFile)
		if err != nil {
			apiErr, ok := err.(*fibe.APIError)
			if ok && apiErr.StatusCode == 403 {
				t.Error("read scope should allow env_defaults access")
			}
		}
	})

	t.Run("no repositories scope denied", func(t *testing.T) {
		t.Parallel()
		noScope := createScopedKey(t, c, "repository-envdef-noscope", []string{"agents:read"})
		_, err := noScope.Repositories.EnvDefaults(ctx(), repository.ID, branch, seededRepositoryEnvFile)
		if err == nil {
			t.Error("expected error when accessing repositories without repositories scope")
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

	repository, branch := repositoryWithBranchFixture(t, c)

	t.Run("user B cannot access primary repository env_defaults", func(t *testing.T) {
		t.Parallel()
		_, err := userB.Repositories.EnvDefaults(ctx(), repository.ID, branch, seededRepositoryEnvFile)
		requireAPIError(t, err, fibe.ErrCodeNotFound, 404)
	})
}
