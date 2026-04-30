package localconversations

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestParseCodexConversation(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "rollout-2026-04-30T10-00-00-12345678-1234-1234-1234-123456789abc.jsonl")
	writeFile(t, path, `
{"type":"session_meta","timestamp":"2026-04-30T10:00:00Z","payload":{"id":"codex-session-id","cwd":"/work","source":"cli"}}
{"type":"response_item","timestamp":"2026-04-30T10:00:01Z","payload":{"type":"message","role":"user","content":[{"type":"text","text":"repo instructions that should not count"}]}}
{"type":"event_msg","timestamp":"2026-04-30T10:00:02Z","payload":{"type":"user_message","message":"Build the local conversation list. Keep it simple."}}
{"type":"event_msg","timestamp":"2026-04-30T10:00:03Z","payload":{"type":"token_count","info":{"total_token_usage":{"input_tokens":10,"output_tokens":5,"total_tokens":15}}}}
{"type":"event_msg","timestamp":"2026-04-30T10:05:02Z","payload":{"type":"user_message","message":"Add tests too"}}
{"type":"event_msg","timestamp":"2026-04-30T10:05:03Z","payload":{"type":"token_count","info":{"total_token_usage":{"input_tokens":20,"output_tokens":7,"total_tokens":27}}}}
`)

	conversation, ok, err := parseCodexConversation(path)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if !ok {
		t.Fatal("expected codex conversation")
	}
	if conversation.UUID != "codex-session-id" {
		t.Fatalf("uuid = %q", conversation.UUID)
	}
	if conversation.UserMessageCount != 2 {
		t.Fatalf("user count = %d", conversation.UserMessageCount)
	}
	if conversation.FirstUserMessageSentence != "Build the local conversation list." {
		t.Fatalf("first sentence = %q", conversation.FirstUserMessageSentence)
	}
	if conversation.TotalTokenCount != 27 {
		t.Fatalf("tokens = %d", conversation.TotalTokenCount)
	}
	if conversation.LastMessageDate == nil || conversation.LastMessageDate.Format("15:04:05") != "10:05:03" {
		t.Fatalf("last date = %v", conversation.LastMessageDate)
	}
}

func TestParseClaudeJSONLConversation(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee.jsonl")
	writeFile(t, path, `
{"type":"user","timestamp":"2026-04-30T10:00:00.000Z","sessionId":"claude-session-id","cwd":"/work","message":{"role":"user","content":"Create the command. Then verify it."}}
{"type":"assistant","timestamp":"2026-04-30T10:00:05.000Z","sessionId":"claude-session-id","message":{"role":"assistant","content":[],"usage":{"input_tokens":10,"cache_creation_input_tokens":2,"cache_read_input_tokens":3,"output_tokens":5}}}
{"type":"user","timestamp":"2026-04-30T10:00:06.000Z","sessionId":"claude-session-id","message":{"role":"user","content":[{"type":"tool_result","content":"not a user prompt"}]}}
{"type":"user","timestamp":"2026-04-30T10:01:00.000Z","sessionId":"claude-session-id","message":{"role":"user","content":[{"type":"text","text":"Add tests"}]}}
`)

	conversation, ok, err := parseClaudeJSONLConversation(path, "claude-code")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if !ok {
		t.Fatal("expected claude conversation")
	}
	if conversation.UUID != "claude-session-id" {
		t.Fatalf("uuid = %q", conversation.UUID)
	}
	if conversation.UserMessageCount != 2 {
		t.Fatalf("user count = %d", conversation.UserMessageCount)
	}
	if conversation.FirstUserMessageSentence != "Create the command." {
		t.Fatalf("first sentence = %q", conversation.FirstUserMessageSentence)
	}
	if conversation.TotalTokenCount != 20 {
		t.Fatalf("tokens = %d", conversation.TotalTokenCount)
	}
}

