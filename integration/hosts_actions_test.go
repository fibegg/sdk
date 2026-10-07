package integration

import (
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/fibegg/sdk/fibe"
)

func testHostParams(prefix string) *fibe.HostCreateParams {
	n := nameCounter.Add(1)
	return &fibe.HostCreateParams{
		Name:          uniqueName(prefix),
		Host:          fmt.Sprintf("10.%d.%d.%d", (n/65536)%256, (n/256)%256, n%256),
		Port:          2222,
		User:          "testuser",
		SSHPrivateKey: "dummy_key",
		AcmeEmail:     ptr("test@example.com"),
		DomainsInput:  ptr(fmt.Sprintf("%s.test.local", uniqueName(prefix))),
	}
}

func TestHosts_GenerateSSHKey(t *testing.T) {
	t.Parallel()
	c := userClient(t)
	hostID := testSSHKeyHostID(t, c)

	t.Run("generates and returns public key", func(t *testing.T) {
		t.Parallel()
		result, err := c.Hosts.GenerateSSHKey(ctx(), hostID)
		requireNoError(t, err)

		if result.PublicKey == "" {
			t.Error("expected non-empty public_key")
		}
	})

	t.Run("read-only key cannot generate ssh key", func(t *testing.T) {
		t.Parallel()
		readOnly := createScopedKey(t, c, "mq-ssh-ro", []string{"hosts:read"})
		_, err := readOnly.Hosts.GenerateSSHKey(ctx(), hostID)
		requireAPIError(t, err, fibe.ErrCodeForbidden, 403)
	})

	t.Run("nonexistent host returns 404", func(t *testing.T) {
		t.Parallel()
		_, err := c.Hosts.GenerateSSHKey(ctx(), 999999999)
		requireAPIError(t, err, fibe.ErrCodeNotFound, 404)
	})
}

func testSSHKeyHostID(t *testing.T, c *fibe.Client) int64 {
	t.Helper()
	raw := os.Getenv("FIBE_TEST_SSH_KEY_HOST_ID")
	if raw == "" {
		t.Skip("set FIBE_TEST_SSH_KEY_HOST_ID to test SSH key generation against a dedicated funded host")
	}

	id, err := strconv.ParseInt(raw, 10, 64)
	requireNoError(t, err, "parse FIBE_TEST_SSH_KEY_HOST_ID")

	mq, err := c.Hosts.Get(ctx(), id)
	requireNoError(t, err, "load FIBE_TEST_SSH_KEY_HOST_ID")
	if !mq.BillingRuntimeActive {
		t.Fatalf("FIBE_TEST_SSH_KEY_HOST_ID=%d must reference a funded Host", id)
	}

	return id
}

func TestHosts_TestConnection(t *testing.T) {
	t.Parallel()
	c := userClient(t)
	hostID := testHostID(t)
	if hostID == 0 {
		t.Skip("set FIBE_TEST_HOST_ID to test connection against a real host")
	}

	t.Run("returns connection test result", func(t *testing.T) {
		t.Parallel()
		result, err := c.Hosts.TestConnection(ctx(), hostID)
		requireNoError(t, err)
		if !result.Success && result.Message == "" && result.Error == "" {
			t.Error("expected connection test result to have success=true or a message/error")
		}
	})

	t.Run("wrong scope returns 403", func(t *testing.T) {
		t.Parallel()
		noScope := createScopedKey(t, c, "mq-conn-noscope", []string{"repositories:read"})
		_, err := noScope.Hosts.TestConnection(ctx(), hostID)
		requireAPIError(t, err, fibe.ErrCodeForbidden, 403)
	})

	t.Run("nonexistent host returns 404", func(t *testing.T) {
		t.Parallel()
		_, err := c.Hosts.TestConnection(ctx(), 999999999)
		requireAPIError(t, err, fibe.ErrCodeNotFound, 404)
	})
}

func TestHosts_StatusTransitions(t *testing.T) {
	t.Parallel()
	c := userClient(t)

	t.Run("create with disabled status", func(t *testing.T) {
		t.Parallel()
		params := testHostParams("mq-disabled")
		params.Status = ptr("disabled")
		mq, err := c.Hosts.Create(ctx(), params)
		requireNoError(t, err)
		t.Cleanup(func() { c.Hosts.Delete(ctx(), mq.ID) })

		if mq.Status != "disabled" {
			t.Errorf("expected status 'disabled', got %q", mq.Status)
		}
	})

	t.Run("reject invalid status on create", func(t *testing.T) {
		t.Parallel()
		params := testHostParams("mq-bogus")
		params.Status = ptr("bogus")
		_, err := c.Hosts.Create(ctx(), params)
		requireAPIError(t, err, fibe.ErrCodeValidationFailed, 422)
	})

	t.Run("disable active host via update", func(t *testing.T) {
		t.Parallel()
		mq, err := c.Hosts.Create(ctx(), testHostParams("mq-to-disable"))
		requireNoError(t, err)
		t.Cleanup(func() { c.Hosts.Delete(ctx(), mq.ID) })

		updated, err := c.Hosts.Update(ctx(), mq.ID, &fibe.HostUpdateParams{
			Status: ptr("disabled"),
		})
		requireNoError(t, err)

		if updated.Status != "disabled" {
			t.Errorf("expected status 'disabled', got %q", updated.Status)
		}
	})

	t.Run("update with empty name returns validation error", func(t *testing.T) {
		t.Parallel()
		mq, err := c.Hosts.Create(ctx(), testHostParams("mq-badupdate"))
		requireNoError(t, err)
		t.Cleanup(func() { c.Hosts.Delete(ctx(), mq.ID) })

		_, err = c.Hosts.Update(ctx(), mq.ID, &fibe.HostUpdateParams{
			Name: ptr(""),
		})
		requireAPIError(t, err, fibe.ErrCodeValidationFailed, 422)
	})
}

