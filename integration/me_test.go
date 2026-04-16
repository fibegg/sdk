package integration

import (
	"testing"

	"github.com/fibegg/sdk/fibe"
)

func TestMe(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	player, err := c.APIKeys.Me(ctx())
	requireNoError(t, err)

	if player.ID == 0 {
		t.Error("expected non-zero player ID")
	}
	if player.Username == "" {
		t.Error("expected non-empty username")
	}
}

func TestMe_Unauthorized(t *testing.T) {
	t.Parallel()
	c := adminClient(t)
	bad := c.WithKey("invalid-token-that-does-not-exist")

	_, err := bad.APIKeys.Me(ctx())
	requireAPIError(t, err, fibe.ErrCodeUnauthorized, 401)
}
