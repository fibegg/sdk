package mcpserver

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLocalConversationsToolsAreRegisteredInLocalTier(t *testing.T) {
	srv := New(Config{APIKey: "pk_test", ToolSet: "core", PipelineCacheSize: 4})
	if err := srv.RegisterAll(); err != nil {
		t.Fatalf("RegisterAll: %v", err)
	}

	for _, name := range []string{"fibe_local_conversations_list", "fibe_local_conversations_get", "fibe_local_conversations_get_message"} {
		tool, ok := srv.dispatcher.lookup(name)
		if !ok {
			t.Fatalf("%s not registered", name)
		}
		if tool.tier != tierLocal {
			t.Fatalf("%s tier = %v, want local", name, tool.tier)
		}
		if advertisedToolNames(srv)[name] {
			t.Fatalf("%s should not be advertised in core mode", name)
		}
	}

	full := New(Config{APIKey: "pk_test", ToolSet: "full", PipelineCacheSize: 4})
	if err := full.RegisterAll(); err != nil {
		t.Fatalf("RegisterAll full: %v", err)
	}
	for _, name := range []string{"fibe_local_conversations_list", "fibe_local_conversations_get", "fibe_local_conversations_get_message"} {
		if !advertisedToolNames(full)[name] {
			t.Fatalf("%s should be advertised in full mode", name)
		}
	}
}

func TestLocalConversationsListTool(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("FIBE_LOCAL_CONVERSATION_PATHS", "")
	writeLocalConversationTestFile(t, filepath.Join(home, ".codex", "sessions", "rollout-2026-04-30T10-00-00-12345678-1234-1234-1234-123456789abc.jsonl"), `
{"type":"session_meta","timestamp":"2026-04-30T10:00:00Z","payload":{"id":"codex-session-id","cwd":"/work","source":"cli"}}
{"type":"event_msg","timestamp":"2026-04-30T10:00:02Z","payload":{"type":"user_message","message":"Build native MCP conversation tools. Keep the list fast. zz-fibe-sdk-mcp-test-native-937201"}}
{"type":"event_msg","timestamp":"2026-04-30T10:00:03Z","payload":{"type":"token_count","info":{"total_token_usage":{"input_tokens":10,"output_tokens":5,"total_tokens":15}}}}
`)

	srv := New(Config{APIKey: "pk_test", ToolSet: "full", PipelineCacheSize: 4})
	if err := srv.RegisterAll(); err != nil {
		t.Fatalf("RegisterAll: %v", err)
	}

	out, err := srv.dispatcher.dispatch(context.Background(), "fibe_local_conversations_list", map[string]any{
		"query": "zz-fibe-sdk-mcp-test-native-937201",
		"limit": 1,
	})
	if err != nil {
		t.Fatalf("dispatch list: %v", err)
	}

	res, ok := out.(localConversationsListResponse)
	if !ok {
		t.Fatalf("response type = %T", out)
	}
	if res.Count != 1 {
		t.Fatalf("count = %d", res.Count)
	}
	if res.Conversations[0].UUID != "codex-session-id" {
		t.Fatalf("uuid = %q", res.Conversations[0].UUID)
	}
	if data, err := json.Marshal(res.Conversations[0]); err != nil {
		t.Fatal(err)
	} else if strings.Contains(string(data), "path") {
		t.Fatalf("mcp conversation should not expose path: %s", data)
	}
	if _, ok := res.Conversations[0].Metadata["cwd"]; ok {
		t.Fatalf("mcp conversation should expose project instead of cwd: %#v", res.Conversations[0].Metadata)
	}
	if res.Conversations[0].Project != "work" {
		t.Fatalf("project = %q", res.Conversations[0].Project)
	}
	if res.Conversations[0].TotalTokenCount != 15 {
		t.Fatalf("tokens = %d", res.Conversations[0].TotalTokenCount)
	}
}

