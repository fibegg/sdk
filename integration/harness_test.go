package integration

import (
	"context"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/fibegg/sdk/fibe"
)

var (
	user          *fibe.Client
	contract      *fibe.Client
	setupOnce     sync.Once
	setupErr      error
	setupSkip     bool
	userKeyScopes []string
	testCtx       = context.Background()
	nameCounter   atomic.Int64
)

func userClient(t *testing.T) *fibe.Client {
	t.Helper()
	setupOnce.Do(func() {
		key := os.Getenv("FIBE_API_KEY")
		if key == "" {
			setupErr = fmt.Errorf("FIBE_API_KEY is required for integration tests")
			setupSkip = true
			return
		}
		domain := os.Getenv("FIBE_DOMAIN")
		if domain == "" {
			domain = "localhost:3000"
		}

		user = integrationClient(key, domain, 2)
		contract = integrationClient(key, domain, 2)

		me, err := contract.APIKeys.Me(ctx())
		if err != nil {
			setupErr = fmt.Errorf("validate FIBE_API_KEY as normal user key: %w", err)
			return
		}
		if hasScope(me.APIKeyScopes, "*") {
			setupErr = fmt.Errorf("FIBE_API_KEY must use explicit normal-user scopes, not wildcard '*'")
			return
		}
		if len(me.APIKeyScopes) == 0 {
			setupErr = fmt.Errorf("FIBE_API_KEY did not report api_key_scopes; cannot prove normal-user coverage")
			return
		}
		userKeyScopes = append([]string(nil), me.APIKeyScopes...)
	})
	if setupErr != nil {
		if setupSkip {
			t.Skipf("skipping integration test: %v", setupErr)
		}
		t.Fatalf("integration test setup failed: %v", setupErr)
	}
	return user
}

func contractClient(t *testing.T) *fibe.Client {
	t.Helper()
	userClient(t)
	return contract
}

func superAdminClient(t *testing.T) *fibe.Client {
	t.Helper()
	key := os.Getenv("FIBE_ADMIN_API_KEY")
	if key == "" {
		t.Skip("FIBE_ADMIN_API_KEY not set — skipping admin-only integration test")
	}
	domain := os.Getenv("FIBE_DOMAIN")
	if domain == "" {
		domain = "localhost:3000"
	}
	client := integrationClient(key, domain, 2)
	me, err := client.APIKeys.Me(ctx())
	requireNoError(t, err, "validate FIBE_ADMIN_API_KEY")
	if !hasScope(me.APIKeyScopes, "*") {
		t.Fatal("FIBE_ADMIN_API_KEY must have wildcard '*' scope")
	}
	return client
}

func integrationClient(key, domain string, maxRetries int) *fibe.Client {
	return fibe.NewClient(
		fibe.WithAPIKey(key),
		fibe.WithDomain(domain),
		fibe.WithMaxRetries(maxRetries),
		fibe.WithRetryDelay(500*time.Millisecond, 5*time.Second),
	)
}

func userScopes(t *testing.T) []string {
	t.Helper()
	userClient(t)
	return append([]string(nil), userKeyScopes...)
}

func hasScope(scopes []string, expected string) bool {
	for _, scope := range scopes {
		if scope == expected {
			return true
		}
	}
	return false
}

func cliAuthArgs(t *testing.T) []string {
	t.Helper()
	key := os.Getenv("FIBE_API_KEY")
	if key == "" {
		t.Skip("FIBE_API_KEY is required for CLI integration tests")
	}
	domain := os.Getenv("FIBE_DOMAIN")
	if domain == "" {
		domain = "localhost:3000"
	}
	return []string{"--api-key", key, "--domain", domain}
}

func ctx() context.Context {
	return testCtx
}

func ctxTimeout(d time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(testCtx, d)
}

func skipThirdpartyIfDisabled(t *testing.T) {
	t.Helper()
	if os.Getenv("FIBE_SKIP_THIRDPARTY_TESTS") == "1" {
		t.Skip("third-party integration disabled by FIBE_SKIP_THIRDPARTY_TESTS=1")
	}
}

func requireNoError(t *testing.T, err error, msgAndArgs ...any) {
	t.Helper()
	if err != nil {
		if len(msgAndArgs) > 0 {
			t.Fatalf("%s: %v", fmt.Sprint(msgAndArgs...), err)
		}
		t.Fatalf("unexpected error: %v", err)
	}
}

func requireAPIError(t *testing.T, err error, expectedCode string, expectedStatus int) *fibe.APIError {
	t.Helper()
	if err == nil {
		t.Fatalf("expected API error with code %q, got nil", expectedCode)
	}
	apiErr, ok := err.(*fibe.APIError)
	if !ok {
		t.Fatalf("expected *fibe.APIError, got %T: %v", err, err)
	}
	if apiErr.Code != expectedCode {
		t.Errorf("expected error code %q, got %q (message: %s)", expectedCode, apiErr.Code, apiErr.Message)
	}
	if expectedStatus > 0 && apiErr.StatusCode != expectedStatus {
		t.Errorf("expected status %d, got %d", expectedStatus, apiErr.StatusCode)
	}
	return apiErr
}

func uniqueName(prefix string) string {
	return fmt.Sprintf("%s-%d-%d", prefix, time.Now().UnixNano(), nameCounter.Add(1))
}

func createScopedKey(t *testing.T, c *fibe.Client, label string, scopes []string) *fibe.Client {
	t.Helper()
	key, err := c.APIKeys.Create(ctx(), &fibe.APIKeyCreateParams{
		Label:  label,
		Scopes: scopes,
	})
	requireNoError(t, err, "create scoped key")
	if key.Token == nil {
		t.Fatal("expected token in create response")
	}
	t.Cleanup(func() {
		if key.ID != nil {
			c.APIKeys.Delete(ctx(), *key.ID)
		}
	})
	return c.WithKey(*key.Token)
}

func userBClient(t *testing.T) *fibe.Client {
	t.Helper()
	c := userClient(t)
	key := os.Getenv("USER_B_API_KEY")
	if key == "" {
		t.Skip("USER_B_API_KEY not set — skipping multi-user test")
	}
	return c.WithKey(key)
}

func rateLimitClient(t *testing.T) *fibe.Client {
	t.Helper()
	_ = userClient(t)
	key := os.Getenv("RATE_LIMIT_TEST_KEY")
	if key == "" {
		t.Skip("RATE_LIMIT_TEST_KEY not set — skipping rate limit test")
	}
	domain := os.Getenv("FIBE_DOMAIN")
	if domain == "" {
		domain = "localhost:3000"
	}
	return fibe.NewClient(
		fibe.WithAPIKey(key),
		fibe.WithDomain(domain),
		fibe.WithMaxRetries(0),
	)
}

func ptr[T any](v T) *T {
	return &v
}
