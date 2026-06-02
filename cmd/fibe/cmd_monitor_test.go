package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"nhooyr.io/websocket"
)

func TestMonitorLogsStreamsNDJSON(t *testing.T) {
	setupAuthTest(t)
	srv := monitorLogsTestServer(t, func(identifier map[string]any) {
		if identifier["channel"] != "ContainerLogsChannel" || identifier["playground_id"] != float64(42) || identifier["service_name"] != "web" {
			t.Fatalf("unexpected identifier: %#v", identifier)
		}
	})
	defer srv.Close()

	t.Setenv("FIBE_DOMAIN", srv.URL)
	t.Setenv("FIBE_API_KEY", "pk_test")

	cmd := RootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--output", "json", "monitor", "logs", "demo", "--service", "web", "--max-lines", "1"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	event := firstNDJSONEvent(t, out.Bytes())
	if event["type"] != "log" || event["line"] != "ready" || event["service"] != "web" {
		t.Fatalf("unexpected NDJSON event: %#v from %s", event, out.String())
	}
}

func TestPlaygroundLogsFollowAliasStreamsAllServices(t *testing.T) {
	setupAuthTest(t)
	srv := monitorLogsTestServer(t, func(identifier map[string]any) {
		if identifier["channel"] != "PlaygroundLogsChannel" || identifier["playground_id"] != float64(42) {
			t.Fatalf("unexpected identifier: %#v", identifier)
		}
		if _, ok := identifier["service_name"]; ok {
			t.Fatalf("all-service follow should not send service_name: %#v", identifier)
		}
	})
	defer srv.Close()

	t.Setenv("FIBE_DOMAIN", srv.URL)
	t.Setenv("FIBE_API_KEY", "pk_test")

	cmd := RootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--output", "json", "playgrounds", "logs", "demo", "--follow", "--max-lines", "1"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	event := firstNDJSONEvent(t, out.Bytes())
	if event["type"] != "log" || event["line"] != "ready" || event["service"] != "web" {
		t.Fatalf("unexpected NDJSON event: %#v from %s", event, out.String())
	}
}

func TestMonitorLogsTargetTrickStreamsNDJSON(t *testing.T) {
	setupAuthTest(t)
	srv := monitorLogsTestServer(t, func(identifier map[string]any) {
		if identifier["channel"] != "ContainerLogsChannel" || identifier["playground_id"] != float64(42) || identifier["service_name"] != "worker" {
			t.Fatalf("unexpected identifier: %#v", identifier)
		}
	})
	defer srv.Close()

	t.Setenv("FIBE_DOMAIN", srv.URL)
	t.Setenv("FIBE_API_KEY", "pk_test")

	cmd := RootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--output", "json", "monitor", "logs", "demo", "--target", "trick", "--service", "worker", "--max-lines", "1"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	event := firstNDJSONEvent(t, out.Bytes())
	if event["type"] != "log" || event["line"] != "ready" {
		t.Fatalf("unexpected NDJSON event: %#v from %s", event, out.String())
	}
}

func TestTricksLogsFollowAliasStreamsService(t *testing.T) {
	setupAuthTest(t)
	srv := monitorLogsTestServer(t, func(identifier map[string]any) {
		if identifier["channel"] != "ContainerLogsChannel" || identifier["playground_id"] != float64(42) || identifier["service_name"] != "worker" {
			t.Fatalf("unexpected identifier: %#v", identifier)
		}
	})
	defer srv.Close()

	t.Setenv("FIBE_DOMAIN", srv.URL)
	t.Setenv("FIBE_API_KEY", "pk_test")

	cmd := RootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--output", "json", "tricks", "logs", "demo", "--service", "worker", "--follow", "--max-lines", "1"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	event := firstNDJSONEvent(t, out.Bytes())
	if event["type"] != "log" || event["line"] != "ready" {
		t.Fatalf("unexpected NDJSON event: %#v from %s", event, out.String())
	}
}

func TestLogsSnapshotJSONOutput(t *testing.T) {
	for name, args := range map[string][]string{
		"playgrounds": {"--output", "json", "playgrounds", "logs", "demo", "--service", "web", "--tail", "12"},
		"tricks":      {"--output", "json", "tricks", "logs", "demo", "--service", "web", "--tail", "12"},
	} {
		t.Run(name, func(t *testing.T) {
			setupAuthTest(t)
			var body map[string]any
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost || r.URL.Path != "/api/playgrounds/demo/logs" {
					t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
				}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Fatalf("decode logs body: %v", err)
				}
				_ = json.NewEncoder(w).Encode(map[string]any{
					"service": "web",
					"lines":   []string{"line1", "line2"},
					"source":  "snapshot",
				})
			}))
			defer srv.Close()

			t.Setenv("FIBE_DOMAIN", srv.URL)
			t.Setenv("FIBE_API_KEY", "pk_test")

			out, err := captureStdout(func() error {
				cmd := RootCmd()
				cmd.SetArgs(args)
				return cmd.Execute()
			})
			if err != nil {
				t.Fatalf("execute: %v", err)
			}
			if body["service"] != "web" || body["tail"] != float64(12) {
				t.Fatalf("unexpected snapshot request body: %#v", body)
			}
			var got map[string]any
			if err := json.Unmarshal([]byte(out), &got); err != nil {
				t.Fatalf("decode snapshot JSON %q: %v", out, err)
			}
			lines := got["lines"].([]any)
			if got["service"] != "web" || len(lines) != 2 || lines[0] != "line1" {
				t.Fatalf("unexpected snapshot JSON: %#v", got)
			}
		})
	}
}

func TestLogsSnapshotRequiresService(t *testing.T) {
	setupAuthTest(t)
	for name, args := range map[string][]string{
		"playgrounds": {"playgrounds", "logs", "demo"},
		"tricks":      {"tricks", "logs", "demo"},
	} {
		t.Run(name, func(t *testing.T) {
			cmd := RootCmd()
			cmd.SetOut(&bytes.Buffer{})
			cmd.SetErr(&bytes.Buffer{})
			cmd.SetArgs(args)
			err := cmd.Execute()
			if err == nil || !strings.Contains(err.Error(), `required flag "service" not set`) {
				t.Fatalf("expected service validation error, got %v", err)
			}
		})
	}
}

func monitorLogsTestServer(t *testing.T, assertIdentifier func(map[string]any)) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/playgrounds/demo":
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 42, "name": "demo", "status": "running"})
		case "/cable":
			conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
				Subprotocols: []string{"actioncable-v1-json"},
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
			assertIdentifier(identifier)

			_ = conn.Write(r.Context(), websocket.MessageText, []byte(`{"message":{"type":"log","stream":"stdout","service":"web","line":"ready"}}`))
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
}

func firstNDJSONEvent(t *testing.T, data []byte) map[string]any {
	t.Helper()
	dec := json.NewDecoder(bytes.NewReader(data))
	var event map[string]any
	if err := dec.Decode(&event); err != nil {
		t.Fatalf("decode NDJSON event from %q: %v", string(data), err)
	}
	return event
}
