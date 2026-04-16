package integration

import (
	"testing"

	"github.com/fibegg/sdk/fibe"
)

func TestProps_CRUD(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	var propID int64

	t.Run("create prop", func(t *testing.T) {
		// Parallel disabled: dependent sequence
		prop, err := c.Props.Create(ctx(), &fibe.PropCreateParams{
			RepositoryURL: "https://github.com/fibegg/sdk",
			Name:          ptr(uniqueName("test-prop")),
		})
		requireNoError(t, err)

		propID = prop.ID
		if prop.RepositoryURL == "" {
			t.Error("expected repository_url")
		}
		if prop.Provider == "" {
			t.Error("expected provider")
		}
	})
	t.Cleanup(func() {
		if propID > 0 {
			c.Props.Delete(ctx(), propID)
		}
	})

	t.Run("list props", func(t *testing.T) {
		t.Parallel()
		result, err := c.Props.List(ctx(), nil)
		requireNoError(t, err)

		if result.Meta.Total == 0 {
			t.Error("expected at least one prop")
		}
	})

	t.Run("get prop detail", func(t *testing.T) {
		t.Parallel()
		if propID == 0 {
			t.Skip("no prop created")
		}
		prop, err := c.Props.Get(ctx(), propID)
		requireNoError(t, err)

		if prop.ID != propID {
			t.Errorf("expected ID %d", propID)
		}
		if prop.DefaultBranch == "" {
			t.Error("expected default_branch")
		}
	})

	t.Run("update prop", func(t *testing.T) {
		t.Parallel()
		if propID == 0 {
			t.Skip("no prop created")
		}
		newName := uniqueName("updated-prop")
		prop, err := c.Props.Update(ctx(), propID, &fibe.PropUpdateParams{
			Name: &newName,
		})
		requireNoError(t, err)

		if prop.Name != newName {
			t.Errorf("expected name %q", newName)
		}
	})

	t.Run("list branches", func(t *testing.T) {
		t.Parallel()
		if propID == 0 {
			t.Skip("no prop created")
		}
		result, err := c.Props.Branches(ctx(), propID, "", 0)
		requireNoError(t, err)
		// Branches may be empty if repo sync hasn't completed, but meta should be valid
		if result.Branches == nil {
			t.Error("expected non-nil branches slice")
		}
	})

	t.Run("sync prop", func(t *testing.T) {
		t.Parallel()
		if propID == 0 {
			t.Skip("no prop created")
		}
		err := c.Props.Sync(ctx(), propID)
		requireNoError(t, err)
	})

	t.Run("with docker compose", func(t *testing.T) {
		t.Parallel()
		result, err := c.Props.WithDockerCompose(ctx(), nil)
		requireNoError(t, err)
		if result.Meta.Page == 0 {
			t.Error("expected meta.page > 0")
		}
	})

	t.Run("delete prop", func(t *testing.T) {
		t.Parallel()
		prop, err := c.Props.Create(ctx(), &fibe.PropCreateParams{
			RepositoryURL: "https://github.com/fibegg/nonexistent-repo-delete-test",
			Name:          ptr(uniqueName("delete-prop")),
		})
		requireNoError(t, err)

		err = c.Props.Delete(ctx(), prop.ID)
		requireNoError(t, err)

		_, err = c.Props.Get(ctx(), prop.ID)
		requireAPIError(t, err, fibe.ErrCodeNotFound, 404)
	})
}

func TestProps_ScopeEnforcement(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	prop, err := c.Props.Create(ctx(), &fibe.PropCreateParams{
		RepositoryURL: "https://github.com/fibegg/scope-test-repo",
		Name:          ptr(uniqueName("scope-prop")),
	})
	requireNoError(t, err)
	t.Cleanup(func() { c.Props.Delete(ctx(), prop.ID) })

	readOnly := createScopedKey(t, c, "props-read", []string{"props:read"})

	t.Run("read key can list", func(t *testing.T) {
		t.Parallel()
		_, err := readOnly.Props.List(ctx(), nil)
		requireNoError(t, err)
	})

	t.Run("read key cannot create", func(t *testing.T) {
		t.Parallel()
		_, err := readOnly.Props.Create(ctx(), &fibe.PropCreateParams{
			RepositoryURL: "https://github.com/org/nope",
		})
		requireAPIError(t, err, fibe.ErrCodeForbidden, 403)
	})

	t.Run("read key cannot delete", func(t *testing.T) {
		t.Parallel()
		err := readOnly.Props.Delete(ctx(), prop.ID)
		requireAPIError(t, err, fibe.ErrCodeForbidden, 403)
	})
}
