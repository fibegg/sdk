package integration

import (
	"fmt"
	"testing"

	"github.com/fibegg/sdk/fibe"
)

func testMarqueeParams(prefix string) *fibe.MarqueeCreateParams {
	n := nameCounter.Add(1)
	return &fibe.MarqueeCreateParams{
		Name:          uniqueName(prefix),
		Host:          fmt.Sprintf("10.%d.%d.%d", (n/65536)%256, (n/256)%256, n%256),
		Port:          2222,
		User:          "testuser",
		SSHPrivateKey: "dummy_key",
		AcmeEmail:     ptr("test@example.com"),
		DomainsInput:  ptr(fmt.Sprintf("%s.test.local", uniqueName(prefix))),
	}
}

func TestMarquees_GenerateSSHKey(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	mq, err := c.Marquees.Create(ctx(), testMarqueeParams("mq-ssh"))
	requireNoError(t, err)
	t.Cleanup(func() { c.Marquees.Delete(ctx(), mq.ID) })

	t.Run("generates and returns public key", func(t *testing.T) {
		t.Parallel()
		result, err := c.Marquees.GenerateSSHKey(ctx(), mq.ID)
		requireNoError(t, err)

		if result.PublicKey == "" {
			t.Error("expected non-empty public_key")
		}
	})

	t.Run("read-only key cannot generate ssh key", func(t *testing.T) {
		t.Parallel()
		readOnly := createScopedKey(t, c, "mq-ssh-ro", []string{"marquees:read"})
		_, err := readOnly.Marquees.GenerateSSHKey(ctx(), mq.ID)
		requireAPIError(t, err, fibe.ErrCodeForbidden, 403)
	})

	t.Run("nonexistent marquee returns 404", func(t *testing.T) {
		t.Parallel()
		_, err := c.Marquees.GenerateSSHKey(ctx(), 999999999)
		requireAPIError(t, err, fibe.ErrCodeNotFound, 404)
	})
}

func TestMarquees_TestConnection(t *testing.T) {
	t.Parallel()
	c := adminClient(t)
	marqueeID := testMarqueeID(t)
	if marqueeID == 0 {
		t.Skip("set FIBE_TEST_MARQUEE_ID to test connection against a real marquee")
	}

	t.Run("returns connection test result", func(t *testing.T) {
		t.Parallel()
		result, err := c.Marquees.TestConnection(ctx(), marqueeID)
		requireNoError(t, err)
		if !result.Success && result.Message == "" && result.Error == "" {
			t.Error("expected connection test result to have success=true or a message/error")
		}
	})

	t.Run("wrong scope returns 403", func(t *testing.T) {
		t.Parallel()
		noScope := createScopedKey(t, c, "mq-conn-noscope", []string{"props:read"})
		_, err := noScope.Marquees.TestConnection(ctx(), marqueeID)
		requireAPIError(t, err, fibe.ErrCodeForbidden, 403)
	})

	t.Run("nonexistent marquee returns 404", func(t *testing.T) {
		t.Parallel()
		_, err := c.Marquees.TestConnection(ctx(), 999999999)
		requireAPIError(t, err, fibe.ErrCodeNotFound, 404)
	})
}

func TestMarquees_StatusTransitions(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	t.Run("create with disabled status", func(t *testing.T) {
		t.Parallel()
		params := testMarqueeParams("mq-disabled")
		params.Status = ptr("disabled")
		mq, err := c.Marquees.Create(ctx(), params)
		requireNoError(t, err)
		t.Cleanup(func() { c.Marquees.Delete(ctx(), mq.ID) })

		if mq.Status != "disabled" {
			t.Errorf("expected status 'disabled', got %q", mq.Status)
		}
	})

	t.Run("reject invalid status on create", func(t *testing.T) {
		t.Parallel()
		params := testMarqueeParams("mq-bogus")
		params.Status = ptr("bogus")
		_, err := c.Marquees.Create(ctx(), params)
		requireAPIError(t, err, fibe.ErrCodeValidationFailed, 422)
	})

	t.Run("disable active marquee via update", func(t *testing.T) {
		t.Parallel()
		mq, err := c.Marquees.Create(ctx(), testMarqueeParams("mq-to-disable"))
		requireNoError(t, err)
		t.Cleanup(func() { c.Marquees.Delete(ctx(), mq.ID) })

		updated, err := c.Marquees.Update(ctx(), mq.ID, &fibe.MarqueeUpdateParams{
			Status: ptr("disabled"),
		})
		requireNoError(t, err)

		if updated.Status != "disabled" {
			t.Errorf("expected status 'disabled', got %q", updated.Status)
		}
	})

	t.Run("update with empty name returns validation error", func(t *testing.T) {
		t.Parallel()
		mq, err := c.Marquees.Create(ctx(), testMarqueeParams("mq-badupdate"))
		requireNoError(t, err)
		t.Cleanup(func() { c.Marquees.Delete(ctx(), mq.ID) })

		_, err = c.Marquees.Update(ctx(), mq.ID, &fibe.MarqueeUpdateParams{
			Name: ptr(""),
		})
		requireAPIError(t, err, fibe.ErrCodeValidationFailed, 422)
	})
}

