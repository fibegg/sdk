package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestAgentChatCommandMapsPositionalMessage(t *testing.T) {
	setupAuthTest(t)

	var body map[string]any
	srv := agentChatTestServer(t, &body)
	defer srv.Close()

	t.Setenv("FIBE_DOMAIN", srv.URL)
	t.Setenv("FIBE_API_KEY", "pk_test")

	cmd := RootCmd()
	cmd.SetArgs([]string{"agent", "chat", "builder", "hello"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	if body["text"] != "hello" {
		t.Fatalf("body text = %#v, want hello", body["text"])
	}
}

func TestAgentSendMessageAliasKeepsTextFlag(t *testing.T) {
	setupAuthTest(t)

	var body map[string]any
	srv := agentChatTestServer(t, &body)
	defer srv.Close()

	t.Setenv("FIBE_DOMAIN", srv.URL)
	t.Setenv("FIBE_API_KEY", "pk_test")

	cmd := RootCmd()
	cmd.SetArgs([]string{"agents", "send-message", "builder", "--text", "old form still works"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	if body["text"] != "old form still works" {
		t.Fatalf("body text = %#v, want old form still works", body["text"])
	}
}

func TestAgentChatReadsMessageFromStdin(t *testing.T) {
	setupAuthTest(t)

	var body map[string]any
	srv := agentChatTestServer(t, &body)
	defer srv.Close()

	t.Setenv("FIBE_DOMAIN", srv.URL)
	t.Setenv("FIBE_API_KEY", "pk_test")

	oldStdin := os.Stdin
	read, write, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	if _, err := write.WriteString("from stdin\n"); err != nil {
		t.Fatalf("write stdin: %v", err)
	}
	if err := write.Close(); err != nil {
		t.Fatalf("close stdin writer: %v", err)
	}
	os.Stdin = read
	t.Cleanup(func() {
		os.Stdin = oldStdin
		_ = read.Close()
	})

	cmd := RootCmd()
	cmd.SetArgs([]string{"agent", "chat", "builder", "-"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	if body["text"] != "from stdin\n" {
		t.Fatalf("body text = %#v, want stdin text", body["text"])
	}
}

func TestAgentChatRejectsTextFlagAndPositionalText(t *testing.T) {
	setupAuthTest(t)

	cmd := RootCmd()
	cmd.SetArgs([]string{"agent", "chat", "builder", "hello", "--text", "also hello"})
	err := cmd.Execute()
	if err == nil {
		t.Fatalf("execute succeeded, want error")
	}
	if got := err.Error(); got != "pass message either as positional text or --text, not both" {
		t.Fatalf("error = %q", got)
	}
}

func agentChatTestServer(t *testing.T, body *map[string]any) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.EscapedPath() != "/api/agents/builder/messages" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.EscapedPath())
		}
		if err := json.NewDecoder(r.Body).Decode(body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"status": "accepted"})
	}))
}
