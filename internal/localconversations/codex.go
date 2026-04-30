package localconversations

import (
	"context"
	"path/filepath"
)

type CodexStrategy struct{}

func (CodexStrategy) Provider() string { return "codex" }

func (CodexStrategy) Aliases() []string {
	return []string{"codex-cli", "openai-codex", "openai"}
}

func (CodexStrategy) DefaultPaths(homeDir string) []string {
	return []string{
		filepath.Join(homeDir, ".codex", "sessions"),
		filepath.Join(homeDir, ".config", "codex", "sessions"),
		filepath.Join(homeDir, ".local", "share", "codex", "sessions"),
	}
}

func (CodexStrategy) Discover(ctx context.Context, opts DiscoverOptions) ([]Conversation, error) {
	candidates, err := discoverFileCandidates(ctx, opts.HomeDir, opts.Paths, func(path string) bool {
		return hasJSONLExt(path)
	}, opts.Query)
	if err != nil {
		return nil, err
	}

	var conversations []Conversation
	for _, candidate := range candidates {
		conversation, ok, err := parseCodexConversation(candidate.Path)
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

func (CodexStrategy) Get(ctx context.Context, opts DiscoverOptions, uuid string) (*ConversationDetail, bool, error) {
	candidates, err := discoverFileCandidates(ctx, opts.HomeDir, opts.Paths, func(path string) bool {
		return hasJSONLExt(path)
	}, "")
	if err != nil {
		return nil, false, err
	}
	prioritizeCandidates(candidates, uuid)

	var prefixMatch *ConversationDetail
	for _, candidate := range candidates {
		detail, ok, err := parseCodexConversationDetail(candidate.Path)
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

func parseCodexConversation(path string) (Conversation, bool, error) {
	detail, ok, err := parseCodexConversationData(path, false)
	return detail.Conversation, ok, err
}

func parseCodexConversationDetail(path string) (ConversationDetail, bool, error) {
	return parseCodexConversationData(path, true)
}

func parseCodexConversationData(path string, includeDetail bool) (ConversationDetail, bool, error) {
	detail := ConversationDetail{}
	detail.Conversation = Conversation{
		Provider: CodexStrategy{}.Provider(),
		Path:     path,
		UUID:     uuidFromPath(path),
	}
	conversation := &detail.Conversation

	var sawCodex bool
	var fallbackUserMessages []string
	err := readJSONL(path, func(item map[string]any) error {
		itemType := stringValue(item["type"])
		payload := mapValue(item["payload"])
		if payload == nil {
			return nil
		}
		sawCodex = true
		timestamp := parseTimeValue(item["timestamp"])
		updateLastTime(&conversation.LastMessageDate, timestamp)
		if includeDetail {
			detail.RawEvents = append(detail.RawEvents, item)
		}

		payloadType := stringValue(payload["type"])
		if itemType == "session_meta" {
			if id := stringValue(payload["id"]); id != "" {
				conversation.UUID = id
			}
			setMetadataString(conversation, "cwd", payload["cwd"])
			setMetadataString(conversation, "model_provider", payload["model_provider"])
			setMetadataString(conversation, "source", payload["source"])
			setMetadataString(conversation, "cli_version", payload["cli_version"])
			return nil
		}

		if payloadType == "user_message" {
			text := stringValue(payload["message"])
			if text == "" {
				text = extractText(payload["text_elements"])
			}
			if includeDetail {
				detail.Messages = append(detail.Messages, ConversationMessage{
					Role:      "user",
					Type:      payloadType,
					Timestamp: timestamp,
					Text:      text,
					Content:   payload["message"],
					Metadata:  messageMetadata(payload, "images", "local_images", "text_elements"),
				})
			}
			if text != "" {
				conversation.UserMessageCount++
				if conversation.FirstUserMessageSentence == "" {
					conversation.FirstUserMessageSentence = firstSentence(text)
				}
			}
			return nil
		}

		if includeDetail {
			if role := stringValue(payload["role"]); role != "" {
				content := payload["content"]
				detail.Messages = append(detail.Messages, ConversationMessage{
					Role:      role,
					Type:      payloadType,
					Timestamp: timestamp,
					Text:      extractText(content),
					Content:   content,
					Metadata:  messageMetadata(payload, "phase"),
				})
			} else if payloadType == "agent_message" {
				detail.Messages = append(detail.Messages, ConversationMessage{
					Role:      "assistant",
					Type:      payloadType,
					Timestamp: timestamp,
					Text:      stringValue(payload["message"]),
					Content:   payload["message"],
					Metadata:  messageMetadata(payload, "phase"),
				})
			} else if payloadType == "agent_reasoning" {
				detail.Messages = append(detail.Messages, ConversationMessage{
					Role:      "assistant",
					Type:      payloadType,
					Timestamp: timestamp,
					Text:      stringValue(payload["text"]),
					Content:   payload["text"],
				})
			}
		}

		if payloadType == "token_count" {
			if total := codexTokenTotal(payload); total > 0 {
				conversation.TotalTokenCount = total
			}
			return nil
		}

		if conversation.UserMessageCount == 0 && stringValue(payload["role"]) == "user" {
			if text := extractText(payload["content"]); text != "" {
				fallbackUserMessages = append(fallbackUserMessages, text)
			}
		}
		return nil
	})
	if err != nil {
		return ConversationDetail{}, false, err
	}

	if conversation.UserMessageCount == 0 && len(fallbackUserMessages) > 0 {
		conversation.UserMessageCount = len(fallbackUserMessages)
		conversation.FirstUserMessageSentence = firstSentence(fallbackUserMessages[0])
	}

	if conversation.UUID == "" {
		conversation.UUID = uuidFromPath(path)
	}
	if !sawCodex || conversation.UUID == "" {
		return ConversationDetail{}, false, nil
	}
	finalizeConversationDetail(&detail)
	return detail, true, nil
}

func codexTokenTotal(payload map[string]any) int64 {
	info := mapValue(payload["info"])
	if info == nil {
		return 0
	}
	if totalUsage := mapValue(info["total_token_usage"]); totalUsage != nil {
		if total := tokenUsageTotal(totalUsage); total > 0 {
			return total
		}
	}
	return tokenUsageTotal(info)
}
