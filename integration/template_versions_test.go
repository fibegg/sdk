package integration

import (
	"testing"

	"github.com/fibegg/sdk/fibe"
)

// Migrated from: 16-template-versions.spec.js
func TestTemplateVersions_Lifecycle(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	categories, err := c.TemplateCategories.List(ctx(), nil)
	requireNoError(t, err)
	if len(categories.Data) == 0 {
		t.Skip("no template categories available")
	}
	catID := categories.Data[0].ID

	tpl, err := c.ImportTemplates.Create(ctx(), &fibe.ImportTemplateCreateParams{
		Name:         uniqueName("version-test"),
		Description:  "version lifecycle test",
		CategoryID:   catID,
		TemplateBody: "services:\n  web:\n    image: nginx:1.0\n",
	})
	requireNoError(t, err)
	t.Cleanup(func() { c.ImportTemplates.Delete(ctx(), *tpl.ID) })

	t.Run("initial version created with template", func(t *testing.T) {
		versions, err := c.ImportTemplates.ListVersions(ctx(), *tpl.ID, nil)
		requireNoError(t, err)

		if len(versions.Data) < 1 {
			t.Fatal("expected at least one version created with template")
		}
	})

	var ver2ID int64
	t.Run("create second version", func(t *testing.T) {
		ver, err := c.ImportTemplates.CreateVersion(ctx(), *tpl.ID, &fibe.ImportTemplateVersionCreateParams{
			TemplateBody: "services:\n  web:\n    image: nginx:2.0\n",
		})
		requireNoError(t, err)

		if ver.ID == nil {
			t.Fatal("expected version ID")
		}
		ver2ID = *ver.ID
		if ver.TemplateBody == "" {
			t.Error("expected template_body in response")
		}
	})

	t.Run("version count incremented", func(t *testing.T) {
		versions, err := c.ImportTemplates.ListVersions(ctx(), *tpl.ID, nil)
		requireNoError(t, err)

		if len(versions.Data) < 2 {
			t.Errorf("expected at least 2 versions, got %d", len(versions.Data))
		}
	})

	t.Run("toggle version public", func(t *testing.T) {
		if ver2ID == 0 {
			t.Skip("no version created")
		}
		ver, err := c.ImportTemplates.TogglePublic(ctx(), *tpl.ID, ver2ID)
		requireNoError(t, err)

		if ver.Public == nil || !*ver.Public {
			t.Error("expected public=true after toggle")
		}

		ver, err = c.ImportTemplates.TogglePublic(ctx(), *tpl.ID, ver2ID)
		requireNoError(t, err)

		if ver.Public == nil || *ver.Public {
			t.Error("expected public=false after second toggle")
		}
	})

	t.Run("delete version", func(t *testing.T) {
		if ver2ID == 0 {
			t.Skip("no version created")
		}
		err := c.ImportTemplates.DestroyVersion(ctx(), *tpl.ID, ver2ID)
		requireNoError(t, err)
	})

	t.Run("search finds template", func(t *testing.T) {
		result, err := c.ImportTemplates.Search(ctx(), "version-test", nil)
		requireNoError(t, err)

		if result.Data == nil {
			t.Error("expected search data to be non-nil")
		}
	})
}
