package localconversations

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const extraPathsEnv = "FIBE_LOCAL_CONVERSATION_PATHS"

// Conversation is the normalized shape exposed by local provider strategies.
// Provider-specific fields should live in Metadata so the core shape can stay
// stable while individual storage formats evolve.
type Conversation struct {
	Provider                 string         `json:"provider" yaml:"provider"`
	Path                     string         `json:"path" yaml:"path"`
	UUID                     string         `json:"uuid" yaml:"uuid"`
	Title                    string         `json:"title,omitempty" yaml:"title,omitempty"`
	Project                  string         `json:"project,omitempty" yaml:"project,omitempty"`
	LastMessageDate          *time.Time     `json:"last_message_date,omitempty" yaml:"last_message_date,omitempty"`
	FirstUserMessageSentence string         `json:"first_user_message_sentence" yaml:"first_user_message_sentence"`
	UserMessageCount         int            `json:"user_message_count" yaml:"user_message_count"`
	TotalTokenCount          int64          `json:"total_token_count" yaml:"total_token_count"`
	Metadata                 map[string]any `json:"metadata,omitempty" yaml:"metadata,omitempty"`
}

type ConversationMessage struct {
	ID                string         `json:"id,omitempty" yaml:"id,omitempty"`
	Position          int            `json:"position,omitempty" yaml:"position,omitempty"`
	Role              string         `json:"role,omitempty" yaml:"role,omitempty"`
	Type              string         `json:"type,omitempty" yaml:"type,omitempty"`
	UUID              string         `json:"uuid,omitempty" yaml:"uuid,omitempty"`
	Timestamp         *time.Time     `json:"timestamp,omitempty" yaml:"timestamp,omitempty"`
	Text              string         `json:"text,omitempty" yaml:"text,omitempty"`
	Content           any            `json:"content,omitempty" yaml:"content,omitempty"`
	ContentOmitted    bool           `json:"content_omitted,omitempty" yaml:"content_omitted,omitempty"`
	TextTruncated     bool           `json:"text_truncated,omitempty" yaml:"text_truncated,omitempty"`
	FullTextCharCount int            `json:"full_text_char_count,omitempty" yaml:"full_text_char_count,omitempty"`
	TokenCount        int64          `json:"token_count,omitempty" yaml:"token_count,omitempty"`
	Metadata          map[string]any `json:"metadata,omitempty" yaml:"metadata,omitempty"`
}

type ConversationDetail struct {
	Conversation `yaml:",inline"`
	Messages     []ConversationMessage `json:"messages,omitempty" yaml:"messages,omitempty"`
	RawEvents    []map[string]any      `json:"raw_events,omitempty" yaml:"raw_events,omitempty"`
}

type Strategy interface {
	Provider() string
	Aliases() []string
	DefaultPaths(homeDir string) []string
	Discover(ctx context.Context, opts DiscoverOptions) ([]Conversation, error)
	Get(ctx context.Context, opts DiscoverOptions, uuid string) (*ConversationDetail, bool, error)
}

type DiscoverOptions struct {
	HomeDir             string
	Paths               []string
	IncludeMetadataOnly bool
	Limit               int
	Query               string
	cursor              *listCursor
}

type ListOptions struct {
	HomeDir             string
	Providers           []string
	Paths               []string
	IncludeMetadataOnly bool
	Limit               int
	Query               string
	Cursor              string
}

type ListPageResult struct {
	Conversations []Conversation
	Count         int
	Limit         int
	Query         string
	HasMore       bool
	NextCursor    string
}

func DefaultStrategies() []Strategy {
	return []Strategy{
		CodexStrategy{},
		ClaudeCodeStrategy{},
		ClaudeDesktopStrategy{},
	}
}

func List(ctx context.Context, opts ListOptions) ([]Conversation, error) {
	page, err := listPage(ctx, opts, false)
	if err != nil {
		return nil, err
	}
	return page.Conversations, nil
}

func ListPage(ctx context.Context, opts ListOptions) (ListPageResult, error) {
	return listPage(ctx, opts, true)
}

