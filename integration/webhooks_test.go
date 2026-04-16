package integration

import (
	"testing"

	"github.com/fibegg/sdk/fibe"
)

func TestWebhookEndpoints_CRUD(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	var endpointID int64

	t.Run("create webhook endpoint", func(t *testing.T) {
		// Parallel disabled: dependent sequence
		ep, err := c.WebhookEndpoints.Create(ctx(), &fibe.WebhookEndpointCreateParams{
			URL:         "https://httpbin.org/post",
			Events:      []string{"playground.created", "playground.status.changed"},
			Description: ptr("integration test webhook"),
			ToolFilters: map[string][]string{"mcp.tool.executed": []string{"deploy", "status"}},
		})
		requireNoError(t, err)

		if ep.ID == nil {
			t.Fatal("expected endpoint ID")
		}
		endpointID = *ep.ID
		if ep.URL != "https://httpbin.org/post" {
			t.Errorf("expected URL, got %q", ep.URL)
		}
		if len(ep.Events) != 2 {
			t.Errorf("expected 2 events, got %d", len(ep.Events))
		}
		if ep.Secret == nil || *ep.Secret == "" {
			t.Error("expected server-generated secret")
		}
		if got := ep.ToolFilters["mcp.tool.executed"]; len(got) != 2 {
			t.Errorf("expected 2 tool filters, got %d", len(got))
		}
	})
	t.Cleanup(func() {
		if endpointID > 0 {
			c.WebhookEndpoints.Delete(ctx(), endpointID)
		}
	})

	t.Run("list webhook endpoints", func(t *testing.T) {
		t.Parallel()
		result, err := c.WebhookEndpoints.List(ctx(), nil)
		requireNoError(t, err)

		if result.Meta.Total == 0 {
			t.Error("expected at least one endpoint")
		}
	})

	t.Run("get webhook endpoint", func(t *testing.T) {
		// Parallel disabled: dependent sequence with update
		if endpointID == 0 {
			t.Skip("no endpoint created")
		}
		ep, err := c.WebhookEndpoints.Get(ctx(), endpointID)
		requireNoError(t, err)

		if *ep.ID != endpointID {
			t.Errorf("expected ID %d", endpointID)
		}
	})

	t.Run("update webhook endpoint", func(t *testing.T) {
		// Parallel disabled: dependent sequence with get
		if endpointID == 0 {
			t.Skip("no endpoint created")
		}
		newDesc := "updated description"
		ep, err := c.WebhookEndpoints.Update(ctx(), endpointID, &fibe.WebhookEndpointUpdateParams{
			Description: &newDesc,
			Enabled:     ptr(false),
		})
		requireNoError(t, err)

		if ep.Description == nil || *ep.Description != newDesc {
			t.Error("expected updated description")
		}
	})

	t.Run("test webhook endpoint", func(t *testing.T) {
		t.Parallel()
		if endpointID == 0 {
			t.Skip("no endpoint created")
		}
		err := c.WebhookEndpoints.Test(ctx(), endpointID)
		requireNoError(t, err)
	})

	t.Run("list deliveries", func(t *testing.T) {
		t.Parallel()
		if endpointID == 0 {
			t.Skip("no endpoint created")
		}
		result, err := c.WebhookEndpoints.ListDeliveries(ctx(), endpointID, nil)
		requireNoError(t, err)

		if result.Data == nil {
			t.Error("expected deliveries data to be non-nil")
		}
	})

	t.Run("event types", func(t *testing.T) {
		t.Parallel()
		types, err := c.WebhookEndpoints.EventTypes(ctx())
		requireNoError(t, err)

		if len(types) == 0 {
			t.Error("expected at least one event type")
		}
	})

	t.Run("delete webhook endpoint", func(t *testing.T) {
		t.Parallel()
		ep, err := c.WebhookEndpoints.Create(ctx(), &fibe.WebhookEndpointCreateParams{
			URL:    "https://httpbin.org/post",
			Events: []string{"playground.created"},
		})
		requireNoError(t, err)

		err = c.WebhookEndpoints.Delete(ctx(), *ep.ID)
		requireNoError(t, err)

		_, err = c.WebhookEndpoints.Get(ctx(), *ep.ID)
		requireAPIError(t, err, fibe.ErrCodeNotFound, 404)
	})
}

func TestWebhookEndpoints_ScopeEnforcement(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	ep, err := c.WebhookEndpoints.Create(ctx(), &fibe.WebhookEndpointCreateParams{
		URL:    "https://httpbin.org/post",
		Events: []string{"playground.created"},
	})
	requireNoError(t, err)
	t.Cleanup(func() { c.WebhookEndpoints.Delete(ctx(), *ep.ID) })

	t.Run("read-only can list", func(t *testing.T) {
		t.Parallel()
		readOnly := createScopedKey(t, c, "wh-read", []string{"webhooks:read"})
		_, err := readOnly.WebhookEndpoints.List(ctx(), nil)
		requireNoError(t, err)
	})

	t.Run("read-only cannot create", func(t *testing.T) {
		t.Parallel()
		readOnly := createScopedKey(t, c, "wh-read2", []string{"webhooks:read"})
		_, err := readOnly.WebhookEndpoints.Create(ctx(), &fibe.WebhookEndpointCreateParams{
			URL:    "https://nope.com",
			Events: []string{"playground.created"},
		})
		requireAPIError(t, err, fibe.ErrCodeForbidden, 403)
	})

	t.Run("no scope returns 403", func(t *testing.T) {
		t.Parallel()
		noScope := createScopedKey(t, c, "no-wh", []string{"agents:read"})
		_, err := noScope.WebhookEndpoints.List(ctx(), nil)
		requireAPIError(t, err, fibe.ErrCodeForbidden, 403)
	})
}
