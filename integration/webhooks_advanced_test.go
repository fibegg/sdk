package integration

import (
	"testing"
	"time"

	"github.com/fibegg/sdk/fibe"
)

// Migrated from: 29-webhooks.spec.js (advanced)
func TestWebhooks_EventTypes(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	types, err := c.WebhookEndpoints.EventTypes(ctx())
	requireNoError(t, err)

	if len(types) == 0 {
		t.Fatal("expected at least one event type")
	}

	knownEvents := map[string]bool{
		"playground.created":        false,
		"playground.status.changed": false,
	}

	for _, et := range types {
		if _, ok := knownEvents[et]; ok {
			knownEvents[et] = true
		}
	}

	for event, found := range knownEvents {
		if !found {
			t.Errorf("expected event type %q in list", event)
		}
	}
}

// Migrated from: 29-webhooks.spec.js
func TestWebhooks_DeliveryHistory(t *testing.T) {
	t.Parallel()
	c := adminClient(t)
	server, requests := newWebhookCaptureServer(t)

	ep, err := c.WebhookEndpoints.Create(ctx(), &fibe.WebhookEndpointCreateParams{
		URL:    server.URL + "/delivery-" + uniqueName(""),
		Secret: uniqueName("delivery-secret"),
		Events: []string{"playground.created"},
	})
	requireNoError(t, err)
	t.Cleanup(func() { c.WebhookEndpoints.Delete(ctx(), *ep.ID) })

	t.Run("test endpoint queues delivery", func(t *testing.T) {
		reqCtx, cancel := ctxTimeout(10 * time.Second)
		defer cancel()

		err := c.WebhookEndpoints.Test(reqCtx, *ep.ID)
		requireNoError(t, err)
		_, _ = pollWebhookRequest(5*time.Second, requests)
	})

	t.Run("deliveries list returns results", func(t *testing.T) {
		reqCtx, cancel := ctxTimeout(10 * time.Second)
		defer cancel()

		result, err := c.WebhookEndpoints.ListDeliveries(reqCtx, *ep.ID, nil)
		requireNoError(t, err)

		if result.Data == nil {
			t.Error("expected deliveries data to be non-nil")
		}
	})

	t.Run("update endpoint events", func(t *testing.T) {
		reqCtx, cancel := ctxTimeout(10 * time.Second)
		defer cancel()

		updated, err := c.WebhookEndpoints.Update(reqCtx, *ep.ID, &fibe.WebhookEndpointUpdateParams{
			Events: []string{"playground.created", "agent.updated", "playground.destroyed"},
		})
		requireNoError(t, err)

		if len(updated.Events) != 3 {
			t.Errorf("expected 3 events after update, got %d", len(updated.Events))
		}
	})

	t.Run("disable and re-enable endpoint", func(t *testing.T) {
		reqCtx, cancel := ctxTimeout(10 * time.Second)
		defer cancel()

		_, err := c.WebhookEndpoints.Update(reqCtx, *ep.ID, &fibe.WebhookEndpointUpdateParams{
			Enabled: ptr(false),
		})
		requireNoError(t, err)

		got, err := c.WebhookEndpoints.Get(reqCtx, *ep.ID)
		requireNoError(t, err)
		if got.Enabled != nil && *got.Enabled {
			t.Error("expected disabled")
		}

		_, err = c.WebhookEndpoints.Update(reqCtx, *ep.ID, &fibe.WebhookEndpointUpdateParams{
			Enabled: ptr(true),
		})
		requireNoError(t, err)
	})
}

// Migrated from: 29-webhooks.spec.js
func TestWebhooks_SecretHandling(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	t.Run("secret shown on create only", func(t *testing.T) {
		t.Parallel()
		ep, err := c.WebhookEndpoints.Create(ctx(), &fibe.WebhookEndpointCreateParams{
			URL:    "https://sdk-webhook-secret.invalid/post",
			Secret: "visible-on-create-only",
			Events: []string{"playground.created"},
		})
		requireNoError(t, err)
		t.Cleanup(func() { c.WebhookEndpoints.Delete(ctx(), *ep.ID) })

		if ep.Secret == nil || *ep.Secret == "" {
			t.Error("secret should be visible on create")
		}

		got, err := c.WebhookEndpoints.Get(ctx(), *ep.ID)
		requireNoError(t, err)

		if got.Secret != nil && *got.Secret != "" {
			t.Error("secret should NOT be visible on get")
		}
	})
}
