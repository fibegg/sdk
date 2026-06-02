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

func TestCableSubscribeLogStream(t *testing.T) {
	apiKey := "fibe_test_secret"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/cable" {
			t.Fatalf("unexpected path %s", r.URL.Path)
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
		var identifier map[string]any
		if err := json.Unmarshal([]byte(subscribe["identifier"]), &identifier); err != nil {
			t.Fatalf("decode identifier: %v", err)
		}
		if identifier["channel"] != "ContainerLogsChannel" || identifier["playground_id"] != float64(42) || identifier["service_name"] != "web" || identifier["tail"] != float64(25) {
			t.Fatalf("unexpected identifier: %#v", identifier)
		}
		if identifier["subscriber_id"] == "" {
			t.Fatalf("subscriber_id missing: %#v", identifier)
		}

		_ = conn.Write(r.Context(), websocket.MessageText, []byte(`{"type":"confirm_subscription"}`))
		_ = conn.Write(r.Context(), websocket.MessageText, []byte(`{"message":{"type":"status","status":"connected"}}`))
		_ = conn.Write(r.Context(), websocket.MessageText, []byte(`{"message":{"type":"log","stream":"stdout","service":"web","line":"ready","timestamp":"2026-06-02T00:00:00Z"}}`))
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL), WithAPIKey(apiKey))
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	events, errs := client.Cable.SubscribeLogStream(ctx, 42, "web", &LogsStreamOptions{Tail: 25})
	var got []LogStreamEvent
	for len(got) < 2 {
		select {
		case ev := <-events:
			got = append(got, ev)
		case err, ok := <-errs:
			if ok {
				t.Fatalf("unexpected error: %v", err)
			}
		case <-ctx.Done():
			t.Fatal("timed out waiting for log stream events")
		}
	}
	if got[0].Type != "status" || got[0].Status != "connected" {
		t.Fatalf("unexpected status event: %#v", got[0])
	}
	if got[1].Type != "log" || got[1].Line != "ready" || got[1].Service != "web" || got[1].ReceivedAt.IsZero() {
		t.Fatalf("unexpected log event: %#v", got[1])
	}
}

func TestPlaygroundLogStreamByIdentifierResolvesID(t *testing.T) {
	apiKey := "fibe_test_secret"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/playgrounds/demo":
			_ = json.NewEncoder(w).Encode(Playground{ID: 42, Name: "demo"})
		case "/cable":
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
			var identifier map[string]any
			if err := json.Unmarshal([]byte(subscribe["identifier"]), &identifier); err != nil {
				t.Fatalf("decode identifier: %v", err)
			}
			if identifier["channel"] != "PlaygroundLogsChannel" || identifier["playground_id"] != float64(42) {
				t.Fatalf("unexpected identifier: %#v", identifier)
			}
			if _, ok := identifier["service_name"]; ok {
				t.Fatalf("all-service stream should not include service_name: %#v", identifier)
			}
			_ = conn.Write(r.Context(), websocket.MessageText, []byte(`{"message":{"type":"log","stream":"stdout","service":"web","line":"all ready"}}`))
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL), WithAPIKey(apiKey))
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	events, errs := client.Playgrounds.LogStreamByIdentifier(ctx, "demo", "", &LogsStreamOptions{MaxLines: 1})
	select {
	case ev := <-events:
		if ev.Type != "log" || ev.Line != "all ready" || ev.Service != "web" {
			t.Fatalf("unexpected event: %#v", ev)
		}
	case err, ok := <-errs:
		if ok {
			t.Fatalf("unexpected error: %v", err)
		}
	case <-ctx.Done():
		t.Fatal("timed out waiting for log event")
	}
}

func TestTrickLogStreamByIdentifierResolvesID(t *testing.T) {
	apiKey := "fibe_test_secret"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/playgrounds/nightly-build":
			_ = json.NewEncoder(w).Encode(Playground{ID: 77, Name: "nightly-build"})
		case "/cable":
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
			var identifier map[string]any
			if err := json.Unmarshal([]byte(subscribe["identifier"]), &identifier); err != nil {
				t.Fatalf("decode identifier: %v", err)
			}
			if identifier["channel"] != "ContainerLogsChannel" || identifier["playground_id"] != float64(77) || identifier["service_name"] != "worker" || identifier["tail"] != float64(10) {
				t.Fatalf("unexpected identifier: %#v", identifier)
			}
			_ = conn.Write(r.Context(), websocket.MessageText, []byte(`{"message":{"type":"log","stream":"stderr","service":"worker","line":"job ready"}}`))
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL), WithAPIKey(apiKey))
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	events, errs := client.Tricks.LogStreamByIdentifier(ctx, "nightly-build", "worker", &LogsStreamOptions{Tail: 10, MaxLines: 1})
	select {
	case ev := <-events:
		if ev.Type != "log" || ev.Line != "job ready" || ev.Service != "worker" || ev.Stream != "stderr" {
			t.Fatalf("unexpected event: %#v", ev)
		}
	case err, ok := <-errs:
		if ok {
			t.Fatalf("unexpected error: %v", err)
		}
	case <-ctx.Done():
		t.Fatal("timed out waiting for trick log event")
	}
}

func TestLogStreamCancellationClosesChannels(t *testing.T) {
	apiKey := "fibe_test_secret"
	subscribed := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/playgrounds/demo":
			_ = json.NewEncoder(w).Encode(Playground{ID: 42, Name: "demo"})
		case "/cable":
			conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
				Subprotocols: []string{actionCableProtocol},
			})
			if err != nil {
				t.Fatalf("accept websocket: %v", err)
			}
			defer conn.Close(websocket.StatusNormalClosure, "")

			if _, _, err := conn.Read(r.Context()); err != nil {
				t.Fatalf("read subscribe: %v", err)
			}
			close(subscribed)
			<-r.Context().Done()
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	client := NewClient(WithBaseURL(server.URL), WithAPIKey(apiKey))
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	events, errs := client.Playgrounds.LogStreamByIdentifier(ctx, "demo", "", nil)

	select {
	case <-subscribed:
	case <-ctx.Done():
		t.Fatal("timed out waiting for subscription")
	}
	cancel()

	deadline := time.After(time.Second)
	for events != nil || errs != nil {
		select {
		case _, ok := <-events:
			if !ok {
				events = nil
			}
		case err, ok := <-errs:
			if !ok {
				errs = nil
				continue
			}
			if err != nil {
				t.Fatalf("unexpected error after cancellation: %v", err)
			}
		case <-deadline:
			t.Fatal("timed out waiting for log stream channels to close")
		}
	}
}
