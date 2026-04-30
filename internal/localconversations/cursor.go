package localconversations

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
)

const localConversationCursorPrefix = "lc1."

type listCursor struct {
	Version             int    `json:"v"`
	Query               string `json:"q,omitempty"`
	HasLastMessageDate  bool   `json:"ht,omitempty"`
	LastMessageUnixNano int64  `json:"lm,omitempty"`
	Provider            string `json:"p,omitempty"`
	UUID                string `json:"u,omitempty"`
	PathHash            string `json:"ph,omitempty"`
}

type conversationSortKey struct {
	hasLastMessageDate  bool
	lastMessageUnixNano int64
	provider            string
	uuid                string
	pathHash            string
}

func newListCursor(query string, conversation Conversation) *listCursor {
	key := sortKeyFromConversation(conversation)
	return &listCursor{
		Version:             1,
		Query:               normalizeListQuery(query),
		HasLastMessageDate:  key.hasLastMessageDate,
		LastMessageUnixNano: key.lastMessageUnixNano,
		Provider:            key.provider,
		UUID:                key.uuid,
		PathHash:            key.pathHash,
	}
}

func encodeListCursor(cursor *listCursor) string {
	if cursor == nil {
		return ""
	}
	data, err := json.Marshal(cursor)
	if err != nil {
		return ""
	}
	return localConversationCursorPrefix + base64.RawURLEncoding.EncodeToString(data)
}

func decodeListCursor(raw string) (*listCursor, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	if !strings.HasPrefix(raw, localConversationCursorPrefix) {
		return nil, fmt.Errorf("invalid local conversations cursor")
	}
	data, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(raw, localConversationCursorPrefix))
	if err != nil {
		return nil, fmt.Errorf("invalid local conversations cursor: %w", err)
	}
	var cursor listCursor
	if err := json.Unmarshal(data, &cursor); err != nil {
		return nil, fmt.Errorf("invalid local conversations cursor: %w", err)
	}
	if cursor.Version != 1 {
		return nil, fmt.Errorf("unsupported local conversations cursor version %d", cursor.Version)
	}
	cursor.Query = normalizeListQuery(cursor.Query)
	return &cursor, nil
}

func normalizeListQuery(query string) string {
	return strings.ToLower(strings.TrimSpace(query))
}

func conversationsAfterCursor(conversations []Conversation, cursor *listCursor) []Conversation {
	if cursor == nil || len(conversations) == 0 {
		return conversations
	}
	out := conversations[:0]
	for _, conversation := range conversations {
		if conversationAfterCursor(conversation, cursor) {
			out = append(out, conversation)
		}
	}
	return out
}

func conversationAfterCursor(conversation Conversation, cursor *listCursor) bool {
	if cursor == nil {
		return true
	}
	return compareConversationSortKeys(sortKeyFromConversation(conversation), cursor.sortKey()) > 0
}

func (cursor *listCursor) sortKey() conversationSortKey {
	if cursor == nil {
		return conversationSortKey{}
	}
	return conversationSortKey{
		hasLastMessageDate:  cursor.HasLastMessageDate,
		lastMessageUnixNano: cursor.LastMessageUnixNano,
		provider:            cursor.Provider,
		uuid:                cursor.UUID,
		pathHash:            cursor.PathHash,
	}
}

func sortKeyFromConversation(conversation Conversation) conversationSortKey {
	key := conversationSortKey{
		provider: strings.TrimSpace(conversation.Provider),
		uuid:     strings.TrimSpace(conversation.UUID),
		pathHash: hashConversationPath(conversation.Path),
	}
	if conversation.LastMessageDate != nil {
		key.hasLastMessageDate = true
		key.lastMessageUnixNano = conversation.LastMessageDate.UTC().UnixNano()
	}
	return key
}

// compareConversationSortKeys returns -1 when left sorts before right in the
// newest-first conversation order, 0 for equal keys, and 1 when left sorts
// after right.
func compareConversationSortKeys(left, right conversationSortKey) int {
	if left.hasLastMessageDate && right.hasLastMessageDate {
		if left.lastMessageUnixNano > right.lastMessageUnixNano {
			return -1
		}
		if left.lastMessageUnixNano < right.lastMessageUnixNano {
			return 1
		}
	} else if left.hasLastMessageDate != right.hasLastMessageDate {
		if left.hasLastMessageDate {
			return -1
		}
		return 1
	}
	if cmp := strings.Compare(left.provider, right.provider); cmp != 0 {
		return cmp
	}
	if cmp := strings.Compare(left.uuid, right.uuid); cmp != 0 {
		return cmp
	}
	if cmp := strings.Compare(left.pathHash, right.pathHash); cmp != 0 {
		return cmp
	}
	return 0
}

func hashConversationPath(path string) string {
	sum := sha256.Sum256([]byte(path))
	return hex.EncodeToString(sum[:])
}
