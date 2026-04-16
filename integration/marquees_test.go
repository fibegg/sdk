package integration

import (
	"testing"

	"github.com/fibegg/sdk/fibe"
)

func TestMarqueesLifecycle(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	// Step 1: Create Marquee
	params := testMarqueeParams("test-mq")
	name := params.Name
	mq, err := c.Marquees.Create(ctx(), params)
	requireNoError(t, err, "failed to create marquee")

	if mq.Name != name {
		t.Errorf("expected marquee name %s, got %s", name, mq.Name)
	}
	t.Cleanup(func() { c.Marquees.Delete(ctx(), mq.ID) })

	// Step 2: Get Marquee
	fetched, err := c.Marquees.Get(ctx(), mq.ID)
	requireNoError(t, err, "failed to get marquee")
	if fetched.ID != mq.ID {
		t.Errorf("expected marquee id %d, got %d", mq.ID, fetched.ID)
	}

	// Step 3: Update Marquee
	newName := name + "-updated"
	updated, err := c.Marquees.Update(ctx(), mq.ID, &fibe.MarqueeUpdateParams{
		Name: &newName,
	})
	requireNoError(t, err, "failed to update marquee")
	if updated.Name != newName {
		t.Errorf("expected marquee name %s, got %s", newName, updated.Name)
	}

	// Step 4: List Marquees
	list, err := c.Marquees.List(ctx(), &fibe.MarqueeListParams{})
	requireNoError(t, err, "failed to list marquees")

	found := false
	for _, m := range list.Data {
		if m.ID == mq.ID {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected to find marquee %d in list", mq.ID)
	}

	// Step 5: Delete Marquee
	err = c.Marquees.Delete(ctx(), mq.ID)
	requireNoError(t, err, "failed to delete marquee")

	// Step 6: Verify deletion
	_, err = c.Marquees.Get(ctx(), mq.ID)
	if err == nil {
		t.Errorf("expected error getting deleted marquee")
	}
}
