package fibe

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type countingTransport struct {
	base  http.RoundTripper
	calls atomic.Int32
}

type transportFunc func(*http.Request) (*http.Response, error)

func (fn transportFunc) RoundTrip(req *http.Request) (*http.Response, error) { return fn(req) }

func (t *countingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	t.calls.Add(1)
	return t.base.RoundTrip(req)
}

func TestClient_AutomaticIdempotencyKeyIsStableAcrossRetries(t *testing.T) {
	var attempts atomic.Int32
	var mu sync.Mutex
	var keys []string
	c, _ := testServerWithRetry(t, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		keys = append(keys, r.Header.Get("Idempotency-Key"))
		mu.Unlock()
		if attempts.Add(1) < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"error":{"code":"INTERNAL_ERROR","message":"retry"}}`))
			return
		}
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(Playground{ID: 1, Name: "test"})
	})

	_, err := c.Playgrounds.Create(context.Background(), &PlaygroundCreateParams{Name: "test", PlayspecID: 1})
	if err != nil {
		t.Fatalf("create after retry: %v", err)
	}
	if len(keys) != 3 || keys[0] == "" {
		t.Fatalf("unexpected idempotency keys: %#v", keys)
	}
	for i, key := range keys[1:] {
		if key != keys[0] {
			t.Fatalf("retry %d changed idempotency key: %#v", i+1, keys)
		}
	}
}

func TestClient_NegativeMaxRetriesStillSendsOneRequest(t *testing.T) {
	var attempts atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		attempts.Add(1)
		_ = json.NewEncoder(w).Encode(Player{ID: 1})
	}))
	defer srv.Close()

	c := NewClient(WithBaseURL(srv.URL), WithMaxRetries(-10), WithDisableAutoConfig())
	if _, err := c.APIKeys.Me(context.Background()); err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if got := attempts.Load(); got != 1 {
		t.Fatalf("requests = %d, want 1", got)
	}
}

func TestClient_RetryUsesInjectedClockAndSleeper(t *testing.T) {
	var attempts atomic.Int32
	c, _ := testServerWithRetry(t, func(w http.ResponseWriter, _ *http.Request) {
		if attempts.Add(1) == 1 {
			w.Header().Set("Retry-After", "Tue, 02 Jan 2024 03:04:08 GMT")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"error":{"message":"retry"}}`))
			return
		}
		_ = json.NewEncoder(w).Encode(Player{ID: 1})
	})
	fixed := time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)
	c.now = func() time.Time { return fixed }
	c.retry.maxDelay = 10 * time.Second
	c.retry.jitter = func() float64 { return 1 }
	var delays []time.Duration
	c.sleep = func(_ context.Context, delay time.Duration) error {
		delays = append(delays, delay)
		return nil
	}

	if _, err := c.APIKeys.Me(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(delays) != 1 || delays[0] != 3*time.Second {
		t.Fatalf("retry delays = %v", delays)
	}
}

func TestClient_DoesNotRetryLocalHookOrMarshalErrors(t *testing.T) {
	var hookCalls atomic.Int64
	transport := &countingTransport{}
	client := NewClient(
		WithAPIKey("test"),
		WithBaseURL("https://example.com"),
		WithHTTPClient(&http.Client{Transport: transport}),
		WithMaxRetries(4),
		WithRequestHook(func(*http.Request) error {
			hookCalls.Add(1)
			return errors.New("blocked locally")
		}),
		WithCircuitBreaker(CircuitBreakerConfig{FailureThreshold: 1, ResetTimeout: time.Hour, HalfOpenRequests: 1}),
	)
	if err := client.do(context.Background(), http.MethodGet, "/hook", nil, nil); err == nil {
		t.Fatal("hook error was ignored")
	}
	if hookCalls.Load() != 1 || transport.calls.Load() != 0 {
		t.Fatalf("hook calls=%d transport calls=%d", hookCalls.Load(), transport.calls.Load())
	}
	if err := client.do(context.Background(), http.MethodGet, "/hook", nil, nil); err == nil {
		t.Fatal("second hook error was ignored")
	}
	if _, ok := client.breakerErrorForTest(); ok {
		t.Fatal("local hook error opened circuit breaker")
	}

	marshalClient := NewClient(
		WithAPIKey("test"),
		WithBaseURL("https://example.com"),
		WithHTTPClient(&http.Client{Transport: transport}),
		WithMaxRetries(4),
	)
	if err := marshalClient.do(context.Background(), http.MethodPost, "/marshal", map[string]any{"bad": func() {}}, nil); err == nil {
		t.Fatal("marshal error was ignored")
	}
	if transport.calls.Load() != 0 {
		t.Fatalf("marshal error reached transport %d times", transport.calls.Load())
	}
}

