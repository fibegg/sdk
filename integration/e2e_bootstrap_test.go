package integration

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
)

const (
	defaultE2EAdminAPIKey = "fibe_test_secret_admin"
	defaultSDKAPIKey      = "fibe_test_secret_sdk"
	defaultSDKUserBAPIKey = "fibe_test_secret_sdk_user_b"
	defaultSDKRateAPIKey  = "fibe_test_secret_sdk_rate_limit"
	e2eUsernameMaxLength  = 39
	seededPropNamePrefix  = "sdk-seed-prop"
	seededPropRepoPrefix  = "https://github.com/fibegg/sdk-sdk-seed"
	seededPropEnvFile     = ".env.example"
)

func TestMain(m *testing.M) {
	cleanup, err := bootstrapDockerE2E()
	if err != nil {
		fmt.Fprintf(os.Stderr, "SDK docker e2e bootstrap failed: %v\n", err)
		os.Exit(1)
	}

	code := m.Run()
	if cleanup != nil {
		if err := cleanup(); err != nil {
			fmt.Fprintf(os.Stderr, "SDK docker e2e cleanup failed: %v\n", err)
		}
	}
	os.Exit(code)
}

func bootstrapDockerE2E() (func() error, error) {
	if !envBool("FIBE_E2E_BOOTSTRAP") && !envBool("SDK_E2E_BOOTSTRAP") {
		return nil, nil
	}

	baseURL := strings.TrimRight(firstEnv("FIBE_DOMAIN", "FIBE_URL", "FIBE_BASE_URL"), "/")
	if baseURL == "" {
		baseURL = "http://localhost:3000"
	}
	adminToken := envDefault("E2E_ADMIN_API_KEY", defaultE2EAdminAPIKey)
	sshKey, err := dockerE2ESSHPrivateKey()
	if err != nil {
		return nil, err
	}

	client := &e2eBootstrapClient{baseURL: baseURL, adminToken: adminToken, http: &http.Client{Timeout: 30 * time.Second}}
	if err := client.waitForFibe(); err != nil {
		return nil, err
	}
	if _, err := client.currentPlayer(adminToken); err != nil {
		return nil, fmt.Errorf("validate e2e admin API key: %w", err)
	}

	runID := e2eRunID()
	tokenSuffix := strings.ReplaceAll(runID, "-", "_")
	token := envDefault("E2E_SDK_API_KEY", defaultSDKAPIKey+"_"+tokenSuffix)
	userBToken := envDefault("E2E_SDK_USER_B_API_KEY", defaultSDKUserBAPIKey+"_"+tokenSuffix)
	rateToken := envDefault("E2E_SDK_RATE_LIMIT_API_KEY", defaultSDKRateAPIKey+"_"+tokenSuffix)
	identities, err := client.ensureIdentities([]map[string]any{
		e2eIdentity("primary", compactE2EUsername("e2e-sdk", runID), token, "sdk-e2e-"+runID+"-key", 0),
		e2eIdentity("user_b", compactE2EUsername("e2e-sdk", runID, "teammate"), userBToken, "sdk-e2e-"+runID+"-user-b-key", 0),
		e2eIdentity("rate_limit", compactE2EUsername("e2e-sdk", runID, "rate-limit"), rateToken, "sdk-e2e-"+runID+"-rate-limit-key", 2),
	})
	if err != nil {
		return nil, err
	}

	if _, err := client.ensurePropFixture(identities["primary"].PlayerID, runID); err != nil {
		return nil, err
	}

	rootDomain := envDefault("TEST_MARQUEE_ROOT_DOMAIN", dockerE2ERootDomain())
	marquee, err := client.ensureMarquee(identities["primary"].PlayerID, map[string]any{
		"name":                    "sdk-e2e-" + runID,
		"host":                    envDefault("TEST_MARQUEE_HOST", "dind-sdk"),
		"port":                    envDefault("TEST_MARQUEE_PORT", "22"),
		"user":                    envDefault("TEST_MARQUEE_USER", "root"),
		"ssh_private_key":         sshKey,
		"domains_input":           rootDomain,
		"status":                  "active",
		"billing_requested_until": time.Now().Add(7 * 24 * time.Hour).UTC().Format(time.RFC3339),
		"https_enabled":           false,
		"tls_certificate_source":  "provided",
	})
	if err != nil {
		return nil, err
	}
	sshKeyMarquee, err := client.ensureMarquee(identities["primary"].PlayerID, map[string]any{
		"name":                    "sdk-e2e-ssh-key-" + runID,
		"host":                    envDefault("TEST_MARQUEE_HOST", "dind-sdk"),
		"port":                    envDefault("TEST_SSH_KEY_MARQUEE_PORT", "2222"),
		"user":                    envDefault("TEST_MARQUEE_USER", "root"),
		"ssh_private_key":         sshKey,
		"domains_input":           rootDomain,
		"status":                  "disabled",
		"billing_requested_until": time.Now().Add(7 * 24 * time.Hour).UTC().Format(time.RFC3339),
		"https_enabled":           false,
		"tls_certificate_source":  "provided",
	})
	if err != nil {
		return nil, err
	}

	if err := client.attachMarquee(identities["user_b"].PlayerID, marquee.ID, false); err != nil {
		return nil, err
	}
	player, err := client.currentPlayer(token)
	if err != nil {
		return nil, err
	}
	_ = player

	os.Setenv("E2E_RUN_ID", runID)
	os.Setenv("FIBE_DOMAIN", baseURL)
	os.Setenv("FIBE_API_KEY", token)
	os.Setenv("FIBE_API_KEY_ID", strconv.FormatInt(identities["primary"].APIKeyID, 10))
	os.Setenv("FIBE_TEST_MARQUEE_ID", strconv.FormatInt(marquee.ID, 10))
	os.Setenv("FIBE_TEST_SSH_KEY_MARQUEE_ID", strconv.FormatInt(sshKeyMarquee.ID, 10))
	os.Setenv("USER_B_API_KEY", userBToken)
	os.Setenv("USER_B_USERNAME", identities["user_b"].Username)
	os.Setenv("RATE_LIMIT_TEST_KEY", rateToken)
	os.Setenv("FIBE_ADMIN_API_KEY", adminToken)

	return func() error {
		var errs []string
		for _, id := range []int64{marquee.ID, sshKeyMarquee.ID} {
			if err := client.deactivateMarquee(id); err != nil {
				errs = append(errs, err.Error())
			}
		}
		if len(errs) > 0 {
			return errors.New(strings.Join(errs, "; "))
		}
		return nil
	}, nil
}