func listPage(ctx context.Context, opts ListOptions, includeLookahead bool) (ListPageResult, error) {
	homeDir := opts.HomeDir
	if homeDir == "" {
		var err error
		homeDir, err = os.UserHomeDir()
		if err != nil {
			return ListPageResult{}, fmt.Errorf("resolve home directory: %w", err)
		}
	}

	providerFilter, err := providerSet(opts.Providers, DefaultStrategies())
	if err != nil {
		return ListPageResult{}, err
	}

	cursor, query, err := resolveListCursor(opts)
	if err != nil {
		return ListPageResult{}, err
	}

	extraPaths := append([]string{}, opts.Paths...)
	if envPaths := os.Getenv(extraPathsEnv); envPaths != "" {
		extraPaths = append(extraPaths, filepath.SplitList(envPaths)...)
	}

	discoverLimit := opts.Limit
	if includeLookahead && discoverLimit > 0 {
		discoverLimit++
	}

	var conversations []Conversation
	for _, strategy := range DefaultStrategies() {
		if len(providerFilter) > 0 && !providerFilter[strategy.Provider()] {
			continue
		}

		paths := append([]string{}, strategy.DefaultPaths(homeDir)...)
		paths = append(paths, extraPaths...)
		found, err := strategy.Discover(ctx, DiscoverOptions{
			HomeDir:             homeDir,
			Paths:               paths,
			IncludeMetadataOnly: opts.IncludeMetadataOnly,
			Limit:               discoverLimit,
			Query:               query,
			cursor:              cursor,
		})
		if err != nil {
			return ListPageResult{}, err
		}
		conversations = append(conversations, found...)
	}

	enrichConversations(homeDir, conversations)
	sortConversations(conversations)
	if cursor != nil {
		conversations = conversationsAfterCursor(conversations, cursor)
	}

	hasMore := false
	if opts.Limit > 0 && len(conversations) > opts.Limit {
		hasMore = includeLookahead
		conversations = conversations[:opts.Limit]
	}

	nextCursor := ""
	if hasMore && len(conversations) > 0 {
		nextCursor = encodeListCursor(newListCursor(query, conversations[len(conversations)-1]))
	}

	return ListPageResult{
		Conversations: conversations,
		Count:         len(conversations),
		Limit:         opts.Limit,
		Query:         query,
		HasMore:       hasMore,
		NextCursor:    nextCursor,
	}, nil
}

func Get(ctx context.Context, uuid string, opts ListOptions) (*ConversationDetail, error) {
	uuid = strings.TrimSpace(uuid)
	if uuid == "" {
		return nil, fmt.Errorf("conversation uuid is required")
	}

	homeDir := opts.HomeDir
	if homeDir == "" {
		var err error
		homeDir, err = os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("resolve home directory: %w", err)
		}
	}

	providerFilter, err := providerSet(opts.Providers, DefaultStrategies())
	if err != nil {
		return nil, err
	}

	extraPaths := append([]string{}, opts.Paths...)
	if envPaths := os.Getenv(extraPathsEnv); envPaths != "" {
		extraPaths = append(extraPaths, filepath.SplitList(envPaths)...)
	}

	for _, strategy := range DefaultStrategies() {
		if len(providerFilter) > 0 && !providerFilter[strategy.Provider()] {
			continue
		}

		paths := append([]string{}, strategy.DefaultPaths(homeDir)...)
		paths = append(paths, extraPaths...)
		found, ok, err := strategy.Get(ctx, DiscoverOptions{
			HomeDir:             homeDir,
			Paths:               paths,
			IncludeMetadataOnly: opts.IncludeMetadataOnly,
		}, uuid)
		if err != nil {
			return nil, err
		}
		if ok {
			enrichConversation(&found.Conversation, loadCodexSessionIndexInfo(homeDir), loadClaudeHistoryInfo(homeDir))
			return found, nil
		}
	}

	return nil, fmt.Errorf("no local conversation found for uuid %q", uuid)
}

type conversationHistoryInfo struct {
	Title   string
	Project string
}

func enrichConversations(homeDir string, conversations []Conversation) {
	codexIndex := loadCodexSessionIndexInfo(homeDir)
	claudeHistory := loadClaudeHistoryInfo(homeDir)
	for i := range conversations {
		enrichConversation(&conversations[i], codexIndex, claudeHistory)
	}
}

func enrichConversation(conversation *Conversation, codexIndex, claudeHistory map[string]conversationHistoryInfo) {
	if conversation == nil {
		return
	}
	if info, ok := codexIndex[conversation.UUID]; ok {
		if conversation.Title == "" {
			conversation.Title = info.Title
		}
		if conversation.Project == "" {
			conversation.Project = info.Project
		}
	}
	if info, ok := claudeHistory[conversation.UUID]; ok {
		if conversation.Title == "" {
			conversation.Title = info.Title
		}
		if conversation.Project == "" {
			conversation.Project = info.Project
		}
	}
	if conversation.Title == "" {
		conversation.Title = metadataString(conversation.Metadata, "title")
	}
	if conversation.Project == "" {
		conversation.Project = projectNameFromPath(metadataString(conversation.Metadata, "project"))
	}
	if conversation.Project == "" {
		conversation.Project = projectNameFromPath(metadataString(conversation.Metadata, "cwd"))
	}
	if conversation.Project == "" {
		conversation.Project = projectNameFromPath(metadataString(conversation.Metadata, "origin_cwd"))
	}
}