func TestHosts_DeleteConflicts(t *testing.T) {
	t.Parallel()
	c := userClient(t)
	hostID := testHostID(t)

	if hostID == 0 {
		t.Skip("set FIBE_TEST_HOST_ID to test host delete conflicts")
	}

	t.Run("delete host with playgrounds returns 409", func(t *testing.T) {
		t.Parallel()
		spec, err := c.Specs.Create(ctx(), &fibe.SpecCreateParams{
			Name:            uniqueName("mq-conflict-spec"),
			BaseComposeYAML: "services:\n  web:\n    image: nginx:alpine\n",
			Services:        []fibe.SpecServiceDef{{Name: "web", Type: fibe.ServiceTypeStatic}},
		})
		requireNoError(t, err)
		t.Cleanup(func() { c.Specs.Delete(ctx(), *spec.ID) })

		pg, err := c.Playgrounds.Create(ctx(), &fibe.PlaygroundCreateParams{
			Name:   uniqueName("mq-conflict-pg"),
			SpecID: *spec.ID,
			HostID: &hostID,
		})
		requireNoError(t, err)
		t.Cleanup(func() { c.Playgrounds.Delete(ctx(), pg.ID) })

		err = c.Hosts.Delete(ctx(), hostID)
		requireAPIError(t, err, fibe.ErrCodeConflict, 409)
	})
}

func TestHosts_IDOR(t *testing.T) {
	t.Parallel()
	c := userClient(t)
	userB := userBClient(t)

	mq, err := c.Hosts.Create(ctx(), testHostParams("mq-idor"))
	requireNoError(t, err)
	t.Cleanup(func() { c.Hosts.Delete(ctx(), mq.ID) })

	t.Run("user B cannot get primary host", func(t *testing.T) {
		t.Parallel()
		_, err := userB.Hosts.Get(ctx(), mq.ID)
		requireAPIError(t, err, fibe.ErrCodeNotFound, 404)
	})

	t.Run("user B cannot update primary host", func(t *testing.T) {
		t.Parallel()
		_, err := userB.Hosts.Update(ctx(), mq.ID, &fibe.HostUpdateParams{
			Name: ptr("hacked"),
		})
		requireAPIError(t, err, fibe.ErrCodeNotFound, 404)
	})

	t.Run("user B cannot delete primary host", func(t *testing.T) {
		t.Parallel()
		err := userB.Hosts.Delete(ctx(), mq.ID)
		requireAPIError(t, err, fibe.ErrCodeNotFound, 404)
	})

	t.Run("user B cannot generate ssh key for primary host", func(t *testing.T) {
		t.Parallel()
		_, err := userB.Hosts.GenerateSSHKey(ctx(), mq.ID)
		requireAPIError(t, err, fibe.ErrCodeNotFound, 404)
	})

	t.Run("user B cannot test connection for primary host", func(t *testing.T) {
		t.Parallel()
		_, err := userB.Hosts.TestConnection(ctx(), mq.ID)
		requireAPIError(t, err, fibe.ErrCodeNotFound, 404)
	})
}

func TestHosts_ScopeEnforcement(t *testing.T) {
	t.Parallel()
	c := userClient(t)

	mq, err := c.Hosts.Create(ctx(), testHostParams("mq-scope"))
	requireNoError(t, err)
	t.Cleanup(func() { c.Hosts.Delete(ctx(), mq.ID) })

	t.Run("read key can list and get", func(t *testing.T) {
		t.Parallel()
		readOnly := createScopedKey(t, c, "mq-read", []string{"hosts:read"})

		_, err := readOnly.Hosts.List(ctx(), nil)
		requireNoError(t, err)

		_, err = readOnly.Hosts.Get(ctx(), mq.ID)
		requireNoError(t, err)
	})

	t.Run("read key cannot create", func(t *testing.T) {
		t.Parallel()
		readOnly := createScopedKey(t, c, "mq-read-create", []string{"hosts:read"})
		params := testHostParams("nope")
		_, err := readOnly.Hosts.Create(ctx(), params)
		requireAPIError(t, err, fibe.ErrCodeForbidden, 403)
	})

	t.Run("read key cannot update", func(t *testing.T) {
		t.Parallel()
		readOnly := createScopedKey(t, c, "mq-read-update", []string{"hosts:read"})
		_, err := readOnly.Hosts.Update(ctx(), mq.ID, &fibe.HostUpdateParams{
			Name: ptr("nope"),
		})
		requireAPIError(t, err, fibe.ErrCodeForbidden, 403)
	})

	t.Run("read key cannot delete", func(t *testing.T) {
		t.Parallel()
		readOnly := createScopedKey(t, c, "mq-read-delete", []string{"hosts:read"})
		err := readOnly.Hosts.Delete(ctx(), mq.ID)
		requireAPIError(t, err, fibe.ErrCodeForbidden, 403)
	})

	t.Run("no host scope denied", func(t *testing.T) {
		t.Parallel()
		noScope := createScopedKey(t, c, "mq-noscope", []string{"agents:read"})
		_, err := noScope.Hosts.List(ctx(), nil)
		requireAPIError(t, err, fibe.ErrCodeForbidden, 403)
	})
}
