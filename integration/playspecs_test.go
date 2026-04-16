package integration

import (
	"testing"

	"github.com/fibegg/sdk/fibe"
)

func TestPlayspecs_CRUD(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	var specID int64

	t.Run("create playspec", func(t *testing.T) {
		// Parallel disabled: dependent sequence
		spec, err := c.Playspecs.Create(ctx(), &fibe.PlayspecCreateParams{
			Name:            uniqueName("test-spec"),
			BaseComposeYAML: "services:\n  web:\n    image: nginx:alpine\n",
			Services:        []fibe.PlayspecServiceDef{{Name: "web", Type: fibe.ServiceTypeStatic}},
		})
		requireNoError(t, err)

		if spec.ID == nil {
			t.Fatal("expected playspec ID")
		}
		specID = *spec.ID
	})
	t.Cleanup(func() {
		if specID > 0 {
			c.Playspecs.Delete(ctx(), specID)
		}
	})

	t.Run("list playspecs", func(t *testing.T) {
		t.Parallel()
		result, err := c.Playspecs.List(ctx(), nil)
		requireNoError(t, err)

		if result.Meta.Total == 0 {
			t.Error("expected at least one playspec")
		}

		found := false
		for _, s := range result.Data {
			if s.ID != nil && *s.ID == specID {
				found = true
				break
			}
		}
		if specID > 0 && !found {
			t.Error("created playspec not found in list")
		}
	})

	t.Run("get playspec detail", func(t *testing.T) {
		t.Parallel()
		if specID == 0 {
			t.Skip("no playspec created")
		}
		spec, err := c.Playspecs.Get(ctx(), specID)
		requireNoError(t, err)

		if spec.Name == "" {
			t.Error("expected name")
		}
	})

	t.Run("update playspec", func(t *testing.T) {
		t.Parallel()
		if specID == 0 {
			t.Skip("no playspec created")
		}
		newName := uniqueName("updated-spec")
		spec, err := c.Playspecs.Update(ctx(), specID, &fibe.PlayspecUpdateParams{
			Name: &newName,
		})
		requireNoError(t, err)

		if spec.Name != newName {
			t.Errorf("expected name %q, got %q", newName, spec.Name)
		}
	})

	t.Run("get services", func(t *testing.T) {
		t.Parallel()
		if specID == 0 {
			t.Skip("no playspec created")
		}
		services, err := c.Playspecs.Services(ctx(), specID)
		requireNoError(t, err)

		if services == nil {
			t.Fatal("expected services to be non-nil")
		}
	})

	t.Run("delete playspec", func(t *testing.T) {
		t.Parallel()
		spec, err := c.Playspecs.Create(ctx(), &fibe.PlayspecCreateParams{
			Name:            uniqueName("delete-spec"),
			BaseComposeYAML: "services:\n  web:\n    image: nginx:alpine\n",
			Services:        []fibe.PlayspecServiceDef{{Name: "web", Type: fibe.ServiceTypeStatic}},
		})
		requireNoError(t, err)

		err = c.Playspecs.Delete(ctx(), *spec.ID)
		requireNoError(t, err)

		_, err = c.Playspecs.Get(ctx(), *spec.ID)
		requireAPIError(t, err, fibe.ErrCodeNotFound, 404)
	})
}

func TestPlayspecs_ValidateCompose(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	t.Run("valid compose", func(t *testing.T) {
		t.Parallel()
		result, err := c.Playspecs.ValidateCompose(ctx(), "services:\n  web:\n    image: nginx\n")
		requireNoError(t, err)

		if result == nil {
			t.Fatal("expected validation result to be non-nil")
		}
	})
}

func TestPlayspecs_ScopeEnforcement(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	spec, err := c.Playspecs.Create(ctx(), &fibe.PlayspecCreateParams{
		Name:            uniqueName("scope-spec"),
		BaseComposeYAML: "services:\n  web:\n    image: nginx:alpine\n",
		Services:        []fibe.PlayspecServiceDef{{Name: "web", Type: fibe.ServiceTypeStatic}},
	})
	requireNoError(t, err)
	t.Cleanup(func() { c.Playspecs.Delete(ctx(), *spec.ID) })

	t.Run("wrong scope returns 403", func(t *testing.T) {
		t.Parallel()
		wrongScope := createScopedKey(t, c, "no-playspecs", []string{"agents:read"})
		_, err := wrongScope.Playspecs.List(ctx(), nil)
		requireAPIError(t, err, fibe.ErrCodeForbidden, 403)
	})

	t.Run("read scope can list but not create", func(t *testing.T) {
		t.Parallel()
		readOnly := createScopedKey(t, c, "specs-read", []string{"playspecs:read"})

		_, err := readOnly.Playspecs.List(ctx(), nil)
		requireNoError(t, err)

		_, err = readOnly.Playspecs.Create(ctx(), &fibe.PlayspecCreateParams{
			Name:            "nope",
			BaseComposeYAML: "services:\n  web:\n    image: nginx:alpine\n",
			Services:        []fibe.PlayspecServiceDef{{Name: "web", Type: fibe.ServiceTypeStatic}},
		})
		requireAPIError(t, err, fibe.ErrCodeForbidden, 403)
	})
}
