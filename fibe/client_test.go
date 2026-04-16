package fibe

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func testServer(t *testing.T, handler http.HandlerFunc) (*Client, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	client := NewClient(
		WithAPIKey("test-key"),
		WithBaseURL(srv.URL),
		WithMaxRetries(0),
	)
	return client, srv
}

func testServerWithRetry(t *testing.T, handler http.HandlerFunc) (*Client, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	client := NewClient(
		WithAPIKey("test-key"),
		WithBaseURL(srv.URL),
		WithMaxRetries(2),
		WithRetryDelay(1*time.Millisecond, 10*time.Millisecond),
	)
	return client, srv
}

func TestClient_AuthorizationHeader(t *testing.T) {
	var gotHeader string
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotHeader = r.Header.Get("Authorization")
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(listEnv([]Playground{}))
	})

	c.Playgrounds.List(context.Background(), nil)
	if gotHeader != "Bearer test-key" {
		t.Errorf("expected 'Bearer test-key', got %q", gotHeader)
	}
}

func TestClient_UserAgent(t *testing.T) {
	var gotUA string
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotUA = r.Header.Get("User-Agent")
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(listEnv([]Playground{}))
	})

	c.Playgrounds.List(context.Background(), nil)
	if gotUA != defaultUserAgent {
		t.Errorf("expected %q, got %q", defaultUserAgent, gotUA)
	}
}

func TestClient_CustomUserAgent(t *testing.T) {
	var gotUA string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUA = r.Header.Get("User-Agent")
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(listEnv([]Playground{}))
	}))
	defer srv.Close()

	c := NewClient(
		WithAPIKey("test"),
		WithBaseURL(srv.URL),
		WithUserAgent("custom/2.0"),
		WithMaxRetries(0),
	)
	c.Playgrounds.List(context.Background(), nil)
	if gotUA != "custom/2.0" {
		t.Errorf("expected 'custom/2.0', got %q", gotUA)
	}
}

func TestClient_APIError(t *testing.T) {
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		json.NewEncoder(w).Encode(apiErrorResponse{
			Error: struct {
				Code    string         `json:"code"`
				Message string         `json:"message"`
				Details map[string]any `json:"details,omitempty"`
			}{
				Code:    ErrCodeNotFound,
				Message: "Playground not found",
			},
		})
	})

	_, err := c.Playgrounds.Get(context.Background(), 999)
	if err == nil {
		t.Fatal("expected error")
	}
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.Code != ErrCodeNotFound {
		t.Errorf("expected code %q, got %q", ErrCodeNotFound, apiErr.Code)
	}
	if apiErr.StatusCode != 404 {
		t.Errorf("expected status 404, got %d", apiErr.StatusCode)
	}
	if !apiErr.IsNotFound() {
		t.Error("expected IsNotFound() to be true")
	}
}

func TestClient_ValidationError(t *testing.T) {
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(422)
		json.NewEncoder(w).Encode(apiErrorResponse{
			Error: struct {
				Code    string         `json:"code"`
				Message string         `json:"message"`
				Details map[string]any `json:"details,omitempty"`
			}{
				Code:    ErrCodeValidationFailed,
				Message: "Validation failed",
				Details: map[string]any{"name": []any{"can't be blank"}},
			},
		})
	})

	_, err := c.Playgrounds.Create(context.Background(), &PlaygroundCreateParams{
		Name:       "test",
		PlayspecID: 1,
	})
	apiErr := err.(*APIError)
	if !apiErr.IsValidation() {
		t.Error("expected IsValidation() to be true")
	}
	if apiErr.Details == nil {
		t.Error("expected details to be present")
	}
}

func TestClient_RateLimitHeaders(t *testing.T) {
	resetTime := time.Now().Add(1 * time.Hour).Unix()
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-RateLimit-Limit", "5000")
		w.Header().Set("X-RateLimit-Remaining", "4999")
		w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", resetTime))
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(listEnv([]Playground{}))
	})

	c.Playgrounds.List(context.Background(), nil)

	rl := c.RateLimit()
	if rl.Limit != 5000 {
		t.Errorf("expected limit 5000, got %d", rl.Limit)
	}
	if rl.Remaining != 4999 {
		t.Errorf("expected remaining 4999, got %d", rl.Remaining)
	}
}

func TestClient_Retry(t *testing.T) {
	attempts := 0
	c, _ := testServerWithRetry(t, func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(503)
			json.NewEncoder(w).Encode(apiErrorResponse{
				Error: struct {
					Code    string         `json:"code"`
					Message string         `json:"message"`
					Details map[string]any `json:"details,omitempty"`
				}{Code: ErrCodeInternalError, Message: "Service unavailable"},
			})
			return
		}
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(listEnv([]Playground{}))
	})

	_, err := c.Playgrounds.List(context.Background(), nil)
	if err != nil {
		t.Fatalf("expected success after retries, got: %v", err)
	}
	if attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}
}

func TestClient_NoRetryOn4xx(t *testing.T) {
	attempts := 0
	c, _ := testServerWithRetry(t, func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(403)
		json.NewEncoder(w).Encode(apiErrorResponse{
			Error: struct {
				Code    string         `json:"code"`
				Message string         `json:"message"`
				Details map[string]any `json:"details,omitempty"`
			}{Code: ErrCodeForbidden, Message: "Forbidden"},
		})
	})

	_, err := c.Playgrounds.List(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if attempts != 1 {
		t.Errorf("expected 1 attempt (no retry on 403), got %d", attempts)
	}
}