func TestMarquees_DeleteConflicts(t *testing.T) {
	t.Parallel()
	c := adminClient(t)
	marqueeID := testMarqueeID(t)

	if marqueeID == 0 {
		t.Skip("set FIBE_TEST_MARQUEE_ID to test marquee delete conflicts")
	}

	t.Run("delete marquee with playgrounds returns 409", func(t *testing.T) {
		t.Parallel()
		spec, err := c.Playspecs.Create(ctx(), &fibe.PlayspecCreateParams{
			Name:            uniqueName("mq-conflict-spec"),
			BaseComposeYAML: "services:\n  web:\n    image: nginx:alpine\n",
			Services:        []fibe.PlayspecServiceDef{{Name: "web", Type: fibe.ServiceTypeStatic}},
		})
		requireNoError(t, err)
		t.Cleanup(func() { c.Playspecs.Delete(ctx(), *spec.ID) })

		pg, err := c.Playgrounds.Create(ctx(), &fibe.PlaygroundCreateParams{
			Name:       uniqueName("mq-conflict-pg"),
			PlayspecID: *spec.ID,
			MarqueeID:  &marqueeID,
		})
		requireNoError(t, err)
		t.Cleanup(func() { c.Playgrounds.Delete(ctx(), pg.ID) })

		err = c.Marquees.Delete(ctx(), marqueeID)
		requireAPIError(t, err, fibe.ErrCodeConflict, 409)
	})
}

func TestMarquees_IDOR(t *testing.T) {
	t.Parallel()
	c := adminClient(t)
	userB := userBClient(t)

	mq, err := c.Marquees.Create(ctx(), testMarqueeParams("mq-idor"))
	requireNoError(t, err)
	t.Cleanup(func() { c.Marquees.Delete(ctx(), mq.ID) })

	t.Run("user B cannot get admin marquee", func(t *testing.T) {
		t.Parallel()
		_, err := userB.Marquees.Get(ctx(), mq.ID)
		requireAPIError(t, err, fibe.ErrCodeNotFound, 404)
	})

	t.Run("user B cannot update admin marquee", func(t *testing.T) {
		t.Parallel()
		_, err := userB.Marquees.Update(ctx(), mq.ID, &fibe.MarqueeUpdateParams{
			Name: ptr("hacked"),
		})
		requireAPIError(t, err, fibe.ErrCodeNotFound, 404)
	})

	t.Run("user B cannot delete admin marquee", func(t *testing.T) {
		t.Parallel()
		err := userB.Marquees.Delete(ctx(), mq.ID)
		requireAPIError(t, err, fibe.ErrCodeNotFound, 404)
	})

	t.Run("user B cannot generate ssh key for admin marquee", func(t *testing.T) {
		t.Parallel()
		_, err := userB.Marquees.GenerateSSHKey(ctx(), mq.ID)
		requireAPIError(t, err, fibe.ErrCodeNotFound, 404)
	})

	t.Run("user B cannot test connection for admin marquee", func(t *testing.T) {
		t.Parallel()
		_, err := userB.Marquees.TestConnection(ctx(), mq.ID)
		requireAPIError(t, err, fibe.ErrCodeNotFound, 404)
	})
}

func TestMarquees_ScopeEnforcement(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	mq, err := c.Marquees.Create(ctx(), testMarqueeParams("mq-scope"))
	requireNoError(t, err)
	t.Cleanup(func() { c.Marquees.Delete(ctx(), mq.ID) })

	t.Run("read key can list and get", func(t *testing.T) {
		t.Parallel()
		readOnly := createScopedKey(t, c, "mq-read", []string{"marquees:read"})

		_, err := readOnly.Marquees.List(ctx(), nil)
		requireNoError(t, err)

		_, err = readOnly.Marquees.Get(ctx(), mq.ID)
		requireNoError(t, err)
	})

	t.Run("read key cannot create", func(t *testing.T) {
		t.Parallel()
		readOnly := createScopedKey(t, c, "mq-read-create", []string{"marquees:read"})
		params := testMarqueeParams("nope")
		_, err := readOnly.Marquees.Create(ctx(), params)
		requireAPIError(t, err, fibe.ErrCodeForbidden, 403)
	})

	t.Run("read key cannot update", func(t *testing.T) {
		t.Parallel()
		readOnly := createScopedKey(t, c, "mq-read-update", []string{"marquees:read"})
		_, err := readOnly.Marquees.Update(ctx(), mq.ID, &fibe.MarqueeUpdateParams{
			Name: ptr("nope"),
		})
		requireAPIError(t, err, fibe.ErrCodeForbidden, 403)
	})

	t.Run("read key cannot delete", func(t *testing.T) {
		t.Parallel()
		readOnly := createScopedKey(t, c, "mq-read-delete", []string{"marquees:read"})
		err := readOnly.Marquees.Delete(ctx(), mq.ID)
		requireAPIError(t, err, fibe.ErrCodeForbidden, 403)
	})

	t.Run("no marquee scope denied", func(t *testing.T) {
		t.Parallel()
		noScope := createScopedKey(t, c, "mq-noscope", []string{"agents:read"})
		_, err := noScope.Marquees.List(ctx(), nil)
		requireAPIError(t, err, fibe.ErrCodeForbidden, 403)
	})
}
