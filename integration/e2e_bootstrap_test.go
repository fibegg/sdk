package integration

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
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

	requestTimeout := time.Duration(envInt("FIBE_E2E_BOOTSTRAP_REQUEST_TIMEOUT_SECONDS", 180)) * time.Second
	client := &e2eBootstrapClient{baseURL: baseURL, adminToken: adminToken, http: &http.Client{Timeout: requestTimeout}}
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
	rootDomain := envDefault("TEST_MARQUEE_ROOT_DOMAIN", dockerE2ERootDomain())
	bootstrap, err := client.bootstrapSDK(runID, map[string]string{
		"primary":    token,
		"user_b":     userBToken,
		"rate_limit": rateToken,
	}, map[string]any{
		"host":            envDefault("TEST_MARQUEE_HOST", "dind-sdk"),
		"port":            envDefault("TEST_MARQUEE_PORT", "22"),
		"ssh_key_port":    envDefault("TEST_SSH_KEY_MARQUEE_PORT", "2222"),
		"user":            envDefault("TEST_MARQUEE_USER", "root"),
		"ssh_private_key": sshKey,
		"root_domain":     rootDomain,
	})
	if err != nil {
		return nil, err
	}
	if !bootstrap.Success {
		return nil, errors.New("bootstrap backdoor returned success=false")
	}
	for key, value := range bootstrap.Env {
		os.Setenv(key, value)
	}
	os.Setenv("FIBE_DOMAIN", baseURL)

	player, err := client.currentPlayer(os.Getenv("FIBE_API_KEY"))
	if err != nil {
		return nil, err
	}
	_ = player

	return func() error {
		var errs []string
		for _, id := range []int64{bootstrap.Marquee.ID, bootstrap.SSHKeyMarquee.ID} {
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

type e2eResource struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type e2eBootstrapPayload struct {
	Success       bool              `json:"success"`
	Env           map[string]string `json:"env"`
	Marquee       e2eResource       `json:"marquee"`
	SSHKeyMarquee e2eResource       `json:"ssh_key_marquee"`
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

func (c *e2eBootstrapClient) bootstrapSDK(runID string, tokens map[string]string, marquee map[string]any) (e2eBootstrapPayload, error) {
	var payload e2eBootstrapPayload
	err := c.requestJSON(http.MethodPost, "/e2e_backdoor/bootstrap", c.adminToken, map[string]any{
		"mode":    "sdk",
		"run_id":  runID,
		"tokens":  tokens,
		"marquee": marquee,
	}, &payload, http.StatusOK)
	return payload, err
}

func (c *e2eBootstrapClient) deactivateMarquee(marqueeID int64) error {
	var payload struct {
		Success bool `json:"success"`
	}
	err := c.requestJSON(http.MethodPost, "/e2e_backdoor/operation", c.adminToken, map[string]any{
		"operation":  "deactivate_marquee",
		"marquee_id": marqueeID,
	}, &payload, http.StatusOK)
	if err != nil {
		return err
	}
	if !payload.Success {
		return errors.New("deactivate_marquee operation returned success=false")
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

	rawID := strings.TrimSpace(os.Getenv("FIBE_TEST_MARQUEE_ID"))
	if rawID == "" {
		t.Fatal("FIBE_TEST_MARQUEE_ID is required after SDK e2e bootstrap")
	}
	id, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil {
		t.Fatalf("parse FIBE_TEST_MARQUEE_ID: %v", err)
	}
	if id <= 0 {
		t.Fatalf("invalid FIBE_TEST_MARQUEE_ID %q", rawID)
	}
	return e2eResource{ID: id, Name: prefix}, true
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
