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
	agentRuntimeReadyTimeout   = 12 * time.Minute
	agentRuntimeIdleTimeout    = 8 * time.Minute
	agentRuntimeSyncTimeout    = 5 * time.Minute
	agentRuntimeSendTimeout    = 60 * time.Second
	agentRuntimeConversationID = "runtime-matrix"
)

type agentRuntimeMatrixCase struct {
	name               string
	provider           string
	providerAPIKeyMode bool
	modelOptions       string
	credentialEnv      string
	credentialAliases  []string
	opencodeProvider   string
	baseURL            string
}

func TestAgentRuntimeMatrix(t *testing.T) {
	skipThirdpartyIfDisabled(t)

	cases := []agentRuntimeMatrixCase{
		{
			name:               "Gemini OAuth",
			provider:           fibe.ProviderGemini,
			providerAPIKeyMode: false,
			modelOptions:       "gemini-2.5-flash-lite",
			credentialEnv:      "FIBE_TEST_AGENT_GEMINI_OAUTH_JSON",
			credentialAliases:  []string{"GEMINI_OAUTH_JSON"},
		},
		{
			name:               "Gemini API key",
			provider:           fibe.ProviderGemini,
			providerAPIKeyMode: true,
			modelOptions:       "gemini-2.5-flash-lite",
			credentialEnv:      "FIBE_TEST_AGENT_GEMINI_API_KEY",
			credentialAliases:  []string{"GEMINI_KEY", "GEMINI_API_KEY", "GOOGLE_GENERATIVE_AI_API_KEY", "GOOGLE_API_KEY"},
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
			credentialAliases:  []string{"OPENCODE_GEMINI_KEY", "GEMINI_API_KEY", "GOOGLE_GENERATIVE_AI_API_KEY", "GOOGLE_API_KEY"},
			opencodeProvider:   "gemini",
		},
	}

	caseFilters := agentRuntimeCaseFilters(os.Getenv("CHAT_E2E_CASE"))
	caseExcludeFilters := agentRuntimeCaseFilters(os.Getenv("CHAT_E2E_CASE_EXCEPT"))
	selectedCases := make([]agentRuntimeMatrixCase, 0, len(cases))
	configuredRows := 0
	for _, tc := range cases {
		if !agentRuntimeCaseSelected(tc, caseFilters, caseExcludeFilters) {
			continue
		}
		selectedCases = append(selectedCases, tc)
		if secret, _ := lookupAgentRuntimeCredential(tc); secret != "" {
			configuredRows++
		}
	}
	if len(selectedCases) == 0 {
		t.Fatalf("no agent runtime matrix rows selected by CHAT_E2E_CASE=%q CHAT_E2E_CASE_EXCEPT=%q", os.Getenv("CHAT_E2E_CASE"), os.Getenv("CHAT_E2E_CASE_EXCEPT"))
	}
	if configuredRows == 0 {
		t.Log("no FIBE_TEST_AGENT_* credential env vars configured for selected rows; each selected row will skip for missing credentials")
	}

	for _, tc := range selectedCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			secret, credentialSource := lookupAgentRuntimeCredential(tc)
			if secret == "" {
				t.Skipf("set one of %s to run this agent runtime matrix row", strings.Join(agentRuntimeCredentialEnvNames(tc), ", "))
			}

			t.Logf("using credential from %s", credentialSource)
			marqueeID := requiredAgentRuntimeMarqueeID(t)
			c := userClient(t)
			runAgentRuntimeMatrixCase(t, c, marqueeID, tc, secret)
		})
	}
}

