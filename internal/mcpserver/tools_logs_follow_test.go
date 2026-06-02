package mcpserver

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"nhooyr.io/websocket"
)

func TestMonitorLogsFollowStreamsPlaygroundLogs(t *testing.T) {
	api := logsFollowAPIServer(t, []string{
		`{"message":{"type":"log","stream":"stdout","service":"web","line":"ready"}}`,
	}, 0, func(identifier map[string]any) {
		if identifier["channel"] != "ContainerLogsChannel" || identifier["playground_id"] != float64(42) || identifier["service_name"] != "web" {
			t.Fatalf("unexpected identifier: %#v", identifier)
		}
	})
	defer api.Close()

	srv := New(Config{APIKey: "pk_test", Domain: api.URL, ToolSet: "full"})
	if err := srv.RegisterAll(); err != nil {
		t.Fatalf("RegisterAll: %v", err)
	}

	out, err := srv.dispatcher.dispatch(context.Background(), "fibe_monitor_logs_follow", map[string]any{
		"id_or_name": "demo",
		"service":    "web",
		"max_lines":  1,
		"duration":   "2s",
	})
	if err != nil {
		t.Fatalf("logs follow dispatch: %v", err)
	}
	result := out.(map[string]any)
	if result["line_count"] != 1 {
		t.Fatalf("unexpected logs follow result: %#v", result)
	}
	lines := result["lines"].([]map[string]string)
	if lines[0]["text"] != "ready" {
		t.Fatalf("unexpected lines: %#v", lines)
	}
}

func TestPlaygroundsLogsFollowCompatibilityAliasStreamsAllServices(t *testing.T) {
	api := logsFollowAPIServer(t, []string{
		`{"message":{"type":"log","stream":"stdout","service":"web","line":"all ready"}}`,
	}, 0, func(identifier map[string]any) {
		if identifier["channel"] != "PlaygroundLogsChannel" || identifier["playground_id"] != float64(42) {
			t.Fatalf("unexpected identifier: %#v", identifier)
		}
		if _, ok := identifier["service_name"]; ok {
			t.Fatalf("all-service follow should not send service_name: %#v", identifier)
		}
	})
	defer api.Close()

	srv := New(Config{APIKey: "pk_test", Domain: api.URL, ToolSet: "full"})
	if err := srv.RegisterAll(); err != nil {
		t.Fatalf("RegisterAll: %v", err)
	}

	out, err := srv.dispatcher.dispatch(context.Background(), "fibe_playgrounds_logs_follow", map[string]any{
		"id_or_name": "demo",
		"max_lines":  1,
		"duration":   "2s",
	})
	if err != nil {
		t.Fatalf("logs follow dispatch: %v", err)
	}
	result := out.(map[string]any)
	if result["target"] != "playground" || result["line_count"] != 1 {
		t.Fatalf("unexpected legacy logs follow result: %#v", result)
	}
}

func TestMonitorLogsFollowStreamsTrickLogs(t *testing.T) {
	api := logsFollowAPIServer(t, []string{
		`{"message":{"type":"log","stream":"stderr","service":"worker","line":"job ready"}}`,
	}, 0, func(identifier map[string]any) {
		if identifier["channel"] != "ContainerLogsChannel" || identifier["playground_id"] != float64(42) || identifier["service_name"] != "worker" {
			t.Fatalf("unexpected identifier: %#v", identifier)
		}
	})
	defer api.Close()

	srv := New(Config{APIKey: "pk_test", Domain: api.URL, ToolSet: "full"})
	if err := srv.RegisterAll(); err != nil {
		t.Fatalf("RegisterAll: %v", err)
	}

	out, err := srv.dispatcher.dispatch(context.Background(), "fibe_monitor_logs_follow", map[string]any{
		"id_or_name": "demo",
		"target":     "trick",
		"service":    "worker",
		"max_lines":  1,
		"duration":   "2s",
	})
	if err != nil {
		t.Fatalf("logs follow dispatch: %v", err)
	}
	result := out.(map[string]any)
	if result["target"] != "trick" || result["line_count"] != 1 {
		t.Fatalf("unexpected trick logs follow result: %#v", result)
	}
	lines := result["lines"].([]map[string]string)
	if lines[0]["service"] != "worker" || lines[0]["text"] != "job ready" {
		t.Fatalf("unexpected trick lines: %#v", lines)
	}
}

