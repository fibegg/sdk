package mcpserver

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestResponseShapeOnlyProjectsObject(t *testing.T) {
	shape, err := parseResponseShape(map[string]any{"only": []any{"id", "name"}})
	if err != nil {
		t.Fatalf("parse shape: %v", err)
	}
	out, err := applyResponseShape(map[string]any{
		"id":     1,
		"name":   "demo",
		"status": "running",
	}, shape)
	if err != nil {
		t.Fatalf("apply shape: %v", err)
	}
	m := out.(map[string]any)
	if m["id"] != float64(1) || m["name"] != "demo" {
		t.Fatalf("projected fields = %#v", m)
	}
	if _, ok := m["status"]; ok {
		t.Fatalf("status should be omitted: %#v", m)
	}
}

func TestResponseShapeOnlyProjectsArrayItems(t *testing.T) {
	shape, err := parseResponseShape(map[string]any{"only": "id,status"})
	if err != nil {
		t.Fatalf("parse shape: %v", err)
	}
	out, err := applyResponseShape([]any{
		map[string]any{"id": 1, "name": "a", "status": "running"},
		map[string]any{"id": 2, "name": "b", "status": "pending"},
	}, shape)
	if err != nil {
		t.Fatalf("apply shape: %v", err)
	}
	rows := out.([]any)
	first := rows[0].(map[string]any)
	if first["id"] != float64(1) || first["status"] != "running" {
		t.Fatalf("first row = %#v", first)
	}
	if _, ok := first["name"]; ok {
		t.Fatalf("name should be omitted: %#v", first)
	}
}

func TestResponseShapeOnlyPreservesListEnvelopeMetadata(t *testing.T) {
	shape, err := parseResponseShape(map[string]any{"only": []string{"id"}})
	if err != nil {
		t.Fatalf("parse shape: %v", err)
	}
	out, err := applyResponseShape(map[string]any{
		"data": []any{
			map[string]any{"id": 1, "name": "a"},
		},
		"meta": map[string]any{"total": 1},
	}, shape)
	if err != nil {
		t.Fatalf("apply shape: %v", err)
	}
	envelope := out.(map[string]any)
	if _, ok := envelope["meta"]; !ok {
		t.Fatalf("meta should be preserved: %#v", envelope)
	}
	row := envelope["data"].([]any)[0].(map[string]any)
	if row["id"] != float64(1) || len(row) != 1 {
		t.Fatalf("row = %#v", row)
	}
}

func TestResponseShapeOnlyProjectsSingleArrayEnvelope(t *testing.T) {
	shape, err := parseResponseShape(map[string]any{"only": []any{"uuid"}})
	if err != nil {
		t.Fatalf("parse shape: %v", err)
	}
	out, err := applyResponseShape(map[string]any{
		"conversations": []any{
			map[string]any{"uuid": "abc", "path": "/tmp/session.jsonl"},
		},
		"count": 1,
	}, shape)
	if err != nil {
		t.Fatalf("apply shape: %v", err)
	}
	envelope := out.(map[string]any)
	if envelope["count"] != float64(1) {
		t.Fatalf("count should be preserved: %#v", envelope)
	}
	row := envelope["conversations"].([]any)[0].(map[string]any)
	if row["uuid"] != "abc" || len(row) != 1 {
		t.Fatalf("conversation row = %#v", row)
	}
}

func TestResponseShapeOutputPathRunsBeforeOnly(t *testing.T) {
	shape, err := parseResponseShape(map[string]any{
		"output_path": "$.data",
		"only":        []any{"id"},
	})
	if err != nil {
		t.Fatalf("parse shape: %v", err)
	}
	out, err := applyResponseShape(map[string]any{
		"data": []any{
			map[string]any{"id": 1, "name": "a"},
		},
		"meta": map[string]any{"total": 1},
	}, shape)
	if err != nil {
		t.Fatalf("apply shape: %v", err)
	}
	rows := out.([]any)
	row := rows[0].(map[string]any)
	if row["id"] != float64(1) || len(row) != 1 {
		t.Fatalf("row = %#v", row)
	}
}

func TestResponseShapeBadOutputPathReturnsError(t *testing.T) {
	shape, err := parseResponseShape(map[string]any{"output_path": "$.missing"})
	if err != nil {
		t.Fatalf("parse shape: %v", err)
	}
	_, err = applyResponseShape(map[string]any{"id": 1}, shape)
	if err == nil || !strings.Contains(err.Error(), "output_path") {
		t.Fatalf("expected output_path error, got %v", err)
	}
}

