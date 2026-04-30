package localconversations

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

type ClaudeCodeStrategy struct{}

func (ClaudeCodeStrategy) Provider() string { return "claude-code" }

func (ClaudeCodeStrategy) Aliases() []string {
	return []string{"claude", "anthropic", "anthropic-claude-code"}
}

func (ClaudeCodeStrategy) DefaultPaths(homeDir string) []string {
	return []string{
		filepath.Join(homeDir, ".claude", "projects"),
		filepath.Join(homeDir, ".config", "claude", "projects"),
		filepath.Join(homeDir, ".local", "share", "claude", "projects"),
	}
}

func (ClaudeCodeStrategy) Discover(ctx context.Context, opts DiscoverOptions) ([]Conversation, error) {
	candidates, err := discoverFileCandidates(ctx, opts.HomeDir, opts.Paths, func(path string) bool {
		return hasJSONLExt(path)
	}, opts.Query)
	if err != nil {
		return nil, err
	}

	var conversations []Conversation
	for _, candidate := range candidates {
		conversation, ok, err := parseClaudeJSONLConversation(candidate.Path, ClaudeCodeStrategy{}.Provider())
		if err != nil || !ok {
			continue
		}
		if !conversationAfterCursor(conversation, opts.cursor) {
			continue
		}
		conversations = append(conversations, conversation)
		if opts.Limit > 0 && len(conversations) >= opts.Limit {
			break
		}
	}
	return conversations, nil
}

func (ClaudeCodeStrategy) Get(ctx context.Context, opts DiscoverOptions, uuid string) (*ConversationDetail, bool, error) {
	return getClaudeJSONLConversation(ctx, opts, uuid, ClaudeCodeStrategy{}.Provider(), func(path string) bool {
		return hasJSONLExt(path)
	})
}

type ClaudeDesktopStrategy struct{}

func (ClaudeDesktopStrategy) Provider() string { return "claude-desktop" }

func (ClaudeDesktopStrategy) Aliases() []string {
	return []string{"desktop", "claude-app"}
}

func (ClaudeDesktopStrategy) DefaultPaths(homeDir string) []string {
	return []string{
		filepath.Join(homeDir, "Library", "Application Support", "Claude", "local-agent-mode-sessions"),
		filepath.Join(homeDir, "Library", "Application Support", "Claude", "claude-code-sessions"),
		filepath.Join(homeDir, ".config", "Claude", "local-agent-mode-sessions"),
		filepath.Join(homeDir, ".config", "Claude", "claude-code-sessions"),
		filepath.Join(homeDir, ".config", "Claude-3p", "local-agent-mode-sessions"),
		filepath.Join(homeDir, ".config", "Claude-3p", "claude-code-sessions"),
		filepath.Join(homeDir, ".config", "claude", "local-agent-mode-sessions"),
		filepath.Join(homeDir, ".config", "claude", "claude-code-sessions"),
		filepath.Join(homeDir, ".var", "app", "com.anthropic.Claude", "config", "Claude", "local-agent-mode-sessions"),
		filepath.Join(homeDir, ".var", "app", "com.anthropic.Claude", "config", "Claude", "claude-code-sessions"),
		filepath.Join(homeDir, ".var", "app", "com.anthropic.Claude", "config", "Claude-3p", "local-agent-mode-sessions"),
		filepath.Join(homeDir, ".var", "app", "com.anthropic.Claude", "config", "Claude-3p", "claude-code-sessions"),
	}
}

func (ClaudeDesktopStrategy) Discover(ctx context.Context, opts DiscoverOptions) ([]Conversation, error) {
	candidates, err := discoverFileCandidates(ctx, opts.HomeDir, opts.Paths, func(path string) bool {
		if hasJSONLExt(path) {
			return strings.Contains(filepath.ToSlash(path), "/.claude/projects/")
		}
		if opts.IncludeMetadataOnly && hasJSONExt(path) {
			base := filepath.Base(path)
			return strings.HasPrefix(base, "local_ditto_") || strings.HasPrefix(base, "local_")
		}
		return false
	}, opts.Query)
	if err != nil {
		return nil, err
	}

	var conversations []Conversation
	for _, candidate := range candidates {
		var conversation Conversation
		if hasJSONLExt(candidate.Path) {
			parsed, ok, err := parseClaudeJSONLConversation(candidate.Path, ClaudeDesktopStrategy{}.Provider())
			if err != nil || !ok {
				continue
			}
			conversation = parsed
		} else if opts.IncludeMetadataOnly {
			parsed, ok, err := parseClaudeDesktopMetadata(candidate.Path)
			if err != nil || !ok {
				continue
			}
			conversation = parsed
		} else {
			continue
		}
		if !conversationAfterCursor(conversation, opts.cursor) {
			continue
		}
		conversations = append(conversations, conversation)
		if opts.Limit > 0 && len(conversations) >= opts.Limit {
			break
		}
	}
	return conversations, nil
}

