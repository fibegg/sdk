package fibe

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"testing"
)

func FuzzFlexibleTimeJSON(f *testing.F) {
	for _, seed := range []string{`null`, `"2026-01-02T03:04:05Z"`, `""`, `123`, `{}`} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, raw string) {
		_, _ = parseFlexibleTimeJSON(json.RawMessage(raw))
	})
}

func FuzzWebhookParsing(f *testing.F) {
	f.Add(`{"event":"playground.created","timestamp":"2026-01-02T03:04:05Z","data":{"id":1}}`)
	f.Add(`{}`)
	f.Fuzz(func(t *testing.T, body string) {
		if int64(len(body)) > maxErrorBody+1 {
			t.Skip()
		}
		mac := hmac.New(sha256.New, []byte("fuzz-secret"))
		_, _ = mac.Write([]byte(body))
		req, err := http.NewRequest(http.MethodPost, "https://example.test/webhook", bytes.NewBufferString(body))
		if err != nil {
			t.Skip()
		}
		req.Header.Set("X-Fibe-Signature", hex.EncodeToString(mac.Sum(nil)))
		payload, err := VerifyWebhookSignature(req, "fuzz-secret")
		if err == nil {
			_, _ = ParseWebhookData(payload)
		}
	})
}

func FuzzProjection(f *testing.F) {
	f.Add(`{"id":1,"name":"demo"}`, "id")
	f.Add(`[]`, "missing")
	f.Fuzz(func(t *testing.T, raw, field string) {
		if len(raw) > 1<<20 || len(field) > 1024 {
			t.Skip()
		}
		var value map[string]any
		if json.Unmarshal([]byte(raw), &value) != nil {
			return
		}
		_ = ProjectFields(value, map[string]bool{field: true})
	})
}

func FuzzRetryAfter(f *testing.F) {
	for _, seed := range []string{"0", "15", "-1", "Sun, 06 Nov 1994 08:49:37 GMT", "invalid"} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, value string) {
		response := &http.Response{Header: make(http.Header)}
		response.Header.Set("Retry-After", value)
		_ = parseRetryAfter(response)
	})
}
