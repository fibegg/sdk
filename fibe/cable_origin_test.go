package fibe

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/coder/websocket"
)

func TestCableOriginUsesCanonicalAPIOrigin(t *testing.T) {
	for _, fixture := range []struct{ name, base, want string }{
		{"public domain", "fibe.gg", "https://fibe.gg"},
		{"local domain", "localhost:3000", "http://localhost:3000"},
		{"HTTPS port and path", "https://api.example.test:8443/api/v2/", "https://api.example.test:8443"},
		{"HTTP path", "http://127.0.0.1:3000/api", "http://127.0.0.1:3000"},
		{"IPv6", "http://[::1]:8080/api", "http://[::1]:8080"},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			client := NewClient(WithDisableAutoConfig(), WithBaseURL(fixture.base))
			got, err := client.cableOrigin()
			if err != nil || got != fixture.want {
				t.Fatalf("origin = %q, %v; want %q", got, err, fixture.want)
			}
		})
	}
}

func TestCableOriginRejectsUnsupportedAPIURLs(t *testing.T) {
	for _, base := range []string{
		"https://fixture:synthetic-password@api.example.test/api",
		"https://api.example.test/api?query=1",
		"https://api.example.test/api#fragment",
		"ftp://api.example.test/api",
		"https:///api",
		"",
	} {
		t.Run(base, func(t *testing.T) {
			client := NewClient(WithDisableAutoConfig(), WithBaseURL(base))
			if origin, err := client.cableOrigin(); err == nil || origin != "" {
				t.Fatalf("unsupported API URL produced origin %q, error %v", origin, err)
			}
		})
	}
}

func TestCableSubscriptionSendsAPIOriginThroughProxy(t *testing.T) {
	const targetOrigin = "http://api.example.test:8087"
	observed := make(chan string, 1)
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		observed <- r.Header.Get("Origin")
		if r.Header.Get("Origin") != targetOrigin || r.Header.Get("Authorization") != "Bearer review-synthetic-key" || r.Host != "api.example.test:8087" {
			http.Error(w, "Invalid target authority or origin", http.StatusForbidden)
			return
		}
		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{Subprotocols: []string{actionCableProtocol}})
		if err != nil {
			t.Error(err)
			return
		}
		defer conn.Close(websocket.StatusNormalClosure, "")
		if _, _, err = conn.Read(r.Context()); err != nil {
			t.Error(err)
			return
		}
		_ = conn.Write(r.Context(), websocket.MessageText, []byte(`{"message":{"event":"origin-verified"}}`))
	}))
	defer proxy.Close()
	proxyURL, err := url.Parse(proxy.URL)
	if err != nil {
		t.Fatal(err)
	}
	client := NewClient(WithDisableAutoConfig(), WithBaseURL(targetOrigin+"/api"), WithAPIKey("review-synthetic-key"),
		WithHTTPClient(&http.Client{Transport: &http.Transport{Proxy: http.ProxyURL(proxyURL)}}))
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	events, failures := client.Cable.SubscribeResource(ctx, "Agent")
	select {
	case event := <-events:
		payload, _ := json.Marshal(event.Message)
		if string(payload) != `{"event":"origin-verified"}` {
			t.Fatalf("unexpected subscription payload %s", payload)
		}
	case failure := <-failures:
		t.Fatalf("subscription failed: %v", failure)
	case <-ctx.Done():
		t.Fatal("subscription timed out")
	}
	if got := <-observed; got != targetOrigin {
		t.Fatalf("proxy handshake origin %q; want target %q", got, targetOrigin)
	}
}

func TestCableSubscriptionSendsHTTPSOrigin(t *testing.T) {
	var expectedOrigin string
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Origin") != expectedOrigin {
			http.Error(w, "Origin does not match HTTPS API origin", http.StatusForbidden)
			return
		}
		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{Subprotocols: []string{actionCableProtocol}})
		if err != nil {
			t.Error(err)
			return
		}
		defer conn.Close(websocket.StatusNormalClosure, "")
		if _, _, err = conn.Read(r.Context()); err != nil {
			t.Error(err)
			return
		}
		_ = conn.Write(r.Context(), websocket.MessageText, []byte(`{"message":{"event":"https-origin-verified"}}`))
	}))
	defer server.Close()
	expectedOrigin = server.URL
	client := NewClient(WithDisableAutoConfig(), WithBaseURL(server.URL+"/api"), WithAPIKey("review-synthetic-key"), WithHTTPClient(server.Client()))
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	events, failures := client.Cable.SubscribeResource(ctx, "Agent")
	select {
	case event := <-events:
		payload, _ := json.Marshal(event.Message)
		if string(payload) != `{"event":"https-origin-verified"}` {
			t.Fatalf("unexpected subscription payload %s", payload)
		}
	case failure := <-failures:
		t.Fatalf("subscription failed: %v", failure)
	case <-ctx.Done():
		t.Fatal("subscription timed out")
	}
}
