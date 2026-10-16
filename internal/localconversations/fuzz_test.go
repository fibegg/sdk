package localconversations

import "testing"

func FuzzConversationIDMatch(f *testing.F) {
	f.Add("019abc-def", "019")
	f.Add("", "")
	f.Fuzz(func(t *testing.T, id, query string) {
		_ = conversationIDMatchScore(id, query)
	})
}
