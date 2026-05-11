package mcpserver

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/fibegg/sdk/fibe"
	"nhooyr.io/websocket"
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
		"id_or_name":           "test-agent",
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

func TestResourceListAgentsPassesIncludeRuntimeStatus(t *testing.T) {
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/agents" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		if got := r.URL.Query().Get("include_runtime_status"); got != "true" {
			t.Fatalf("include_runtime_status = %q", got)
		}
		if got := r.URL.Query().Get("per_page"); got != "100" {
			t.Fatalf("per_page = %q", got)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{
					"id":       1,
					"name":     "agent-1",
					"provider": "openai-codex",
					"runtime_status": map[string]any{
						"status":            "running",
						"runtime_reachable": true,
						"queue_count":       1,
					},
				},
			},
			"meta": map[string]any{"page": 1, "per_page": 100, "total": 1},
		})
	}))
	defer api.Close()

	srv := New(Config{APIKey: "pk_test", Domain: api.URL, ToolSet: "full"})
	if err := srv.RegisterAll(); err != nil {
		t.Fatalf("RegisterAll: %v", err)
	}

	out, err := srv.dispatcher.dispatch(context.Background(), "fibe_resource_list", map[string]any{
		"resource": "agents",
		"params": map[string]any{
			"include_runtime_status": true,
			"per_page":               100,
		},
	})
	if err != nil {
		t.Fatalf("dispatch: %v", err)
	}
	if out == nil {
		t.Fatal("expected list result")
	}
}

func TestAgentAttachmentResourceMutateAndGet(t *testing.T) {
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
			if string(content) != "hello" || header.Filename != "hello.txt" {
				t.Fatalf("unexpected uploaded file %q %q", header.Filename, string(content))
			}
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]any{"filename": "runtime-hello.txt"})
		case 2:
			if r.Method != http.MethodGet || r.URL.Path != "/api/agents/test-agent/uploads/runtime-hello.txt" {
				t.Fatalf("unexpected download request %s %s", r.Method, r.URL.Path)
			}
			if got := r.URL.Query().Get("conversation_id"); got != "thread-1" {
				t.Fatalf("download conversation_id = %q", got)
			}
			w.Header().Set("Content-Type", "text/plain")
			w.Header().Set("Content-Disposition", `inline; filename="runtime-hello.txt"`)
			_, _ = w.Write([]byte("hello"))
		default:
			t.Fatalf("unexpected extra request %d: %s %s", step, r.Method, r.URL.String())
		}
	}))
	defer api.Close()

	srv := New(Config{APIKey: "pk_test", Domain: api.URL, ToolSet: "full"})
	if err := srv.RegisterAll(); err != nil {
		t.Fatalf("RegisterAll: %v", err)
	}

	upload, err := srv.dispatcher.dispatch(context.Background(), "fibe_resource_mutate", map[string]any{
		"resource":  "agent",
		"operation": "upload_attachment",
		"payload": map[string]any{
			"id_or_name":      "test-agent",
			"content_base64":  "aGVsbG8=",
			"filename":        "hello.txt",
			"conversation_id": "thread-1",
		},
	})
	if err != nil {
		t.Fatalf("upload dispatch: %v", err)
	}
	if upload.(*fibe.AgentUploadResult).Filename != "runtime-hello.txt" {
		t.Fatalf("unexpected upload result: %#v", upload)
	}

	download, err := srv.dispatcher.dispatch(context.Background(), "fibe_resource_get", map[string]any{
		"resource":         "agent_attachment",
		"agent_id_or_name": "test-agent",
		"filename":         "runtime-hello.txt",
		"conversation_id":  "thread-1",
	})
	if err != nil {
		t.Fatalf("download dispatch: %v", err)
	}
	result := download.(map[string]any)
	if result["content_base64"] != "aGVsbG8=" || result["content_type"] != "text/plain" || result["size"] != 5 {
		t.Fatalf("unexpected download result: %#v", result)
	}
	if step != 2 {
		t.Fatalf("expected 2 requests, got %d", step)
	}
}

func TestResourceWatchAgentsUsesAnyCable(t *testing.T) {
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/cable" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		if !strings.Contains(r.Header.Get("Sec-WebSocket-Protocol"), "fibe-api-key.") {
			t.Fatalf("missing api key subprotocol: %s", r.Header.Get("Sec-WebSocket-Protocol"))
		}
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
		var identifier map[string]string
		if err := json.Unmarshal([]byte(body["identifier"]), &identifier); err != nil {
			t.Fatalf("decode identifier: %v", err)
		}
		if identifier["channel"] != "ApiResourceChannel" || identifier["resource"] != "Agent" {
			t.Fatalf("unexpected identifier: %#v", identifier)
		}
		_ = conn.Write(r.Context(), websocket.MessageText, []byte(`{"type":"confirm_subscription"}`))
		_ = conn.Write(r.Context(), websocket.MessageText, []byte(`{"message":{"event":"updated","id":1}}`))
	}))
	defer api.Close()

	srv := New(Config{APIKey: "pk_test", Domain: api.URL, ToolSet: "full"})
	if err := srv.RegisterAll(); err != nil {
		t.Fatalf("RegisterAll: %v", err)
	}

	out, err := srv.dispatcher.dispatch(context.Background(), "fibe_resource_watch", map[string]any{
		"resource":   "agents",
		"max_events": 1,
		"duration":   "2s",
	})
	if err != nil {
		t.Fatalf("watch dispatch: %v", err)
	}
	result := out.(map[string]any)
	if result["count"] != 1 {
		t.Fatalf("unexpected watch result: %#v", result)
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
		"id_or_name":      "test-agent",
		"conversation_id": "thread-1",
		"title":           "Project One",
	}); err != nil {
		t.Fatalf("create dispatch: %v", err)
	}
	if _, err := srv.dispatcher.dispatch(context.Background(), "fibe_agents_live_state", map[string]any{
		"id_or_name":      "test-agent",
		"conversation_id": "thread-1",
	}); err != nil {
		t.Fatalf("live dispatch: %v", err)
	}
	if _, err := srv.dispatcher.dispatch(context.Background(), "fibe_agents_interrupt", map[string]any{
		"id_or_name":      "test-agent",
		"conversation_id": "thread-1",
	}); err != nil {
		t.Fatalf("interrupt dispatch: %v", err)
	}
	if _, err := srv.dispatcher.dispatch(context.Background(), "fibe_agents_delete_conversation", map[string]any{
		"id_or_name":      "test-agent",
		"conversation_id": "thread-1",
		"confirm":         true,
	}); err != nil {
		t.Fatalf("delete dispatch: %v", err)
	}
	if step != 4 {
		t.Fatalf("expected 4 requests, got %d", step)
	}
}
