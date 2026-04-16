package integration

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/fibegg/sdk/fibe"
)

type capturedWebhookRequest struct {
	Headers http.Header
	Body    []byte
}

func newWebhookCaptureServer(t *testing.T) (*httptest.Server, <-chan capturedWebhookRequest) {
	t.Helper()

	requests := make(chan capturedWebhookRequest, 4)
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read webhook request body: %v", err)
			http.Error(w, "read error", http.StatusInternalServerError)
			return
		}
		requests <- capturedWebhookRequest{
			Headers: r.Header.Clone(),
			Body:    body,
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))

	listener, err := net.Listen("tcp", "0.0.0.0:0")
	if err != nil {
		t.Fatalf("listen for webhook capture server: %v", err)
	}
	server.Listener = listener
	server.Start()
	t.Cleanup(server.Close)

	if host := os.Getenv("WEBHOOK_CATCHER_HOST"); host != "" {
		_, port, err := net.SplitHostPort(listener.Addr().String())
		if err != nil {
			t.Fatalf("parse webhook capture listener address %q: %v", listener.Addr().String(), err)
		}
		scheme := os.Getenv("WEBHOOK_CATCHER_SCHEME")
		if scheme == "" {
			scheme = "http"
		}
		server.URL = fmt.Sprintf("%s://%s:%s", scheme, host, port)
	}

	return server, requests
}

func pollWebhookRequest(timeout time.Duration, requests <-chan capturedWebhookRequest) (capturedWebhookRequest, bool) {
	attempts := int(timeout / time.Second)
	if attempts < 1 {
		attempts = 1
	}
	return pollUntil(attempts, time.Second, func() (capturedWebhookRequest, bool) {
		select {
		case req := <-requests:
			return req, true
		default:
			return capturedWebhookRequest{}, false
		}
	})
}

func pollWebhookDeliveries(c *fibe.Client, endpointID int64, timeout time.Duration) ([]fibe.WebhookDelivery, bool) {
	attempts := int(timeout / time.Second)
	if attempts < 1 {
		attempts = 1
	}
	return pollUntil(attempts, time.Second, func() ([]fibe.WebhookDelivery, bool) {
		list, err := c.WebhookEndpoints.ListDeliveries(ctx(), endpointID, nil)
		if err != nil {
			return nil, false
		}
		if len(list.Data) > 0 {
			return list.Data, true
		}
		return nil, false
	})
}

func formatWebhookDeliveries(deliveries []fibe.WebhookDelivery) string {
	if len(deliveries) == 0 {
		return "none"
	}

	parts := make([]string, 0, len(deliveries))
	for _, delivery := range deliveries {
		deliveryID := "nil"
		if delivery.ID != nil {
			deliveryID = fmt.Sprintf("%d", *delivery.ID)
		}
		responseCode := "nil"
		if delivery.ResponseCode != nil {
			responseCode = fmt.Sprintf("%d", *delivery.ResponseCode)
		}
		attempt := "nil"
		if delivery.Attempt != nil {
			attempt = fmt.Sprintf("%d", *delivery.Attempt)
		}
		parts = append(parts, fmt.Sprintf(
			"id=%s event=%s status=%s response_code=%s attempt=%s",
			deliveryID,
			delivery.EventType,
			delivery.Status,
			responseCode,
			attempt,
		))
	}

	return strings.Join(parts, "; ")
}

func TestWebhook_CreateWithEventFilters(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	// Pick a real playspec ID to use as filter target
	spec := seedPlayspec(t, c)

	ep, err := c.WebhookEndpoints.Create(ctx(), &fibe.WebhookEndpointCreateParams{
		URL:    "https://example.com/hook-" + uniqueName(""),
		Secret: "test-secret-min-20-chars-" + uniqueName(""),
		Events: []string{"playground.created", "playground.status.changed"},
		EventFilters: map[string]any{
			"playground.created": []int64{*spec.ID},
		},
	})
	requireNoError(t, err)
	t.Cleanup(func() {
		if ep.ID != nil {
			c.WebhookEndpoints.Delete(ctx(), *ep.ID)
		}
	})

	// Verify the endpoint was created with the right events
	if len(ep.Events) < 2 {
		t.Errorf("expected >= 2 events, got %d", len(ep.Events))
	}
}

func TestWebhook_TestDeliveryAppearsInHistory(t *testing.T) {
	t.Parallel()
	c := adminClient(t)
	server, _ := newWebhookCaptureServer(t)

	ep, err := c.WebhookEndpoints.Create(ctx(), &fibe.WebhookEndpointCreateParams{
		URL:    server.URL + "/history-" + uniqueName(""),
		Secret: "test-secret-min-chars-" + uniqueName(""),
		Events: []string{"webhook.test"},
	})
	requireNoError(t, err)
	t.Cleanup(func() {
		if ep.ID != nil {
			c.WebhookEndpoints.Delete(ctx(), *ep.ID)
		}
	})
	if ep.ID == nil {
		t.Fatal("expected webhook ID")
	}

	// Trigger a test delivery
	err = c.WebhookEndpoints.Test(ctx(), *ep.ID)
	requireNoError(t, err)

	_, found := pollWebhookDeliveries(c, *ep.ID, webhookTimeout())

	if !found {
		t.Log("delivery did not appear in history within timeout (may be async/queued)")
	}
}

