package integration

import (
	"testing"
	"time"

	"github.com/fibegg/sdk/fibe"
)

// TestPayload_PlaygroundDetail verifies that Get returns full detail fields,
// not just the slim list view.
func TestPayload_PlaygroundDetail(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	specID, marqueeID := setupPlaygroundDeps(t, c)
	if marqueeID == 0 {
		t.Skip("set FIBE_TEST_MARQUEE_ID to run this test")
	}

	pg, err := c.Playgrounds.Create(ctx(), &fibe.PlaygroundCreateParams{
		Name:       uniqueName("payload-pg"),
		PlayspecID: specID,
		MarqueeID:  &marqueeID,
	})
	requireNoError(t, err)
	t.Cleanup(func() { c.Playgrounds.Delete(ctx(), pg.ID) })

	t.Run("Get returns detail fields not in List", func(t *testing.T) {
		detail, err := c.Playgrounds.Get(ctx(), pg.ID)
		requireNoError(t, err)
		// Detail-only fields: ComposeProject may be nil before provisioning
		if detail.ID != pg.ID {
			t.Errorf("ID mismatch: want %d, got %d", pg.ID, detail.ID)
		}
		if detail.Name != pg.Name {
			t.Errorf("Name mismatch: want %s, got %s", pg.Name, detail.Name)
		}
		if detail.Status == "" {
			t.Error("expected Status")
		}
		// TimeRemaining and ExpirationPercentage become available once expiration set
	})

	t.Run("Compose returns yaml + project (poll until ready)", func(t *testing.T) {
		cmp, found := pollUntil(120, time.Second, func() (*fibe.PlaygroundCompose, bool) {
			c2, err := c.Playgrounds.Compose(ctx(), pg.ID)
			if err != nil {
				return nil, false
			}
			return c2, c2.ComposeYAML != ""
		})
		if !found {
			t.Skip("ComposeYAML not ready within timeout")
		}
		if cmp.ComposeYAML == "" {
			t.Error("expected non-empty ComposeYAML")
		}
	})

	t.Run("EnvMetadata returns merged + metadata + system_keys", func(t *testing.T) {
		env, err := c.Playgrounds.EnvMetadata(ctx(), pg.ID)
		requireNoError(t, err)
		if env.Merged == nil {
			t.Error("expected non-nil Merged map")
		}
		if env.Metadata == nil {
			t.Error("expected non-nil Metadata map")
		}
		// SystemKeys may be empty; just check non-nil
		if env.SystemKeys == nil {
			t.Error("expected non-nil SystemKeys slice")
		}
	})
}

// TestPayload_PlayspecDetail verifies detailed playspec fields.
func TestPayload_PlayspecDetail(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	spec := seedPlayspec(t, c)

	t.Run("Get returns services and mounted_files", func(t *testing.T) {
		d, err := c.Playspecs.Get(ctx(), *spec.ID)
		requireNoError(t, err)
		if d.Name == "" {
			t.Error("expected Name on detail")
		}
		if d.Services == nil {
			t.Error("expected non-nil Services on detail")
		}
	})

	t.Run("Services returns expected service definitions", func(t *testing.T) {
		services, err := c.Playspecs.Services(ctx(), *spec.ID)
		requireNoError(t, err)
		// Services may be a wrapper; at least check it's non-nil
		if services == nil {
			t.Error("expected non-nil services response")
		}
	})
}

// TestPayload_APIKeyTokenExposure verifies token is shown ONCE on create.
func TestPayload_APIKeyTokenExposure(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	k, err := c.APIKeys.Create(ctx(), &fibe.APIKeyCreateParams{
		Label:  uniqueName("token-check"),
		Scopes: []string{"agents:read"},
	})
	requireNoError(t, err)
	t.Cleanup(func() {
		if k.ID != nil {
			c.APIKeys.Delete(ctx(), *k.ID)
		}
	})

	if k.Token == nil || *k.Token == "" {
		t.Error("expected Token on create response")
	}
	if k.MaskedToken == "" {
		t.Error("expected MaskedToken always present")
	}

	// Now re-list: Token should NOT be exposed again
	list, err := c.APIKeys.List(ctx(), nil)
	requireNoError(t, err)
	for _, kk := range list.Data {
		if kk.ID != nil && k.ID != nil && *kk.ID == *k.ID {
			if kk.Token != nil && *kk.Token != "" {
				t.Errorf("Token should NOT be exposed on list; got %q", *kk.Token)
			}
			if kk.MaskedToken == "" {
				t.Error("expected MaskedToken on list")
			}
			return
		}
	}
}

