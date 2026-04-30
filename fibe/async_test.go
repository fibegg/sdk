package fibe

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestPollAsync_ImmediateSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"request_id": "req-123",
			"status":     "success",
			"data":       "hello",
		})
	}))
	defer srv.Close()

	c := NewClient(WithBaseURL(srv.URL), WithAPIKey("test"), WithMaxRetries(0))
	result, err := c.PollAsync(context.Background(), "/status/req-123", &AsyncPollOptions{
		Interval: 50 * time.Millisecond,
		Timeout:  1 * time.Second,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "success" {
		t.Fatalf("expected status=success, got %q", result.Status)
	}
	if result.Payload["data"] != "hello" {
		t.Fatalf("expected payload.data=hello, got %v", result.Payload["data"])
	}
}

func TestPollAsync_PendingThenSuccess(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := calls.Add(1)
		if n <= 2 {
			json.NewEncoder(w).Encode(map[string]any{
				"request_id": "req-456",
				"status":     "running",
			})
			return
		}
		json.NewEncoder(w).Encode(map[string]any{
			"request_id": "req-456",
			"status":     "success",
			"result":     "done",
		})
	}))
	defer srv.Close()

	c := NewClient(WithBaseURL(srv.URL), WithAPIKey("test"), WithMaxRetries(0))
	result, err := c.PollAsync(context.Background(), "/status/req-456", &AsyncPollOptions{
		Interval: 50 * time.Millisecond,
		Timeout:  5 * time.Second,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "success" {
		t.Fatalf("expected success, got %q", result.Status)
	}
	if calls.Load() < 3 {
		t.Fatalf("expected at least 3 poll calls, got %d", calls.Load())
	}
}

func TestPollAsync_Error(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"request_id": "req-789",
			"status":     "error",
			"error":      "connection refused",
		})
	}))
	defer srv.Close()

	c := NewClient(WithBaseURL(srv.URL), WithAPIKey("test"), WithMaxRetries(0))
	result, err := c.PollAsync(context.Background(), "/status/req-789", &AsyncPollOptions{
		Interval: 50 * time.Millisecond,
		Timeout:  1 * time.Second,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "error" {
		t.Fatalf("expected error status, got %q", result.Status)
	}
	if result.Error != "connection refused" {
		t.Fatalf("expected error message 'connection refused', got %q", result.Error)
	}
}

func TestPollAsync_Legacy422AsyncErrorPayload(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		json.NewEncoder(w).Encode(map[string]any{
			"request_id": "req-422",
			"status":     "error",
			"error":      "remote command failed",
		})
	}))
	defer srv.Close()

	c := NewClient(WithBaseURL(srv.URL), WithAPIKey("test"), WithMaxRetries(0))
	result, err := c.PollAsync(context.Background(), "/status/req-422", &AsyncPollOptions{
		Interval: 50 * time.Millisecond,
		Timeout:  1 * time.Second,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "error" || result.Error != "remote command failed" {
		t.Fatalf("expected async error result, got %#v", result)
	}
}

func TestPollAsync_Legacy422APIErrorPayload(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{
				"code":    "REMOTE_REQUEST_FAILED",
				"message": "connection refused",
			},
		})
	}))
	defer srv.Close()

	c := NewClient(WithBaseURL(srv.URL), WithAPIKey("test"), WithMaxRetries(0))
	result, err := c.PollAsync(context.Background(), "/status/req-422", &AsyncPollOptions{
		Interval: 50 * time.Millisecond,
		Timeout:  1 * time.Second,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "error" || result.Error != "connection refused" {
		t.Fatalf("expected API error converted to async error, got %#v", result)
	}
}

func TestPollAsync_404BecomesMissing(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]any{"error": "Request not found"})
	}))
	defer srv.Close()

	c := NewClient(WithBaseURL(srv.URL), WithAPIKey("test"), WithMaxRetries(0))
	result, err := c.PollAsync(context.Background(), "/status/missing", &AsyncPollOptions{
		Interval: 50 * time.Millisecond,
		Timeout:  1 * time.Second,
	})
	if err == nil {
		t.Fatal("expected missing request error")
	}
	if result == nil || result.Status != "missing" {
		t.Fatalf("expected missing result, got %#v", result)
	}
}

func TestPollAsync_EmptyStatusPath(t *testing.T) {
	c := NewClient(WithAPIKey("test"), WithMaxRetries(0))
	_, err := c.PollAsync(context.Background(), " ", &AsyncPollOptions{
		Interval: 50 * time.Millisecond,
		Timeout:  1 * time.Second,
	})
	if err == nil {
		t.Fatal("expected empty status path error")
	}
}

func TestPollAsync_MalformedPayload(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("{not-json"))
	}))
	defer srv.Close()

	c := NewClient(WithBaseURL(srv.URL), WithAPIKey("test"), WithMaxRetries(0))
	_, err := c.PollAsync(context.Background(), "/status/bad", &AsyncPollOptions{
		Interval: 50 * time.Millisecond,
		Timeout:  1 * time.Second,
	})
	if err == nil {
		t.Fatal("expected malformed payload error")
	}
}

