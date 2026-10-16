package mcpserver

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
)

type securityTestSession struct {
	id string
}

func (*securityTestSession) Initialize()                                         {}
func (*securityTestSession) Initialized() bool                                   { return true }
func (*securityTestSession) NotificationChannel() chan<- mcp.JSONRPCNotification { return nil }
func (s *securityTestSession) SessionID() string                                 { return s.id }

func TestResolveClientRequireAuthDoesNotUseServerDefault(t *testing.T) {
	srv := New(Config{APIKey: "server-secret", Domain: "fibe.gg", RequireAuth: true})
	srv.httpMode = true
	srv.baseCli = srv.buildBaseClient()
	if _, err := srv.resolveClient(context.Background()); err == nil || !strings.Contains(err.Error(), "no API key resolved") {
		t.Fatalf("error = %v", err)
	}
}

func TestResolveClientRejectsDomainOverrideWithDefaultKey(t *testing.T) {
	srv := New(Config{
		APIKey:         "server-secret",
		Domain:         "https://api.example.com",
		AllowedDomains: []string{"https://other.example.com"},
	})
	srv.httpMode = true
	srv.baseCli = srv.buildBaseClient()
	ctx := context.WithValue(context.Background(), ctxKeyDomain{}, "https://other.example.com")
	if _, err := srv.resolveClient(ctx); err == nil || !strings.Contains(err.Error(), "requires request or session credentials") {
		t.Fatalf("error = %v", err)
	}
}

func TestResolveClientAllowsPairedAllowlistedDomainOverride(t *testing.T) {
	srv := New(Config{
		APIKey:         "server-secret",
		Domain:         "https://api.example.com",
		AllowedDomains: []string{"https://other.example.com"},
	})
	srv.httpMode = true
	srv.baseCli = srv.buildBaseClient()
	ctx := context.WithValue(context.Background(), ctxKeyAPIKey{}, "tenant-secret")
	ctx = context.WithValue(ctx, ctxKeyDomain{}, "https://other.example.com")
	client, err := srv.resolveClient(ctx)
	if err != nil {
		t.Fatalf("resolve client: %v", err)
	}
	if client.BaseURL() != "https://other.example.com" {
		t.Fatalf("base URL = %q", client.BaseURL())
	}
}

func TestResolveClientAlwaysIsolatesSessionClientState(t *testing.T) {
	srv := New(Config{APIKey: "server-secret", Domain: "fibe.gg"})
	srv.baseCli = srv.buildBaseClient()
	ctxA := srv.mcp.WithContext(context.Background(), &securityTestSession{id: "a"})
	ctxB := srv.mcp.WithContext(context.Background(), &securityTestSession{id: "b"})
	clientA, err := srv.resolveClient(ctxA)
	if err != nil {
		t.Fatal(err)
	}
	clientB, err := srv.resolveClient(ctxB)
	if err != nil {
		t.Fatal(err)
	}
	if clientA == clientB || clientA == srv.baseCli || clientB == srv.baseCli {
		t.Fatal("sessions must have distinct client reliability state")
	}
}

func TestResolveClientRejectsCredentialChangeWithinSession(t *testing.T) {
	srv := New(Config{Domain: "fibe.gg", RequireAuth: true})
	srv.httpMode = true
	srv.baseCli = srv.buildBaseClient()
	session := &securityTestSession{id: "same"}
	ctx := srv.mcp.WithContext(context.WithValue(context.Background(), ctxKeyAPIKey{}, "first"), session)
	if _, err := srv.resolveClient(ctx); err != nil {
		t.Fatal(err)
	}
	changed := srv.mcp.WithContext(context.WithValue(context.Background(), ctxKeyAPIKey{}, "second"), session)
	if _, err := srv.resolveClient(changed); err == nil || !strings.Contains(err.Error(), "pinned") {
		t.Fatalf("error = %v", err)
	}
}

func TestResolveClientRejectsHeaderConflictBeforeClientBuild(t *testing.T) {
	srv := New(Config{Domain: "fibe.gg", RequireAuth: true})
	srv.httpMode = true
	session := &securityTestSession{id: "same"}
	ctx := srv.mcp.WithContext(context.Background(), session)
	srv.setSessionProfile(ctx, "saved", "profile-key", "https://fibe.gg")
	conflicting := srv.mcp.WithContext(context.WithValue(context.Background(), ctxKeyAPIKey{}, "header-key"), session)
	if _, err := srv.resolveClient(conflicting); err == nil || !strings.Contains(err.Error(), "pinned") {
		t.Fatalf("error = %v", err)
	}
}

