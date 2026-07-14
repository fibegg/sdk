package fibe

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestVerifyWebhookSignature_Valid(t *testing.T) {
	secret := "test-secret"
	body := []byte(`{"event":"playground.created","timestamp":"2024-01-01T00:00:00Z","data":{}}`)

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	signature := hex.EncodeToString(mac.Sum(nil))

	req, _ := http.NewRequest("POST", "/webhook", bytes.NewReader(body))
	req.Header.Set("X-Fibe-Signature", signature)

	payload, err := VerifyWebhookSignature(req, secret)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if payload.Event != "playground.created" {
		t.Errorf("expected event 'playground.created', got %q", payload.Event)
	}
}

func signedWebhookRequest(t *testing.T, body []byte, secret string) *http.Request {
	t.Helper()
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(body)
	req, err := http.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("X-Fibe-Signature", hex.EncodeToString(mac.Sum(nil)))
	return req
}

func TestVerifyWebhookSignature_Invalid(t *testing.T) {
	body := []byte(`{"event":"test"}`)

	req, _ := http.NewRequest("POST", "/webhook", bytes.NewReader(body))
	req.Header.Set("X-Fibe-Signature", "invalid-signature")

	_, err := VerifyWebhookSignature(req, "secret")
	if err == nil {
		t.Fatal("expected error for invalid signature")
	}
}

func TestVerifyWebhookSignature_MissingHeader(t *testing.T) {
	body := []byte(`{"event":"test"}`)
	req, _ := http.NewRequest("POST", "/webhook", bytes.NewReader(body))

	_, err := VerifyWebhookSignature(req, "secret")
	if err == nil {
		t.Fatal("expected error for missing signature header")
	}
}

func TestVerifyWebhookSignature_RejectsEmptySecret(t *testing.T) {
	req := signedWebhookRequest(t, []byte(`{"event":"test"}`), "")
	if _, err := VerifyWebhookSignature(req, ""); err == nil {
		t.Fatal("expected empty secret error")
	}
}

func TestVerifyWebhookSignature_RejectsMissingRequestBody(t *testing.T) {
	if _, err := VerifyWebhookSignature(nil, "secret"); err == nil || !strings.Contains(err.Error(), "body is missing") {
		t.Fatalf("nil request error = %v", err)
	}
	req := &http.Request{Header: http.Header{"X-Fibe-Signature": []string{strings.Repeat("0", sha256.Size*2)}}}
	if _, err := VerifyWebhookSignature(req, "secret"); err == nil || !strings.Contains(err.Error(), "body is missing") {
		t.Fatalf("nil body error = %v", err)
	}
}

func TestParseWebhookDataRejectsNilPayload(t *testing.T) {
	if _, err := ParseWebhookData(nil); err == nil || !strings.Contains(err.Error(), "payload is nil") {
		t.Fatalf("error = %v", err)
	}
}

func TestVerifyWebhookSignature_RejectsOversizedSignedPrefix(t *testing.T) {
	body := []byte(strings.Repeat("a", int(maxErrorBody)+1))
	req := signedWebhookRequest(t, body[:maxErrorBody], "secret")
	req.Body = io.NopCloser(bytes.NewReader(body))
	if _, err := VerifyWebhookSignature(req, "secret"); err == nil || !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("error = %v, want oversized body error", err)
	}
}

func TestVerifyWebhookSignatureWithMaxAge_RequiresTimestamp(t *testing.T) {
	req := signedWebhookRequest(t, []byte(`{"event":"test","data":{}}`), "secret")
	if _, err := VerifyWebhookSignatureWithMaxAge(req, "secret", time.Minute); err == nil || !strings.Contains(err.Error(), "required") {
		t.Fatalf("error = %v", err)
	}
}

func TestVerifyWebhookSignatureWithMaxAge_RejectsFutureTimestamp(t *testing.T) {
	body := []byte(`{"event":"test","timestamp":"` + time.Now().Add(10*time.Minute).UTC().Format(time.RFC3339) + `","data":{}}`)
	req := signedWebhookRequest(t, body, "secret")
	if _, err := VerifyWebhookSignatureWithMaxAge(req, "secret", time.Hour); err == nil || !strings.Contains(err.Error(), "future") {
		t.Fatalf("error = %v", err)
	}
}
