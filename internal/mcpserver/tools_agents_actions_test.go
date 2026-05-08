package mcpserver

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestAgentSendMessageToolUploadsAttachmentsAndPassesConversationControls(t *testing.T) {
	tmp, err := os.CreateTemp(t.TempDir(), "agent-attachment-*.txt")
	if err != nil {
		t.Fatalf("CreateTemp: %v", err)
	}
	if _, err := tmp.WriteString("hello"); err != nil {
		t.Fatalf("write temp: %v", err)
	}
	if err := tmp.Close(); err != nil {
		t.Fatalf("close temp: %v", err)
	}

	step := 0
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		step++
		switch step {
		case 1:
			if r.Method != http.MethodPost || r.URL.Path != "/api/agents/test-agent/uploads" {
				t.Fatalf("unexpected upload request %s %s", r.Method, r.URL.Path)
			}
			if err := r.ParseMultipartForm(8 << 20); err != nil {
				t.Fatalf("ParseMultipartForm: %v", err)
			}
			if got := r.FormValue("conversation_id"); got != "thread-1" {
				t.Fatalf("upload conversation_id = %q", got)
			}
			file, header, err := r.FormFile("file")
			if err != nil {
				t.Fatalf("FormFile: %v", err)
			}
			defer file.Close()
			content, err := io.ReadAll(file)
			if err != nil {
				t.Fatalf("ReadAll: %v", err)
			}
			if string(content) != "hello" || header.Filename == "" {
				t.Fatalf("unexpected uploaded file %q %q", header.Filename, string(content))
			}
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]any{"filename": "uploaded.txt"})
		case 2:
			if r.Method != http.MethodPost || r.URL.Path != "/api/agents/test-agent/chat" {
				t.Fatalf("unexpected chat request %s %s", r.Method, r.URL.Path)
			}
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode chat body: %v", err)
			}
			if body["text"] != "hello" || body["conversation_id"] != "thread-1" || body["busy_policy"] != "queue" {
				t.Fatalf("unexpected chat body: %#v", body)
			}
			attachments, ok := body["attachmentFilenames"].([]any)
			if !ok || len(attachments) != 2 || attachments[0] != "uploaded.txt" || attachments[1] != "manual.txt" {
				t.Fatalf("unexpected attachment filenames: %#v", body["attachmentFilenames"])
			}
			w.WriteHeader(http.StatusAccepted)
			_ = json.NewEncoder(w).Encode(map[string]any{"status": "accepted"})
		default:
			t.Fatalf("unexpected extra request %d: %s %s", step, r.Method, r.URL.String())
		}
	}))
	defer api.Close()

	srv := New(Config{APIKey: "pk_test", Domain: api.URL, ToolSet: "full"})
	if err := srv.RegisterAll(); err != nil {
		t.Fatalf("RegisterAll: %v", err)
	}

	out, err := srv.dispatcher.dispatch(context.Background(), "fibe_agents_send_message", map[string]any{
		"agent_id":             "test-agent",
		"text":                 "hello",
		"conversation_id":      "thread-1",
		"busy_policy":          "queue",
		"attachment_paths":     []any{tmp.Name()},
		"attachment_filenames": []any{"manual.txt"},
	})
	if err != nil {
		t.Fatalf("dispatch: %v", err)
	}
	result := out.(map[string]any)
	if result["status"] != "accepted" || step != 2 {
		t.Fatalf("unexpected result=%#v step=%d", result, step)
	}
}

func TestAgentConversationToolsDispatchScopedRequests(t *testing.T) {
	step := 0
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		step++
		switch step {
		case 1:
			if r.Method != http.MethodPost || r.URL.Path != "/api/agents/test-agent/conversations" {
				t.Fatalf("unexpected create request %s %s", r.Method, r.URL.Path)
			}
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode create body: %v", err)
			}
			if body["conversation_id"] != "thread-1" || body["title"] != "Project One" {
				t.Fatalf("unexpected create body: %#v", body)
			}
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]any{"id": "thread-1"})
		case 2:
			if r.Method != http.MethodGet || r.URL.Path != "/api/agents/test-agent/live_state" {
				t.Fatalf("unexpected live request %s %s", r.Method, r.URL.Path)
			}
			if got := r.URL.Query().Get("conversation_id"); got != "thread-1" {
				t.Fatalf("live conversation_id = %q", got)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"content": map[string]any{"conversationId": "thread-1", "isProcessing": true}})
		case 3:
			if r.Method != http.MethodPost || r.URL.Path != "/api/agents/test-agent/interrupt" {
				t.Fatalf("unexpected interrupt request %s %s", r.Method, r.URL.Path)
			}
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode interrupt body: %v", err)
			}
			if body["conversation_id"] != "thread-1" {
				t.Fatalf("unexpected interrupt body: %#v", body)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"interrupted": true})
		case 4:
			if r.Method != http.MethodDelete || r.URL.Path != "/api/agents/test-agent/conversations" {
				t.Fatalf("unexpected delete request %s %s", r.Method, r.URL.Path)
			}
			if got := r.URL.Query().Get("conversation_id"); got != "thread-1" {
				t.Fatalf("delete conversation_id = %q", got)
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected extra request %d: %s %s", step, r.Method, r.URL.String())
		}
	}))
	defer api.Close()

	srv := New(Config{APIKey: "pk_test", Domain: api.URL, ToolSet: "full"})
	if err := srv.RegisterAll(); err != nil {
		t.Fatalf("RegisterAll: %v", err)
	}

	if _, err := srv.dispatcher.dispatch(context.Background(), "fibe_agents_create_conversation", map[string]any{
		"agent_id":        "test-agent",
		"conversation_id": "thread-1",
		"title":           "Project One",
	}); err != nil {
		t.Fatalf("create dispatch: %v", err)
	}
	if _, err := srv.dispatcher.dispatch(context.Background(), "fibe_agents_live_state", map[string]any{
		"agent_id":        "test-agent",
		"conversation_id": "thread-1",
	}); err != nil {
		t.Fatalf("live dispatch: %v", err)
	}
	if _, err := srv.dispatcher.dispatch(context.Background(), "fibe_agents_interrupt", map[string]any{
		"agent_id":        "test-agent",
		"conversation_id": "thread-1",
	}); err != nil {
		t.Fatalf("interrupt dispatch: %v", err)
	}
	if _, err := srv.dispatcher.dispatch(context.Background(), "fibe_agents_delete_conversation", map[string]any{
		"agent_id":        "test-agent",
		"conversation_id": "thread-1",
		"confirm":         true,
	}); err != nil {
		t.Fatalf("delete dispatch: %v", err)
	}
	if step != 4 {
		t.Fatalf("expected 4 requests, got %d", step)
	}
}