func (ClaudeDesktopStrategy) Get(ctx context.Context, opts DiscoverOptions, uuid string) (*ConversationDetail, bool, error) {
	return getClaudeJSONLConversation(ctx, opts, uuid, ClaudeDesktopStrategy{}.Provider(), func(path string) bool {
		if hasJSONLExt(path) {
			return strings.Contains(filepath.ToSlash(path), "/.claude/projects/")
		}
		if opts.IncludeMetadataOnly && hasJSONExt(path) {
			base := filepath.Base(path)
			return strings.HasPrefix(base, "local_ditto_") || strings.HasPrefix(base, "local_")
		}
		return false
	})
}

func parseClaudeJSONLConversation(path, provider string) (Conversation, bool, error) {
	detail, ok, err := parseClaudeJSONLConversationData(path, provider, false)
	return detail.Conversation, ok, err
}

func parseClaudeJSONLConversationDetail(path, provider string) (ConversationDetail, bool, error) {
	return parseClaudeJSONLConversationData(path, provider, true)
}

func parseClaudeJSONLConversationData(path, provider string, includeDetail bool) (ConversationDetail, bool, error) {
	detail := ConversationDetail{}
	detail.Conversation = Conversation{
		Provider: provider,
		Path:     path,
		UUID:     uuidFromPath(path),
	}
	if strings.Contains(filepath.ToSlash(path), "/subagents/") {
		detail.UUID = fileStem(path)
	}
	conversation := &detail.Conversation

	var sawClaude bool
	pathUUID := uuidFromPath(path)
	err := readJSONL(path, func(item map[string]any) error {
		sessionID := stringValue(item["sessionId"])
		if sessionID != "" {
			sawClaude = true
			if conversation.UUID == "" || conversation.UUID == pathUUID {
				conversation.UUID = sessionID
			}
		}
		timestamp := parseTimeValue(item["timestamp"])
		updateLastTime(&conversation.LastMessageDate, timestamp)
		if includeDetail {
			detail.RawEvents = append(detail.RawEvents, item)
		}

		setMetadataString(conversation, "cwd", item["cwd"])
		setMetadataString(conversation, "git_branch", item["gitBranch"])
		setMetadataString(conversation, "version", item["version"])
		setMetadataString(conversation, "entrypoint", item["entrypoint"])
		setMetadataBool(conversation, "is_sidechain", item["isSidechain"])
		setMetadataString(conversation, "parent_uuid", item["parentUuid"])

		message := mapValue(item["message"])
		if message == nil {
			return nil
		}

		tokenCount := int64(0)
		if usage := mapValue(message["usage"]); usage != nil {
			tokenCount = tokenUsageTotal(usage)
			conversation.TotalTokenCount += tokenCount
		}

		if includeDetail {
			role := stringValue(message["role"])
			if role == "" {
				role = stringValue(item["type"])
			}
			content := message["content"]
			detail.Messages = append(detail.Messages, ConversationMessage{
				Role:       role,
				Type:       stringValue(item["type"]),
				UUID:       stringValue(item["uuid"]),
				Timestamp:  timestamp,
				Text:       extractText(content),
				Content:    content,
				TokenCount: tokenCount,
				Metadata:   messageMetadata(item, "cwd", "gitBranch", "version", "entrypoint", "parentUuid", "requestId", "userType", "isSidechain"),
			})
		}

		if stringValue(item["type"]) != "user" || stringValue(message["role"]) != "user" {
			return nil
		}
		text := extractText(message["content"])
		if text == "" {
			return nil
		}

		conversation.UserMessageCount++
		if conversation.FirstUserMessageSentence == "" {
			conversation.FirstUserMessageSentence = firstSentence(text)
		}
		return nil
	})
	if err != nil {
		return ConversationDetail{}, false, err
	}

	if conversation.UUID == "" {
		conversation.UUID = uuidFromPath(path)
	}
	if !sawClaude || conversation.UUID == "" {
		return ConversationDetail{}, false, nil
	}
	finalizeConversationDetail(&detail)
	return detail, true, nil
}