func runAgentRuntimeMatrixCase(t *testing.T, c *fibe.Client, marqueeID int64, tc agentRuntimeMatrixCase, secret string) {
	t.Helper()
	agentRuntimeProgressf("%s: creating %s runtime (model=%s api_key_mode=%t)", tc.name, tc.provider, tc.modelOptions, tc.providerAPIKeyMode)

	syncEnabled := true
	syscheckEnabled := false
	providerAPIKeyMode := tc.providerAPIKeyMode
	apiKeyID := createAgentRuntimeAPIKey(t, c, tc)
	params := &fibe.AgentCreateParams{
		Name:               uniqueName("fx-agent-runtime"),
		Provider:           tc.provider,
		APIKeyID:           &apiKeyID,
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

	agentRuntimeProgressf("%s: authenticating agent %d", tc.name, agent.ID)
	authParams := &fibe.AgentAuthenticateParams{Token: &secret}
	if tc.opencodeProvider != "" {
		authParams.OpenCodeProvider = &tc.opencodeProvider
	}
	if tc.baseURL != "" {
		authParams.BaseURL = &tc.baseURL
	}
	authenticated, err := c.Agents.AuthenticateWithParams(ctx(), agent.ID, authParams)
	requireNoError(t, err, "authenticate agent")
	if !authenticated.Authenticated {
		t.Fatal("expected authenticated agent")
	}

	agentRuntimeProgressf("%s: starting chat for agent %d on marquee %d", tc.name, agent.ID, marqueeID)
	chat, err := c.Agents.StartChat(ctx(), agent.ID, marqueeID)
	requireNoError(t, err, "start chat")
	if chat.ID == 0 {
		t.Fatal("expected started chat ID")
	}

	agentRuntimeProgressf("%s: waiting for runtime to become running, authenticated, and idle", tc.name)
	waitForAgentRuntimeStatus(t, c, agent.ID, agentRuntimeReadyTimeout, 2*time.Second, "runtime to become running, authenticated, and idle", func(status *fibe.AgentRuntimeStatus) bool {
		return status.Status == "running" && status.RuntimeReachable && status.Authenticated && !status.IsProcessing && status.QueueCount == 0
	})

	conversationID := uniqueName(agentRuntimeConversationID)
	agentRuntimeProgressf("%s: creating runtime conversation %s", tc.name, conversationID)
	_, err = c.Agents.CreateConversationByIdentifier(ctx(), fmt.Sprint(agent.ID), &fibe.AgentConversationParams{
		ConversationID: conversationID,
		Title:          "Runtime matrix",
	})
	requireNoError(t, err, "create runtime conversation")

	firstPrompt := agentRuntimeInitialMessage()
	agentRuntimeProgressf("%s: sending initial prompt", tc.name)
	sendAgentRuntimeMessage(t, c, agent.ID, firstPrompt, conversationID)

	agentRuntimeProgressf("%s: verifying first assistant response synced", tc.name)
	waitForAgentRuntimeAssistantData(t, c, agent.ID, conversationID, agentRuntimeIdleTimeout, 1)
	waitForAgentRuntimeIdle(t, c, agent.ID, tc.name, "first assistant response")

	followupCount := agentRuntimeFollowupCount()
	for i := 2; i <= followupCount+1; i++ {
		agentRuntimeProgressf("%s: sending follow-up prompt %d/%d", tc.name, i-1, followupCount)
		sendAgentRuntimeMessage(t, c, agent.ID, fmt.Sprintf("Runtime matrix follow-up %d. Reply with one concise sentence.", i), conversationID)
		expectedAssistantMessages := i
		if minEntries := agentRuntimeMinEntries(); minEntries < expectedAssistantMessages {
			expectedAssistantMessages = minEntries
		}
		agentRuntimeProgressf("%s: verifying assistant response %d synced", tc.name, expectedAssistantMessages)
		waitForAgentRuntimeAssistantData(t, c, agent.ID, conversationID, agentRuntimeIdleTimeout, expectedAssistantMessages)
		waitForAgentRuntimeIdle(t, c, agent.ID, tc.name, fmt.Sprintf("assistant response %d", expectedAssistantMessages))
	}

	agentRuntimeProgressf("%s: waiting for synced messages/activity", tc.name)
	messages, activity := waitForAgentRuntimeAssistantData(t, c, agent.ID, conversationID, agentRuntimeSyncTimeout, agentRuntimeMinEntries())
	t.Logf("messages for %s:\n%s", tc.name, prettyAgentRuntimeJSON(messages.Content))
	t.Logf("activity for %s:\n%s", tc.name, prettyAgentRuntimeJSON(activity.Content))
	agentRuntimeProgressf("%s: completed", tc.name)
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

func createAgentRuntimeAPIKey(t *testing.T, c *fibe.Client, tc agentRuntimeMatrixCase) int64 {
	t.Helper()

	agentAccessible := true
	key, err := c.APIKeys.Create(ctx(), &fibe.APIKeyCreateParams{
		Label:           uniqueName("fx-agent-runtime-key"),
		Scopes:          userScopes(t),
		AgentAccessible: &agentAccessible,
	})
	requireNoError(t, err, "create agent runtime API key")
	if key.ID == nil || *key.ID <= 0 {
		t.Fatalf("expected API key ID for %s", tc.name)
	}

	t.Cleanup(func() {
		if err := c.APIKeys.Delete(ctx(), *key.ID); err != nil {
			t.Logf("cleanup agent runtime API key %d: %v", *key.ID, err)
		}
	})

	if !key.AgentAccessible {
		t.Fatalf("expected API key %d to be agent-accessible for %s", *key.ID, tc.name)
	}

	return *key.ID
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

func agentRuntimeCaseFilters(raw string) []string {
	var filters []string
	for _, part := range strings.Split(raw, ",") {
		filter := strings.TrimSpace(part)
		if filter != "" {
			filters = append(filters, filter)
		}
	}
	return filters
}

func agentRuntimeCaseSelected(tc agentRuntimeMatrixCase, includeFilters []string, excludeFilters []string) bool {
	if len(includeFilters) > 0 && !agentRuntimeCaseMatchesAny(tc, includeFilters) {
		return false
	}
	return !agentRuntimeCaseMatchesAny(tc, excludeFilters)
}

func agentRuntimeCaseMatchesAny(tc agentRuntimeMatrixCase, filters []string) bool {
	for _, filter := range filters {
		if agentRuntimeCaseMatches(tc, filter) {
			return true
		}
	}
	return false
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
		t.Fatal("set FIBE_TEST_MARQUEE_ID to run selected agent runtime matrix rows")
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

func sendAgentRuntimeMessage(t *testing.T, c *fibe.Client, agentID int64, text string, conversationID string) {
	t.Helper()

	params := &fibe.AgentChatParams{
		Text:           text,
		ConversationID: conversationID,
		BusyPolicy:     "queue",
	}
	result, err := chatEventuallyAcceptedResult(c, agentID, params, agentChatProbeAttempts, agentChatProbeDelay, agentRuntimeSendTimeout)
	if err != nil {
		diagnoseAgentRuntimeSendError(t, c, agentID, conversationID, err)
		requireNoError(t, err, "send runtime message")
	}
	t.Logf("runtime message accepted; response=%s", prettyAgentRuntimeJSON(result))
}

func diagnoseAgentRuntimeSendError(t *testing.T, c *fibe.Client, agentID int64, conversationID string, sendErr error) {
	t.Helper()

	params := &fibe.AgentDataParams{ConversationID: conversationID}
	reqCtx, cancel := ctxTimeout(10 * time.Second)
	messages, messagesErr := c.Agents.GetMessagesByIdentifierWithParams(reqCtx, fmt.Sprint(agentID), params)
	cancel()

	reqCtx, cancel = ctxTimeout(10 * time.Second)
	activity, activityErr := c.Agents.GetActivityByIdentifierWithParams(reqCtx, fmt.Sprint(agentID), params)
	cancel()

	reqCtx, cancel = ctxTimeout(10 * time.Second)
	providerTraffic, providerTrafficErr := c.Agents.GetProviderTrafficByIdentifierWithParams(reqCtx, fmt.Sprint(agentID), params)
	cancel()

	lastStatus, statusErr := agentRuntimeLatestStatus(c, agentID)
	statusLastError := ""
	if lastStatus != nil && lastStatus.LastError != nil {
		statusLastError = *lastStatus.LastError
	}

	failIfAgentRuntimeProviderUnavailable(
		t,
		sendErr.Error(),
		statusLastError,
		agentRuntimeDataContent(messages),
		agentRuntimeDataContent(activity),
	)
	failIfAgentRuntimeProviderTrafficUnavailable(t, agentRuntimeDataContent(providerTraffic))

	t.Logf(
		"send runtime message diagnostics after error: send_error=%v last_status=%s status_error=%v runtime_last_error=%q messages_error=%v activity_error=%v provider_traffic_error=%v provider_traffic_count=%d messages_excerpt=%q activity_excerpt=%q",
		sendErr,
		agentRuntimeStatusSummary(lastStatus),
		statusErr,
		statusLastError,
		messagesErr,
		activityErr,
		providerTrafficErr,
		agentRuntimeTopLevelCount(agentRuntimeDataContent(providerTraffic)),
		agentRuntimeDataExcerpt(messages),
		agentRuntimeDataExcerpt(activity),
	)
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
	lastProgressAt := time.Now()

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
		if time.Since(lastProgressAt) >= 30*time.Second {
			agentRuntimeProgressf(
				"still waiting for %s; last_status=%s last_error=%v",
				description,
				agentRuntimeStatusSummary(lastStatus),
				lastErr,
			)
			lastProgressAt = time.Now()
		}
		time.Sleep(pollInterval)
	}

	t.Fatalf("timed out waiting for %s; last_status=%s last_error=%v", description, prettyAgentRuntimeJSON(lastStatus), lastErr)
	return nil
}

func agentRuntimeProgressf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "[agent-runtime] "+format+"\n", args...)
}

func agentRuntimeStatusSummary(status *fibe.AgentRuntimeStatus) string {
	if status == nil {
		return "nil"
	}
	return fmt.Sprintf(
		"status=%s reachable=%t authenticated=%t processing=%t queue=%d last_error=%q",
		status.Status,
		status.RuntimeReachable,
		status.Authenticated,
		status.IsProcessing,
		status.QueueCount,
		agentRuntimeStatusLastError(status),
	)
}

func waitForAgentRuntimeIdle(t *testing.T, c *fibe.Client, agentID int64, caseName, phase string) {
	t.Helper()

	agentRuntimeProgressf("%s: waiting for runtime idle after %s", caseName, phase)
	waitForAgentRuntimeStatus(t, c, agentID, agentRuntimeIdleTimeout, 2*time.Second, fmt.Sprintf("runtime idle after %s", phase), func(status *fibe.AgentRuntimeStatus) bool {
		return status.Status == "running" && status.RuntimeReachable && status.Authenticated && !status.IsProcessing && status.QueueCount == 0
	})
}

func waitForAgentRuntimeAssistantData(t *testing.T, c *fibe.Client, agentID int64, conversationID string, timeout time.Duration, minAssistantMessages int) (*fibe.AgentData, *fibe.AgentData) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	var messages *fibe.AgentData
	var activity *fibe.AgentData
	var messagesErr error
	var activityErr error
	var providerTrafficErr error
	var messageCount int
	var assistantCount int
	var activityCount int
	var providerTrafficCount int

	for time.Now().Before(deadline) {
		params := &fibe.AgentDataParams{ConversationID: conversationID}
		reqCtx, cancel := ctxTimeout(10 * time.Second)
		messages, messagesErr = c.Agents.GetMessagesByIdentifierWithParams(reqCtx, fmt.Sprint(agentID), params)
		cancel()

		reqCtx, cancel = ctxTimeout(10 * time.Second)
		activity, activityErr = c.Agents.GetActivityByIdentifierWithParams(reqCtx, fmt.Sprint(agentID), params)
		cancel()

		reqCtx, cancel = ctxTimeout(10 * time.Second)
		providerTraffic, providerTrafficErr := c.Agents.GetProviderTrafficByIdentifierWithParams(reqCtx, fmt.Sprint(agentID), params)
		cancel()

		if messagesErr == nil && activityErr == nil {
			messageCount = agentRuntimeTopLevelCount(messages.Content)
			assistantCount = agentRuntimeAssistantMessageCount(messages.Content)
			activityCount = agentRuntimeTopLevelCount(activity.Content)
			providerTrafficCount = agentRuntimeTopLevelCount(agentRuntimeDataContent(providerTraffic))
			if assistantCount >= minAssistantMessages && activityCount > 0 {
				return messages, activity
			}
			failIfAgentRuntimeProviderUnavailable(t, messages.Content, activity.Content)
			if providerTrafficErr == nil {
				failIfAgentRuntimeProviderTrafficUnavailable(t, agentRuntimeDataContent(providerTraffic))
			}
		}

		time.Sleep(2 * time.Second)
	}

	lastStatus, statusErr := agentRuntimeLatestStatus(c, agentID)
	t.Fatalf(
		"timed out waiting for synced assistant output; message_count=%d assistant_count=%d activity_count=%d provider_traffic_count=%d messages_error=%v activity_error=%v provider_traffic_error=%v last_status=%s status_error=%v messages_excerpt=%q activity_excerpt=%q",
		messageCount,
		assistantCount,
		activityCount,
		providerTrafficCount,
		messagesErr,
		activityErr,
		providerTrafficErr,
		agentRuntimeStatusSummary(lastStatus),
		statusErr,
		agentRuntimeDataExcerpt(messages),
		agentRuntimeDataExcerpt(activity),
	)
	return nil, nil
}

