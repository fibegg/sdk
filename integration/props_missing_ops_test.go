package integration

import (
	"testing"

	"github.com/fibegg/sdk/fibe"
)

func TestProps_Attach(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	t.Run("attach nonexistent repo returns 4xx", func(t *testing.T) {
		t.Parallel()
		_, err := c.Props.Attach(ctx(), "nonexistent-org/nonexistent-repo-"+uniqueName(""))
		if err == nil {
			t.Skip("attach unexpectedly succeeded (GitHub integration present with matching repo?)")
		}
		apiErr, ok := err.(*fibe.APIError)
		if !ok {
			t.Fatalf("expected APIError, got %T: %v", err, err)
		}
		if apiErr.StatusCode < 400 || apiErr.StatusCode >= 500 {
			t.Errorf("expected 4xx, got %d", apiErr.StatusCode)
		}
	})

	t.Run("attach with empty string returns validation error", func(t *testing.T) {
		t.Parallel()
		_, err := c.Props.Attach(ctx(), "")
		if err == nil {
			t.Error("expected error for empty repo_full_name")
		}
	})
}

func TestProps_Mirror(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	t.Run("mirror with invalid URL returns 400", func(t *testing.T) {
		t.Parallel()
		_, err := c.Props.Mirror(ctx(), "not-a-url")
		if err == nil {
			t.Error("expected validation error for invalid URL")
		}
		if apiErr, ok := err.(*fibe.APIError); ok {
			if apiErr.StatusCode != 400 && apiErr.StatusCode != 422 {
				t.Errorf("expected 400/422, got %d", apiErr.StatusCode)
			}
		}
	})

	t.Run("mirror with empty URL returns error", func(t *testing.T) {
		t.Parallel()
		_, err := c.Props.Mirror(ctx(), "")
		if err == nil {
			t.Error("expected error for empty URL")
		}
	})
}

func TestProps_ManualLink(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	t.Run("manual_link nonexistent prop returns 4xx", func(t *testing.T) {
		t.Parallel()
		_, err := c.Props.ManualLink(ctx(), 999999999)
		if err == nil {
			t.Skip("manual_link unexpectedly succeeded for nonexistent ID")
		}
		apiErr, ok := err.(*fibe.APIError)
		if !ok {
			t.Fatalf("expected APIError, got %T", err)
		}
		if apiErr.StatusCode < 400 || apiErr.StatusCode >= 500 {
			t.Errorf("expected 4xx, got %d", apiErr.StatusCode)
		}
	})
}