func TestLocalConversationsListToolCursorPagination(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("FIBE_LOCAL_CONVERSATION_PATHS", "")
	newPath := filepath.Join(home, ".codex", "sessions", "rollout-2026-04-30T12-00-00-12345678-1234-1234-1234-123456789abc.jsonl")
	middlePath := filepath.Join(home, ".codex", "sessions", "rollout-2026-04-30T11-00-00-22345678-1234-1234-1234-123456789abc.jsonl")
	oldPath := filepath.Join(home, ".codex", "sessions", "rollout-2026-04-30T10-00-00-32345678-1234-1234-1234-123456789abc.jsonl")
	writeLocalConversationTestFile(t, newPath, `
{"type":"session_meta","timestamp":"2026-04-30T12:00:00Z","payload":{"id":"new-session","cwd":"/work"}}
{"type":"event_msg","timestamp":"2026-04-30T12:00:02Z","payload":{"type":"user_message","message":"Newest prompt"}}
`)
	writeLocalConversationTestFile(t, middlePath, `
{"type":"session_meta","timestamp":"2026-04-30T11:00:00Z","payload":{"id":"middle-session","cwd":"/work"}}
{"type":"event_msg","timestamp":"2026-04-30T11:00:02Z","payload":{"type":"user_message","message":"Middle prompt"}}
`)
	writeLocalConversationTestFile(t, oldPath, `
{"type":"session_meta","timestamp":"2026-04-30T10:00:00Z","payload":{"id":"old-session","cwd":"/work"}}
{"type":"event_msg","timestamp":"2026-04-30T10:00:02Z","payload":{"type":"user_message","message":"Old prompt"}}
`)
	setLocalConversationTestFileMTime(t, newPath, time.Date(2026, 4, 30, 12, 0, 0, 0, time.UTC))
	setLocalConversationTestFileMTime(t, middlePath, time.Date(2026, 4, 30, 11, 0, 0, 0, time.UTC))
	setLocalConversationTestFileMTime(t, oldPath, time.Date(2026, 4, 30, 10, 0, 0, 0, time.UTC))

	srv := New(Config{APIKey: "pk_test", ToolSet: "full", PipelineCacheSize: 4})
	if err := srv.RegisterAll(); err != nil {
		t.Fatalf("RegisterAll: %v", err)
	}

	firstOut, err := srv.dispatcher.dispatch(context.Background(), "fibe_local_conversations_list", map[string]any{"limit": 1})
	if err != nil {
		t.Fatalf("dispatch first page: %v", err)
	}
	first := firstOut.(localConversationsListResponse)
	if first.Count != 1 || first.Conversations[0].UUID != "new-session" || !first.HasMore || first.NextCursor == "" || !first.LimitReached {
		t.Fatalf("first page = %#v", first)
	}

	secondOut, err := srv.dispatcher.dispatch(context.Background(), "fibe_local_conversations_list", map[string]any{
		"limit":  1,
		"cursor": first.NextCursor,
	})
	if err != nil {
		t.Fatalf("dispatch second page: %v", err)
	}
	second := secondOut.(localConversationsListResponse)
	if second.Count != 1 || second.Conversations[0].UUID != "middle-session" || !second.HasMore || second.NextCursor == "" {
		t.Fatalf("second page = %#v", second)
	}
}