func parseClaudeDesktopMetadata(path string) (Conversation, bool, error) {
	detail, ok, err := parseClaudeDesktopMetadataDetail(path)
	return detail.Conversation, ok, err
}

func parseClaudeDesktopMetadataDetail(path string) (ConversationDetail, bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return ConversationDetail{}, false, err
	}
	var item map[string]any
	if err := json.Unmarshal(data, &item); err != nil {
		return ConversationDetail{}, false, err
	}

	sessionID := stringValue(item["sessionId"])
	cliSessionID := stringValue(item["cliSessionId"])
	if sessionID == "" && cliSessionID == "" {
		return ConversationDetail{}, false, nil
	}

	detail := ConversationDetail{
		RawEvents: []map[string]any{item},
	}
	detail.Conversation = Conversation{
		Provider: ClaudeDesktopStrategy{}.Provider(),
		Path:     path,
		UUID:     sessionID,
		Metadata: map[string]any{"metadata_only": true},
	}
	conversation := &detail.Conversation
	if conversation.UUID == "" {
		conversation.UUID = cliSessionID
	}
	updateLastTime(&conversation.LastMessageDate, parseTimeValue(item["lastActivityAt"]))
	if conversation.LastMessageDate == nil {
		updateLastTime(&conversation.LastMessageDate, parseTimeValue(item["createdAt"]))
	}

	if text := stringValue(item["initialMessage"]); text != "" {
		conversation.FirstUserMessageSentence = firstSentence(text)
		conversation.UserMessageCount = 1
		detail.Messages = append(detail.Messages, ConversationMessage{
			Role:      "user",
			Type:      "initial_message",
			Timestamp: parseTimeValue(item["createdAt"]),
			Text:      text,
			Content:   text,
		})
	} else if title := stringValue(item["title"]); title != "" {
		conversation.FirstUserMessageSentence = firstSentence(title)
	}
	if turns := int64Value(item["completedTurns"]); turns > 0 {
		conversation.UserMessageCount = int(turns)
	}

	setMetadataString(conversation, "cli_session_id", cliSessionID)
	setMetadataString(conversation, "cwd", item["cwd"])
	setMetadataString(conversation, "origin_cwd", item["originCwd"])
	setMetadataString(conversation, "model", item["model"])
	setMetadataString(conversation, "effort", item["effort"])
	setMetadataString(conversation, "permission_mode", item["permissionMode"])
	setMetadataString(conversation, "title", item["title"])
	setMetadataString(conversation, "title_source", item["titleSource"])
	setMetadataBool(conversation, "is_archived", item["isArchived"])

	finalizeConversationDetail(&detail)
	return detail, true, nil
}

func getClaudeJSONLConversation(ctx context.Context, opts DiscoverOptions, uuid, provider string, accept func(string) bool) (*ConversationDetail, bool, error) {
	candidates, err := discoverFileCandidates(ctx, opts.HomeDir, opts.Paths, accept, "")
	if err != nil {
		return nil, false, err
	}
	prioritizeCandidates(candidates, uuid)

	var prefixMatch *ConversationDetail
	for _, candidate := range candidates {
		var (
			detail ConversationDetail
			ok     bool
			err    error
		)
		if hasJSONLExt(candidate.Path) {
			detail, ok, err = parseClaudeJSONLConversationDetail(candidate.Path, provider)
		} else {
			detail, ok, err = parseClaudeDesktopMetadataDetail(candidate.Path)
		}
		if err != nil || !ok {
			continue
		}
		score := conversationIDMatchScore(detail.UUID, uuid)
		if score == 2 {
			return &detail, true, nil
		}
		if score == 1 && prefixMatch == nil {
			copy := detail
			prefixMatch = &copy
		}
	}
	if prefixMatch != nil {
		return prefixMatch, true, nil
	}
	return nil, false, nil
}
