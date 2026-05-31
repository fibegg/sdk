package integration

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/fibegg/sdk/fibe"
)

const (
	agentChatHealthProbeAttempts = 10
	agentChatHealthProbeDelay    = 2 * time.Second
	agentChatHealthProbeTimeout  = 3 * time.Second
	// Must exceed the server-side agent chat reachability window
	// (45 attempts * 2s delay) so the SDK doesn't give up first.
	agentChatProbeAttempts = 50
	agentChatProbeDelay    = 2 * time.Second
	agentChatProbeTimeout  = 5 * time.Second
)

func isTransientAgentChatError(err error) bool {
	if err == nil {
		return false
	}
	errText := err.Error()
	if err == context.DeadlineExceeded || strings.Contains(errText, context.DeadlineExceeded.Error()) {
		return true
	}
	apiErr, ok := err.(*fibe.APIError)
	if !ok {
		return false
	}
	if apiErr.StatusCode == http.StatusNotFound || apiErr.StatusCode == http.StatusConflict {
		return true
	}
	if apiErr.StatusCode != http.StatusUnprocessableEntity {
		return false
	}
	return strings.Contains(apiErr.Message, "AGENT_BUSY") ||
		(strings.Contains(errText, "AGENT_COMMUNICATION_FAILED") && strings.Contains(errText, "HTTP 5")) ||
		strings.Contains(apiErr.Message, "Agent unreachable") ||
		strings.Contains(apiErr.Message, "No running AgentChat") ||
		strings.Contains(apiErr.Message, "Agent is not currently running")
}

func sendAgentChatWithTimeout(c *fibe.Client, agentID int64, params *fibe.AgentChatParams, timeout time.Duration) (map[string]any, error) {
	reqCtx, cancel := ctxTimeout(timeout)
	defer cancel()
	return c.Agents.Chat(reqCtx, agentID, params)
}

func chatEventuallyAcceptedResult(c *fibe.Client, agentID int64, params *fibe.AgentChatParams, attempts int, delay, timeout time.Duration) (map[string]any, error) {
	var lastErr error
	for attempt := 1; attempt <= attempts; attempt++ {
		result, err := sendAgentChatWithTimeout(c, agentID, params, timeout)
		if err == nil {
			return result, nil
		}
		lastErr = err
		if !isTransientAgentChatError(err) {
			return nil, err
		}
		if attempt < attempts {
			time.Sleep(delay)
		}
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("agent chat did not become ready")
	}
	return nil, fmt.Errorf("agent chat did not become ready after %d attempts: %w", attempts, lastErr)
}

func chatEventuallyAccepted(c *fibe.Client, agentID int64, params *fibe.AgentChatParams, attempts int, delay, timeout time.Duration) error {
	_, err := chatEventuallyAcceptedResult(c, agentID, params, attempts, delay, timeout)
	return err
}

func waitForAgentChatHealth(chatURL string, attempts int, delay, timeout time.Duration) error {
	healthURL := strings.TrimRight(chatURL, "/") + "/api/health"
	client := &http.Client{Timeout: timeout}
	var lastErr error

	for attempt := 1; attempt <= attempts; attempt++ {
		resp, err := client.Get(healthURL)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				return nil
			}
			lastErr = fmt.Errorf("health returned status %d", resp.StatusCode)
		} else {
			lastErr = err
		}

		if attempt < attempts {
			time.Sleep(delay)
		}
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("agent chat health did not become ready")
	}

	return fmt.Errorf("agent chat health did not become ready after %d attempts: %w", attempts, lastErr)
}

func bootstrapOpencodeChat(t *testing.T, c *fibe.Client) *fibe.Agent {
	t.Helper()

	marqueeID := testMarqueeID(t)
	if marqueeID == 0 {
		t.Skip("set FIBE_TEST_MARQUEE_ID to bootstrap an agent chat")
	}

	secret, modelOptions, opencodeProvider, credentialSource := opencodeChatCredential(t)
	providerAPIKeyMode := true
	params := &fibe.AgentCreateParams{
		Name:               uniqueName("fx-agent"),
		Provider:           fibe.ProviderOpenCode,
		ProviderAPIKeyMode: &providerAPIKeyMode,
	}
	if modelOptions != "" {
		params.ModelOptions = &modelOptions
	}

	ag, err := c.Agents.Create(ctx(), params)
	requireNoError(t, err, "seed opencode agent")
	t.Cleanup(func() { c.Agents.Delete(ctx(), ag.ID) })
	t.Logf("using OpenCode credential from %s", credentialSource)

	authParams := &fibe.AgentAuthenticateParams{Token: &secret, OpenCodeProvider: &opencodeProvider}
	authenticated, err := c.Agents.AuthenticateWithParams(ctx(), ag.ID, authParams)
	requireNoError(t, err)
	if !authenticated.Authenticated {
		t.Fatal("expected authenticated opencode agent")
	}

	chat, err := c.Agents.StartChat(ctx(), ag.ID, marqueeID)
	requireNoError(t, err)
	if chat.ChatURL == nil || *chat.ChatURL == "" {
		t.Fatal("expected started chat to expose chat_url")
	}

	if err := waitForAgentChatHealth(*chat.ChatURL, agentChatHealthProbeAttempts, agentChatHealthProbeDelay, agentChatHealthProbeTimeout); err != nil {
		t.Logf("agent chat direct health probe did not become ready at %s; continuing with API retries: %v", *chat.ChatURL, err)
	}

	waitForAgentRuntimeStatus(t, c, ag.ID, agentRuntimeReadyTimeout, agentChatHealthProbeDelay, "agent chat runtime to become running and idle", func(status *fibe.AgentRuntimeStatus) bool {
		return status.Status == "running" && status.RuntimeReachable && status.Authenticated && !status.IsProcessing && status.QueueCount == 0
	})

	return ag
}

