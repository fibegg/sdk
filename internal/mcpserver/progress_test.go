package mcpserver

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fibegg/sdk/fibe"
	"github.com/mark3labs/mcp-go/mcp"
)

func TestAsyncSDKProgressEmitsMCPNotifications(t *testing.T) {
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.EscapedPath() {
		case "/api/marquees/3/ssh_keys":
			if r.Method != http.MethodPost {
				t.Fatalf("unexpected method %s", r.Method)
			}
			w.WriteHeader(http.StatusAccepted)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"request_id": "req-ssh",
				"status":     "queued",
			})
		case "/api/async_requests/req-ssh":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"request_id": "req-ssh",
				"status":     "success",
				"public_key": "ssh-rsa AAAATEST",
			})
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.EscapedPath())
		}
	}))
	defer api.Close()

	srv := New(Config{APIKey: "pk_test", Domain: api.URL, ToolSet: "full"})
	if err := srv.RegisterAll(); err != nil {
		t.Fatalf("RegisterAll: %v", err)
	}
	session := &progressTestSession{
		ch:   make(chan mcp.JSONRPCNotification, 4),
		meta: map[string]any{"progressToken": "async-token"},
	}
	ctx := srv.mcp.WithContext(context.Background(), session)

	out, err := srv.dispatcher.dispatch(ctx, "fibe_resource_mutate", map[string]any{
		"resource":  "marquee",
		"operation": "generate_ssh_key",
		"payload":   map[string]any{"id_or_name": "3"},
	})
	if err != nil {
		t.Fatalf("resource mutate: %v", err)
	}
	result := out.(*fibe.SSHKeyResult)
	if result.PublicKey != "ssh-rsa AAAATEST" {
		t.Fatalf("unexpected result: %#v", result)
	}

	notifications := collectProgressNotifications(t, session.ch, 2)
	first := notifications[0].Params.AdditionalFields
	second := notifications[1].Params.AdditionalFields
	if first["progressToken"] != "async-token" || first["message"] != "async req-ssh: queued" {
		t.Fatalf("unexpected accepted progress notification: %#v", first)
	}
	if second["progressToken"] != "async-token" || second["message"] != "async req-ssh: success" {
		t.Fatalf("unexpected final progress notification: %#v", second)
	}
}
