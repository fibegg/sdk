package fibe

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestPollAsyncRejectsUnestablishedCompletion(t *testing.T) {
	for _, status := range []any{"future-state", "", nil, 7, map[string]any{"value": "success"}} {
		t.Run(strings.ReplaceAll(strings.TrimSpace(toJSON(status)), "/", "_"), func(t *testing.T) {
			var calls atomic.Int32
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				_ = json.NewEncoder(w).Encode(map[string]any{"request_id": "req-protocol", "status": status, "partial": "kept"})
			}))
			defer srv.Close()
			c := NewClient(WithBaseURL(srv.URL), WithAPIKey("synthetic"), WithMaxRetries(0))
			result, err := c.PollAsync(context.Background(), "/status", nil)
			if err == nil || !strings.Contains(err.Error(), "unexpected async status") || result == nil || result.Status == "success" || result.RequestID != "req-protocol" || result.Payload["partial"] != "kept" || calls.Load() != 1 {
				t.Fatalf("result=%#v error=%v calls=%d", result, err, calls.Load())
			}
		})
	}
}

func toJSON(value any) string { raw, _ := json.Marshal(value); return string(raw) }

func TestPollAsyncMissingEnvelopeStatusAndLegacyPlainPayload(t *testing.T) {
	for _, envelope := range []bool{true, false} {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			payload := map[string]any{"id": 12}
			if envelope {
				payload["request_id"] = "req-without-status"
			}
			_ = json.NewEncoder(w).Encode(payload)
		}))
		c := NewClient(WithBaseURL(srv.URL), WithAPIKey("synthetic"), WithMaxRetries(0))
		result, err := c.PollAsync(context.Background(), "/status", nil)
		srv.Close()
		if envelope && (err == nil || result == nil || result.Status == "success") {
			t.Fatalf("missing status accepted: %#v %v", result, err)
		}
		if !envelope && (err != nil || result.Status != "success" || result.Payload["id"] != float64(12)) {
			t.Fatalf("legacy payload rejected: %#v %v", result, err)
		}
	}
}

func TestDoAsyncUnknownStatusDoesNotResubmitMutation(t *testing.T) {
	var mutations atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			mutations.Add(1)
			w.WriteHeader(http.StatusAccepted)
			_, _ = w.Write([]byte(`{"request_id":"req-one","status":"queued"}`))
			return
		}
		_, _ = w.Write([]byte(`{"request_id":"req-one","status":"future"}`))
	}))
	defer srv.Close()
	c := NewClient(WithBaseURL(srv.URL), WithAPIKey("synthetic"), WithMaxRetries(0))
	if err := c.doAsync(context.Background(), http.MethodPost, "/create", "/status/%s", map[string]any{"name": "test"}, nil); err == nil || mutations.Load() != 1 {
		t.Fatalf("error=%v mutations=%d", err, mutations.Load())
	}
}

func TestPollAsyncDeadlineCancelsBlockedStatusRequest(t *testing.T) {
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case <-release:
		}
	}))
	defer srv.Close()
	defer close(release)
	c := NewClient(WithBaseURL(srv.URL), WithAPIKey("synthetic"), WithMaxRetries(0))
	start := time.Now()
	_, err := c.PollAsync(context.Background(), "/status", &AsyncPollOptions{Timeout: 60 * time.Millisecond})
	if !errors.Is(err, context.DeadlineExceeded) || time.Since(start) > time.Second {
		t.Fatalf("error=%v elapsed=%s", err, time.Since(start))
	}
}
