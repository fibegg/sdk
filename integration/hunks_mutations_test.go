package integration

import (
	"testing"
	"time"

	"github.com/fibegg/sdk/fibe"
)

// Migrated from: 40-hunks.spec.js
func TestHunks_CRUD(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	prop, err := c.Props.Create(ctx(), &fibe.PropCreateParams{
		RepositoryURL: "https://github.com/octocat/Hello-World",
		Name:          ptr(uniqueName("hunk-prop")),
	})
	requireNoError(t, err)
	t.Cleanup(func() { c.Props.Delete(ctx(), prop.ID) })

	t.Run("list hunks", func(t *testing.T) {
		t.Parallel()
		result, err := c.Hunks.List(ctx(), prop.ID, nil)
		if err != nil {
			apiErr, ok := err.(*fibe.APIError)
			if ok && apiErr.Code == "FEATURE_DISABLED" {
				t.Skip("hunks feature not enabled")
			}
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Meta.Total < 0 {
			t.Error("expected non-negative total in hunks list")
		}
	})

	t.Run("ingest hunks", func(t *testing.T) {
		t.Parallel()
		err := c.Hunks.Ingest(ctx(), prop.ID, false)
		if err != nil {
			apiErr, ok := err.(*fibe.APIError)
			if ok && apiErr.Code == "FEATURE_DISABLED" {
				t.Skip("hunks feature not enabled")
			}
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("next hunk with processor", func(t *testing.T) {
		t.Parallel()
		processorName := uniqueName("test-processor")
		claimNext := func(propID int64) (*fibe.Hunk, bool) {
			return pollUntil(20, time.Second, func() (*fibe.Hunk, bool) {
				nextHunk, err := c.Hunks.Next(ctx(), propID, processorName)
				if err != nil {
					if apiErr, ok := err.(*fibe.APIError); ok && (apiErr.Code == "FEATURE_DISABLED" || apiErr.StatusCode == 404) {
						return nil, false
					}
					t.Fatalf("unexpected error: %v", err)
				}
				if nextHunk.ID == nil || nextHunk.Status == "" {
					return nil, false
				}
				return nextHunk, true
			})
		}

		hunk, found := claimNext(prop.ID)
		if !found {
			seededPropID, _ := seedPropWithHunks(t, c)
			if seededPropID != 0 && seededPropID != prop.ID {
				hunk, found = claimNext(seededPropID)
			}
		}
		if !found {
			t.Skip("no pending hunks available for this processor")
		}
		if hunk.ID != nil && hunk.Status == "" {
			t.Error("hunk has an ID but no status")
		}
	})

	t.Run("filter hunks by status", func(t *testing.T) {
		t.Parallel()
		result, err := c.Hunks.List(ctx(), prop.ID, &fibe.HunkListParams{
			Status: "pending",
		})
		if err != nil {
			apiErr, ok := err.(*fibe.APIError)
			if ok && apiErr.Code == "FEATURE_DISABLED" {
				t.Skip("hunks feature not enabled")
			}
			t.Fatalf("unexpected error: %v", err)
		}

		for _, h := range result.Data {
			if h.Status != "pending" {
				t.Errorf("expected status 'pending', got %q", h.Status)
			}
		}
	})
}

// Migrated from: 41-mutations-crud.spec.js
func TestMutations_CRUD(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	prop, err := c.Props.Create(ctx(), &fibe.PropCreateParams{
		RepositoryURL: "https://github.com/octocat/Spoon-Knife",
		Name:          ptr(uniqueName("mut-prop")),
	})
	requireNoError(t, err)
	t.Cleanup(func() { c.Props.Delete(ctx(), prop.ID) })

	var mutID int64

	t.Run("create mutation", func(t *testing.T) {
		// Parallelism disabled: flow dependencies below
		mut, err := c.Mutations.Create(ctx(), prop.ID, &fibe.MutationCreateParams{
			GitDiff:        "--- a/test.rb\n+++ b/test.rb\n@@ -1 +1 @@\n-old\n+new",
			FoundCommitSHA: "abc123def456",
			Branch:         "main",
		})
		requireNoError(t, err)

		if mut.ID == nil {
			t.Fatal("expected mutation ID")
		}
		mutID = *mut.ID
	})

	t.Run("list mutations", func(t *testing.T) {
		t.Parallel()
		result, err := c.Mutations.List(ctx(), prop.ID, nil)
		requireNoError(t, err)

		if result.Meta.Total == 0 {
			t.Error("expected at least one mutation")
		}
	})

	t.Run("filter by status", func(t *testing.T) {
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

	t.Run("update mutation status to cured", func(t *testing.T) {
		t.Parallel()
		if mutID == 0 {
			t.Skip("no mutation created")
		}
		curedSHA := "def456abc789"
		mut, err := c.Mutations.Update(ctx(), prop.ID, mutID, &fibe.MutationUpdateParams{
			Status:         ptr("cured"),
			CuredCommitSHA: &curedSHA,
		})
		requireNoError(t, err)

		if mut.Status != "cured" {
			t.Errorf("expected status 'cured', got %q", mut.Status)
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
					t.Error("expected ascending order by created_at")
				}
			}
		}
	})
}
