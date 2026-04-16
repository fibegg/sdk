package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
)

func TestRequestID_ReturnedInHeader(t *testing.T) {
	t.Parallel()
	c := adminClient(t)
	apiKey := os.Getenv("FIBE_API_KEY")
	if apiKey == "" {
		t.Skip("FIBE_API_KEY not set")
	}

	url := c.BaseURL() + "/api/me"

	req, err := http.NewRequestWithContext(context.Background(), "GET", url, nil)
	requireNoError(t, err)
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := http.DefaultClient.Do(req)
	requireNoError(t, err)
	defer resp.Body.Close()

	requestID := resp.Header.Get("X-Request-Id")
	if requestID == "" {
		t.Error("expected X-Request-Id response header")
	}
}

func TestIdempotencyKey(t *testing.T) {
	t.Parallel()
	c := adminClient(t)
	apiKey := os.Getenv("FIBE_API_KEY")
	if apiKey == "" {
		t.Skip("FIBE_API_KEY not set")
	}

	url := c.BaseURL() + "/api/agents"
	agentName := uniqueName("idempotent-agent")
	body := fmt.Sprintf(`{"agent":{"name":"%s","provider":"gemini"}}`, agentName)
	idempotencyKey := uniqueName("idem-key")

	doReq := func() (*http.Response, error) {
		req, err := http.NewRequestWithContext(ctx(), "POST", url, strings.NewReader(body))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+apiKey)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", idempotencyKey)
		return http.DefaultClient.Do(req)
	}

	resp1, err := doReq()
	requireNoError(t, err)
	body1, _ := io.ReadAll(resp1.Body)
	resp1.Body.Close()

	if resp1.StatusCode != 201 {
		t.Fatalf("first request: expected 201, got %d — %s", resp1.StatusCode, string(body1))
	}

	var agent1 struct {
		ID int64 `json:"id"`
	}
	json.Unmarshal(body1, &agent1)
	t.Cleanup(func() { c.Agents.Delete(ctx(), agent1.ID) })

	resp2, err := doReq()
	requireNoError(t, err)
	body2, _ := io.ReadAll(resp2.Body)
	resp2.Body.Close()

	replayed := resp2.Header.Get("X-Idempotent-Replayed")
	if replayed != "true" {
		t.Errorf("expected X-Idempotent-Replayed: true on second request, got headers: %v, body: %s", resp2.Header, string(body2))
	}

	if resp2.StatusCode != 201 {
		t.Errorf("replayed request: expected same 201 status, got %d", resp2.StatusCode)
	}
}

func jsonBody(s string) io.Reader {
	return bytes.NewBufferString(s)
}
