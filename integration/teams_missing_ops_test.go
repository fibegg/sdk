package integration

import (
	"testing"

	"github.com/fibegg/sdk/fibe"
)

func TestTeams_TransferLeadership(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	team := seedTeam(t, c)

	t.Run("transfer to nonexistent player returns 4xx", func(t *testing.T) {
		t.Parallel()
		_, err := c.Teams.TransferLeadership(ctx(), team.ID, 999999999)
		if err == nil {
			t.Error("expected error when transferring to nonexistent player")
		}
	})

	t.Run("transfer to self returns 4xx or is a no-op", func(t *testing.T) {
		t.Parallel()
		// current player ID is team.CreatorID (admin created the team)
		_, err := c.Teams.TransferLeadership(ctx(), team.ID, team.CreatorID)
		// Self-transfer typically returns 400/422 or is idempotent
		if err != nil {
			if apiErr, ok := err.(*fibe.APIError); ok {
				if apiErr.StatusCode < 400 || apiErr.StatusCode >= 500 {
					t.Errorf("expected 4xx or success, got %d", apiErr.StatusCode)
				}
			}
		}
	})
}
