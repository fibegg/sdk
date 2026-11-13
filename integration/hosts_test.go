package integration

import (
	"testing"

	"github.com/fibegg/sdk/fibe"
)

func TestHostsLifecycle(t *testing.T) {
	t.Parallel()
	c := userClient(t)

	params := testHostParams("test-host")
	name := params.Name
	host, err := c.Hosts.Create(ctx(), params)
	requireNoError(t, err, "failed to create host")

	if host.Name != name {
		t.Errorf("expected host name %s, got %s", name, host.Name)
	}
	t.Cleanup(func() { c.Hosts.Delete(ctx(), host.ID) })

	fetched, err := c.Hosts.Get(ctx(), host.ID)
	requireNoError(t, err, "failed to get host")
	if fetched.ID != host.ID {
		t.Errorf("expected host id %d, got %d", host.ID, fetched.ID)
	}

	newName := name + "-updated"
	updated, err := c.Hosts.Update(ctx(), host.ID, &fibe.HostUpdateParams{
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
		if m.ID == host.ID {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected to find host %d in list", host.ID)
	}

	err = c.Hosts.Delete(ctx(), host.ID)
	requireNoError(t, err, "failed to delete host")

	_, err = c.Hosts.Get(ctx(), host.ID)
	if err == nil {
		t.Errorf("expected error getting deleted host")
	}
}