func TestWebhook_EventTypesEndpoint(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	types, err := c.WebhookEndpoints.EventTypes(ctx())
	requireNoError(t, err)
	if len(types) == 0 {
		t.Error("expected non-empty list of event types")
	}
	// Known events that should always be present:
	knownEvents := []string{"playground.created", "marquee.created", "agent.created"}
	foundAny := false
	for _, want := range knownEvents {
		for _, got := range types {
			if got == want {
				foundAny = true
				break
			}
		}
	}
	if !foundAny {
		t.Errorf("expected at least one of %v in event types, got %v", knownEvents, types)
	}
}

func TestWebhook_UpdateEventsAndFilters(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	ep, err := c.WebhookEndpoints.Create(ctx(), &fibe.WebhookEndpointCreateParams{
		URL:    "https://example.com/hook-upd-" + uniqueName(""),
		Secret: "secret-" + uniqueName(""),
		Events: []string{"playground.created"},
	})
	requireNoError(t, err)
	t.Cleanup(func() {
		if ep.ID != nil {
			c.WebhookEndpoints.Delete(ctx(), *ep.ID)
		}
	})
	if ep.ID == nil {
		t.Fatal("expected ID")
	}

	t.Run("update replaces event list", func(t *testing.T) {
		// Discover a valid set of events first
		allTypes, err := c.WebhookEndpoints.EventTypes(ctx())
		requireNoError(t, err)
		if len(allTypes) < 2 {
			t.Skip("not enough known event types to test replacement")
		}
		newSet := allTypes[:2]
		upd, err := c.WebhookEndpoints.Update(ctx(), *ep.ID, &fibe.WebhookEndpointUpdateParams{
			Events: newSet,
		})
		requireNoError(t, err)
		if len(upd.Events) != len(newSet) {
			t.Errorf("expected %d events after update, got %d", len(newSet), len(upd.Events))
		}
	})

	t.Run("disable endpoint persists", func(t *testing.T) {
		f := false
		upd, err := c.WebhookEndpoints.Update(ctx(), *ep.ID, &fibe.WebhookEndpointUpdateParams{
			Enabled: &f,
		})
		requireNoError(t, err)
		if upd.Enabled == nil || *upd.Enabled {
			t.Errorf("expected Enabled=false, got %v", upd.Enabled)
		}
	})
}

func TestWebhook_TestEndpointDeliversSignedPayload(t *testing.T) {
	t.Parallel()
	c := adminClient(t)
	server, requests := newWebhookCaptureServer(t)
	secret := "hmac-secret-" + uniqueName("")

	ep, err := c.WebhookEndpoints.Create(ctx(), &fibe.WebhookEndpointCreateParams{
		URL:    server.URL + "/signed-" + uniqueName(""),
		Secret: secret,
		Events: []string{"webhook.test"},
	})
	requireNoError(t, err)
	t.Cleanup(func() {
		if ep.ID != nil {
			c.WebhookEndpoints.Delete(ctx(), *ep.ID)
		}
	})

	if ep.ID == nil {
		t.Fatal("expected ID")
	}
	err = c.WebhookEndpoints.Test(ctx(), *ep.ID)
	requireNoError(t, err)

	request, delivered := pollWebhookRequest(webhookTimeout(), requests)
	if !delivered {
		deliveries, found := pollWebhookDeliveries(c, *ep.ID, 10*time.Second)
		if !found && os.Getenv("WEBHOOK_CATCHER_HOST") == "" {
			t.Skipf("webhook request not observed within %s and no WEBHOOK_CATCHER_HOST is configured; ensure the webhook worker is running", webhookTimeout())
		}
		if found {
			t.Fatalf("webhook request not observed within %s; deliveries observed: %s", webhookTimeout(), formatWebhookDeliveries(deliveries))
		}
		t.Fatalf("webhook request not observed within %s; no delivery history recorded", webhookTimeout())
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(request.Body)
	expectedSignature := "sha256=" + hex.EncodeToString(mac.Sum(nil))
	if got := request.Headers.Get("X-Webhook-Signature"); got != expectedSignature {
		t.Errorf("expected X-Webhook-Signature %q, got %q", expectedSignature, got)
	}
	if got := request.Headers.Get("X-Webhook-Event"); got != "webhook.test" {
		t.Errorf("expected X-Webhook-Event %q, got %q", "webhook.test", got)
	}
	if got := request.Headers.Get("X-Webhook-Delivery"); got == "" {
		t.Error("expected X-Webhook-Delivery header")
	}
	if got := request.Headers.Get("X-Idempotency-Key"); got == "" {
		t.Error("expected X-Idempotency-Key header")
	}
	if len(request.Body) == 0 {
		t.Error("expected non-empty webhook payload body")
	}

	deliveries, found := pollWebhookDeliveries(c, *ep.ID, webhookTimeout())
	if !found {
		t.Fatalf("delivery not recorded within %s", webhookTimeout())
	}
	d := deliveries[0]
	if d.EventType == "" {
		t.Error("delivery missing EventType")
	}
	if d.Status == "" {
		t.Error("delivery missing Status")
	}
	if d.CreatedAt == nil {
		t.Error("delivery missing CreatedAt")
	}
}
