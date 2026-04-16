package integration

import (
	"testing"

	"github.com/fibegg/sdk/fibe"
)

func TestImportTemplates_CRUD(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	categories, err := c.TemplateCategories.List(ctx(), nil)
	requireNoError(t, err, "list categories")
	if len(categories.Data) == 0 {
		t.Skip("no template categories available")
	}
	catID := categories.Data[0].ID

	var templateID int64

	t.Run("create template", func(t *testing.T) {
		// Parallelism disabled: down-stream tests rely on templateID
		tpl, err := c.ImportTemplates.Create(ctx(), &fibe.ImportTemplateCreateParams{
			Name:         uniqueName("test-template"),
			Description:  "integration test template",
			CategoryID:   catID,
			TemplateBody: "services:\n  web:\n    image: nginx:alpine\n",
		})
		requireNoError(t, err)

		if tpl.ID == nil {
			t.Fatal("expected template ID")
		}
		templateID = *tpl.ID
	})
	t.Cleanup(func() {
		if templateID > 0 {
			c.ImportTemplates.Delete(ctx(), templateID)
		}
	})

	t.Run("list templates", func(t *testing.T) {
		t.Parallel()
		result, err := c.ImportTemplates.List(ctx(), nil)
		requireNoError(t, err)

		if result.Meta.Total == 0 {
			t.Error("expected at least one template")
		}
	})

	t.Run("get template detail", func(t *testing.T) {
		t.Parallel()
		if templateID == 0 {
			t.Skip("no template created")
		}
		tpl, err := c.ImportTemplates.Get(ctx(), templateID)
		requireNoError(t, err)

		if tpl.Name == "" {
			t.Error("expected template name")
		}
	})

	t.Run("update template", func(t *testing.T) {
		t.Parallel()
		if templateID == 0 {
			t.Skip("no template created")
		}
		newName := uniqueName("updated-tpl")
		tpl, err := c.ImportTemplates.Update(ctx(), templateID, &fibe.ImportTemplateUpdateParams{
			Name: &newName,
		})
		requireNoError(t, err)

		if tpl.Name != newName {
			t.Errorf("expected name %q", newName)
		}
	})

	t.Run("search templates", func(t *testing.T) {
		t.Parallel()
		if templateID == 0 {
			t.Skip("no template created")
		}
		result, err := c.ImportTemplates.Search(ctx(), "test-template", nil)
		requireNoError(t, err)

		if result.Data == nil {
			t.Error("expected search data to be non-nil")
		}
	})

	t.Run("list versions", func(t *testing.T) {
		t.Parallel()
		if templateID == 0 {
			t.Skip("no template created")
		}
		result, err := c.ImportTemplates.ListVersions(ctx(), templateID, nil)
		requireNoError(t, err)

		if len(result.Data) == 0 {
			t.Error("expected at least one version (created with template)")
		}
	})

	t.Run("create version", func(t *testing.T) {
		t.Parallel()
		if templateID == 0 {
			t.Skip("no template created")
		}
		ver, err := c.ImportTemplates.CreateVersion(ctx(), templateID, &fibe.ImportTemplateVersionCreateParams{
			TemplateBody: "services:\n  api:\n    image: node:20\n",
		})
		requireNoError(t, err)

		if ver.TemplateBody == "" {
			t.Error("expected template_body in response")
		}
	})

	t.Run("upload image", func(t *testing.T) {
		t.Parallel()
		if templateID == 0 {
			t.Skip("no template created")
		}
		// A tiny 1x1 base64 GIF would actually be image/gif, let's use a dummy payload, but it will 400 because Marcel strict mode rejects invalid base64 padding or content type if invalid.
		// For the sake of the test we expect a 400 with "Invalid base64 data" or similar from the API,
		// but since we want to test success, we will supply a true base64 encoded PNG.
		// "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVQYV2NgYAAAAAMAAWgmWQ0AAAAASUVORK5CYII=" is 1x1 black pixel PNG.
		b64png := "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVQYV2NgYAAAAAMAAWgmWQ0AAAAASUVORK5CYII="
		_, err := c.ImportTemplates.UploadImage(ctx(), templateID, &fibe.UploadImageParams{
			Filename:    "test.png",
			ImageData:   b64png,
			ContentType: "image/png",
		})
		requireNoError(t, err)
	})

	t.Run("fork template", func(t *testing.T) {
		t.Parallel()
		if templateID == 0 {
			t.Skip("no template created")
		}
		// First we must enable a public version since it checks for `latest_public_version`
		ver, err := c.ImportTemplates.CreateVersion(ctx(), templateID, &fibe.ImportTemplateVersionCreateParams{
			TemplateBody: "services:\n  api:\n    image: node:20\n",
			Public:       ptr(true),
		})
		requireNoError(t, err)

		// Wait, Fibe backend does not let you fork your own template!
		// if source.player_id == current_player.id -> "Cannot fork your own template"
		// We expect a 422 Unprocessable Content.
		_, err = c.ImportTemplates.Fork(ctx(), templateID)
		requireAPIError(t, err, "FORK_FAILED", 422)

		if ver.ID == nil {
			t.Error("expected version ID from CreateVersion")
		}
	})

	t.Run("delete template", func(t *testing.T) {
		t.Parallel()
		tpl, err := c.ImportTemplates.Create(ctx(), &fibe.ImportTemplateCreateParams{
			Name:         uniqueName("delete-tpl"),
			CategoryID:   catID,
			TemplateBody: "services:\n  web:\n    image: nginx\n",
		})
		requireNoError(t, err)

		err = c.ImportTemplates.Delete(ctx(), *tpl.ID)
		requireNoError(t, err)
	})
}

func TestTemplateCategories_List(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	result, err := c.TemplateCategories.List(ctx(), nil)
	requireNoError(t, err)

	for _, cat := range result.Data {
		if cat.Name == "" {
			t.Error("expected category name")
		}
		if cat.Slug == "" {
			t.Error("expected category slug")
		}
	}
}