func TestMonitorLogsFollowStopsAtMaxLines(t *testing.T) {
	api := logsFollowAPIServer(t, []string{
		`{"message":{"type":"log","stream":"stdout","service":"web","line":"first"}}`,
		`{"message":{"type":"log","stream":"stdout","service":"web","line":"second"}}`,
	}, 0, func(identifier map[string]any) {
		if identifier["channel"] != "ContainerLogsChannel" || identifier["service_name"] != "web" {
			t.Fatalf("unexpected identifier: %#v", identifier)
		}
	})
	defer api.Close()

	srv := New(Config{APIKey: "pk_test", Domain: api.URL, ToolSet: "full"})
	if err := srv.RegisterAll(); err != nil {
		t.Fatalf("RegisterAll: %v", err)
	}

	out, err := srv.dispatcher.dispatch(context.Background(), "fibe_monitor_logs_follow", map[string]any{
		"id_or_name": "demo",
		"service":    "web",
		"max_lines":  1,
		"duration":   "2s",
	})
	if err != nil {
		t.Fatalf("logs follow dispatch: %v", err)
	}
	result := out.(map[string]any)
	lines := result["lines"].([]map[string]string)
	if result["line_count"] != 1 || len(lines) != 1 || lines[0]["text"] != "first" {
		t.Fatalf("expected max_lines to stop after first line, got %#v", result)
	}
}

func TestMonitorLogsFollowStopsAfterDuration(t *testing.T) {
	api := logsFollowAPIServer(t, nil, 200*time.Millisecond, func(identifier map[string]any) {
		if identifier["channel"] != "ContainerLogsChannel" || identifier["service_name"] != "web" {
			t.Fatalf("unexpected identifier: %#v", identifier)
		}
	})
	defer api.Close()

	srv := New(Config{APIKey: "pk_test", Domain: api.URL, ToolSet: "full"})
	if err := srv.RegisterAll(); err != nil {
		t.Fatalf("RegisterAll: %v", err)
	}

	out, err := srv.dispatcher.dispatch(context.Background(), "fibe_monitor_logs_follow", map[string]any{
		"id_or_name": "demo",
		"service":    "web",
		"duration":   "20ms",
		"max_lines":  10,
	})
	if err != nil {
		t.Fatalf("logs follow dispatch: %v", err)
	}
	result := out.(map[string]any)
	if result["line_count"] != 0 || result["count"] != 0 {
		t.Fatalf("expected duration stop without events, got %#v", result)
	}
}

func TestMonitorLogsFollowEmitsProgressNotifications(t *testing.T) {
	api := logsFollowAPIServer(t, []string{
		`{"message":{"type":"status","status":"connected"}}`,
		`{"message":{"type":"log","stream":"stdout","service":"web","line":"ready"}}`,
	}, 0, func(identifier map[string]any) {
		if identifier["channel"] != "ContainerLogsChannel" || identifier["service_name"] != "web" {
			t.Fatalf("unexpected identifier: %#v", identifier)
		}
	})
	defer api.Close()

	srv := New(Config{APIKey: "pk_test", Domain: api.URL, ToolSet: "full"})
	if err := srv.RegisterAll(); err != nil {
		t.Fatalf("RegisterAll: %v", err)
	}
	session := &progressTestSession{
		ch:   make(chan mcp.JSONRPCNotification, 4),
		meta: map[string]any{"progressToken": "logs-token"},
	}
	ctx := srv.mcp.WithContext(context.Background(), session)

	out, err := srv.dispatcher.dispatch(ctx, "fibe_monitor_logs_follow", map[string]any{
		"id_or_name": "demo",
		"service":    "web",
		"max_lines":  1,
		"duration":   "2s",
	})
	if err != nil {
		t.Fatalf("logs follow dispatch: %v", err)
	}
	if out.(map[string]any)["line_count"] != 1 {
		t.Fatalf("unexpected logs follow result: %#v", out)
	}

	notifications := collectProgressNotifications(t, session.ch, 2)
	if notifications[0].Method != "notifications/progress" || notifications[1].Method != "notifications/progress" {
		t.Fatalf("unexpected progress methods: %#v", notifications)
	}
	first := notifications[0].Params.AdditionalFields
	second := notifications[1].Params.AdditionalFields
	if first["progressToken"] != "logs-token" || first["message"] != "connected" {
		t.Fatalf("unexpected status progress notification: %#v", first)
	}
	if second["progressToken"] != "logs-token" || second["message"] != "ready" {
		t.Fatalf("unexpected log progress notification: %#v", second)
	}
}

