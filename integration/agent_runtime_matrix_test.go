package integration

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/fibegg/sdk/fibe"
)

const (
	agentRuntimeReadyTimeout      = 12 * time.Minute
	agentRuntimeProcessingTimeout = 2 * time.Minute
	agentRuntimeIdleTimeout       = 8 * time.Minute
	agentRuntimeSyncTimeout       = 5 * time.Minute
)

type agentRuntimeMatrixCase struct {
	name               string
	provider           string
	providerAPIKeyMode bool
	modelOptions       string
	credentialEnv      string
	credentialAliases  []string
}

func TestAgentRuntimeMatrix(t *testing.T) {
	cases := []agentRuntimeMatrixCase{
		{
			name:               "Gemini OAuth",
			provider:           fibe.ProviderGemini,
			providerAPIKeyMode: false,
			modelOptions:       "pro",
			credentialEnv:      "FIBE_TEST_AGENT_GEMINI_OAUTH_JSON",
			credentialAliases:  []string{"GEMINI_OAUTH_JSON"},
		},
		{
			name:               "Gemini API key",
			provider:           fibe.ProviderGemini,
			providerAPIKeyMode: true,
			modelOptions:       "flash-lite",
			credentialEnv:      "FIBE_TEST_AGENT_GEMINI_API_KEY",
			credentialAliases:  []string{"GEMINI_KEY"},
		},
		{
			name:               "Claude manual",
			provider:           fibe.ProviderClaudeCode,
			providerAPIKeyMode: false,
			modelOptions:       "haiku",
			credentialEnv:      "FIBE_TEST_AGENT_CLAUDE_CODE_OAUTH_TOKEN",
			credentialAliases:  []string{"CLAUDE_CODE_OAUTH_TOKEN"},
		},
		{
			name:               "Claude API key",
			provider:           fibe.ProviderClaudeCode,
			providerAPIKeyMode: true,
			modelOptions:       "haiku",
			credentialEnv:      "FIBE_TEST_AGENT_ANTHROPIC_API_KEY",
			credentialAliases:  []string{"ANTHROPIC_KEY"},
		},
		{
			name:               "Codex auth JSON",
			provider:           fibe.ProviderOpenAICodex,
			providerAPIKeyMode: false,
			modelOptions:       "gpt-5.4-mini",
			credentialEnv:      "FIBE_TEST_AGENT_CODEX_AUTH_JSON",
			credentialAliases:  []string{"CODEX_AUTH_JSON"},
		},
		{
			name:               "Codex API key",
			provider:           fibe.ProviderOpenAICodex,
			providerAPIKeyMode: true,
			modelOptions:       "gpt-5.4-mini",
			credentialEnv:      "FIBE_TEST_AGENT_OPENAI_API_KEY",
			credentialAliases:  []string{"OPENAI_KEY"},
		},
		{
			name:               "Cursor API key",
			provider:           fibe.ProviderCursor,
			providerAPIKeyMode: true,
			credentialEnv:      "FIBE_TEST_AGENT_CURSOR_API_KEY",
			credentialAliases:  []string{"CURSOR_KEY"},
		},
		{
			name:               "OpenCode OpenRouter",
			provider:           fibe.ProviderOpenCode,
			providerAPIKeyMode: true,
			modelOptions:       "deepseek/deepseek-chat-v3.1",
			credentialEnv:      "FIBE_TEST_AGENT_OPENCODE_OPENROUTER_API_KEY",
			credentialAliases:  []string{"OPENCODE_OPENROUTER_KEY"},
		},
		{
			name:               "OpenCode Anthropic",
			provider:           fibe.ProviderOpenCode,
			providerAPIKeyMode: true,
			modelOptions:       "anthropic/claude-sonnet-4",
			credentialEnv:      "FIBE_TEST_AGENT_OPENCODE_ANTHROPIC_API_KEY",
			credentialAliases:  []string{"OPENCODE_ANTHROPIC_KEY"},
		},
		{
			name:               "OpenCode OpenAI",
			provider:           fibe.ProviderOpenCode,
			providerAPIKeyMode: true,
			modelOptions:       "openai/gpt-4.1",
			credentialEnv:      "FIBE_TEST_AGENT_OPENCODE_OPENAI_API_KEY",
			credentialAliases:  []string{"OPENCODE_OPENAI_KEY"},
		},
		{
			name:               "OpenCode Gemini",
			provider:           fibe.ProviderOpenCode,
			providerAPIKeyMode: true,
			modelOptions:       "google/gemini-2.5-pro",
			credentialEnv:      "FIBE_TEST_AGENT_OPENCODE_GEMINI_API_KEY",
			credentialAliases:  []string{"OPENCODE_GEMINI_KEY"},
		},
	}

	caseFilter := strings.TrimSpace(os.Getenv("CHAT_E2E_CASE"))
	configuredRows := 0
	for _, tc := range cases {
		if caseFilter != "" && !agentRuntimeCaseMatches(tc, caseFilter) {
			continue
		}
		if secret, _ := lookupAgentRuntimeCredential(tc); secret != "" {
			configuredRows++
		}
	}
	if configuredRows == 0 {
		t.Log("no FIBE_TEST_AGENT_* credential env vars configured; all agent runtime matrix rows will skip")
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			if caseFilter != "" && !agentRuntimeCaseMatches(tc, caseFilter) {
				t.Skipf("filtered by CHAT_E2E_CASE=%q", caseFilter)
			}

			secret, credentialSource := lookupAgentRuntimeCredential(tc)
			if secret == "" {
				t.Skipf("set one of %s to run this agent runtime matrix row", strings.Join(agentRuntimeCredentialEnvNames(tc), ", "))
			}

			t.Logf("using credential from %s", credentialSource)
			marqueeID := requiredAgentRuntimeMarqueeID(t)
			c := adminClient(t)
			runAgentRuntimeMatrixCase(t, c, marqueeID, tc, secret)
		})
	}
}

