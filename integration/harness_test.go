package integration

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/fibegg/sdk/fibe"
)

var (
	admin     *fibe.Client
	setupOnce sync.Once
	setupErr  error
	testCtx   = context.Background()
	nameCounter atomic.Int64
)

func adminClient(t *testing.T) *fibe.Client {
	t.Helper()
	setupOnce.Do(func() {
		key := os.Getenv("FIBE_API_KEY")
		if key == "" {
			setupErr = fmt.Errorf("FIBE_API_KEY is required for integration tests")
			return
		}
		domain := os.Getenv("FIBE_DOMAIN")
		if domain == "" {
			domain = "localhost:3000"
		}

		admin = fibe.NewClient(
			fibe.WithAPIKey(key),
			fibe.WithDomain(domain),
			fibe.WithMaxRetries(2),
			fibe.WithRetryDelay(500*time.Millisecond, 5*time.Second),
			fibe.WithRequestHook(func(req *http.Request) error {
				req.Header.Set("X-Fibe-Test-Bypass", "true")
				return nil
			}),
		)
	})
	if setupErr != nil {
		t.Skipf("skipping integration test: %v", setupErr)
	}
	return admin
}

func ctx() context.Context {
	return testCtx
}

func ctxTimeout(d time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(testCtx, d)
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
	c := adminClient(t)
	key := os.Getenv("USER_B_API_KEY")
	if key == "" {
		t.Skip("USER_B_API_KEY not set — skipping multi-user test")
	}
	return c.WithKey(key)
}

func rateLimitClient(t *testing.T) *fibe.Client {
	t.Helper()
	_ = adminClient(t)
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