func agentRuntimeLatestStatus(c *fibe.Client, agentID int64) (*fibe.AgentRuntimeStatus, error) {
	reqCtx, cancel := ctxTimeout(10 * time.Second)
	defer cancel()
	return c.Agents.RuntimeStatus(reqCtx, agentID)
}

func failIfAgentRuntimeProviderUnavailable(t *testing.T, contents ...any) {
	t.Helper()

	if agentRuntimeProviderQuotaExhausted(contents...) {
		t.Fatalf("provider quota or rate limit exhausted for this credential/model; excerpt=%q", agentRuntimeContentExcerpt(contents))
	}
	if agentRuntimeProviderCredentialsMissing(contents...) {
		t.Fatalf("provider credentials/auth unavailable; excerpt=%q", agentRuntimeContentExcerpt(contents))
	}
}

func failIfAgentRuntimeProviderTrafficUnavailable(t *testing.T, content any) {
	t.Helper()

	if agentRuntimeProviderQuotaExhausted(content) {
		t.Fatal("provider quota or rate limit exhausted for this credential/model; provider traffic contains quota/rate-limit marker")
	}
	if agentRuntimeProviderCredentialsMissing(content) {
		t.Fatal("provider credentials/auth unavailable; provider traffic contains auth/credential marker")
	}
}

