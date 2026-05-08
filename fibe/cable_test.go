package fibe

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"nhooyr.io/websocket"
)

func TestCableSubscribeResource(t *testing.T) {
	apiKey := "fibe_test_secret"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/cable" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		protocols := r.Header.Values("Sec-WebSocket-Protocol")
		if !strings.Contains(strings.Join(protocols, ","), apiKeyProtocolPrefix) {
			t.Fatalf("missing api key subprotocol: %#v", protocols)
		}
		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
			Subprotocols: []string{actionCableProtocol},
		})
		if err != nil {
			t.Fatalf("accept websocket: %v", err)
		}
		defer conn.Close(websocket.StatusNormalClosure, "")

		_, data, err := conn.Read(r.Context())
		if err != nil {
			t.Fatalf("read subscribe: %v", err)
		}
		var subscribe map[string]string
		if err := json.Unmarshal(data, &subscribe); err != nil {
			t.Fatalf("decode subscribe: %v", err)
		}
		if subscribe["command"] != "subscribe" {
			t.Fatalf("unexpected subscribe body: %#v", subscribe)
		}
		var identifier map[string]string
		if err := json.Unmarshal([]byte(subscribe["identifier"]), &identifier); err != nil {
			t.Fatalf("decode identifier: %v", err)
		}
		if identifier["channel"] != "ApiResourceChannel" || identifier["resource"] != "Agent" {
			t.Fatalf("unexpected identifier: %#v", identifier)
		}
		_ = conn.Write(r.Context(), websocket.MessageText, []byte(`{"type":"confirm_subscription"}`))
		_ = conn.Write(r.Context(), websocket.MessageText, []byte(`{"message":{"event":"updated","id":42}}`))
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL), WithAPIKey(apiKey))
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	events, errs := client.Cable.SubscribeResource(ctx, "Agent")
	select {
	case ev := <-events:
		message := ev.Message.(map[string]any)
		if message["event"] != "updated" || message["id"] != float64(42) || ev.Resource != "Agent" {
			t.Fatalf("unexpected event: %#v", ev)
		}
	case err := <-errs:
		t.Fatalf("unexpected error: %v", err)
	case <-ctx.Done():
		t.Fatal("timed out waiting for event")
	}
}
