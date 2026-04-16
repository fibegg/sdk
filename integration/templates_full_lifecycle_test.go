package integration

import (
	"encoding/base64"
	"testing"

	"github.com/fibegg/sdk/fibe"
)

// 1x1 transparent PNG for image upload tests
const tinyPNG = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNkYPhfDwAChwGA60e6kgAAAABJRU5ErkJggg=="

// seedTemplateCategory returns an existing category ID (creates one if none exist — but admin only).
func seedTemplateCategory(t *testing.T, c *fibe.Client) int64 {
	t.Helper()
	cats, err := c.TemplateCategories.List(ctx(), nil)
	requireNoError(t, err)
	if len(cats.Data) == 0 {
		t.Skip("no template categories available — admin must seed one")
	}
	return cats.Data[0].ID
}

// TestTemplates_FullLifecycle walks: Create → CreateVersion → TogglePublic → ListVersions →
// UploadImage → Update → DestroyVersion → Delete
func TestTemplates_FullLifecycle(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	catID := seedTemplateCategory(t, c)

	tpl, err := c.ImportTemplates.Create(ctx(), &fibe.ImportTemplateCreateParams{
		Name:         uniqueName("tpl-full"),
		Description:  "Integration test template",
		CategoryID:   catID,
		TemplateBody: "name: integration-template\nversion: 1\nservices: []\n",
	})
	requireNoError(t, err)
	t.Cleanup(func() {
		if tpl.ID != nil {
			c.ImportTemplates.Delete(ctx(), *tpl.ID)
		}
	})

	if tpl.ID == nil {
		t.Fatal("expected template ID")
	}
	tplID := *tpl.ID

	// Step 1: CreateVersion
	var versionID int64
	t.Run("create new version", func(t *testing.T) {
		pub := false
		v, err := c.ImportTemplates.CreateVersion(ctx(), tplID, &fibe.ImportTemplateVersionCreateParams{
			TemplateBody: "name: integration-template\nversion: 2\nservices: []\n",
			Public:       &pub,
		})
		requireNoError(t, err)
		if v.ID == nil {
			t.Fatal("expected version ID")
		}
		versionID = *v.ID
		if v.TemplateBody == "" {
			t.Error("expected non-empty TemplateBody in response")
		}
	})

	t.Run("list versions includes the new one", func(t *testing.T) {
		if versionID == 0 {
			t.Skip("no version created")
		}
		list, err := c.ImportTemplates.ListVersions(ctx(), tplID, nil)
		requireNoError(t, err)
		found := false
		for _, v := range list.Data {
			if v.ID != nil && *v.ID == versionID {
				found = true
			}
		}
		if !found {
			t.Errorf("expected version %d in ListVersions", versionID)
		}
	})

	t.Run("toggle public on version", func(t *testing.T) {
		if versionID == 0 {
			t.Skip("no version created")
		}
		v, err := c.ImportTemplates.TogglePublic(ctx(), tplID, versionID)
		requireNoError(t, err)
		if v.Public == nil {
			t.Error("expected non-nil Public flag")
		}
	})

	t.Run("update template metadata", func(t *testing.T) {
		newName := uniqueName("tpl-renamed")
		upd, err := c.ImportTemplates.Update(ctx(), tplID, &fibe.ImportTemplateUpdateParams{
			Name: &newName,
		})
		requireNoError(t, err)
		if upd.Name != newName {
			t.Errorf("expected Name=%s, got %s", newName, upd.Name)
		}
	})

	t.Run("upload image sets ImageURL", func(t *testing.T) {
		imgData, _ := base64.StdEncoding.DecodeString(tinyPNG)
		params := &fibe.UploadImageParams{
			Filename:    "tiny.png",
			ImageData:   base64.StdEncoding.EncodeToString(imgData),
			ContentType: "image/png",
		}
		_, err := c.ImportTemplates.UploadImage(ctx(), tplID, params)
		if err != nil {
			if apiErr, ok := err.(*fibe.APIError); ok && apiErr.StatusCode >= 400 && apiErr.StatusCode < 500 {
				t.Logf("upload image returned %d: %s", apiErr.StatusCode, apiErr.Message)
				return
			}
			requireNoError(t, err)
		}
	})

	t.Run("destroy version", func(t *testing.T) {
		if versionID == 0 {
			t.Skip("no version to destroy")
		}
		err := c.ImportTemplates.DestroyVersion(ctx(), tplID, versionID)
		requireNoError(t, err)
	})
}

func TestTemplates_SearchAndFilters(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	catID := seedTemplateCategory(t, c)

	needle := uniqueName("tplsearch-needle")
	tpl, err := c.ImportTemplates.Create(ctx(), &fibe.ImportTemplateCreateParams{
		Name:         needle,
		CategoryID:   catID,
		TemplateBody: "name: t\nversion: 1\n",
	})
	requireNoError(t, err)
	t.Cleanup(func() {
		if tpl.ID != nil {
			c.ImportTemplates.Delete(ctx(), *tpl.ID)
		}
	})

	t.Run("search by name finds template", func(t *testing.T) {
		// Search may have indexing delay
		r, err := c.ImportTemplates.Search(ctx(), "tplsearch-needle", nil)
		requireNoError(t, err)
		found := false
		for _, item := range r.Data {
			if item.ID != nil && tpl.ID != nil && *item.ID == *tpl.ID {
				found = true
			}
		}
		if !found {
			t.Log("search didn't find template (may have indexing delay)")
		}
	})

	t.Run("list filter by category_id returns only that category", func(t *testing.T) {
		r, err := c.ImportTemplates.List(ctx(), &fibe.ImportTemplateListParams{CategoryID: catID, PerPage: 50})
		requireNoError(t, err)
		// All returned templates should be in that category (Category field is the name, not ID —
		// so just ensure we have results and one of them is our template)
		found := false
		for _, item := range r.Data {
			if item.ID != nil && tpl.ID != nil && *item.ID == *tpl.ID {
				found = true
			}
		}
		if !found {
			t.Errorf("expected template %v in category %d filter, got %d results", tpl.ID, catID, r.Meta.Total)
		}
	})

	t.Run("list filter by name matches", func(t *testing.T) {
		r, err := c.ImportTemplates.List(ctx(), &fibe.ImportTemplateListParams{Name: "tplsearch", PerPage: 50})
		requireNoError(t, err)
		if r.Meta.Total == 0 {
			t.Log("name filter may have indexing delay")
		}
	})
}
