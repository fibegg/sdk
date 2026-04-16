package integration

import (
	"testing"

	"github.com/fibegg/sdk/fibe"
)

// seedPropWithHunks finds a prop that has hunks, or skips.
func seedPropWithHunks(t *testing.T, c *fibe.Client) (propID, hunkID int64) {
	t.Helper()
	list, err := c.Props.List(ctx(), &fibe.PropListParams{PerPage: 50})
	if skipIfFeatureDisabled(t, err, "props") {
		return 0, 0
	}
	requireNoError(t, err)
	for _, p := range list.Data {
		hunks, err := c.Hunks.List(ctx(), p.ID, &fibe.HunkListParams{PerPage: 1})
		if err != nil {
			if apiErr, ok := err.(*fibe.APIError); ok && apiErr.Code == fibe.ErrCodeFeatureDisabled {
				t.Skip("hunks feature disabled")
			}
			continue
		}
		if len(hunks.Data) > 0 && hunks.Data[0].ID != nil {
			return p.ID, *hunks.Data[0].ID
		}
	}
	t.Skip("no prop has hunks — seed prop and trigger ingestion first")
	return 0, 0
}

func TestHunks_Get(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	propID, hunkID := seedPropWithHunks(t, c)
	if propID == 0 {
		return
	}

	t.Run("get hunk returns full detail with diff", func(t *testing.T) {
		h, err := c.Hunks.Get(ctx(), propID, hunkID)
		requireNoError(t, err)
		if h.ID == nil || *h.ID != hunkID {
			t.Errorf("expected ID %d, got %v", hunkID, h.ID)
		}
		if h.FilePath == "" {
			t.Error("expected non-empty FilePath")
		}
		if h.CommitSHA == "" {
			t.Error("expected non-empty CommitSHA")
		}
		// Detail endpoint should return diff_content
		if h.DiffContent == "" {
			t.Log("no DiffContent returned (may be empty for some hunks)")
		}
	})

	t.Run("get nonexistent hunk returns 404", func(t *testing.T) {
		t.Parallel()
		_, err := c.Hunks.Get(ctx(), propID, 999999999)
		requireAPIError(t, err, fibe.ErrCodeNotFound, 404)
	})
}

func TestHunks_Update(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	propID, hunkID := seedPropWithHunks(t, c)
	if propID == 0 {
		return
	}

	t.Run("update sets processor_name when no status", func(t *testing.T) {
		processor := "integration-test-processor"
		h, err := c.Hunks.Update(ctx(), propID, hunkID, &fibe.HunkUpdateParams{
			ProcessorName: &processor,
		})
		requireNoError(t, err)
		if h.ProcessorName == nil || *h.ProcessorName != processor {
			t.Errorf("expected ProcessorName=%q, got %v", processor, h.ProcessorName)
		}
	})

	t.Run("update with invalid status returns 4xx", func(t *testing.T) {
		t.Parallel()
		bad := "totally-invalid-status"
		_, err := c.Hunks.Update(ctx(), propID, hunkID, &fibe.HunkUpdateParams{
			Status: &bad,
		})
		if err == nil {
			t.Error("expected validation error for invalid status")
		}
	})
}