func TestClient_CircuitCountsFinalTransientTransportFailure(t *testing.T) {
	client := NewClient(
		WithBaseURL("https://example.com"),
		WithHTTPClient(&http.Client{Transport: transportFunc(func(*http.Request) (*http.Response, error) {
			return nil, io.ErrUnexpectedEOF
		})}),
		WithMaxRetries(0),
		WithCircuitBreaker(CircuitBreakerConfig{FailureThreshold: 1, ResetTimeout: time.Hour, HalfOpenRequests: 1}),
	)
	if err := client.do(context.Background(), http.MethodGet, "/fixture", nil, nil); !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("first error = %v", err)
	}
	if err := client.do(context.Background(), http.MethodGet, "/fixture", nil, nil); err == nil {
		t.Fatal("circuit remained closed after final transient failure")
	} else {
		var open *CircuitOpenError
		if !errors.As(err, &open) {
			t.Fatalf("second error = %T %v", err, err)
		}
	}
}

func (c *Client) breakerErrorForTest() (error, bool) {
	if c.breaker != nil && !c.breaker.allow() {
		return &CircuitOpenError{}, true
	}
	if c.breaker != nil {
		c.breaker.releaseProbe()
	}
	return nil, false
}

func TestClient_DoesNotSendEmptyAuthorizationHeader(t *testing.T) {
	var authorization string
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		authorization = r.Header.Get("Authorization")
		_ = json.NewEncoder(w).Encode(Player{ID: 1})
	})
	c.cfg.apiKey = ""

	if _, err := c.APIKeys.Me(context.Background()); err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if authorization != "" {
		t.Fatalf("Authorization = %q, want absent", authorization)
	}
}

func TestClient_CapturesHeadersBeforeResponseHookError(t *testing.T) {
	c, _ := testServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-Request-Id", "req-hook-error")
		w.Header().Set("X-RateLimit-Limit", "42")
		w.Header().Set("X-RateLimit-Remaining", "41")
		_ = json.NewEncoder(w).Encode(Player{ID: 1})
	})
	c.cfg.responseHook = func(*http.Response) error { return errors.New("hook rejected response") }

	if _, err := c.APIKeys.Me(context.Background()); err == nil || !strings.Contains(err.Error(), "hook rejected response") {
		t.Fatalf("error = %v", err)
	}
	if got := c.LastRequestID(); got != "req-hook-error" {
		t.Fatalf("request ID = %q", got)
	}
	if got := c.RateLimit(); got.Limit != 42 || got.Remaining != 41 {
		t.Fatalf("rate limit = %#v", got)
	}
}

func TestClient_WithTimeoutClonesCustomHTTPClient(t *testing.T) {
	original := &http.Client{Timeout: 10 * time.Second}
	c := NewClient(WithTimeout(125*time.Millisecond), WithHTTPClient(original), WithDisableAutoConfig())
	if c.http == original {
		t.Fatal("client must clone the caller-provided http.Client")
	}
	if c.http.Timeout != 125*time.Millisecond {
		t.Fatalf("timeout = %s, want 125ms", c.http.Timeout)
	}
	if original.Timeout != 10*time.Second {
		t.Fatalf("caller client was mutated: %s", original.Timeout)
	}
}

func TestClient_RejectsTrailingJSONValue(t *testing.T) {
	c, _ := testServer(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"data":[],"meta":{}} {"extra":true}`))
	})
	_, err := c.Playgrounds.List(context.Background(), nil)
	if err == nil || !strings.Contains(err.Error(), "multiple JSON values") {
		t.Fatalf("error = %v, want multiple JSON values", err)
	}
}

func TestClient_DecodeFailureDoesNotPartiallyMutateResult(t *testing.T) {
	c, _ := testServer(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"id":1,"name":"partial"} trailing`))
	})
	result := Player{ID: 99, Username: "original"}
	err := c.do(context.Background(), http.MethodGet, "/fixture", nil, &result)
	if err == nil {
		t.Fatal("expected trailing JSON error")
	}
	if result.ID != 99 || result.Username != "original" {
		t.Fatalf("caller result was partially mutated: %#v", result)
	}
}

func TestResolveBaseURLSupportsIPv6Loopback(t *testing.T) {
	got, err := resolveBaseURL("[::1]:3000")
	if err != nil {
		t.Fatalf("resolve IPv6 loopback: %v", err)
	}
	if got != "http://[::1]:3000" {
		t.Fatalf("base URL = %q", got)
	}
}

func TestResolveBaseURLRejectsCredentialsAndFragments(t *testing.T) {
	for _, raw := range []string{"https://user:pass@example.com", "https://example.com/#fragment", "ftp://example.com"} {
		t.Run(raw, func(t *testing.T) {
			if _, err := resolveBaseURL(raw); err == nil {
				t.Fatalf("resolveBaseURL(%q) unexpectedly succeeded", raw)
			}
		})
	}
}