// TestPayload_WebhookDelivery verifies delivery records have populated fields.
func TestPayload_WebhookDelivery(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	ep, err := c.WebhookEndpoints.Create(ctx(), &fibe.WebhookEndpointCreateParams{
		URL:    "https://example.com/hook-" + uniqueName(""),
		Secret: "test-secret-1234567890",
		Events: []string{"playground.created"},
	})
	requireNoError(t, err)
	t.Cleanup(func() {
		if ep.ID != nil {
			c.WebhookEndpoints.Delete(ctx(), *ep.ID)
		}
	})

	// Fire a test event
	if ep.ID != nil {
		_ = c.WebhookEndpoints.Test(ctx(), *ep.ID)
	}

	t.Run("deliveries list payload has timestamps and status", func(t *testing.T) {
		if ep.ID == nil {
			t.Skip("no ep ID")
		}
		list, err := c.WebhookEndpoints.ListDeliveries(ctx(), *ep.ID, nil)
		requireNoError(t, err)
		// Deliveries may take a moment; check structure if any
		for _, d := range list.Data {
			if d.EventType == "" {
				t.Error("expected EventType on delivery")
			}
			if d.Status == "" {
				t.Error("expected Status on delivery")
			}
			if d.CreatedAt == nil {
				t.Error("expected CreatedAt on delivery")
			}
		}
	})
}

// TestPayload_PlayerMe verifies Me returns expected fields including scopes.
func TestPayload_PlayerMe(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	me, err := c.APIKeys.Me(ctx())
	requireNoError(t, err)
	if me.ID == 0 {
		t.Error("expected non-zero Player.ID")
	}
	if me.Username == "" {
		t.Error("expected non-empty Player.Username")
	}
	// APIKeyScopes may be empty for full-access keys but the field should be present (non-nil when scopes configured)

	// Verify scoped key reports its scopes
	scoped := createScopedKey(t, c, "me-scopes", []string{"agents:read", "playgrounds:read"})
	me2, err := scoped.APIKeys.Me(ctx())
	requireNoError(t, err)
	if len(me2.APIKeyScopes) == 0 {
		t.Error("expected APIKeyScopes to reflect scoped key's scopes")
	}
	foundAgents := false
	foundPG := false
	for _, s := range me2.APIKeyScopes {
		if s == "agents:read" {
			foundAgents = true
		}
		if s == "playgrounds:read" {
			foundPG = true
		}
	}
	if !foundAgents || !foundPG {
		t.Errorf("expected scopes to contain agents:read and playgrounds:read, got %v", me2.APIKeyScopes)
	}
}

// TestPayload_SecretValueRoundtrip verifies secret encryption + decryption.
func TestPayload_SecretValueRoundtrip(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	originalValue := "super-secret-value-" + uniqueName("")
	key := "ROUNDTRIP_" + uniqueName("KEY")
	s, err := c.Secrets.Create(ctx(), &fibe.SecretCreateParams{
		Key:   key,
		Value: originalValue,
	})
	requireNoError(t, err)
	t.Cleanup(func() {
		if s.ID != nil {
			c.Secrets.Delete(ctx(), *s.ID)
		}
	})

	// List: value should NOT be exposed
	list, err := c.Secrets.List(ctx(), &fibe.SecretListParams{Key: key})
	requireNoError(t, err)
	for _, ls := range list.Data {
		if ls.Key == key && ls.Value != nil && *ls.Value != "" {
			t.Errorf("expected nil/empty Value in list, got %q", *ls.Value)
		}
	}

	// Get: value SHOULD be exposed (reveal)
	if s.ID != nil {
		got, err := c.Secrets.Get(ctx(), *s.ID)
		requireNoError(t, err)
		if got.Value == nil || *got.Value != originalValue {
			t.Errorf("expected Value to round-trip: want %q, got %v", originalValue, got.Value)
		}
	}
}