func TestListFiltersProvidersAndSorts(t *testing.T) {
	home := t.TempDir()
	codexDir := filepath.Join(home, ".codex", "sessions")
	claudeDir := filepath.Join(home, ".claude", "projects", "-tmp")
	if err := os.MkdirAll(codexDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(claudeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(codexDir, "rollout-2026-04-30T10-00-00-12345678-1234-1234-1234-123456789abc.jsonl"), `
{"type":"session_meta","timestamp":"2026-04-30T10:00:00Z","payload":{"id":"codex-session-id"}}
{"type":"event_msg","timestamp":"2026-04-30T10:00:02Z","payload":{"type":"user_message","message":"Codex prompt"}}
`)
	writeFile(t, filepath.Join(claudeDir, "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb.jsonl"), `
{"type":"user","timestamp":"2026-04-30T11:00:00Z","sessionId":"claude-session-id","message":{"role":"user","content":"Claude prompt"}}
`)

	conversations, err := List(context.Background(), ListOptions{HomeDir: home, Providers: []string{"claude"}})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(conversations) != 1 {
		t.Fatalf("len = %d", len(conversations))
	}
	if conversations[0].Provider != "claude-code" {
		t.Fatalf("provider = %q", conversations[0].Provider)
	}
}

func TestListSearchesAllProvidersByDefault(t *testing.T) {
	home := t.TempDir()
	writeFile(t, filepath.Join(home, ".codex", "sessions", "rollout-2026-04-30T10-00-00-12345678-1234-1234-1234-123456789abc.jsonl"), `
{"type":"session_meta","timestamp":"2026-04-30T10:00:00Z","payload":{"id":"codex-session-id"}}
{"type":"event_msg","timestamp":"2026-04-30T10:00:02Z","payload":{"type":"user_message","message":"Codex prompt"}}
`)
	writeFile(t, filepath.Join(home, ".claude", "projects", "-tmp", "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb.jsonl"), `
{"type":"user","timestamp":"2026-04-30T11:00:00Z","sessionId":"claude-session-id","message":{"role":"user","content":"Claude prompt"}}
`)

	conversations, err := List(context.Background(), ListOptions{HomeDir: home})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(conversations) != 2 {
		t.Fatalf("len = %d", len(conversations))
	}
	providers := []string{conversations[0].Provider, conversations[1].Provider}
	if !contains(providers, "codex") || !contains(providers, "claude-code") {
		t.Fatalf("providers = %#v", providers)
	}
}

func TestListEnrichesClaudeTitleAndProjectFromHistory(t *testing.T) {
	home := t.TempDir()
	writeFile(t, filepath.Join(home, ".claude", "projects", "-tmp", "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb.jsonl"), `
{"type":"user","timestamp":"2026-04-30T11:00:00Z","sessionId":"claude-session-id","cwd":"/Users/vvsk/play","message":{"role":"user","content":"Long first user message that should not become the title."}}
`)
	writeFile(t, filepath.Join(home, ".claude", "history.jsonl"), `
{"display":"/resume","project":"/Users/vvsk/play/fibe","sessionId":"claude-session-id"}
{"display":"Fix tutorial marquee invariant","project":"/Users/vvsk/play/fibe","sessionId":"claude-session-id"}
{"display":"later follow-up should not replace title","project":"/Users/vvsk/play/fibe","sessionId":"claude-session-id"}
`)

	conversations, err := List(context.Background(), ListOptions{HomeDir: home, Providers: []string{"claude"}})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(conversations) != 1 {
		t.Fatalf("len = %d", len(conversations))
	}
	if conversations[0].Title != "Fix tutorial marquee invariant" {
		t.Fatalf("title = %q", conversations[0].Title)
	}
	if conversations[0].Project != "fibe" {
		t.Fatalf("project = %q", conversations[0].Project)
	}
}

func TestListEnrichesCodexTitleFromSessionIndex(t *testing.T) {
	home := t.TempDir()
	writeFile(t, filepath.Join(home, ".codex", "sessions", "rollout-2026-04-30T10-00-00-12345678-1234-1234-1234-123456789abc.jsonl"), `
{"type":"session_meta","timestamp":"2026-04-30T10:00:00Z","payload":{"id":"codex-session-id","cwd":"/Users/vvsk/play/sdk"}}
{"type":"event_msg","timestamp":"2026-04-30T10:00:02Z","payload":{"type":"user_message","message":"Long first user message that should not become the title."}}
`)
	writeFile(t, filepath.Join(home, ".codex", "session_index.jsonl"), `
{"id":"codex-session-id","thread_name":"Add local conversations list","updated_at":"2026-04-30T10:10:00Z"}
`)

	conversations, err := List(context.Background(), ListOptions{HomeDir: home, Providers: []string{"codex"}})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(conversations) != 1 {
		t.Fatalf("len = %d", len(conversations))
	}
	if conversations[0].Title != "Add local conversations list" {
		t.Fatalf("title = %q", conversations[0].Title)
	}
	if conversations[0].Project != "sdk" {
		t.Fatalf("project = %q", conversations[0].Project)
	}
}

func TestListLimitParsesNewestCandidatesFirst(t *testing.T) {
	home := t.TempDir()
	codexDir := filepath.Join(home, ".codex", "sessions")
	newPath := filepath.Join(codexDir, "rollout-2026-04-30T11-00-00-bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb.jsonl")
	oldPath := filepath.Join(codexDir, "rollout-2026-04-30T10-00-00-aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa.jsonl")
	writeFile(t, oldPath, `
{"type":"session_meta","timestamp":"2026-04-30T10:00:00Z","payload":{"id":"old-session"}}
{"type":"event_msg","timestamp":"2026-04-30T10:00:01Z","payload":{"type":"user_message","message":"Old prompt"}}
`)
	writeFile(t, newPath, `
{"type":"session_meta","timestamp":"2026-04-30T11:00:00Z","payload":{"id":"new-session"}}
{"type":"event_msg","timestamp":"2026-04-30T11:00:01Z","payload":{"type":"user_message","message":"New prompt"}}
`)
	oldTime := time.Date(2026, 4, 30, 10, 0, 0, 0, time.UTC)
	newTime := time.Date(2026, 4, 30, 11, 0, 0, 0, time.UTC)
	if err := os.Chtimes(oldPath, oldTime, oldTime); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(newPath, newTime, newTime); err != nil {
		t.Fatal(err)
	}

	conversations, err := List(context.Background(), ListOptions{HomeDir: home, Providers: []string{"codex"}, Limit: 1})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(conversations) != 1 {
		t.Fatalf("len = %d", len(conversations))
	}
	if conversations[0].UUID != "new-session" {
		t.Fatalf("uuid = %q", conversations[0].UUID)
	}
}

func TestListPageCursorPagination(t *testing.T) {
	home := t.TempDir()
	codexDir := filepath.Join(home, ".codex", "sessions")
	newPath := filepath.Join(codexDir, "rollout-2026-04-30T12-00-00-bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb.jsonl")
	middlePath := filepath.Join(codexDir, "rollout-2026-04-30T11-00-00-aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa.jsonl")
	oldPath := filepath.Join(codexDir, "rollout-2026-04-30T10-00-00-cccccccc-cccc-cccc-cccc-cccccccccccc.jsonl")
	writeFile(t, newPath, `
{"type":"session_meta","timestamp":"2026-04-30T12:00:00Z","payload":{"id":"new-session"}}
{"type":"event_msg","timestamp":"2026-04-30T12:00:01Z","payload":{"type":"user_message","message":"New prompt"}}
`)
	writeFile(t, middlePath, `
{"type":"session_meta","timestamp":"2026-04-30T11:00:00Z","payload":{"id":"middle-session"}}
{"type":"event_msg","timestamp":"2026-04-30T11:00:01Z","payload":{"type":"user_message","message":"Middle prompt"}}
`)
	writeFile(t, oldPath, `
{"type":"session_meta","timestamp":"2026-04-30T10:00:00Z","payload":{"id":"old-session"}}
{"type":"event_msg","timestamp":"2026-04-30T10:00:01Z","payload":{"type":"user_message","message":"Old prompt"}}
`)
	setMTime(t, newPath, time.Date(2026, 4, 30, 12, 0, 0, 0, time.UTC))
	setMTime(t, middlePath, time.Date(2026, 4, 30, 11, 0, 0, 0, time.UTC))
	setMTime(t, oldPath, time.Date(2026, 4, 30, 10, 0, 0, 0, time.UTC))

	first, err := ListPage(context.Background(), ListOptions{HomeDir: home, Providers: []string{"codex"}, Limit: 1})
	if err != nil {
		t.Fatalf("first page: %v", err)
	}
	if first.Count != 1 || first.Conversations[0].UUID != "new-session" || !first.HasMore || first.NextCursor == "" {
		t.Fatalf("first page = %#v", first)
	}

	second, err := ListPage(context.Background(), ListOptions{HomeDir: home, Providers: []string{"codex"}, Limit: 1, Cursor: first.NextCursor})
	if err != nil {
		t.Fatalf("second page: %v", err)
	}
	if second.Count != 1 || second.Conversations[0].UUID != "middle-session" || !second.HasMore || second.NextCursor == "" {
		t.Fatalf("second page = %#v", second)
	}

	third, err := ListPage(context.Background(), ListOptions{HomeDir: home, Providers: []string{"codex"}, Limit: 1, Cursor: second.NextCursor})
	if err != nil {
		t.Fatalf("third page: %v", err)
	}
	if third.Count != 1 || third.Conversations[0].UUID != "old-session" || third.HasMore || third.NextCursor != "" {
		t.Fatalf("third page = %#v", third)
	}
}

func TestListPageCursorReusesAndValidatesQuery(t *testing.T) {
	home := t.TempDir()
	codexDir := filepath.Join(home, ".codex", "sessions")
	newPath := filepath.Join(codexDir, "rollout-2026-04-30T12-00-00-bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb.jsonl")
	middlePath := filepath.Join(codexDir, "rollout-2026-04-30T11-00-00-aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa.jsonl")
	oldPath := filepath.Join(codexDir, "rollout-2026-04-30T10-00-00-cccccccc-cccc-cccc-cccc-cccccccccccc.jsonl")
	writeFile(t, newPath, `
{"type":"session_meta","timestamp":"2026-04-30T12:00:00Z","payload":{"id":"new-session"}}
{"type":"event_msg","timestamp":"2026-04-30T12:00:01Z","payload":{"type":"user_message","message":"Needle prompt one"}}
`)
	writeFile(t, middlePath, `
{"type":"session_meta","timestamp":"2026-04-30T11:00:00Z","payload":{"id":"middle-session"}}
{"type":"event_msg","timestamp":"2026-04-30T11:00:01Z","payload":{"type":"user_message","message":"Needle prompt two"}}
`)
	writeFile(t, oldPath, `
{"type":"session_meta","timestamp":"2026-04-30T10:00:00Z","payload":{"id":"old-session"}}
{"type":"event_msg","timestamp":"2026-04-30T10:00:01Z","payload":{"type":"user_message","message":"Other prompt"}}
`)
	setMTime(t, newPath, time.Date(2026, 4, 30, 12, 0, 0, 0, time.UTC))
	setMTime(t, middlePath, time.Date(2026, 4, 30, 11, 0, 0, 0, time.UTC))
	setMTime(t, oldPath, time.Date(2026, 4, 30, 10, 0, 0, 0, time.UTC))

	first, err := ListPage(context.Background(), ListOptions{HomeDir: home, Providers: []string{"codex"}, Query: "Needle", Limit: 1})
	if err != nil {
		t.Fatalf("first page: %v", err)
	}
	if first.Count != 1 || first.Conversations[0].UUID != "new-session" || !first.HasMore || first.NextCursor == "" {
		t.Fatalf("first page = %#v", first)
	}

	second, err := ListPage(context.Background(), ListOptions{HomeDir: home, Providers: []string{"codex"}, Limit: 1, Cursor: first.NextCursor})
	if err != nil {
		t.Fatalf("second page: %v", err)
	}
	if second.Query != "needle" || second.Count != 1 || second.Conversations[0].UUID != "middle-session" || second.HasMore {
		t.Fatalf("second page = %#v", second)
	}

	if _, err := ListPage(context.Background(), ListOptions{HomeDir: home, Providers: []string{"codex"}, Query: "other", Limit: 1, Cursor: first.NextCursor}); err == nil || !strings.Contains(err.Error(), "does not match query") {
		t.Fatalf("expected query mismatch error, got %v", err)
	}
}

func TestGetCodexConversationDetail(t *testing.T) {
	home := t.TempDir()
	path := filepath.Join(home, ".codex", "sessions", "rollout-2026-04-30T10-00-00-12345678-1234-1234-1234-123456789abc.jsonl")
	writeFile(t, path, `
{"type":"session_meta","timestamp":"2026-04-30T10:00:00Z","payload":{"id":"codex-session-id","cwd":"/work","source":"cli"}}
{"type":"event_msg","timestamp":"2026-04-30T10:00:02Z","payload":{"type":"user_message","message":"Build detail output. Include full content."}}
{"type":"response_item","timestamp":"2026-04-30T10:00:03Z","payload":{"type":"message","role":"assistant","content":[{"type":"text","text":"Done"}]}}
`)

	detail, err := Get(context.Background(), "codex-session-id", ListOptions{
		HomeDir:   home,
		Providers: []string{"codex"},
		Paths:     []string{filepath.Join(home, "missing")},
	})
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if detail.UUID != "codex-session-id" {
		t.Fatalf("uuid = %q", detail.UUID)
	}
	if len(detail.Messages) != 2 {
		t.Fatalf("messages len = %d", len(detail.Messages))
	}
	if detail.Messages[0].ID != "position:1" || detail.Messages[0].Position != 1 {
		t.Fatalf("first message identity = %#v", detail.Messages[0])
	}
	message, ok := MessageByID(detail, "1")
	if !ok || message.ID != "position:1" {
		t.Fatalf("MessageByID = %#v ok=%v", message, ok)
	}
	if detail.Messages[0].Text != "Build detail output. Include full content." {
		t.Fatalf("message text = %q", detail.Messages[0].Text)
	}
	if len(detail.RawEvents) != 3 {
		t.Fatalf("raw events len = %d", len(detail.RawEvents))
	}
}

func TestSearchRootsWorkForListAndGet(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "nested", "rollout-2026-04-30T10-00-00-12345678-1234-1234-1234-123456789abc.jsonl")
	writeFile(t, path, `
{"type":"session_meta","timestamp":"2026-04-30T10:00:00Z","payload":{"id":"codex-search-root-id"}}
{"type":"event_msg","timestamp":"2026-04-30T10:00:02Z","payload":{"type":"user_message","message":"Search root prompt"}}
`)

	conversations, err := List(context.Background(), ListOptions{HomeDir: t.TempDir(), Paths: []string{root}})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(conversations) != 1 || conversations[0].UUID != "codex-search-root-id" {
		t.Fatalf("conversations = %#v", conversations)
	}

	detail, err := Get(context.Background(), "codex-search-root", ListOptions{HomeDir: t.TempDir(), Paths: []string{root}})
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if detail.UUID != "codex-search-root-id" {
		t.Fatalf("uuid = %q", detail.UUID)
	}
}