func runAgentRuntimeMatrixCase(t *testing.T, c *fibe.Client, marqueeID int64, tc agentRuntimeMatrixCase, secret string) {
	t.Helper()

	syncEnabled := true
	syscheckEnabled := false
	providerAPIKeyMode := tc.providerAPIKeyMode
	params := &fibe.AgentCreateParams{
		Name:               uniqueName("fx-agent-runtime"),
		Provider:           tc.provider,
		SyncEnabled:        &syncEnabled,
		SyscheckEnabled:    &syscheckEnabled,
		ProviderAPIKeyMode: &providerAPIKeyMode,
	}
	if tc.modelOptions != "" {
		modelOptions := tc.modelOptions
		params.ModelOptions = &modelOptions
	}

	agent, err := c.Agents.Create(ctx(), params)
	requireNoError(t, err, "create agent")
	t.Cleanup(func() {
		cleanupAgentRuntimeMatrixCase(t, c, agent.ID)
	})

	authenticated, err := c.Agents.Authenticate(ctx(), agent.ID, nil, &secret)
	requireNoError(t, err, "authenticate agent")
	if !authenticated.Authenticated {
		t.Fatal("expected authenticated agent")
	}

	chat, err := c.Agents.StartChat(ctx(), agent.ID, marqueeID)
	requireNoError(t, err, "start chat")
	if chat.ID == 0 {
		t.Fatal("expected started chat ID")
	}

	waitForAgentRuntimeStatus(t, c, agent.ID, agentRuntimeReadyTimeout, 2*time.Second, "runtime to become running and idle", func(status *fibe.AgentRuntimeStatus) bool {
		return status.Status == "running" && status.RuntimeReachable && !status.IsProcessing && status.QueueCount == 0
	})

	firstPrompt := agentRuntimeInitialMessage()
	sendAgentRuntimeMessage(t, c, agent.ID, firstPrompt)

	waitForAgentRuntimeStatus(t, c, agent.ID, agentRuntimeProcessingTimeout, 250*time.Millisecond, "runtime to report processing", func(status *fibe.AgentRuntimeStatus) bool {
		return status.Status == "running" && status.RuntimeReachable && status.IsProcessing
	})
	waitForAgentRuntimeStatus(t, c, agent.ID, agentRuntimeIdleTimeout, 2*time.Second, "runtime to return to idle", func(status *fibe.AgentRuntimeStatus) bool {
		return status.Status == "running" && status.RuntimeReachable && !status.IsProcessing && status.QueueCount == 0
	})

	followupCount := agentRuntimeFollowupCount()
	for i := 2; i <= followupCount+1; i++ {
		sendAgentRuntimeMessage(t, c, agent.ID, fmt.Sprintf("Runtime matrix follow-up %d. Reply with one concise sentence.", i))
		waitForAgentRuntimeStatus(t, c, agent.ID, agentRuntimeIdleTimeout, 2*time.Second, fmt.Sprintf("runtime to return to idle after prompt %d", i), func(status *fibe.AgentRuntimeStatus) bool {
			return status.Status == "running" && status.RuntimeReachable && !status.IsProcessing && status.QueueCount == 0
		})
	}

	messages, activity := waitForAgentRuntimeSyncedData(t, c, agent.ID, agentRuntimeSyncTimeout, agentRuntimeMinEntries())
	t.Logf("messages for %s:\n%s", tc.name, prettyAgentRuntimeJSON(messages.Content))
	t.Logf("activity for %s:\n%s", tc.name, prettyAgentRuntimeJSON(activity.Content))
}