func TestPollAsync_Timeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"request_id": "req-stuck",
			"status":     "running",
		})
	}))
	defer srv.Close()

	c := NewClient(WithBaseURL(srv.URL), WithAPIKey("test"), WithMaxRetries(0))
	_, err := c.PollAsync(context.Background(), "/status/req-stuck", &AsyncPollOptions{
		Interval: 50 * time.Millisecond,
		Timeout:  200 * time.Millisecond,
	})
	if err == nil {
		t.Fatal("expected timeout error")
	}
}

func TestPollAsync_ContextCancellation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"request_id": "req-stuck",
			"status":     "running",
		})
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	c := NewClient(WithBaseURL(srv.URL), WithAPIKey("test"), WithMaxRetries(0))
	_, err := c.PollAsync(ctx, "/status/req-stuck", &AsyncPollOptions{
		Interval: time.Second,
		Timeout:  time.Minute,
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context canceled, got %v", err)
	}
}

func TestDoAsync_PassthroughFor200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"id":     42,
			"status": "running",
		})
	}))
	defer srv.Close()

	c := NewClient(WithBaseURL(srv.URL), WithAPIKey("test"), WithMaxRetries(0))
	var result PlaygroundStatus
	err := c.doAsync(context.Background(), http.MethodPost, "/api/playgrounds/42/action", "/api/playgrounds/42/action/%s", nil, &result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != 42 {
		t.Fatalf("expected ID=42, got %d", result.ID)
	}
}

func TestDoAsync_202ThenPollSuccess(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := calls.Add(1)
		if n == 1 {
			// First call: return 202 Accepted
			w.WriteHeader(http.StatusAccepted)
			json.NewEncoder(w).Encode(map[string]any{
				"request_id": "req-async",
				"status":     "queued",
			})
			return
		}
		// Poll calls: return success
		json.NewEncoder(w).Encode(map[string]any{
			"request_id": "req-async",
			"status":     "success",
			"id":         99,
		})
	}))
	defer srv.Close()

	c := NewClient(WithBaseURL(srv.URL), WithAPIKey("test"), WithMaxRetries(0))
	var result PlaygroundStatus
	err := c.doAsync(context.Background(), http.MethodPost, "/api/playgrounds/99/action", "/api/playgrounds/99/action/%s", nil, &result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != 99 {
		t.Fatalf("expected ID=99, got %d", result.ID)
	}
}

func TestDoAsync_UsesStatusURLWhenPresent(t *testing.T) {
	var polledPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			w.WriteHeader(http.StatusAccepted)
			json.NewEncoder(w).Encode(map[string]any{
				"request_id": "req-async",
				"status":     "queued",
				"status_url": "/custom/status/req-async",
			})
			return
		}
		polledPath = r.URL.Path
		json.NewEncoder(w).Encode(map[string]any{
			"request_id": "req-async",
			"status":     "success",
			"id":         101,
		})
	}))
	defer srv.Close()

	c := NewClient(WithBaseURL(srv.URL), WithAPIKey("test"), WithMaxRetries(0))
	var result PlaygroundStatus
	err := c.doAsync(context.Background(), http.MethodPost, "/api/playgrounds/101/action", "/wrong/%s", nil, &result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if polledPath != "/custom/status/req-async" {
		t.Fatalf("expected status_url to be polled, got %q", polledPath)
	}
	if result.ID != 101 {
		t.Fatalf("expected ID=101, got %d", result.ID)
	}
}

func TestDoAsync_EmptyRequestIDWithoutStatusURL(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(map[string]any{"status": "queued"})
	}))
	defer srv.Close()

	c := NewClient(WithBaseURL(srv.URL), WithAPIKey("test"), WithMaxRetries(0))
	err := c.doAsync(context.Background(), http.MethodPost, "/api/playgrounds/1/action", "/status/%s", nil, nil)
	if err == nil {
		t.Fatal("expected missing request_id error")
	}
}

func TestAsyncResult_Helpers(t *testing.T) {
	pending := AsyncResult{Status: "queued"}
	if pending.IsComplete() {
		t.Error("queued should not be complete")
	}
	if !pending.IsPending() {
		t.Error("queued should be pending")
	}

	running := AsyncResult{Status: "running"}
	if running.IsComplete() {
		t.Error("running should not be complete")
	}
	if !running.IsPending() {
		t.Error("running should be pending")
	}

	success := AsyncResult{Status: "success"}
	if !success.IsComplete() {
		t.Error("success should be complete")
	}
	if success.IsPending() {
		t.Error("success should not be pending")
	}

	errResult := AsyncResult{Status: "error"}
	if !errResult.IsComplete() {
		t.Error("error should be complete")
	}
}