type e2eBootstrapClient struct {
	baseURL    string
	adminToken string
	http       *http.Client
}

type e2eIdentityResult struct {
	PlayerID int64  `json:"player_id"`
	Username string `json:"username"`
	APIKeyID int64  `json:"api_key_id"`
}

type e2eResource struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

func (c *e2eBootstrapClient) waitForFibe() error {
	deadline := time.Now().Add(time.Duration(envInt("FIBE_E2E_BOOTSTRAP_TIMEOUT_SECONDS", 600)) * time.Second)
	var lastErr error
	for time.Now().Before(deadline) {
		resp, err := c.http.Get(c.baseURL + "/up")
		if err == nil && resp.StatusCode >= 200 && resp.StatusCode < 300 {
			resp.Body.Close()
			return nil
		}
		if resp != nil {
			lastErr = fmt.Errorf("HTTP %d", resp.StatusCode)
			resp.Body.Close()
		} else {
			lastErr = err
		}
		time.Sleep(time.Second)
	}
	return fmt.Errorf("Fibe not ready at %s/up: %w", c.baseURL, lastErr)
}

func (c *e2eBootstrapClient) currentPlayer(token string) (e2eResource, error) {
	var player e2eResource
	err := c.requestJSON(http.MethodGet, "/api/me", token, nil, &player, http.StatusOK)
	if err != nil {
		return player, err
	}
	if player.ID == 0 {
		return player, errors.New("/api/me returned no player id")
	}
	return player, nil
}

func (c *e2eBootstrapClient) ensureIdentities(identities []map[string]any) (map[string]e2eIdentityResult, error) {
	var payload struct {
		Success    bool                `json:"success"`
		Identities []e2eIdentityResult `json:"identities"`
	}
	if err := c.requestJSON(http.MethodPost, "/e2e_backdoor/identity", c.adminToken, map[string]any{"identities": identities}, &payload, http.StatusOK); err != nil {
		return nil, err
	}
	if !payload.Success {
		return nil, errors.New("identity backdoor returned success=false")
	}
	if len(payload.Identities) != len(identities) {
		return nil, errors.New("identity backdoor returned unexpected identity count")
	}
	result := make(map[string]e2eIdentityResult, len(identities))
	for i, identity := range identities {
		role, _ := identity["role"].(string)
		result[role] = payload.Identities[i]
	}
	return result, nil
}