func TestClient_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(listEnv([]Playground{}))
	}))
	defer srv.Close()

	c := NewClient(
		WithAPIKey("test"),
		WithBaseURL(srv.URL),
		WithMaxRetries(0),
	)

	_, err := c.Playgrounds.List(ctx, nil)
	if err == nil {
		t.Fatal("expected error from cancelled context")
	}
}

func TestClient_Delete204(t *testing.T) {
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		w.WriteHeader(204)
	})

	err := c.Playgrounds.Delete(context.Background(), 42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestClient_JSONContentType(t *testing.T) {
	var gotCT string
	c, _ := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotCT = r.Header.Get("Content-Type")
		w.WriteHeader(201)
		json.NewEncoder(w).Encode(Playground{ID: 1, Name: "test"})
	})

	c.Playgrounds.Create(context.Background(), &PlaygroundCreateParams{
		Name:       "test",
		PlayspecID: 1,
	})

	if gotCT != "application/json" {
		t.Errorf("expected 'application/json', got %q", gotCT)
	}
}

func TestClient_WithKey(t *testing.T) {
	var gotKeys []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotKeys = append(gotKeys, r.Header.Get("Authorization"))
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(Player{ID: 1, Username: "test"})
	}))
	defer srv.Close()

	admin := NewClient(
		WithAPIKey("admin-key"),
		WithBaseURL(srv.URL),
		WithMaxRetries(0),
	)
	reader := admin.WithKey("reader-key")
	other := admin.WithKey("other-key")

	admin.APIKeys.Me(context.Background())
	reader.APIKeys.Me(context.Background())
	other.APIKeys.Me(context.Background())

	expected := []string{"Bearer admin-key", "Bearer reader-key", "Bearer other-key"}
	for i, want := range expected {
		if i >= len(gotKeys) {
			t.Fatalf("missing request %d", i)
		}
		if gotKeys[i] != want {
			t.Errorf("request %d: expected %q, got %q", i, want, gotKeys[i])
		}
	}
}

func TestClient_WithKeySharesTransport(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(Player{ID: 1})
	}))
	defer srv.Close()

	parent := NewClient(WithAPIKey("key1"), WithBaseURL(srv.URL), WithMaxRetries(0))
	child := parent.WithKey("key2")

	if parent.BaseURL() != child.BaseURL() {
		t.Errorf("expected same base URL, got %q vs %q", parent.BaseURL(), child.BaseURL())
	}
}

func TestClient_WithKeyIndependentState(t *testing.T) {
	resetTime := time.Now().Add(1 * time.Hour).Unix()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-RateLimit-Remaining", "100")
		w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", resetTime))
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(listEnv([]Playground{}))
	}))
	defer srv.Close()

	parent := NewClient(WithAPIKey("key1"), WithBaseURL(srv.URL), WithMaxRetries(0))
	child := parent.WithKey("key2")

	parent.Playgrounds.List(context.Background(), nil)

	if parent.RateLimit().Remaining != 100 {
		t.Error("parent should have updated rate limit")
	}
	if child.RateLimit().Remaining != 0 {
		t.Error("child should have independent rate limit state")
	}
}

func TestDomainResolution(t *testing.T) {
	tests := []struct {
		domain string
		want   string
	}{
		{"fibe.gg", "https://fibe.gg"},
		{"staging.fibe.gg", "https://staging.fibe.gg"},
		{"localhost:3000", "http://localhost:3000"},
		{"127.0.0.1:3000", "http://127.0.0.1:3000"},
		{"http://localhost:3000", "http://localhost:3000"},
		{"https://custom.example.com", "https://custom.example.com"},
		{"https://fibe.gg/", "https://fibe.gg"},
		{"rails.test:3000", "http://rails.test:3000"},
		{"app.local:3000", "http://app.local:3000"},
		{"dev.internal:8080", "http://dev.internal:8080"},
	}

	for _, tt := range tests {
		cfg := &clientConfig{domain: tt.domain}
		got := cfg.baseURL()
		if got != tt.want {
			t.Errorf("domain %q: expected %q, got %q", tt.domain, tt.want, got)
		}
	}
}

func TestClient_EnvFallback(t *testing.T) {
	t.Setenv("FIBE_API_KEY", "env-key")
	t.Setenv("FIBE_DOMAIN", "staging.fibe.gg")

	c := NewClient(WithMaxRetries(0))

	if c.BaseURL() != "https://staging.fibe.gg" {
		t.Errorf("expected https://staging.fibe.gg, got %s", c.BaseURL())
	}
}

func TestClient_ExplicitOverridesEnv(t *testing.T) {
	t.Setenv("FIBE_API_KEY", "env-key")
	t.Setenv("FIBE_DOMAIN", "staging.fibe.gg")

	c := NewClient(
		WithAPIKey("explicit-key"),
		WithDomain("custom.example.com"),
		WithMaxRetries(0),
	)

	if c.BaseURL() != "https://custom.example.com" {
		t.Errorf("expected https://custom.example.com, got %s", c.BaseURL())
	}
}
