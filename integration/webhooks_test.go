package integration

import (
	"reflect"
	"testing"

	"github.com/fibegg/sdk/fibe"
)

func TestWebhookEndpoints_CRUD(t *testing.T) {
	t.Parallel()
	c := userClient(t)
	server, _ := newWebhookCaptureServer(t)

	var endpointID int64
	endpointURL := server.URL + "/crud-" + uniqueName("")

	t.Run("create webhook endpoint", func(t *testing.T) {
		// Parallel disabled: dependent sequence
		ep, err := c.WebhookEndpoints.Create(ctx(), &fibe.WebhookEndpointCreateParams{
			URL:         endpointURL,
			Events:      []string{"playground.created", "playground.status.changed"},
			Description: ptr("integration test webhook"),
			ToolFilters: map[string][]string{"playground.created": {"deploy", "status"}},
		})
		requireNoError(t, err)

		if ep.ID == nil {
			t.Fatal("expected endpoint ID")
		}
		endpointID = *ep.ID
		if ep.URL != endpointURL {
			t.Errorf("expected URL, got %q", ep.URL)
		}
		if len(ep.Events) != 2 {
			t.Errorf("expected 2 events, got %d", len(ep.Events))
		}
		if ep.Secret == nil || *ep.Secret == "" {
			t.Error("expected server-generated secret")
		}
		if want := (map[string][]string{"playground.created": {"deploy", "status"}}); !reflect.DeepEqual(ep.ToolFilters, want) {
			t.Errorf("expected tool filters %v, got %v", want, ep.ToolFilters)
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
		if want := (map[string][]string{"playground.created": {"deploy", "status"}}); !reflect.DeepEqual(ep.ToolFilters, want) {
			t.Errorf("expected persisted tool filters %v, got %v", want, ep.ToolFilters)
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
			ToolFilters: map[string][]string{"playground.created": {}},
		})
		requireNoError(t, err)

		if ep.Description == nil || *ep.Description != newDesc {
			t.Error("expected updated description")
		}
		got, err := c.WebhookEndpoints.Get(ctx(), endpointID)
		requireNoError(t, err)
		want := map[string][]string{"playground.created": {}}
		if !reflect.DeepEqual(ep.ToolFilters, want) || !reflect.DeepEqual(got.ToolFilters, want) {
			t.Errorf("expected explicit empty filter to persist, update=%v get=%v", ep.ToolFilters, got.ToolFilters)
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
			URL:    server.URL + "/delete-" + uniqueName(""),
			Events: []string{"playground.created"},
		})
		requireNoError(t, err)

		err = c.WebhookEndpoints.Delete(ctx(), *ep.ID)
		requireNoError(t, err)

		_, err = c.WebhookEndpoints.Get(ctx(), *ep.ID)
		requireAPIError(t, err, fibe.ErrCodeNotFound, 404)
	})
}

func TestWebhookToolFilters_Validation(t *testing.T) {
	t.Parallel()
	c := userClient(t)
	for _, tc := range []struct {
		name    string
		filters map[string][]string
	}{
		{"unknown event", map[string][]string{"mcp.tool.executed": {"deploy"}}},
		{"blank tool name", map[string][]string{"playground.created": {" "}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			endpointURL := "https://sdk-webhook-validation.invalid/" + uniqueName("")
			_, err := c.WebhookEndpoints.Create(ctx(), &fibe.WebhookEndpointCreateParams{
				URL:         endpointURL,
				Events:      []string{"playground.created"},
				ToolFilters: tc.filters,
			})
			requireAPIError(t, err, fibe.ErrCodeValidationFailed, 422)

			valid := map[string][]string{"playground.created": {"deploy"}}
			ep, err := c.WebhookEndpoints.Create(ctx(), &fibe.WebhookEndpointCreateParams{
				URL:         endpointURL,
				Events:      []string{"playground.created"},
				ToolFilters: valid,
			})
			requireNoError(t, err)
			t.Cleanup(func() { c.WebhookEndpoints.Delete(ctx(), *ep.ID) })
			_, err = c.WebhookEndpoints.Update(ctx(), *ep.ID, &fibe.WebhookEndpointUpdateParams{ToolFilters: tc.filters})
			requireAPIError(t, err, fibe.ErrCodeValidationFailed, 422)
			got, err := c.WebhookEndpoints.Get(ctx(), *ep.ID)
			requireNoError(t, err)
			if !reflect.DeepEqual(got.ToolFilters, valid) {
				t.Errorf("invalid update changed persisted filters: got %v, want %v", got.ToolFilters, valid)
			}
		})
	}
}
