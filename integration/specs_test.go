package integration

import (
	"testing"

	"github.com/fibegg/sdk/fibe"
)

func TestSpecs_CRUD(t *testing.T) {
	t.Parallel()
	c := userClient(t)

	var specID int64

	t.Run("create spec", func(t *testing.T) {
		// Parallel disabled: dependent sequence
		spec, err := c.Specs.Create(ctx(), &fibe.SpecCreateParams{
			Name:            uniqueName("test-spec"),
			BaseComposeYAML: "services:\n  web:\n    image: nginx:alpine\n",
			Services:        []fibe.SpecServiceDef{{Name: "web", Type: fibe.ServiceTypeStatic}},
		})
		requireNoError(t, err)

		if spec.ID == nil {
			t.Fatal("expected spec ID")
		}
		specID = *spec.ID
	})
	t.Cleanup(func() {
		if specID > 0 {
			c.Specs.Delete(ctx(), specID)
		}
	})

	t.Run("list specs", func(t *testing.T) {
		t.Parallel()
		result, err := c.Specs.List(ctx(), nil)
		requireNoError(t, err)

		if result.Meta.Total == 0 {
			t.Error("expected at least one spec")
		}

		found := false
		for _, s := range result.Data {
			if s.ID != nil && *s.ID == specID {
				found = true
				break
			}
		}
		if specID > 0 && !found {
			t.Error("created spec not found in list")
		}
	})

	t.Run("get spec detail", func(t *testing.T) {
		t.Parallel()
		if specID == 0 {
			t.Skip("no spec created")
		}
		spec, err := c.Specs.Get(ctx(), specID)
		requireNoError(t, err)

		if spec.Name == "" {
			t.Error("expected name")
		}
	})

	t.Run("update spec", func(t *testing.T) {
		t.Parallel()
		if specID == 0 {
			t.Skip("no spec created")
		}
		newName := uniqueName("updated-spec")
		spec, err := c.Specs.Update(ctx(), specID, &fibe.SpecUpdateParams{
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
			t.Skip("no spec created")
		}
		services, err := c.Specs.Services(ctx(), specID)
		requireNoError(t, err)

		if services == nil {
			t.Fatal("expected services to be non-nil")
		}
	})

	t.Run("delete spec", func(t *testing.T) {
		t.Parallel()
		spec, err := c.Specs.Create(ctx(), &fibe.SpecCreateParams{
			Name:            uniqueName("delete-spec"),
			BaseComposeYAML: "services:\n  web:\n    image: nginx:alpine\n",
			Services:        []fibe.SpecServiceDef{{Name: "web", Type: fibe.ServiceTypeStatic}},
		})
		requireNoError(t, err)

		err = c.Specs.Delete(ctx(), *spec.ID)
		requireNoError(t, err)

		_, err = c.Specs.Get(ctx(), *spec.ID)
		requireAPIError(t, err, fibe.ErrCodeNotFound, 404)
	})
}

func TestSpecs_ValidateCompose(t *testing.T) {
	t.Parallel()
	c := userClient(t)

	t.Run("valid compose", func(t *testing.T) {
		t.Parallel()
		result, err := c.Specs.ValidateCompose(ctx(), "services:\n  web:\n    image: nginx\n")
		requireNoError(t, err)

		if result == nil {
			t.Fatal("expected validation result to be non-nil")
		}
	})
}

func TestSpecs_ScopeEnforcement(t *testing.T) {
	t.Parallel()
	c := userClient(t)

	spec, err := c.Specs.Create(ctx(), &fibe.SpecCreateParams{
		Name:            uniqueName("scope-spec"),
		BaseComposeYAML: "services:\n  web:\n    image: nginx:alpine\n",
		Services:        []fibe.SpecServiceDef{{Name: "web", Type: fibe.ServiceTypeStatic}},
	})
	requireNoError(t, err)
	t.Cleanup(func() { c.Specs.Delete(ctx(), *spec.ID) })

	t.Run("wrong scope returns 403", func(t *testing.T) {
		t.Parallel()
		wrongScope := createScopedKey(t, c, "no-specs", []string{"agents:read"})
		_, err := wrongScope.Specs.List(ctx(), nil)
		requireAPIError(t, err, fibe.ErrCodeForbidden, 403)
	})

	t.Run("read scope can list but not create", func(t *testing.T) {
		t.Parallel()
		readOnly := createScopedKey(t, c, "specs-read", []string{"specs:read"})

		_, err := readOnly.Specs.List(ctx(), nil)
		requireNoError(t, err)

		_, err = readOnly.Specs.Create(ctx(), &fibe.SpecCreateParams{
			Name:            "nope",
			BaseComposeYAML: "services:\n  web:\n    image: nginx:alpine\n",
			Services:        []fibe.SpecServiceDef{{Name: "web", Type: fibe.ServiceTypeStatic}},
		})
		requireAPIError(t, err, fibe.ErrCodeForbidden, 403)
	})
}
