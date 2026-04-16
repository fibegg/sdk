package fibe

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"testing"
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