func TestMonitorLogsFollowSchemaAndCatalogExposure(t *testing.T) {
	srv := New(Config{APIKey: "pk_test", ToolSet: "core", PipelineCacheSize: 4})
	if err := srv.RegisterAll(); err != nil {
		t.Fatalf("RegisterAll: %v", err)
	}

	schema := srv.toolSchemas["fibe_monitor_logs_follow"]
	props := schema["properties"].(map[string]any)
	for _, want := range []string{"id_or_name", "target", "service", "tail", "duration", "max_lines"} {
		if _, ok := props[want]; !ok {
			t.Fatalf("monitor logs follow schema missing %q: %#v", want, props)
		}
	}
	targets := schemaPropertyEnum(t, schema, "target")
	if !containsString(targets, "playground") || !containsString(targets, "trick") {
		t.Fatalf("target enum missing playground/trick: %#v", targets)
	}
	if _, ok := srv.toolSchemas["fibe_playgrounds_logs_follow"]; !ok {
		t.Fatalf("legacy playground logs follow schema missing")
	}

	out, err := srv.dispatcher.dispatch(context.Background(), "fibe_tools_catalog", map[string]any{
		"tier":           "core",
		"name_pattern":   "logs_follow",
		"include_schema": true,
	})
	if err != nil {
		t.Fatalf("fibe_tools_catalog: %v", err)
	}
	tools := catalogToolsFromResult(t, out)
	canonical := catalogTool(tools, "fibe_monitor_logs_follow")
	legacy := catalogTool(tools, "fibe_playgrounds_logs_follow")
	if canonical == nil || legacy == nil {
		t.Fatalf("catalog missing logs follow tools: %#v", tools)
	}
	if canonical["advertised"] != true || legacy["advertised"] != true {
		t.Fatalf("logs follow tools should be advertised in core catalog: canonical=%#v legacy=%#v", canonical, legacy)
	}
	inputSchema := canonical["input_schema"].(map[string]any)
	targets = schemaPropertyEnum(t, inputSchema, "target")
	if !containsString(targets, "playground") || !containsString(targets, "trick") {
		t.Fatalf("catalog input schema target enum missing playground/trick: %#v", targets)
	}
}

func logsFollowAPIServer(t *testing.T, frames []string, hold time.Duration, assertIdentifier func(map[string]any)) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/playgrounds/demo":
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 42, "name": "demo", "status": "running"})
		case "/cable":
			conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{Subprotocols: []string{"actioncable-v1-json"}})
			if err != nil {
				t.Fatalf("accept websocket: %v", err)
			}
			defer conn.Close(websocket.StatusNormalClosure, "")
			_, data, err := conn.Read(r.Context())
			if err != nil {
				t.Fatalf("read subscribe: %v", err)
			}
			var body map[string]string
			if err := json.Unmarshal(data, &body); err != nil {
				t.Fatalf("decode subscribe: %v", err)
			}
			var identifier map[string]any
			if err := json.Unmarshal([]byte(body["identifier"]), &identifier); err != nil {
				t.Fatalf("decode identifier: %v", err)
			}
			assertIdentifier(identifier)
			for _, frame := range frames {
				if err := conn.Write(r.Context(), websocket.MessageText, []byte(frame)); err != nil {
					return
				}
			}
			if hold <= 0 {
				hold = 20 * time.Millisecond
			}
			select {
			case <-r.Context().Done():
			case <-time.After(hold):
			}
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
}

type progressTestSession struct {
	ch   chan mcp.JSONRPCNotification
	meta map[string]any
}

func (s *progressTestSession) Initialize() {}

func (s *progressTestSession) Initialized() bool { return true }

func (s *progressTestSession) NotificationChannel() chan<- mcp.JSONRPCNotification {
	return s.ch
}

func (s *progressTestSession) SessionID() string { return "progress-test" }

func (s *progressTestSession) RequestMeta() map[string]any { return s.meta }

func collectProgressNotifications(t *testing.T, ch <-chan mcp.JSONRPCNotification, count int) []mcp.JSONRPCNotification {
	t.Helper()
	notifications := make([]mcp.JSONRPCNotification, 0, count)
	deadline := time.After(time.Second)
	for len(notifications) < count {
		select {
		case notification := <-ch:
			notifications = append(notifications, notification)
		case <-deadline:
			t.Fatalf("timed out waiting for %d progress notifications, got %#v", count, notifications)
		}
	}
	return notifications
}