func TestLocalConversationsGetToolChatAndFullViews(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("FIBE_LOCAL_CONVERSATION_PATHS", "")
	writeLocalConversationTestFile(t, filepath.Join(home, ".codex", "sessions", "rollout-2026-04-30T10-00-00-12345678-1234-1234-1234-123456789abc.jsonl"), `
{"type":"session_meta","timestamp":"2026-04-30T10:00:00Z","payload":{"id":"codex-session-id","cwd":"/work","source":"cli"}}
{"type":"event_msg","timestamp":"2026-04-30T10:00:02Z","payload":{"type":"user_message","message":"Hello long user"}}
{"type":"event_msg","timestamp":"2026-04-30T10:00:03Z","payload":{"type":"agent_message","message":"Hi long assistant!"}}
{"type":"event_msg","timestamp":"2026-04-30T10:00:04Z","payload":{"type":"user_message","message":"Show full data"}}
`)

	srv := New(Config{APIKey: "pk_test", ToolSet: "full", PipelineCacheSize: 4})
	if err := srv.RegisterAll(); err != nil {
		t.Fatalf("RegisterAll: %v", err)
	}

	chatOut, err := srv.dispatcher.dispatch(context.Background(), "fibe_local_conversations_get", map[string]any{
		"uuid":                    "codex-session",
		"view":                    "chat",
		"user_message_limit":      5,
		"assistant_message_limit": 2,
	})
	if err != nil {
		t.Fatalf("dispatch get chat: %v", err)
	}
	chat, ok := chatOut.(localConversationGetResponse)
	if !ok {
		t.Fatalf("chat response type = %T", chatOut)
	}
	if chat.View != "chat" {
		t.Fatalf("view = %q", chat.View)
	}
	if len(chat.Chat) != 2 {
		t.Fatalf("chat turns = %d", len(chat.Chat))
	}
	if chat.Chat[0]["user"] != "Hello" || chat.Chat[0]["codex"] != "Hi" {
		t.Fatalf("first chat turn = %#v", chat.Chat[0])
	}

	userOut, err := srv.dispatcher.dispatch(context.Background(), "fibe_local_conversations_get", map[string]any{
		"uuid":               "codex-session",
		"view":               "user-messages",
		"user_message_limit": 7,
	})
	if err != nil {
		t.Fatalf("dispatch get user messages: %v", err)
	}
	userMessages := userOut.(localConversationGetResponse)
	if len(userMessages.UserMessages) != 2 {
		t.Fatalf("user messages = %d", len(userMessages.UserMessages))
	}
	if userMessages.UserMessages[0].ID != "position:1" || userMessages.UserMessages[0].Text != "Hello l" || !userMessages.UserMessages[0].TextTruncated {
		t.Fatalf("first user message = %#v", userMessages.UserMessages[0])
	}

	fullOut, err := srv.dispatcher.dispatch(context.Background(), "fibe_local_conversations_get", map[string]any{
		"uuid": "codex-session-id",
		"view": "full",
	})
	if err != nil {
		t.Fatalf("dispatch get full: %v", err)
	}
	full := fullOut.(localConversationGetResponse)
	if len(full.Messages) != 3 || len(full.RawEvents) != 4 {
		t.Fatalf("limited full response messages=%d raw=%d", len(full.Messages), len(full.RawEvents))
	}
	if full.Messages[0].ID == "" || full.Messages[0].Position != 1 {
		t.Fatalf("message identity = %#v", full.Messages[0])
	}
	if !full.Messages[0].ContentOmitted {
		t.Fatalf("message content should be omitted in previews: %#v", full.Messages[0])
	}
}

func TestLocalConversationsGetMessageTool(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("FIBE_LOCAL_CONVERSATION_PATHS", "")
	writeLocalConversationTestFile(t, filepath.Join(home, ".codex", "sessions", "rollout-2026-04-30T10-00-00-12345678-1234-1234-1234-123456789abc.jsonl"), `
{"type":"session_meta","timestamp":"2026-04-30T10:00:00Z","payload":{"id":"codex-session-id","cwd":"/work","source":"cli"}}
{"type":"event_msg","timestamp":"2026-04-30T10:00:02Z","payload":{"type":"user_message","message":"Full user message"}}
{"type":"event_msg","timestamp":"2026-04-30T10:00:03Z","payload":{"type":"agent_message","message":"Full assistant message"}}
`)

	srv := New(Config{APIKey: "pk_test", ToolSet: "full", PipelineCacheSize: 4})
	if err := srv.RegisterAll(); err != nil {
		t.Fatalf("RegisterAll: %v", err)
	}

	out, err := srv.dispatcher.dispatch(context.Background(), "fibe_local_conversations_get_message", map[string]any{
		"uuid":       "codex-session-id",
		"message_id": "position:2",
	})
	if err != nil {
		t.Fatalf("dispatch get message: %v", err)
	}
	res := out.(localConversationGetMessageResponse)
	if res.ConversationUUID != "codex-session-id" || res.Role != "assistant" || res.Type != "agent_message" || res.Content != "Full assistant message" {
		t.Fatalf("message response = %#v", res)
	}
	if data, err := json.Marshal(res); err != nil {
		t.Fatal(err)
	} else if strings.Contains(string(data), "text") || strings.Contains(string(data), "message_count") {
		t.Fatalf("get_message response should be minimal: %s", data)
	}
}

