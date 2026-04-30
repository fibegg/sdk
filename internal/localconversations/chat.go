package localconversations

import "strings"

type ChatTurn map[string]string

func ChatTranscript(detail *ConversationDetail) []ChatTurn {
	if detail == nil {
		return nil
	}
	turns := buildChatTranscript(detail, true)
	if len(turns) == 0 {
		turns = buildChatTranscript(detail, false)
	}
	return turns
}

func UserMessages(detail *ConversationDetail) []ConversationMessage {
	if detail == nil {
		return nil
	}
	messages := userMessages(detail, true)
	if len(messages) == 0 {
		messages = userMessages(detail, false)
	}
	return messages
}

func userMessages(detail *ConversationDetail, strict bool) []ConversationMessage {
	var messages []ConversationMessage
	for _, message := range detail.Messages {
		if strings.TrimSpace(message.Text) == "" {
			continue
		}
		if isChatUserMessage(detail.Provider, message, strict) {
			messages = append(messages, message)
		}
	}
	return messages
}

func buildChatTranscript(detail *ConversationDetail, strict bool) []ChatTurn {
	assistantKey := chatAssistantKey(detail.Provider)
	var turns []ChatTurn
	var current ChatTurn

	flush := func() {
		if len(current) == 0 {
			return
		}
		if current["user"] == "" && current[assistantKey] == "" {
			current = nil
			return
		}
		turns = append(turns, current)
		current = nil
	}

	for _, message := range detail.Messages {
		text := strings.TrimSpace(message.Text)
		if text == "" {
			continue
		}

		if isChatUserMessage(detail.Provider, message, strict) {
			flush()
			current = ChatTurn{"user": text}
			continue
		}

		if isChatAssistantMessage(detail.Provider, message, strict) {
			if current == nil {
				current = ChatTurn{}
			}
			appendChatText(current, assistantKey, text)
		}
	}
	flush()

	return turns
}

func isChatUserMessage(provider string, message ConversationMessage, strict bool) bool {
	if message.Role != "user" {
		return false
	}
	if !strict {
		return true
	}
	switch provider {
	case "codex":
		return message.Type == "user_message"
	case "claude-code", "claude-desktop":
		return message.Type == "user" || message.Type == "initial_message"
	default:
		return true
	}
}

func isChatAssistantMessage(provider string, message ConversationMessage, strict bool) bool {
	if message.Role != "assistant" {
		return false
	}
	if !strict {
		return true
	}
	switch provider {
	case "codex":
		return message.Type == "agent_message"
	case "claude-code", "claude-desktop":
		return message.Type == "assistant"
	default:
		return true
	}
}

func appendChatText(turn ChatTurn, key, text string) {
	if existing := turn[key]; existing != "" {
		turn[key] = existing + "\n\n" + text
		return
	}
	turn[key] = text
}

func chatAssistantKey(provider string) string {
	switch provider {
	case "claude-code":
		return "claude_code"
	case "claude-desktop":
		return "claude_desktop"
	default:
		return strings.ReplaceAll(provider, "-", "_")
	}
}