func TestListQueryUsesRipgrepPrefilter(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake rg shell script is POSIX-only")
	}
	home := t.TempDir()
	path := filepath.Join(home, ".codex", "sessions", "rollout-2026-04-30T10-00-00-12345678-1234-1234-1234-123456789abc.jsonl")
	writeFile(t, path, `
{"type":"session_meta","timestamp":"2026-04-30T10:00:00Z","payload":{"id":"rg-session-id"}}
{"type":"event_msg","timestamp":"2026-04-30T10:00:02Z","payload":{"type":"user_message","message":"Needle only fake ripgrep returns this file."}}
`)

	binDir := t.TempDir()
	rgPath := filepath.Join(binDir, "rg")
	writeFile(t, rgPath, "#!/bin/sh\nprintf '%s\\n' \"$FIBE_TEST_RG_MATCH\"\n")
	if err := os.Chmod(rgPath, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("FIBE_TEST_RG_MATCH", path)
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	conversations, err := List(context.Background(), ListOptions{
		HomeDir:   home,
		Providers: []string{"codex"},
		Query:     "needle",
	})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(conversations) != 1 || conversations[0].UUID != "rg-session-id" {
		t.Fatalf("conversations = %#v", conversations)
	}
}

func TestListQueryFallsBackWhenRipgrepIsUnavailable(t *testing.T) {
	home := t.TempDir()
	writeFile(t, filepath.Join(home, ".codex", "sessions", "rollout-2026-04-30T10-00-00-12345678-1234-1234-1234-123456789abc.jsonl"), `
{"type":"session_meta","timestamp":"2026-04-30T10:00:00Z","payload":{"id":"fallback-session-id"}}
{"type":"event_msg","timestamp":"2026-04-30T10:00:02Z","payload":{"type":"user_message","message":"This raw body contains the fallback-only phrase."}}
`)
	t.Setenv("PATH", t.TempDir())

	conversations, err := List(context.Background(), ListOptions{
		HomeDir:   home,
		Providers: []string{"codex"},
		Query:     "fallback-only phrase",
	})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(conversations) != 1 || conversations[0].UUID != "fallback-session-id" {
		t.Fatalf("conversations = %#v", conversations)
	}
}

