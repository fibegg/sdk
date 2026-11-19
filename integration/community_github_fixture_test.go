package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"strings"
	"testing"
	"time"
)

// bindCommunityGitHubFixture uses the operator's normal browser flow. Community
// production servers do not expose the Enterprise E2E bootstrap backdoor.
func bindCommunityGitHubFixture(t *testing.T, baseURL, token string, expectedPlayer int64) {
	t.Helper()
	email := firstEnv("FIBE_NATIVE_ADMIN_EMAIL", "FIBE_NATIVE_EMAIL")
	password := firstEnv("FIBE_NATIVE_ADMIN_PASSWORD", "FIBE_NATIVE_PASSWORD")
	if email == "" || password == "" {
		t.Fatal("Community GitHub fixture binding requires the existing operator's email and password")
	}
	jar, err := cookiejar.New(nil)
	requireNoError(t, err, "create private fixture cookie jar")
	client := &http.Client{Jar: jar, Timeout: 30 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	var csrf string
	var owner int64
	request := func(method, path string, body any) json.RawMessage {
		t.Helper()
		var reader io.Reader
		if body != nil {
			data, err := json.Marshal(body)
			requireNoError(t, err, "encode fixture browser request")
			reader = bytes.NewReader(data)
		}
		req, err := http.NewRequestWithContext(ctx(), method, strings.TrimRight(baseURL, "/")+path, reader)
		requireNoError(t, err, "create fixture browser request")
		req.Header.Set("Accept", "application/json")
		req.Header.Set("Origin", strings.TrimRight(baseURL, "/"))
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		if csrf != "" {
			req.Header.Set("X-CSRF-Token", csrf)
		}
		if owner > 0 {
			req.Header.Set("X-Fibe-Owner-Type", "Player")
			req.Header.Set("X-Fibe-Owner-ID", fmt.Sprint(owner))
		}
		resp, err := client.Do(req)
		requireNoError(t, err, "send fixture browser request")
		defer resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			t.Fatalf("fixture browser %s %s returned HTTP %d", method, path, resp.StatusCode)
		}
		raw, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
		requireNoError(t, err, "read fixture browser response")
		return raw
	}
	bootstrap := func() {
		t.Helper()
		raw := request(http.MethodGet, "/ui-api/v1/bootstrap", nil)
		var envelope struct {
			Data struct {
				CSRF string `json:"csrf_token"`
			} `json:"data"`
		}
		requireNoError(t, json.Unmarshal(raw, &envelope), "decode fixture bootstrap")
		if envelope.Data.CSRF == "" {
			t.Fatal("fixture bootstrap omitted CSRF token")
		}
		csrf = envelope.Data.CSRF
	}
	bootstrap()
	request(http.MethodPost, "/ui-api/v1/session", map[string]string{"email": email, "password": password})
	bootstrap()
	var me struct {
		Data struct {
			ID int64 `json:"id"`
		} `json:"data"`
	}
	requireNoError(t, json.Unmarshal(request(http.MethodGet, "/ui-api/v1/me", nil), &me), "decode fixture identity")
	if me.Data.ID != expectedPlayer {
		t.Fatal("fixture browser and SDK key must belong to the same Player")
	}
	owner = me.Data.ID
	request(http.MethodPost, "/ui-api/v1/security/sudo", map[string]string{"sudo_password": password})
	bootstrap()
	request(http.MethodPost, "/ui-api/v1/git_connections", map[string]string{"provider": "github", "base_url": "https://api.github.com", "token": token})
	// The cookie jar and supplied credentials stay in memory; no CLI profile is written.
}
