package integration

import (
	"testing"

	"github.com/fibegg/sdk/fibe"
)

func TestImportTemplates_Launch(t *testing.T) {
	t.Parallel()
	c := adminClient(t)
	marqueeID := testMarqueeID(t)
	if marqueeID == 0 {
		t.Skip("set FIBE_TEST_MARQUEE_ID to test template launch")
	}

	categories, err := c.TemplateCategories.List(ctx(), nil)
	requireNoError(t, err)
	if len(categories.Data) == 0 {
		t.Skip("no template categories available to seed Launch")
	}

	tpl, err := c.ImportTemplates.Create(ctx(), &fibe.ImportTemplateCreateParams{
		Name:         uniqueName("launch-template"),
		Description:  "integration launch template",
		CategoryID:   categories.Data[0].ID,
		TemplateBody: "services:\n  web:\n    image: nginx:alpine\n",
	})
	requireNoError(t, err)
	if tpl.ID == nil {
		t.Fatal("expected seeded template ID")
	}
	tplID := *tpl.ID
	t.Cleanup(func() { c.ImportTemplates.Delete(ctx(), tplID) })

	t.Run("launch from template returns playground id or error", func(t *testing.T) {
		result, err := c.ImportTemplates.LaunchWithParams(ctx(), tplID, &fibe.ImportTemplateLaunchParams{
			MarqueeID: marqueeID,
			Name:      uniqueName("launch-pg"),
		})
		if err != nil {
			if apiErr, ok := err.(*fibe.APIError); ok {
				// Runtime provisioning can still fail if the target marquee is not usable.
				if apiErr.StatusCode == 422 || apiErr.StatusCode == 400 || apiErr.StatusCode == 404 {
					t.Skipf("template launch requires prerequisites (%s): %s", apiErr.Code, apiErr.Message)
				}
			}
			requireNoError(t, err)
		}
		if result.ID == 0 && result.Name == "" && result.Status == "" {
			t.Error("expected at least one of ID/Name/Status in LaunchResult")
		}
		if result.ID != 0 {
			t.Cleanup(func() { c.Playgrounds.Delete(ctx(), result.ID) })
		}
	})
}