func agentRuntimeProviderQuotaExhausted(contents ...any) bool {
	for _, content := range contents {
		text := strings.ToLower(agentRuntimeContentText(content))
		if text == "" {
			continue
		}
		for _, marker := range []string{
			"resource_exhausted",
			"model_capacity_exhausted",
			"terminalquotaerror",
			"quota exceeded",
			"quota will reset",
			"quota/rate limit exhausted",
			"quota or rate limit exhausted",
			"exhausted your daily quota",
			"exhausted your capacity",
			"rate limited",
			"currently overloaded",
			"code 429",
			"code: 429",
			"code\":429",
			"status 429",
			"statuscode\":429",
			"generate_content_free_tier_requests",
		} {
			if strings.Contains(text, marker) {
				return true
			}
		}
	}
	return false
}

func agentRuntimeProviderCredentialsMissing(contents ...any) bool {
	for _, content := range contents {
		text := strings.ToLower(agentRuntimeContentText(content))
		if text == "" {
			continue
		}
		for _, marker := range []string{
			"authentication failed",
			"credentials do not have access",
			"credentials are missing",
			"do not have access",
			"does not have access",
			"missing credentials",
			"missing api key",
			"api key is missing",
			"authentication required",
			"provider authentication failed",
		} {
			if strings.Contains(text, marker) {
				return true
			}
		}
	}
	return false
}