func opencodeChatCredential(t *testing.T) (string, string, string, string) {
	t.Helper()

	cases := []agentRuntimeMatrixCase{
		{
			name:               "OpenCode OpenRouter",
			provider:           fibe.ProviderOpenCode,
			providerAPIKeyMode: true,
			modelOptions:       "google/gemini-2.5-flash-lite",
			credentialEnv:      "FIBE_TEST_AGENT_OPENCODE_OPENROUTER_API_KEY",
			credentialAliases:  []string{"OPENCODE_OPENROUTER_KEY"},
			opencodeProvider:   "openrouter",
		},
		{
			name:               "OpenCode Anthropic",
			provider:           fibe.ProviderOpenCode,
			providerAPIKeyMode: true,
			modelOptions:       "anthropic/claude-haiku-4-5",
			credentialEnv:      "FIBE_TEST_AGENT_OPENCODE_ANTHROPIC_API_KEY",
			credentialAliases:  []string{"OPENCODE_ANTHROPIC_KEY"},
			opencodeProvider:   "anthropic",
		},
		{
			name:               "OpenCode OpenAI",
			provider:           fibe.ProviderOpenCode,
			providerAPIKeyMode: true,
			modelOptions:       "openai/gpt-5-mini",
			credentialEnv:      "FIBE_TEST_AGENT_OPENCODE_OPENAI_API_KEY",
			credentialAliases:  []string{"OPENCODE_OPENAI_KEY"},
			opencodeProvider:   "openai",
		},
		{
			name:               "OpenCode Gemini",
			provider:           fibe.ProviderOpenCode,
			providerAPIKeyMode: true,
			modelOptions:       "google/gemini-2.5-flash-lite",
			credentialEnv:      "FIBE_TEST_AGENT_OPENCODE_GEMINI_API_KEY",
			credentialAliases:  []string{"OPENCODE_GEMINI_KEY"},
			opencodeProvider:   "gemini",
		},
	}

	var names []string
	for _, tc := range cases {
		if secret, source := lookupAgentRuntimeCredential(tc); secret != "" {
			return secret, tc.modelOptions, tc.opencodeProvider, source
		}
		names = append(names, agentRuntimeCredentialEnvNames(tc)...)
	}

	t.Skipf("set one of %s to bootstrap an OpenCode agent chat", strings.Join(names, ", "))
	return "", "", "", ""
}

func TestAgents_Chat(t *testing.T) {
	c := userClient(t)

	t.Run("chat accepts text message", func(t *testing.T) {
		ag := bootstrapOpencodeChat(t, c)
		params := &fibe.AgentChatParams{
			Text: "hello from integration test",
		}
		err := chatEventuallyAccepted(c, ag.ID, params, agentChatProbeAttempts, agentChatProbeDelay, agentChatProbeTimeout)
		requireNoError(t, err)
	})

	t.Run("chat with empty text is rejected server-side", func(t *testing.T) {
		t.Parallel()
		ag := seedAgent(t, c, fibe.ProviderGemini)
		_, err := c.Agents.Chat(ctx(), ag.ID, &fibe.AgentChatParams{Text: ""})
		if err == nil {
			t.Error("expected error for empty text")
		}
	})
}

func TestAgents_Authenticate(t *testing.T) {
	t.Parallel()
	c := userClient(t)

	ag := seedAgent(t, c, fibe.ProviderGemini)

	t.Run("authenticate accepts arbitrary token without create-time validation", func(t *testing.T) {
		t.Parallel()
		bogus := "not-a-real-oauth-token"
		authenticated, err := c.Agents.Authenticate(ctx(), ag.ID, nil, &bogus)
		requireNoError(t, err)
		if !authenticated.Authenticated {
			t.Fatal("expected arbitrary credential to mark agent authenticated")
		}
	})

	t.Run("authenticate without code or token is idempotent noop or 4xx", func(t *testing.T) {
		t.Parallel()
		_, err := c.Agents.Authenticate(ctx(), ag.ID, nil, nil)
		// Backend behavior: may be a no-op (returns current agent) or 4xx validation.
		// What we want to guard against is a 500.
		if err != nil {
			if apiErr, ok := err.(*fibe.APIError); ok && apiErr.StatusCode >= 500 {
				t.Errorf("expected 2xx/4xx, got 5xx: %v", err)
			}
		}
	})
}

func TestAgents_RevokeGitHubToken(t *testing.T) {
	t.Parallel()
	c := userClient(t)

	ag := seedAgent(t, c, fibe.ProviderGemini)

	t.Run("revoke on fresh agent returns graceful response or error", func(t *testing.T) {
		_, err := c.Agents.RevokeGitHubToken(ctx(), ag.ID)
		// Either success (no token to revoke is fine) or structured 4xx
		if err != nil {
			if apiErr, ok := err.(*fibe.APIError); ok {
				if apiErr.StatusCode >= 500 {
					t.Errorf("expected 2xx/4xx, got 5xx: %v", err)
				}
			}
		}
	})
}

func TestAgents_UpdateActivity(t *testing.T) {
	t.Parallel()
	c := userClient(t)

	ag := seedAgent(t, c, fibe.ProviderGemini)

	t.Run("update activity with JSON and read back", func(t *testing.T) {
		payload := `[{"type":"run","ts":1,"status":"ok"}]`
		err := c.Agents.UpdateActivity(ctx(), ag.ID, payload)
		requireNoError(t, err)
		data, err := c.Agents.GetActivity(ctx(), ag.ID)
		requireNoError(t, err)
		if data == nil || data.Content == nil {
			t.Error("expected activity content echoed back")
		}
	})
}
