package integration

import (
	"testing"

	"github.com/fibegg/sdk/fibe"
)

func TestHostsLifecycle(t *testing.T) {
	t.Parallel()
	c := userClient(t)

	params := testHostParams("test-mq")
	name := params.Name
	mq, err := c.Hosts.Create(ctx(), params)
	requireNoError(t, err, "failed to create host")

	if mq.Name != name {
		t.Errorf("expected host name %s, got %s", name, mq.Name)
	}
	t.Cleanup(func() { c.Hosts.Delete(ctx(), mq.ID) })

	fetched, err := c.Hosts.Get(ctx(), mq.ID)
	requireNoError(t, err, "failed to get host")
	if fetched.ID != mq.ID {
		t.Errorf("expected host id %d, got %d", mq.ID, fetched.ID)
	}

	newName := name + "-updated"
	updated, err := c.Hosts.Update(ctx(), mq.ID, &fibe.HostUpdateParams{
		Name: &newName,
	})
	requireNoError(t, err, "failed to update host")
	if updated.Name != newName {
		t.Errorf("expected host name %s, got %s", newName, updated.Name)
	}

	list, err := c.Hosts.List(ctx(), &fibe.HostListParams{})
	requireNoError(t, err, "failed to list hosts")

	found := false
	for _, m := range list.Data {
		if m.ID == mq.ID {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected to find host %d in list", mq.ID)
	}

	err = c.Hosts.Delete(ctx(), mq.ID)
	requireNoError(t, err, "failed to delete host")

	_, err = c.Hosts.Get(ctx(), mq.ID)
	if err == nil {
		t.Errorf("expected error getting deleted host")
	}
}