func TestAgentRuntimeProviderFailureDetection(t *testing.T) {
	t.Run("quota marker wins for Gemini terminal quota output", func(t *testing.T) {
		content := map[string]any{
			"activity_type": "error",
			"message":       "Provider turn failed",
			"details":       "TerminalQuotaError: You have exhausted your daily quota. code: 429",
		}

		if !agentRuntimeProviderQuotaExhausted(content) {
			t.Fatal("expected Gemini terminal quota output to be detected")
		}
	})

	t.Run("auth failure remains terminal after an earlier assistant response", func(t *testing.T) {
		content := []map[string]any{
			{
				"role": "assistant",
				"body": "docker e2e provider probe complete.",
			},
			{
				"activity_type": "error",
				"details":       "Authentication failed for Gemini: credentials are missing.",
			},
		}

		if !agentRuntimeProviderCredentialsMissing(content) {
			t.Fatal("expected runtime auth failure to be detected even after an assistant response")
		}
	})
}

func agentRuntimeDataExcerpt(data *fibe.AgentData) string {
	if data == nil {
		return ""
	}
	return agentRuntimeContentExcerpt(data.Content)
}

func agentRuntimeDataContent(data *fibe.AgentData) any {
	if data == nil {
		return nil
	}
	return data.Content
}

