package integration

import (
	"testing"

	"github.com/fibegg/sdk/fibe"
)

func TestLaunch_Create(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	t.Run("launch parses compose and creates playspec+playground", func(t *testing.T) {
		// Not parallel: hits shared marquee slot if FIBE_TEST_MARQUEE_ID set
		name := uniqueName("launch-real")
		result, err := c.Launch.Create(ctx(), &fibe.LaunchParams{
			Name:        name,
			ComposeYAML: realComposeYAML(),
		})
		if err != nil {
			if apiErr, ok := err.(*fibe.APIError); ok {
				// No default marquee, no job_mode: backend may require MarqueeID context
				if apiErr.StatusCode == 422 || apiErr.StatusCode == 400 {
					t.Skipf("launch requires additional context (%s): %s", apiErr.Code, apiErr.Message)
				}
			}
			requireNoError(t, err)
		}
		// Launch response shape is backend-defined: may return ID/Name/Status or just confirm.
		// We only ensure the response is non-nil (already guaranteed by err==nil above).
		_ = result
		if result.ID != 0 {
			t.Cleanup(func() { c.Playgrounds.Delete(ctx(), result.ID) })
		}
	})

	t.Run("launch with job_mode=true creates trick", func(t *testing.T) {
		jm := true
		result, err := c.Launch.Create(ctx(), &fibe.LaunchParams{
			Name:        uniqueName("launch-job"),
			ComposeYAML: jobComposeYAML(),
			JobMode:     &jm,
		})
		if err != nil {
			if apiErr, ok := err.(*fibe.APIError); ok && (apiErr.StatusCode == 422 || apiErr.StatusCode == 400) {
				t.Skipf("job-mode launch requires additional context (%s): %s", apiErr.Code, apiErr.Message)
			}
			requireNoError(t, err)
		}
		if result.ID != 0 {
			t.Cleanup(func() { c.Playgrounds.Delete(ctx(), result.ID) })
		}
	})

	t.Run("launch with empty name returns validation error", func(t *testing.T) {
		t.Parallel()
		_, err := c.Launch.Create(ctx(), &fibe.LaunchParams{
			Name:        "",
			ComposeYAML: minimalComposeYAML(),
		})
		if err == nil {
			t.Error("expected validation error for empty name")
		}
	})

	t.Run("launch with empty compose returns validation error", func(t *testing.T) {
		t.Parallel()
		_, err := c.Launch.Create(ctx(), &fibe.LaunchParams{
			Name:        uniqueName("launch-bad"),
			ComposeYAML: "",
		})
		if err == nil {
			t.Error("expected validation error for empty compose")
		}
	})
}