func TestListQueryMatchesCandidatePath(t *testing.T) {
	home := t.TempDir()
	writeFile(t, filepath.Join(home, ".codex", "sessions", "path-match-2026-04-30T10-00-00-12345678-1234-1234-1234-123456789abc.jsonl"), `
{"type":"session_meta","timestamp":"2026-04-30T10:00:00Z","payload":{"id":"path-session-id"}}
{"type":"event_msg","timestamp":"2026-04-30T10:00:02Z","payload":{"type":"user_message","message":"Prompt without the query term."}}
`)

	conversations, err := List(context.Background(), ListOptions{
		HomeDir:   home,
		Providers: []string{"codex"},
		Query:     "path-match",
	})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(conversations) != 1 || conversations[0].UUID != "path-session-id" {
		t.Fatalf("conversations = %#v", conversations)
	}
}

func TestChatTranscriptFoldsUserAndAssistantTurns(t *testing.T) {
	detail := &ConversationDetail{
		Conversation: Conversation{Provider: "codex"},
		Messages: []ConversationMessage{
			{Role: "developer", Type: "message", Text: "ignore developer setup"},
			{Role: "user", Type: "message", Text: "ignore injected user context"},
			{Role: "user", Type: "user_message", Text: "Hello"},
			{Role: "assistant", Type: "agent_message", Text: "Hi!"},
			{Role: "assistant", Type: "message", Text: "Hi!"},
			{Role: "assistant", Type: "reasoning", Text: "ignore reasoning"},
			{Role: "user", Type: "user_message", Text: "Next"},
			{Role: "assistant", Type: "agent_message", Text: "Answer 1"},
			{Role: "assistant", Type: "agent_message", Text: "Answer 2"},
		},
	}

	turns := ChatTranscript(detail)
	if len(turns) != 2 {
		t.Fatalf("turns len = %d", len(turns))
	}
	if turns[0]["user"] != "Hello" || turns[0]["codex"] != "Hi!" {
		t.Fatalf("first turn = %#v", turns[0])
	}
	if turns[1]["user"] != "Next" || turns[1]["codex"] != "Answer 1\n\nAnswer 2" {
		t.Fatalf("second turn = %#v", turns[1])
	}

	userMessages := UserMessages(detail)
	if len(userMessages) != 2 || userMessages[0].Text != "Hello" || userMessages[1].Text != "Next" {
		t.Fatalf("user messages = %#v", userMessages)
	}
}

func TestMissingAndPermissionDeniedPathsAreSkipped(t *testing.T) {
	home := t.TempDir()
	paths := []string{filepath.Join(home, "missing")}
	if runtime.GOOS != "windows" {
		locked := filepath.Join(home, "locked")
		if err := os.MkdirAll(locked, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(locked, 0); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.Chmod(locked, 0o755) })
		paths = append(paths, locked)
	}

	conversations, err := List(context.Background(), ListOptions{HomeDir: home, Providers: []string{"codex"}, Paths: paths})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(conversations) != 0 {
		t.Fatalf("len = %d", len(conversations))
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func setMTime(t *testing.T, path string, timestamp time.Time) {
	t.Helper()
	if err := os.Chtimes(path, timestamp, timestamp); err != nil {
		t.Fatal(err)
	}
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if strings.EqualFold(value, want) {
			return true
		}
	}
	return false
}