func agentRuntimeStatusLastError(status *fibe.AgentRuntimeStatus) string {
	if status == nil || status.LastError == nil {
		return ""
	}
	return *status.LastError
}

func agentRuntimeContentExcerpt(content any) string {
	text := strings.TrimSpace(agentRuntimeContentText(content))
	if text == "" {
		return ""
	}
	text = strings.Join(strings.Fields(text), " ")
	const limit = 1500
	runes := []rune(text)
	if len(runes) <= limit {
		return text
	}
	return string(runes[:limit]) + "...(truncated)"
}

func agentRuntimeContentText(content any) string {
	if content == nil {
		return ""
	}
	if text, ok := content.(string); ok {
		return text
	}
	bytes, err := json.Marshal(content)
	if err != nil {
		return fmt.Sprint(content)
	}
	return string(bytes)
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

func agentRuntimeAssistantMessageCount(content any) int {
	if content == nil {
		return 0
	}

	switch value := content.(type) {
	case []any:
		count := 0
		for _, item := range value {
			if agentRuntimeMessageRole(item) == "assistant" {
				count++
			}
		}
		return count
	case []map[string]any:
		count := 0
		for _, item := range value {
			if agentRuntimeMessageRole(item) == "assistant" {
				count++
			}
		}
		return count
	case []map[string]string:
		count := 0
		for _, item := range value {
			if agentRuntimeMessageRole(item) == "assistant" {
				count++
			}
		}
		return count
	case string:
		var decoded any
		if json.Unmarshal([]byte(value), &decoded) == nil {
			return agentRuntimeAssistantMessageCount(decoded)
		}
	}

	return 0
}

func agentRuntimeMessageRole(item any) string {
	switch value := item.(type) {
	case map[string]any:
		if role, ok := value["role"].(string); ok {
			return role
		}
	case map[string]string:
		return value["role"]
	}
	return ""
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