func agentRuntimeCredentialEnvNames(tc agentRuntimeMatrixCase) []string {
	names := []string{tc.credentialEnv}
	names = append(names, tc.credentialAliases...)
	return names
}

func lookupAgentRuntimeCredential(tc agentRuntimeMatrixCase) (string, string) {
	for _, envName := range agentRuntimeCredentialEnvNames(tc) {
		value := strings.TrimSpace(os.Getenv(envName))
		if value != "" {
			return value, envName
		}
	}
	return "", ""
}

func agentRuntimeCaseMatches(tc agentRuntimeMatrixCase, filter string) bool {
	normalizedFilter := strings.ToLower(strings.TrimSpace(filter))
	if normalizedFilter == "" {
		return true
	}
	haystack := strings.ToLower(strings.Join([]string{
		tc.name,
		tc.provider,
		tc.modelOptions,
		tc.credentialEnv,
		strings.Join(tc.credentialAliases, " "),
	}, " "))
	return strings.Contains(haystack, normalizedFilter)
}

func agentRuntimeInitialMessage() string {
	for _, envName := range []string{"FIBE_TEST_AGENT_MESSAGE", "MESSAGE"} {
		message := strings.TrimSpace(os.Getenv(envName))
		if message != "" {
			return message
		}
	}

	return strings.Join([]string{
		"You are running inside an integration test for the agent runtime matrix.",
		"If a shell or tool call is available, run a harmless step such as `pwd && printf runtime-matrix-probe && sleep 5`.",
		"Do not modify files. Finish with one short sentence that says runtime matrix probe complete.",
	}, " ")
}

func agentRuntimeFollowupCount() int {
	return agentRuntimeEnvInt([]string{"FIBE_TEST_AGENT_FOLLOWUPS", "CHAT_E2E_FOLLOWUPS"}, 5)
}

func agentRuntimeMinEntries() int {
	return agentRuntimeEnvInt([]string{"FIBE_TEST_AGENT_MIN_ENTRIES", "CHAT_E2E_MIN_ENTRIES"}, 5)
}

func agentRuntimeEnvInt(envNames []string, fallback int) int {
	for _, envName := range envNames {
		raw := strings.TrimSpace(os.Getenv(envName))
		if raw == "" {
			continue
		}
		value, err := strconv.Atoi(raw)
		if err == nil && value >= 0 {
			return value
		}
	}
	return fallback
}

func requiredAgentRuntimeMarqueeID(t *testing.T) int64 {
	t.Helper()

	raw := strings.TrimSpace(os.Getenv("FIBE_TEST_MARQUEE_ID"))
	if raw == "" {
		t.Skip("set FIBE_TEST_MARQUEE_ID to run agent runtime matrix rows")
	}
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		t.Fatalf("invalid FIBE_TEST_MARQUEE_ID %q: %v", raw, err)
	}
	if id <= 0 {
		t.Fatalf("invalid FIBE_TEST_MARQUEE_ID %q: must be greater than zero", raw)
	}
	return id
}