func TestAuthSetRollsBackWhenCandidateClientCannotBeResolved(t *testing.T) {
	srv := New(Config{APIKey: "server", Domain: "https://api.example.com", AllowedDomains: []string{"https://api.example.com"}})
	if err := srv.RegisterAll(); err != nil {
		t.Fatal(err)
	}
	srv.httpMode = true
	srv.setSessionProfile(context.Background(), "previous", "previous-key", "https://api.example.com")
	tool, ok := srv.dispatcher.lookup("fibe_auth_set")
	if !ok {
		t.Fatal("fibe_auth_set not registered")
	}
	_, err := tool.handler(context.Background(), nil, map[string]any{
		"api_key": "new-key", "domain": "https://not-allowed.example.com", "validate": true,
	})
	if err == nil || !strings.Contains(err.Error(), "credentials NOT saved") {
		t.Fatalf("error = %v", err)
	}
	state := srv.sessionFor(context.Background())
	if state.profile != "previous" || state.apiKey != "previous-key" || state.domain != "https://api.example.com" {
		t.Fatalf("session was not rolled back: %#v", state)
	}
}

func TestHTTPTransportDeniesLocalCapabilitiesByDefault(t *testing.T) {
	srv := New(Config{APIKey: "server", Domain: "fibe.gg", ToolSet: "full"})
	if err := srv.RegisterAll(); err != nil {
		t.Fatal(err)
	}
	srv.httpMode = true
	if _, err := srv.dispatcher.dispatch(context.Background(), "fibe_run", map[string]any{"args": []any{"status"}}); err == nil || !strings.Contains(err.Error(), "local access") {
		t.Fatalf("error = %v", err)
	}
}

func TestHTTPTransportDetectsNestedLocalCapabilities(t *testing.T) {
	tool := &toolImpl{name: "fibe_resource_mutate", tier: tierBase}
	args := map[string]any{"payload": map[string]any{"content_path": "/tmp/private"}}
	if !isLocalCapability(tool, args) {
		t.Fatal("nested content_path was not classified as a local capability")
	}
}

func TestServeHTTPRequiresAuthOnNonLoopbackBind(t *testing.T) {
	srv := New(DefaultConfig())
	if err := srv.ServeHTTP(context.Background(), ":0", true); err == nil || !strings.Contains(err.Error(), "require-auth") {
		t.Fatalf("error = %v", err)
	}
}

func TestSecureHTTPHandlerRejectsCrossOriginRequest(t *testing.T) {
	srv := New(DefaultConfig())
	handler := srv.secureHTTPHandler(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }))
	req := httptest.NewRequest(http.MethodPost, "http://localhost/mcp", strings.NewReader("{}"))
	req.Header.Set("Origin", "https://attacker.example")
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	if res.Code != http.StatusForbidden {
		t.Fatalf("status = %d", res.Code)
	}
}

func TestSecureHTTPHandlerAllowsExplicitBrowserOrigin(t *testing.T) {
	srv := New(Config{AllowedDomains: []string{"https://console.example"}})
	handler := srv.secureHTTPHandler(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }))
	req := httptest.NewRequest(http.MethodPost, "http://localhost/mcp", strings.NewReader("{}"))
	req.Header.Set("Origin", "https://console.example")
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	if res.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusNoContent)
	}
}

func TestServeHTTPCancelledContextShutsDownBothTransports(t *testing.T) {
	for _, streamable := range []bool{false, true} {
		server := New(DefaultConfig())
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		err := server.ServeHTTP(ctx, "127.0.0.1:0", streamable)
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("streamable=%v error=%v, want context cancellation", streamable, err)
		}
	}
}