func TestLocalConversationToolSchemasAreAgentFocused(t *testing.T) {
	srv := New(Config{APIKey: "pk_test", ToolSet: "full", PipelineCacheSize: 4})
	if err := srv.RegisterAll(); err != nil {
		t.Fatalf("RegisterAll: %v", err)
	}

	listProps := schemaProperties(t, srv.toolSchemas["fibe_local_conversations_list"])
	for _, want := range []string{"query", "limit", "cursor", "only", "output_path"} {
		if _, ok := listProps[want]; !ok {
			t.Fatalf("list schema missing %q: %#v", want, listProps)
		}
	}
	for _, forbidden := range []string{"provider", "providers", "home_dir", "include_metadata_only", "paths", "path", "search_roots", "message_limit", "raw_event_limit"} {
		if _, ok := listProps[forbidden]; ok {
			t.Fatalf("list schema should not expose %q: %#v", forbidden, listProps)
		}
	}

	getProps := schemaProperties(t, srv.toolSchemas["fibe_local_conversations_get"])
	for _, want := range []string{"uuid", "view", "user_message_limit", "assistant_message_limit", "only", "output_path"} {
		if _, ok := getProps[want]; !ok {
			t.Fatalf("get schema missing %q: %#v", want, getProps)
		}
	}
	for _, forbidden := range []string{"provider", "providers", "home_dir", "include_metadata_only", "paths", "path", "search_roots", "message_limit", "raw_event_limit"} {
		if _, ok := getProps[forbidden]; ok {
			t.Fatalf("get schema should not expose %q: %#v", forbidden, getProps)
		}
	}
	views := schemaPropertyEnum(t, srv.toolSchemas["fibe_local_conversations_get"], "view")
	for _, want := range []string{"messages", "chat", "user-messages", "full"} {
		if !containsString(views, want) {
			t.Fatalf("view enum missing %q: %#v", want, views)
		}
	}
	if containsString(views, "summary") {
		t.Fatalf("view enum should not include summary: %#v", views)
	}

	getMessageProps := schemaProperties(t, srv.toolSchemas["fibe_local_conversations_get_message"])
	for _, want := range []string{"uuid", "message_id", "only"} {
		if _, ok := getMessageProps[want]; !ok {
			t.Fatalf("get_message schema missing %q: %#v", want, getMessageProps)
		}
	}
	if _, ok := getMessageProps["search_roots"]; ok {
		t.Fatalf("get_message schema should not expose search_roots: %#v", getMessageProps)
	}
	if _, ok := getMessageProps["output_path"]; ok {
		t.Fatalf("get_message schema should not expose output_path: %#v", getMessageProps)
	}
}

func schemaProperties(t *testing.T, schema map[string]any) map[string]any {
	t.Helper()
	props, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatalf("schema properties is %T", schema["properties"])
	}
	return props
}

func writeLocalConversationTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func setLocalConversationTestFileMTime(t *testing.T, path string, timestamp time.Time) {
	t.Helper()
	if err := os.Chtimes(path, timestamp, timestamp); err != nil {
		t.Fatal(err)
	}
}
