package integration

import (
	"testing"

	"github.com/fibegg/sdk/fibe"
)

// Migrated from: 23-feedbacks.spec.js
func TestFeedbacks_CRUD(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	agent, err := c.Agents.Create(ctx(), &fibe.AgentCreateParams{
		Name:     uniqueName("fb-agent"),
		Provider: fibe.ProviderGemini,
	})
	requireNoError(t, err)
	t.Cleanup(func() { c.Agents.Delete(ctx(), agent.ID) })

	var fbID int64

	t.Run("create feedback", func(t *testing.T) {
		// Parallelism disabled: dependent variables
		fb, err := c.Feedbacks.Create(ctx(), agent.ID, &fibe.FeedbackCreateParams{
			SourceType:     "AgentMutter",
			SourceID:       ptr(int64(1)),
			SelectionStart: ptr(0),
			SelectionEnd:   ptr(10),
			SelectedText:   ptr("selected text here"),
			Comment:        ptr("test feedback comment"),
		})
		requireNoError(t, err)

		if fb.ID == nil {
			t.Fatal("expected feedback ID")
		}
		fbID = *fb.ID
	})
	t.Cleanup(func() {
		if fbID > 0 {
			c.Feedbacks.Delete(ctx(), agent.ID, fbID)
		}
	})

	t.Run("list feedbacks", func(t *testing.T) {
		result, err := c.Feedbacks.List(ctx(), agent.ID, nil)
		requireNoError(t, err)

		if result.Meta.Total == 0 {
			t.Error("expected at least one feedback")
		}
	})

	t.Run("get feedback", func(t *testing.T) {
		if fbID == 0 {
			t.Skip("no feedback created")
		}
		fb, err := c.Feedbacks.Get(ctx(), agent.ID, fbID)
		requireNoError(t, err)

		if fb.Comment != "test feedback comment" {
			t.Errorf("expected comment 'test feedback comment', got %q", fb.Comment)
		}
	})

	t.Run("update feedback", func(t *testing.T) {
		if fbID == 0 {
			t.Skip("no feedback created")
		}
		fb, err := c.Feedbacks.Update(ctx(), agent.ID, fbID, &fibe.FeedbackUpdateParams{
			Comment: "updated comment",
		})
		requireNoError(t, err)

		if fb.Comment != "updated comment" {
			t.Error("expected updated comment")
		}
	})

	t.Run("filter by source_type", func(t *testing.T) {
		result, err := c.Feedbacks.List(ctx(), agent.ID, &fibe.FeedbackListParams{
			SourceType: "AgentMutter",
		})
		requireNoError(t, err)

		for _, fb := range result.Data {
			if fb.SourceType != "AgentMutter" {
				t.Errorf("expected source_type AgentMutter, got %q", fb.SourceType)
			}
		}
	})

	t.Run("delete feedback", func(t *testing.T) {
		fb, err := c.Feedbacks.Create(ctx(), agent.ID, &fibe.FeedbackCreateParams{
			SourceType:     "Artefact",
			SourceID:       ptr(int64(1)),
			SelectionStart: ptr(0),
			SelectionEnd:   ptr(5),
			SelectedText:   ptr("text"),
			Comment:        ptr("delete me"),
		})
		requireNoError(t, err)

		err = c.Feedbacks.Delete(ctx(), agent.ID, *fb.ID)
		requireNoError(t, err)

		_, err = c.Feedbacks.Get(ctx(), agent.ID, *fb.ID)
		requireAPIError(t, err, fibe.ErrCodeNotFound, 404)
	})
}

// Migrated from: 24-mutters.spec.js
func TestMutters_Read(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	agent, err := c.Agents.Create(ctx(), &fibe.AgentCreateParams{
		Name:     uniqueName("mutter-agent"),
		Provider: fibe.ProviderGemini,
	})
	requireNoError(t, err)
	t.Cleanup(func() { c.Agents.Delete(ctx(), agent.ID) })

	t.Run("get mutter for agent without data", func(t *testing.T) {
		t.Parallel()
		_, err := c.Mutters.Get(ctx(), agent.ID, nil)
		if err != nil {
			apiErr, ok := err.(*fibe.APIError)
			if ok && apiErr.StatusCode == 404 {
				return
			}
			t.Fatalf("expected 404 or success, got: %v", err)
		}
	})

	t.Run("nonexistent agent returns 404", func(t *testing.T) {
		t.Parallel()
		_, err := c.Mutters.Get(ctx(), 999999999, nil)
		requireAPIError(t, err, fibe.ErrCodeNotFound, 404)
	})

	t.Run("create mutter item", func(t *testing.T) {
		t.Parallel()
		mutter, err := c.Mutters.CreateItem(ctx(), agent.ID, &fibe.MutterItemParams{
			Type: "info",
			Body: "test mutter body",
		})
		requireNoError(t, err)
		if mutter.Content == nil {
			t.Errorf("expected content to be returned on create")
		} else if items, ok := mutter.Content["items"].([]any); ok {
			if len(items) == 0 {
				t.Errorf("expected items in content, got 0")
			}
		} else {
			t.Errorf("expected items array in content")
		}
	})
}