func TestRequestCredentialExtraction(t *testing.T) {
	server := New(DefaultConfig())
	request := httptest.NewRequest(http.MethodPost, "http://localhost/mcp", nil)
	request.Header.Set("Authorization", "bEaReR fixture-token")
	request.Header.Set("X-Fibe-Domain", "https://api.example.com")
	ctx := server.injectAuthFromRequest(context.Background(), request)
	if got := apiKeyFromContext(ctx); got != "fixture-token" {
		t.Fatalf("api key = %q", got)
	}
	if got := domainFromContext(ctx); got != "https://api.example.com" {
		t.Fatalf("domain = %q", got)
	}

	request.Header.Set("Authorization", "Basic ignored")
	request.Header.Set("X-Fibe-API-Key", "fallback-key")
	ctx = server.injectAuthFromRequest(context.Background(), request)
	if got := apiKeyFromContext(ctx); got != "fallback-key" {
		t.Fatalf("fallback api key = %q", got)
	}
}

func TestCoerceInt64RejectsOverflow(t *testing.T) {
	for _, value := range []any{uint64(math.MaxUint64), math.Inf(1), float64(math.MaxInt64) * 2} {
		if got, ok := coerceInt64(value); ok {
			t.Fatalf("coerceInt64(%v) = %d, want rejection", value, got)
		}
	}
}

func TestNumericCoercionRejectsNonFiniteAndOutOfRangeValues(t *testing.T) {
	for _, value := range []any{math.Inf(1), math.NaN(), math.Exp2(64), "18446744073709551616", "-1"} {
		if got, ok := coerceUint64(value); ok {
			t.Fatalf("coerceUint64(%v) = %d, want rejection", value, got)
		}
	}
	for _, value := range []any{math.Inf(1), math.Inf(-1), math.NaN(), "Inf", json.Number("NaN")} {
		if got, ok := coerceFloat64(value); ok {
			t.Fatalf("coerceFloat64(%v) = %v, want rejection", value, got)
		}
	}
	if _, ok := argInt64(map[string]any{"id": math.Exp2(63)}, "id"); ok {
		t.Fatal("argInt64 accepted overflowing float")
	}
}

func TestPipelineCacheDeleteSessionAndCopyOnRead(t *testing.T) {
	cache := newPipelineCache(4, 1024)
	id, _, err := cache.Put("session", map[string]any{"ok": true})
	if err != nil {
		t.Fatal(err)
	}
	first, ok := cache.Get("session", id)
	if !ok {
		t.Fatal("cache miss")
	}
	first[0] = 'X'
	second, _ := cache.Get("session", id)
	if second[0] == 'X' {
		t.Fatal("cache returned mutable internal payload")
	}
	cache.DeleteSession("session")
	if _, ok := cache.Get("session", id); ok {
		t.Fatal("session entries were not removed")
	}
}

func TestPipelineCacheUsesInjectedClockAndExpiresAllStaleEntries(t *testing.T) {
	now := time.Unix(1000, 0)
	cache := newPipelineCache(4, 1024)
	cache.now = func() time.Time { return now }
	first, _, err := cache.Put("session", map[string]any{"first": true})
	if err != nil {
		t.Fatal(err)
	}
	now = now.Add(time.Minute)
	second, _, err := cache.Put("session", map[string]any{"second": true})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := cache.Get("session", first); !ok {
		t.Fatal("fresh entry was removed")
	}
	now = now.Add(pipelineCacheTTL)
	if entries, _ := cache.Stats(); entries != 0 {
		t.Fatalf("expired entries = %d, want 0", entries)
	}
	if _, ok := cache.Get("session", second); ok {
		t.Fatal("expired entry remained accessible")
	}
}

func TestReadLocalFileRejectsSymlink(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target")
	if err := os.WriteFile(target, []byte("secret"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "link")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	if _, err := readLocalFile(link); err == nil || !strings.Contains(err.Error(), "non-symlink") {
		t.Fatalf("error = %v", err)
	}
}

func TestAuditRedactionIsRecursiveAndCaseInsensitive(t *testing.T) {
	redacted := redactSensitive(map[string]any{
		"Nested": map[string]any{"API-KEY": "secret", "safe": "ok"},
		"items":  []any{map[string]any{"private_key": "pem", "request_content": "source"}},
	})
	nested := redacted["Nested"].(map[string]any)
	if nested["API-KEY"] != "[redacted]" || nested["safe"] != "ok" {
		t.Fatalf("nested redaction = %#v", nested)
	}
	items := redacted["items"].([]any)
	if items[0].(map[string]any)["private_key"] != "[redacted]" {
		t.Fatalf("array redaction = %#v", items)
	}
	if items[0].(map[string]any)["request_content"] != "[redacted]" {
		t.Fatalf("content suffix redaction = %#v", items)
	}
}