func sendAgentRuntimeMessage(t *testing.T, c *fibe.Client, agentID int64, text string) {
	t.Helper()

	params := &fibe.AgentChatParams{Text: text}
	err := chatEventuallyAccepted(c, agentID, params, 10, 2*time.Second, 10*time.Second)
	requireNoError(t, err, "send runtime message")
}

func waitForAgentRuntimeStatus(
	t *testing.T,
	c *fibe.Client,
	agentID int64,
	timeout time.Duration,
	pollInterval time.Duration,
	description string,
	predicate func(*fibe.AgentRuntimeStatus) bool,
) *fibe.AgentRuntimeStatus {
	t.Helper()

	deadline := time.Now().Add(timeout)
	var lastStatus *fibe.AgentRuntimeStatus
	var lastErr error

	for time.Now().Before(deadline) {
		reqCtx, cancel := ctxTimeout(10 * time.Second)
		status, err := c.Agents.RuntimeStatus(reqCtx, agentID)
		cancel()
		if err == nil {
			lastStatus = status
			lastErr = nil
			if predicate(status) {
				return status
			}
		} else {
			lastErr = err
		}
		time.Sleep(pollInterval)
	}

	t.Fatalf("timed out waiting for %s; last_status=%s last_error=%v", description, prettyAgentRuntimeJSON(lastStatus), lastErr)
	return nil
}

func waitForAgentRuntimeSyncedData(t *testing.T, c *fibe.Client, agentID int64, timeout time.Duration, minEntries int) (*fibe.AgentData, *fibe.AgentData) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	var messages *fibe.AgentData
	var activity *fibe.AgentData
	var messagesErr error
	var activityErr error
	var messageCount int
	var activityCount int

	for time.Now().Before(deadline) {
		reqCtx, cancel := ctxTimeout(10 * time.Second)
		messages, messagesErr = c.Agents.GetMessages(reqCtx, agentID)
		cancel()

		reqCtx, cancel = ctxTimeout(10 * time.Second)
		activity, activityErr = c.Agents.GetActivity(reqCtx, agentID)
		cancel()

		if messagesErr == nil && activityErr == nil {
			messageCount = agentRuntimeTopLevelCount(messages.Content)
			activityCount = agentRuntimeTopLevelCount(activity.Content)
			if messageCount > minEntries && activityCount > minEntries {
				return messages, activity
			}
		}

		time.Sleep(2 * time.Second)
	}

	t.Fatalf(
		"timed out waiting for synced messages/activity; message_count=%d activity_count=%d messages_error=%v activity_error=%v",
		messageCount,
		activityCount,
		messagesErr,
		activityErr,
	)
	return nil, nil
}

func agentRuntimeTopLevelCount(content any) int {
	if content == nil {
		return 0
	}

	switch value := content.(type) {
	case []any:
		return len(value)
	case []map[string]any:
		return len(value)
	case []map[string]string:
		return len(value)
	case string:
		var decoded any
		if json.Unmarshal([]byte(value), &decoded) == nil {
			return agentRuntimeTopLevelCount(decoded)
		}
		return 0
	}

	reflected := reflect.ValueOf(content)
	switch reflected.Kind() {
	case reflect.Array, reflect.Slice:
		return reflected.Len()
	default:
		return 0
	}
}

func cleanupAgentRuntimeMatrixCase(t *testing.T, c *fibe.Client, agentID int64) {
	t.Helper()

	reqCtx, cancel := ctxTimeout(5 * time.Minute)
	_, err := c.Agents.PurgeChat(reqCtx, agentID)
	cancel()
	if err != nil {
		if apiErr, ok := err.(*fibe.APIError); !ok || apiErr.StatusCode != 404 {
			t.Logf("cleanup purge_chat failed for agent %d: %v", agentID, err)
		}
	}

	reqCtx, cancel = ctxTimeout(2 * time.Minute)
	err = c.Agents.Delete(reqCtx, agentID)
	cancel()
	if err != nil {
		if apiErr, ok := err.(*fibe.APIError); !ok || apiErr.StatusCode != 404 {
			t.Logf("cleanup delete failed for agent %d: %v", agentID, err)
		}
	}
}

func prettyAgentRuntimeJSON(value any) string {
	if value == nil {
		return "null"
	}
	bytes, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Sprintf("%#v", value)
	}
	return string(bytes)
}