func (c *e2eBootstrapClient) ensureMarquee(playerID int64, attrs map[string]any) (e2eResource, error) {
	var payload struct {
		Success bool        `json:"success"`
		Marquee e2eResource `json:"marquee"`
	}
	err := c.requestJSON(http.MethodPost, "/e2e_backdoor/marquee", c.adminToken, map[string]any{
		"player_id": playerID,
		"marquee":   attrs,
		"funded":    true,
	}, &payload, http.StatusOK)
	if err != nil {
		return e2eResource{}, err
	}
	if !payload.Success {
		return e2eResource{}, errors.New("marquee backdoor returned success=false")
	}
	return payload.Marquee, nil
}

func (c *e2eBootstrapClient) attachMarquee(playerID, marqueeID int64, funded bool) error {
	var payload struct {
		Success bool `json:"success"`
	}
	err := c.requestJSON(http.MethodPost, "/e2e_backdoor/ensure_player_marquee", c.adminToken, map[string]any{
		"player_id":  playerID,
		"marquee_id": marqueeID,
		"funded":     funded,
	}, &payload, http.StatusOK)
	if err != nil {
		return err
	}
	if !payload.Success {
		return errors.New("ensure_player_marquee returned success=false")
	}
	return nil
}

func (c *e2eBootstrapClient) ensurePropFixture(playerID int64, runID string) (e2eResource, error) {
	var payload struct {
		Success bool        `json:"success"`
		Prop    e2eResource `json:"prop"`
		Branch  string      `json:"branch"`
	}
	err := c.requestJSON(http.MethodPost, "/e2e_backdoor/prop_fixture", c.adminToken, map[string]any{
		"player_id":      playerID,
		"name":           seededPropNamePrefix + "-" + runID,
		"repository_url": seededPropRepoPrefix + "-" + runID,
		"default_branch": "main",
	}, &payload, http.StatusOK)
	if err != nil {
		return e2eResource{}, err
	}
	if !payload.Success {
		return e2eResource{}, errors.New("prop fixture backdoor returned success=false")
	}
	if payload.Prop.ID == 0 || payload.Branch == "" {
		return e2eResource{}, errors.New("prop fixture backdoor returned incomplete fixture")
	}
	return payload.Prop, nil
}

func (c *e2eBootstrapClient) deactivateMarquee(marqueeID int64) error {
	var payload struct {
		Success bool `json:"success"`
	}
	err := c.requestJSON(http.MethodPost, "/e2e_backdoor/deactivate_marquee", c.adminToken, map[string]any{
		"marquee_id": marqueeID,
	}, &payload, http.StatusOK)
	if err != nil {
		return err
	}
	if !payload.Success {
		return errors.New("deactivate_marquee returned success=false")
	}
	return nil
}

func (c *e2eBootstrapClient) requestJSON(method, path, token string, body any, out any, okStatuses ...int) error {
	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(raw)
	}
	req, err := http.NewRequest(method, c.baseURL+path, reader)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	for _, status := range okStatuses {
		if resp.StatusCode == status {
			if len(raw) == 0 || out == nil {
				return nil
			}
			return json.Unmarshal(raw, out)
		}
	}
	return fmt.Errorf("%s %s failed with HTTP %d: %s", method, path, resp.StatusCode, string(raw))
}

func e2eIdentity(role, username, apiKey, label string, rateLimit int) map[string]any {
	identity := map[string]any{
		"role":          role,
		"username":      username,
		"api_key":       apiKey,
		"api_key_label": label,
		"email":         username + "@e2e.fibe.gg",
		"github_handle": username,
		"github_uid":    "e2e-" + username,
	}
	if rateLimit > 0 {
		identity["rate_limit_rph_override"] = rateLimit
	}
	return identity
}

func dockerE2ESSHPrivateKey() (string, error) {
	if value := os.Getenv("TEST_MARQUEE_PRIVATE_KEY"); value != "" {
		return value, nil
	}
	path := os.Getenv("TEST_MARQUEE_PRIVATE_KEY_PATH")
	if path == "" {
		return "", errors.New("TEST_MARQUEE_PRIVATE_KEY or TEST_MARQUEE_PRIVATE_KEY_PATH is required for SDK e2e bootstrap")
	}
	deadline := time.Now().Add(time.Duration(envInt("FIBE_E2E_BOOTSTRAP_FILE_TIMEOUT_SECONDS", 600)) * time.Second)
	for time.Now().Before(deadline) {
		raw, err := os.ReadFile(path)
		if err == nil && len(raw) > 0 {
			return string(raw), nil
		}
		time.Sleep(time.Second)
	}
	return "", fmt.Errorf("timed out waiting for %s", path)
}

