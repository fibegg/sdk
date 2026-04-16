package integration

import (
	"testing"

	"github.com/fibegg/sdk/fibe"
)

func TestRepoStatus_Check(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	t.Run("check with valid github URLs", func(t *testing.T) {
		t.Parallel()
		result, err := c.RepoStatus.Check(ctx(), []string{
			"https://github.com/nginx/nginx",
		})
		requireNoError(t, err)

		if len(result.Repos) == 0 {
			t.Error("expected at least one repo status entry")
		}
		for _, repo := range result.Repos {
			if repo.URL == "" {
				t.Error("expected non-empty URL in repo status")
			}
			if repo.Status == "" {
				t.Error("expected non-empty status in repo status")
			}
		}
	})

	t.Run("check with empty URLs", func(t *testing.T) {
		t.Parallel()
		result, err := c.RepoStatus.Check(ctx(), []string{})
		if err != nil {
			return
		}
		if len(result.Repos) != 0 {
			t.Error("expected empty result for empty input")
		}
	})

	t.Run("check with invalid URL", func(t *testing.T) {
		t.Parallel()
		result, err := c.RepoStatus.Check(ctx(), []string{
			"not-a-url",
		})
		if err != nil {
			return
		}
		for _, repo := range result.Repos {
			if repo.Error == "" && repo.Status == "" {
				t.Error("expected error or status for invalid URL")
			}
		}
	})
}

func TestRepoStatus_ScopeEnforcement(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	t.Run("requires appropriate scope", func(t *testing.T) {
		t.Parallel()
		noScope := createScopedKey(t, c, "repo-noscope", []string{"agents:read"})
		_, err := noScope.RepoStatus.Check(ctx(), []string{"https://github.com/nginx/nginx"})
		if err == nil {
			t.Error("repo_status should enforce scopes — request without repo scope should be rejected")
			return
		}
		requireAPIError(t, err, fibe.ErrCodeForbidden, 403)
	})
}
