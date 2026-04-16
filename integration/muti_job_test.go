package integration

import (
	"testing"

	"github.com/fibegg/sdk/fibe"
)

// Migrated from: 34-playspec-muti-job.spec.js
func TestMutiJob_PlayspecConfig(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	prop, err := c.Props.Create(ctx(), &fibe.PropCreateParams{
		RepositoryURL: "https://github.com/octocat/" + uniqueName("Hello-World"),
		Name:          ptr(uniqueName("muti-prop")),
	})
	requireNoError(t, err)
	t.Cleanup(func() { c.Props.Delete(ctx(), prop.ID) })

	t.Run("create playspec with muti config", func(t *testing.T) {
		t.Parallel()
		spec, err := c.Playspecs.Create(ctx(), &fibe.PlayspecCreateParams{
			Name:            uniqueName("muti-spec"),
			BaseComposeYAML: "services:\n  app:\n    image: alpine:latest\n",
			Services:        []fibe.PlayspecServiceDef{{Name: "app", Type: fibe.ServiceTypeStatic}},
			MutiConfig: map[string]any{
				"enabled":  true,
				"prop_id":  prop.ID,
				"agent_id": nil,
			},
		})
		requireNoError(t, err)
		t.Cleanup(func() { c.Playspecs.Delete(ctx(), *spec.ID) })

		detail, err := c.Playspecs.Get(ctx(), *spec.ID)
		requireNoError(t, err)

		if detail.MutiConfig == nil {
			t.Error("expected muti_config in detail")
		}
	})
}

// Migrated from: 34-playspec-muti-job.spec.js (mutation lifecycle)
func TestMutiJob_MutationLifecycle(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	prop, err := c.Props.Create(ctx(), &fibe.PropCreateParams{
		RepositoryURL: "https://github.com/octocat/" + uniqueName("Spoon-Knife"),
		Name:          ptr(uniqueName("mut-lifecycle")),
	})
	requireNoError(t, err)
	t.Cleanup(func() { c.Props.Delete(ctx(), prop.ID) })

	diff := "--- a/app/models/user.rb\n+++ b/app/models/user.rb\n@@ -10,7 +10,7 @@\n-  validates :email\n+  # validates :email"

	var mutID int64

	t.Run("create mutation", func(t *testing.T) {
		// Parallelism disabled: dependent variables downstream
		mut, err := c.Mutations.Create(ctx(), prop.ID, &fibe.MutationCreateParams{
			GitDiff:        diff,
			FoundCommitSHA: "abc123",
			Branch:         "main",
		})
		requireNoError(t, err)
		mutID = *mut.ID

		if mut.Status != "active" {
			t.Errorf("expected initial status 'active', got %q", mut.Status)
		}
	})

	t.Run("list shows mutation", func(t *testing.T) {
		t.Parallel()
		result, err := c.Mutations.List(ctx(), prop.ID, nil)
		requireNoError(t, err)

		found := false
		for _, m := range result.Data {
			if m.ID != nil && *m.ID == mutID {
				found = true
			}
		}
		if mutID > 0 && !found {
			t.Error("created mutation not found in list")
		}
	})

	t.Run("filter by status=active", func(t *testing.T) {
		t.Parallel()
		result, err := c.Mutations.List(ctx(), prop.ID, &fibe.MutationListParams{
			Status: "active",
		})
		requireNoError(t, err)

		for _, m := range result.Data {
			if m.Status != "active" {
				t.Errorf("expected status 'active', got %q", m.Status)
			}
		}
	})

	t.Run("update to cured", func(t *testing.T) {
		t.Parallel()
		if mutID == 0 {
			t.Skip("no mutation")
		}
		sha := "def456"
		mut, err := c.Mutations.Update(ctx(), prop.ID, mutID, &fibe.MutationUpdateParams{
			Status:         ptr("cured"),
			CuredCommitSHA: &sha,
		})
		requireNoError(t, err)

		if mut.Status != "cured" {
			t.Errorf("expected 'cured', got %q", mut.Status)
		}
		if mut.CuredCommitSHA == nil || *mut.CuredCommitSHA != sha {
			t.Error("expected cured_commit_sha")
		}
		if mut.CuredAt == nil {
			t.Error("expected cured_at timestamp")
		}
	})

	t.Run("create and kill mutation", func(t *testing.T) {
		t.Parallel()
		mut, err := c.Mutations.Create(ctx(), prop.ID, &fibe.MutationCreateParams{
			GitDiff:        diff,
			FoundCommitSHA: "kill123",
			Branch:         "develop",
		})
		requireNoError(t, err)

		killed, err := c.Mutations.Update(ctx(), prop.ID, *mut.ID, &fibe.MutationUpdateParams{
			Status: ptr("killed"),
		})
		requireNoError(t, err)

		if killed.Status != "killed" {
			t.Errorf("expected 'killed', got %q", killed.Status)
		}
	})

	t.Run("sort by created_at_asc", func(t *testing.T) {
		t.Parallel()
		result, err := c.Mutations.List(ctx(), prop.ID, &fibe.MutationListParams{
			Sort: "created_at_asc",
		})
		requireNoError(t, err)

		if len(result.Data) >= 2 {
			if result.Data[0].CreatedAt != nil && result.Data[1].CreatedAt != nil {
				if result.Data[0].CreatedAt.After(*result.Data[1].CreatedAt) {
					t.Error("expected ascending order")
				}
			}
		}
	})

	t.Run("sort by cured_at_desc", func(t *testing.T) {
		t.Parallel()
		result, err := c.Mutations.List(ctx(), prop.ID, &fibe.MutationListParams{
			Sort: "cured_at_desc",
		})
		requireNoError(t, err)

		if result.Data == nil {
			t.Error("expected data to be non-nil")
		}
	})
}