func TestDispatcherResponseShapeLocalConversationsListOnly(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("FIBE_LOCAL_CONVERSATION_PATHS", "")
	writeLocalConversationTestFile(t, home+"/.codex/sessions/rollout-2026-04-30T10-00-00-12345678-1234-1234-1234-123456789abc.jsonl", `
{"type":"session_meta","timestamp":"2026-04-30T10:00:00Z","payload":{"id":"codex-session-id","cwd":"/work","source":"cli"}}
{"type":"event_msg","timestamp":"2026-04-30T10:00:02Z","payload":{"type":"user_message","message":"Build response shaping. zz-fibe-sdk-response-shape-819251"}}
`)

	srv := New(Config{APIKey: "pk_test", ToolSet: "full", PipelineCacheSize: 4})
	if err := srv.RegisterAll(); err != nil {
		t.Fatalf("RegisterAll: %v", err)
	}
	out, err := srv.dispatcher.dispatch(context.Background(), "fibe_local_conversations_list", map[string]any{
		"query": "zz-fibe-sdk-response-shape-819251",
		"only":  []any{"uuid"},
	})
	if err != nil {
		t.Fatalf("dispatch: %v", err)
	}
	envelope := out.(map[string]any)
	row := envelope["conversations"].([]any)[0].(map[string]any)
	if row["uuid"] != "codex-session-id" || len(row) != 1 {
		t.Fatalf("conversation row = %#v", row)
	}
	if _, ok := envelope["count"]; !ok {
		t.Fatalf("count should be preserved: %#v", envelope)
	}
}

func TestDispatcherResponseShapeLocalConversationsGetOutputPath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("FIBE_LOCAL_CONVERSATION_PATHS", "")
	writeLocalConversationTestFile(t, home+"/.codex/sessions/rollout-2026-04-30T10-00-00-12345678-1234-1234-1234-123456789abc.jsonl", `
{"type":"session_meta","timestamp":"2026-04-30T10:00:00Z","payload":{"id":"codex-session-id","cwd":"/work","source":"cli"}}
{"type":"event_msg","timestamp":"2026-04-30T10:00:02Z","payload":{"type":"user_message","message":"Build response shaping."}}
`)

	srv := New(Config{APIKey: "pk_test", ToolSet: "full", PipelineCacheSize: 4})
	if err := srv.RegisterAll(); err != nil {
		t.Fatalf("RegisterAll: %v", err)
	}
	out, err := srv.dispatcher.dispatch(context.Background(), "fibe_local_conversations_get", map[string]any{
		"uuid":        "codex-session-id",
		"output_path": "$.conversation.uuid",
	})
	if err != nil {
		t.Fatalf("dispatch: %v", err)
	}
	if out != "codex-session-id" {
		t.Fatalf("output path result = %#v", out)
	}
}

func TestDispatcherResponseShapeGenericResourceListDoesNotRejectOnly(t *testing.T) {
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/playgrounds" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{"id": 1, "name": "pg", "status": "running"},
			},
			"meta": map[string]any{"page": 1, "per_page": 25, "total": 1},
		})
	}))
	t.Cleanup(api.Close)

	srv := New(Config{APIKey: "pk_test", Domain: api.URL, ToolSet: "full", PipelineCacheSize: 4})
	if err := srv.RegisterAll(); err != nil {
		t.Fatalf("RegisterAll: %v", err)
	}
	out, err := srv.dispatcher.dispatch(context.Background(), "fibe_resource_list", map[string]any{
		"resource": "playground",
		"params":   map[string]any{"per_page": 1},
		"only":     []any{"id", "name"},
	})
	if err != nil {
		t.Fatalf("dispatch: %v", err)
	}
	envelope := out.(map[string]any)
	if _, ok := envelope["Meta"]; !ok {
		t.Fatalf("Meta should be preserved: %#v", envelope)
	}
	row := envelope["Data"].([]any)[0].(map[string]any)
	if row["id"] != float64(1) || row["name"] != "pg" {
		t.Fatalf("row = %#v", row)
	}
	if _, ok := row["status"]; ok {
		t.Fatalf("status should be omitted: %#v", row)
	}
}

func TestResponseShapeToolSchemasAdvertiseOnlyAndOutputPath(t *testing.T) {
	srv := New(Config{APIKey: "pk_test", ToolSet: "full", PipelineCacheSize: 4})
	if err := srv.RegisterAll(); err != nil {
		t.Fatalf("RegisterAll: %v", err)
	}

	schema := srv.toolSchemas["fibe_resource_list"]
	props := schema["properties"].(map[string]any)
	only := props["only"].(map[string]any)
	if only["type"] != "array" {
		t.Fatalf("only schema = %#v", only)
	}
	if _, ok := props["fields"]; ok {
		t.Fatalf("fields alias should not be advertised: %#v", props["fields"])
	}
	outputPath := props["output_path"].(map[string]any)
	if outputPath["type"] != "string" {
		t.Fatalf("output_path schema = %#v", outputPath)
	}
}