func TestReadLimitedDetectsOverflow(t *testing.T) {
	_, err := readLimited(strings.NewReader("123456"), 5, "test body")
	if err == nil || !strings.Contains(err.Error(), "exceeds 5 bytes") {
		t.Fatalf("error = %v", err)
	}
}

func TestClient_MultipartRetriesSeekableReaderWithStableKey(t *testing.T) {
	var attempts atomic.Int32
	var keys []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		keys = append(keys, r.Header.Get("Idempotency-Key"))
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Errorf("parse multipart: %v", err)
		}
		file, _, err := r.FormFile("file")
		if err != nil {
			t.Errorf("form file: %v", err)
		} else {
			data, _ := io.ReadAll(file)
			_ = file.Close()
			if string(data) != "payload" {
				t.Errorf("payload = %q", data)
			}
		}
		if attempts.Add(1) == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"error":{"message":"retry"}}`))
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	}))
	defer srv.Close()
	c := NewClient(WithBaseURL(srv.URL), WithMaxRetries(1), WithRetryDelay(time.Millisecond, time.Millisecond), WithDisableAutoConfig())
	var result map[string]any
	if err := c.doMultipart(context.Background(), http.MethodPost, "/upload", nil, "file", "test.txt", strings.NewReader("payload"), &result); err != nil {
		t.Fatalf("multipart: %v", err)
	}
	if len(keys) != 2 || keys[0] == "" || keys[0] != keys[1] {
		t.Fatalf("idempotency keys = %#v", keys)
	}
}

func TestClient_MultipartDoesNotRetryNonReplayableReader(t *testing.T) {
	var attempts atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		_, _ = io.Copy(io.Discard, r.Body)
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"error":{"message":"retry"}}`))
	}))
	defer srv.Close()
	c := NewClient(WithBaseURL(srv.URL), WithMaxRetries(2), WithRetryDelay(time.Millisecond, time.Millisecond), WithDisableAutoConfig())
	err := c.doMultipart(context.Background(), http.MethodPost, "/upload", nil, "file", "test.txt", bytes.NewBufferString("payload"), nil)
	if err == nil {
		t.Fatal("expected server error")
	}
	if attempts.Load() != 1 {
		t.Fatalf("attempts = %d, want 1", attempts.Load())
	}
}

func TestClient_DownloadRedirectUsesConfiguredTransportWithoutCredentials(t *testing.T) {
	var externalAuthorization string
	external := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		externalAuthorization = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/octet-stream")
		_, _ = w.Write([]byte("artifact"))
	}))
	defer external.Close()
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Location", external.URL+"/artifact.bin")
		w.WriteHeader(http.StatusFound)
	}))
	defer api.Close()

	transport := &countingTransport{base: http.DefaultTransport}
	c := NewClient(WithAPIKey("secret"), WithBaseURL(api.URL), WithHTTPClient(&http.Client{Transport: transport}), WithMaxRetries(0))
	body, filename, _, err := c.doDownload(context.Background(), "/download")
	if err != nil {
		t.Fatalf("download: %v", err)
	}
	defer body.Close()
	data, _ := io.ReadAll(body)
	if string(data) != "artifact" || filename != "artifact.bin" {
		t.Fatalf("data=%q filename=%q", data, filename)
	}
	if externalAuthorization != "" {
		t.Fatalf("credential leaked to redirect target: %q", externalAuthorization)
	}
	if transport.calls.Load() != 2 {
		t.Fatalf("transport calls = %d, want 2", transport.calls.Load())
	}
}

func TestClient_DownloadPreservesCredentialsForSameOriginRedirect(t *testing.T) {
	var redirectedAuthorization string
	var api *httptest.Server
	api = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/download":
			w.Header().Set("Location", api.URL+"/protected/artifact.bin")
			w.WriteHeader(http.StatusFound)
		case "/protected/artifact.bin":
			redirectedAuthorization = r.Header.Get("Authorization")
			_, _ = w.Write([]byte("artifact"))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer api.Close()
	c := NewClient(WithAPIKey("secret"), WithBaseURL(api.URL), WithMaxRetries(0))
	body, _, _, err := c.doDownload(context.Background(), "/download")
	if err != nil {
		t.Fatal(err)
	}
	defer body.Close()
	if redirectedAuthorization != "Bearer secret" {
		t.Fatalf("same-origin authorization = %q", redirectedAuthorization)
	}
}

func TestClient_DownloadRejectsRelativeRedirect(t *testing.T) {
	c, _ := testServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Location", "/other")
		w.WriteHeader(http.StatusFound)
	})
	if _, _, _, err := c.doDownload(context.Background(), "/download"); err == nil || !strings.Contains(err.Error(), "invalid download redirect") {
		t.Fatalf("error = %v", err)
	}
}
