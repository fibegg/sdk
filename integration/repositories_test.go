package integration

import (
	"testing"

	"github.com/fibegg/sdk/fibe"
)

func TestProps_CRUD(t *testing.T) {
	t.Parallel()
	c := userClient(t)

	var repositoryID int64

	t.Run("create repository", func(t *testing.T) {
		// Parallel disabled: dependent sequence
		repository := createWritableGiteaRepository(t, c, "test-repository")

		repositoryID = repository.ID
		if repository.RepositoryURL == "" {
			t.Error("expected repository_url")
		}
		if repository.Provider == "" {
			t.Error("expected provider")
		}
	})
	t.Cleanup(func() {
		if repositoryID > 0 {
			c.Repositories.Delete(ctx(), repositoryID)
		}
	})

	t.Run("list repositories", func(t *testing.T) {
		t.Parallel()
		result, err := c.Repositories.List(ctx(), nil)
		requireNoError(t, err)

		if result.Meta.Total == 0 {
			t.Error("expected at least one repository")
		}
	})

	t.Run("get repository detail", func(t *testing.T) {
		t.Parallel()
		if repositoryID == 0 {
			t.Skip("no repository created")
		}
		repository, err := c.Repositories.Get(ctx(), repositoryID)
		requireNoError(t, err)

		if repository.ID != repositoryID {
			t.Errorf("expected ID %d", repositoryID)
		}
		if repository.DefaultBranch == "" {
			t.Error("expected default_branch")
		}
	})

	t.Run("update repository", func(t *testing.T) {
		t.Parallel()
		if repositoryID == 0 {
			t.Skip("no repository created")
		}
		newName := uniqueName("updated-repository")
		repository, err := c.Repositories.Update(ctx(), repositoryID, &fibe.RepositoryUpdateParams{
			Name: &newName,
		})
		requireNoError(t, err)

		if repository.Name != newName {
			t.Errorf("expected name %q", newName)
		}
	})

	t.Run("list branches", func(t *testing.T) {
		t.Parallel()
		if repositoryID == 0 {
			t.Skip("no repository created")
		}
		result, err := c.Repositories.Branches(ctx(), repositoryID, "", 0)
		requireNoError(t, err)
		// Branches may be empty if repo sync hasn't completed, but meta should be valid
		if result.Branches == nil {
			t.Error("expected non-nil branches slice")
		}
	})

	t.Run("sync repository", func(t *testing.T) {
		t.Parallel()
		skipThirdpartyIfDisabled(t)
		if repositoryID == 0 {
			t.Skip("no repository created")
		}
		err := c.Repositories.Sync(ctx(), repositoryID)
		requireNoError(t, err)
	})

	t.Run("delete repository", func(t *testing.T) {
		t.Parallel()
		repository := createWritableGiteaRepository(t, c, "delete-repository")

		err := c.Repositories.Delete(ctx(), repository.ID)
		requireNoError(t, err)

		_, err = c.Repositories.Get(ctx(), repository.ID)
		requireAPIError(t, err, fibe.ErrCodeNotFound, 404)
	})
}
