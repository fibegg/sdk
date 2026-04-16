package fibe

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type WebhookPayload struct {
	Event     string         `json:"event"`
	Timestamp string         `json:"timestamp"`
	Data      map[string]any `json:"data"`
}

// VerifyWebhookSignature verifies the HMAC-SHA256 signature on a webhook request.
// It does NOT check payload age — use VerifyWebhookSignatureWithMaxAge for replay protection.
func VerifyWebhookSignature(r *http.Request, secret string) (*WebhookPayload, error) {
	return VerifyWebhookSignatureWithMaxAge(r, secret, 0)
}

// VerifyWebhookSignatureWithMaxAge verifies the signature and optionally rejects
// payloads older than maxAge. This prevents replay attacks where a valid signed
// payload is re-sent after the fact.
//
// Recommended maxAge: 5 * time.Minute (industry standard used by Stripe, GitHub).
func VerifyWebhookSignatureWithMaxAge(r *http.Request, secret string, maxAge time.Duration) (*WebhookPayload, error) {
	signature := r.Header.Get("X-Fibe-Signature")
	if signature == "" {
		return nil, fmt.Errorf("fibe: missing X-Fibe-Signature header")
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1*1024*1024))
	if err != nil {
		return nil, fmt.Errorf("fibe: read body: %w", err)
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(signature), []byte(expected)) {
		return nil, fmt.Errorf("fibe: invalid signature")
	}

	var payload WebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("fibe: decode payload: %w", err)
	}

	if maxAge > 0 && payload.Timestamp != "" {
		ts, err := time.Parse(time.RFC3339, payload.Timestamp)
		if err != nil {
			return nil, fmt.Errorf("fibe: invalid timestamp format: %w", err)
		}
		if time.Since(ts) > maxAge {
			return nil, fmt.Errorf("fibe: payload too old (%s ago, max %s)", time.Since(ts).Round(time.Second), maxAge)
		}
	}

	return &payload, nil
}

// ParseWebhookData extracts the strongly typed struct from the raw WebhookPayload Data map
// depending on the event prefix (e.g., "playground.created" returns a *Playground).
func ParseWebhookData(payload *WebhookPayload) (any, error) {
	dataBytes, err := json.Marshal(payload.Data)
	if err != nil {
		return nil, fmt.Errorf("fibe: marshal webhook data: %w", err)
	}

	prefix := payload.Event
	if dot := strings.Index(prefix, "."); dot != -1 {
		prefix = prefix[:dot]
	}

	switch prefix {
	case "playground":
		var p Playground
		err = json.Unmarshal(dataBytes, &p)
		return &p, err
	case "marquee":
		var m Marquee
		err = json.Unmarshal(dataBytes, &m)
		return &m, err
	case "prop":
		var p Prop
		err = json.Unmarshal(dataBytes, &p)
		return &p, err
	case "playspec":
		var p Playspec
		err = json.Unmarshal(dataBytes, &p)
		return &p, err
	case "agent", "mutter":
		var a Agent
		err = json.Unmarshal(dataBytes, &a)
		return &a, err
	case "template":
		var t ImportTemplate
		err = json.Unmarshal(dataBytes, &t)
		return &t, err
	case "artefact":
		var a Artefact
		err = json.Unmarshal(dataBytes, &a)
		return &a, err
	case "feedback":
		var f Feedback
		err = json.Unmarshal(dataBytes, &f)
		return &f, err
	case "api_key":
		var k APIKey
		err = json.Unmarshal(dataBytes, &k)
		return &k, err
	case "secret":
		var s Secret
		err = json.Unmarshal(dataBytes, &s)
		return &s, err
	case "webhook_endpoint":
		var w WebhookEndpoint
		err = json.Unmarshal(dataBytes, &w)
		return &w, err
	default:
		return payload.Data, nil
	}
}