func dockerE2ERootDomain() string {
	return "sdk." + envDefault("FIBE_E2E_DNS_SUFFIX", "e2e.fibe.test")
}

func e2eRunID() string {
	raw := firstEnv("E2E_RUN_ID", "FIBE_E2E_RUN_ID")
	if raw == "" {
		raw = fmt.Sprintf("%s-%d", time.Now().UTC().Format("20060102150405"), os.Getpid())
	}
	return e2eSlug(raw, 24)
}

func ensureFundedPrivateE2EMarquee(t *testing.T, prefix string) (e2eResource, bool) {
	t.Helper()
	if !envBool("FIBE_E2E_BOOTSTRAP") && !envBool("SDK_E2E_BOOTSTRAP") {
		return e2eResource{}, false
	}

	apiKey := os.Getenv("FIBE_API_KEY")
	if apiKey == "" {
		t.Fatal("FIBE_API_KEY is required to create private Docker E2E Marquee")
	}
	baseURL := strings.TrimRight(firstEnv("FIBE_DOMAIN", "FIBE_URL", "FIBE_BASE_URL"), "/")
	if baseURL == "" {
		baseURL = "http://localhost:3000"
	}
	sshKey, err := dockerE2ESSHPrivateKey()
	if err != nil {
		t.Fatalf("load Docker E2E Marquee SSH key: %v", err)
	}

	client := &e2eBootstrapClient{
		baseURL:    baseURL,
		adminToken: envDefault("E2E_ADMIN_API_KEY", defaultE2EAdminAPIKey),
		http:       &http.Client{Timeout: integrationHTTPTimeout()},
	}
	player, err := client.currentPlayer(apiKey)
	if err != nil {
		t.Fatalf("load primary Docker E2E player: %v", err)
	}

	rootDomain := envDefault("TEST_MARQUEE_ROOT_DOMAIN", dockerE2ERootDomain())
	marquee, err := client.ensureMarquee(player.ID, map[string]any{
		"name":                    uniqueName(prefix),
		"host":                    uniqueHost(),
		"port":                    2222,
		"user":                    "testuser",
		"ssh_private_key":         sshKey,
		"domains_input":           fmt.Sprintf("%s.%s", e2eSlug(uniqueName(prefix), 63), rootDomain),
		"status":                  "active",
		"billing_requested_until": time.Now().Add(7 * 24 * time.Hour).UTC().Format(time.RFC3339),
		"https_enabled":           false,
		"tls_certificate_source":  "provided",
	})
	if err != nil {
		t.Fatalf("create funded private Docker E2E Marquee: %v", err)
	}
	return marquee, true
}

func compactE2EUsername(parts ...string) string {
	value := e2eSlug(strings.Join(parts, "-"), 0)
	if len(value) <= e2eUsernameMaxLength {
		return value
	}

	suffix := e2eDigest(value)
	headLength := e2eUsernameMaxLength - len(suffix) - 1
	head := strings.Trim(value[:headLength], "-")
	if head == "" {
		head = "e2e"
	}
	return head + "-" + suffix
}

func e2eDigest(value string) string {
	hash := fnv.New32a()
	_, _ = hash.Write([]byte(value))
	return fmt.Sprintf("%08x", hash.Sum32())
}

func e2eSlug(value string, maxLength int) string {
	var builder strings.Builder
	lastDash := false
	for _, r := range strings.ToLower(value) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			builder.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash && builder.Len() > 0 {
			builder.WriteByte('-')
			lastDash = true
		}
	}
	result := strings.Trim(builder.String(), "-")
	if maxLength > 0 && len(result) > maxLength {
		result = strings.Trim(result[:maxLength], "-")
	}
	if result == "" {
		return "run"
	}
	return result
}

func envBool(name string) bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(name))) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func envDefault(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}

func firstEnv(names ...string) string {
	for _, name := range names {
		if value := strings.TrimSpace(os.Getenv(name)); value != "" {
			return value
		}
	}
	return ""
}

func envInt(name string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 1 {
		return fallback
	}
	return parsed
}