func loadCodexSessionIndexInfo(homeDir string) map[string]conversationHistoryInfo {
	path := filepath.Join(homeDir, ".codex", "session_index.jsonl")
	infoBySession := map[string]conversationHistoryInfo{}
	err := readJSONL(path, func(item map[string]any) error {
		sessionID := stringValue(item["id"])
		if sessionID == "" {
			return nil
		}
		info := infoBySession[sessionID]
		if info.Title == "" {
			info.Title = strings.TrimSpace(stringValue(item["thread_name"]))
		}
		infoBySession[sessionID] = info
		return nil
	})
	if err != nil && !shouldSkipPathError(err) {
		return map[string]conversationHistoryInfo{}
	}
	return infoBySession
}

func loadClaudeHistoryInfo(homeDir string) map[string]conversationHistoryInfo {
	path := filepath.Join(homeDir, ".claude", "history.jsonl")
	infoBySession := map[string]conversationHistoryInfo{}
	err := readJSONL(path, func(item map[string]any) error {
		sessionID := stringValue(item["sessionId"])
		if sessionID == "" {
			return nil
		}
		info := infoBySession[sessionID]
		if info.Title == "" {
			if title := historyDisplayTitle(stringValue(item["display"])); title != "" {
				info.Title = title
			}
		}
		if info.Project == "" {
			info.Project = projectNameFromPath(stringValue(item["project"]))
		}
		infoBySession[sessionID] = info
		return nil
	})
	if err != nil && !shouldSkipPathError(err) {
		return map[string]conversationHistoryInfo{}
	}
	return infoBySession
}

func historyDisplayTitle(value string) string {
	value = strings.TrimSpace(value)
	if value == "" || strings.HasPrefix(value, "/") || strings.HasPrefix(value, "[Pasted ") {
		return ""
	}
	if idx := strings.IndexByte(value, '\n'); idx >= 0 {
		value = strings.TrimSpace(value[:idx])
	}
	return firstSentence(value)
}

func metadataString(metadata map[string]any, key string) string {
	if metadata == nil {
		return ""
	}
	return strings.TrimSpace(stringValue(metadata[key]))
}

func projectNameFromPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	return filepath.Base(filepath.Clean(path))
}

func MessageByID(detail *ConversationDetail, messageID string) (ConversationMessage, bool) {
	messageID = strings.TrimSpace(messageID)
	if detail == nil || messageID == "" {
		return ConversationMessage{}, false
	}
	for _, message := range detail.Messages {
		if messageIDMatches(message, messageID) {
			return message, true
		}
	}
	return ConversationMessage{}, false
}

func finalizeConversationDetail(detail *ConversationDetail) {
	if detail == nil {
		return
	}
	for i := range detail.Messages {
		message := &detail.Messages[i]
		message.Position = i + 1
		if message.ID == "" {
			if message.UUID != "" {
				message.ID = message.UUID
			} else {
				message.ID = fmt.Sprintf("position:%d", message.Position)
			}
		}
	}
}

func messageIDMatches(message ConversationMessage, messageID string) bool {
	if strings.EqualFold(message.ID, messageID) {
		return true
	}
	if message.UUID != "" && strings.EqualFold(message.UUID, messageID) {
		return true
	}
	if message.Position > 0 {
		position := fmt.Sprintf("%d", message.Position)
		if messageID == position || strings.EqualFold(messageID, "position:"+position) {
			return true
		}
	}
	return false
}

func sortConversations(conversations []Conversation) {
	sort.SliceStable(conversations, func(i, j int) bool {
		return compareConversationSortKeys(sortKeyFromConversation(conversations[i]), sortKeyFromConversation(conversations[j])) < 0
	})
}

func resolveListCursor(opts ListOptions) (*listCursor, string, error) {
	cursor, err := decodeListCursor(opts.Cursor)
	if err != nil {
		return nil, "", err
	}

	query := strings.TrimSpace(opts.Query)
	if cursor == nil {
		return nil, query, nil
	}

	if query == "" {
		return cursor, cursor.Query, nil
	}
	if normalizeListQuery(query) != normalizeListQuery(cursor.Query) {
		return nil, "", fmt.Errorf("cursor query %q does not match query %q", cursor.Query, query)
	}
	return cursor, query, nil
}

func providerSet(filters []string, strategies []Strategy) (map[string]bool, error) {
	if len(filters) == 0 {
		return nil, nil
	}

	aliases := make(map[string]string)
	for _, strategy := range strategies {
		aliases[strategy.Provider()] = strategy.Provider()
		for _, alias := range strategy.Aliases() {
			aliases[alias] = strategy.Provider()
		}
	}

	set := make(map[string]bool)
	for _, raw := range filters {
		for _, part := range strings.Split(raw, ",") {
			key := strings.ToLower(strings.TrimSpace(part))
			if key == "" {
				continue
			}
			if key == "claude" {
				set["claude-code"] = true
				set["claude-desktop"] = true
				continue
			}
			provider, ok := aliases[key]
			if !ok {
				return nil, fmt.Errorf("unknown local conversation provider %q", key)
			}
			set[provider] = true
		}
	}
	return set, nil
}
