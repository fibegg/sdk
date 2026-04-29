package integration

import (
	"os"
	"testing"

	"github.com/fibegg/sdk/fibe"
)

// Migrated from: 46-auth-rate-limits.spec.js
func TestAuthRateLimits_Headers(t *testing.T) {
	t.Parallel()
	apiKey := os.Getenv("FIBE_API_KEY")
	if apiKey == "" {
		t.Skip("FIBE_API_KEY not set")
	}
	base := adminClient(t).BaseURL()
	c := fibe.NewClient(fibe.WithAPIKey(apiKey), fibe.WithBaseURL(base))

	c.APIKeys.Me(ctx())

	rl := c.RateLimit()
	if rl.Limit == 0 {
		t.Skip("Rate limiting disabled on server (no X-RateLimit-Limit header)")
	}
	if rl.Remaining == 0 {
		t.Error("expected non-zero remaining")
	}
}

func TestAuthRateLimits_RateLimitedKey(t *testing.T) {
	t.Parallel()
	rl := rateLimitClient(t)

	var rateLimited bool
	for i := 0; i < 3; i++ {
		_, err := rl.APIKeys.Me(ctx())
		if err != nil {
			apiErr, ok := err.(*fibe.APIError)
			if ok && apiErr.IsRateLimited() {
				rateLimited = true
				if apiErr.RetryAfter == 0 {
					t.Error("rate limited response should include Retry-After > 0")
				}
				break
			}
		}
	}

	if !rateLimited {
		t.Skip("rate limit not triggered after 3 requests — key may have higher limit; test requires a low-limit key")
	}
}
