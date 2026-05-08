package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/fibegg/sdk/fibe"
)

func TestResolveCLIAuthDefaultsToProductionProfile(t *testing.T) {
	setupAuthTest(t)

	got := resolveCLIAuth()
	if got.Profile != "default" {
		t.Fatalf("profile = %q, want default", got.Profile)
	}
	if got.Domain != "fibe.gg" {
		t.Fatalf("domain = %q, want fibe.gg", got.Domain)
	}
	if got.APIKey != "" || got.AuthSource != "none" {
		t.Fatalf("auth = %q from %q, want empty/none", got.APIKey, got.AuthSource)
	}
	if base := effectiveBaseURL(got.Domain); base != "https://fibe.gg" {
		t.Fatalf("base URL = %q, want https://fibe.gg", base)
	}
}

func TestResolveCLIAuthUsesEnvOnlyWhenNoProfileConfigured(t *testing.T) {
	setupAuthTest(t)
	t.Setenv("FIBE_API_KEY", "fibe_test_env")
	t.Setenv("FIBE_DOMAIN", "next.fibe.live")

	got := resolveCLIAuth()
	if got.APIKey != "fibe_test_env" || got.AuthSource != "FIBE_API_KEY env" {
		t.Fatalf("auth = %q from %q, want env", got.APIKey, got.AuthSource)
	}
	if got.Domain != "next.fibe.live" || got.DomainSource != "FIBE_DOMAIN env" {
		t.Fatalf("domain = %q from %q, want env domain", got.Domain, got.DomainSource)
	}
}

func TestResolveCLIAuthPreservesExplicitHTTPEnvDomain(t *testing.T) {
	setupAuthTest(t)
	t.Setenv("FIBE_API_KEY", "fibe_test_env")
	t.Setenv("FIBE_DOMAIN", "http://app-playwright-web:3001")

	got := resolveCLIAuth()
	if got.Domain != "http://app-playwright-web:3001" {
		t.Fatalf("domain = %q, want explicit HTTP domain", got.Domain)
	}
	if base := effectiveBaseURL(got.Domain); base != "http://app-playwright-web:3001" {
		t.Fatalf("base URL = %q, want http://app-playwright-web:3001", base)
	}
}

func TestResolveCLIAuthProfileBeatsEnv(t *testing.T) {
	setupAuthTest(t)
	if err := saveAuthProfile("staging", "next.fibe.live", "fibe_test_profile", 42); err != nil {
		t.Fatalf("save profile: %v", err)
	}
	t.Setenv("FIBE_API_KEY", "fibe_test_env")
	t.Setenv("FIBE_DOMAIN", "env.example.com")

	got := resolveCLIAuth()
	if got.Profile != "staging" {
		t.Fatalf("profile = %q, want staging", got.Profile)
	}
	if got.APIKey != "fibe_test_profile" || got.AuthSource != "profile staging" {
		t.Fatalf("auth = %q from %q, want profile", got.APIKey, got.AuthSource)
	}
	if got.Domain != "next.fibe.live" {
		t.Fatalf("domain = %q, want next.fibe.live", got.Domain)
	}
	if strings.Join(got.IgnoredEnv, ",") != "FIBE_DOMAIN,FIBE_API_KEY" {
		t.Fatalf("ignored env = %#v", got.IgnoredEnv)
	}
}

func TestResolveCLIAuthProfileFlagSelectsNamedProfile(t *testing.T) {
	setupAuthTest(t)
	if err := saveAuthProfile("default", "fibe.gg", "fibe_live_default", 1); err != nil {
		t.Fatalf("save default: %v", err)
	}
	if err := saveAuthProfile("staging", "next.fibe.live", "fibe_test_staging", 2); err != nil {
		t.Fatalf("save staging: %v", err)
	}
	if err := newCLIConfigStore(defaultCLIConfigPath()).setActive("default"); err != nil {
		t.Fatalf("set active: %v", err)
	}
	flagProfile = "staging"

	got := resolveCLIAuth()
	if got.Profile != "staging" || got.APIKey != "fibe_test_staging" || got.Domain != "next.fibe.live" {
		t.Fatalf("resolved = %#v, want staging profile", got)
	}
}

func TestRootLoginAPIKeyCreatesDefaultProfile(t *testing.T) {
	setupAuthTest(t)
	const key = "fibe_test_login_key"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/me" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer "+key {
			t.Fatalf("authorization = %q", got)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"id": 7, "username": "alice"})
	}))
	defer srv.Close()

	cmd := RootCmd()
	cmd.SetArgs([]string{"login", "--api-key", key, "--domain", srv.URL})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("login: %v", err)
	}

	entry, err := fibe.NewCredentialStore(fibe.DefaultCredentialPath()).GetProfile("default")
	if err != nil {
		t.Fatalf("get profile: %v", err)
	}
	if entry == nil || entry.APIKey != key {
		t.Fatalf("entry = %#v, want key", entry)
	}
	if active := newCLIConfigStore(defaultCLIConfigPath()).loadActiveForTest(t); active != "default" {
		t.Fatalf("active = %q, want default", active)
	}
}

func TestAuthUseSwitchesActiveProfile(t *testing.T) {
	setupAuthTest(t)
	if err := saveAuthProfile("staging", "next.fibe.live", "fibe_test_profile", 42); err != nil {
		t.Fatalf("save profile: %v", err)
	}
	if err := newCLIConfigStore(defaultCLIConfigPath()).setActive("default"); err != nil {
		t.Fatalf("set active: %v", err)
	}

	cmd := authUseCmd()
	cmd.SetArgs([]string{"staging"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("auth use: %v", err)
	}

	got := resolveCLIAuth()
	if got.Profile != "staging" {
		t.Fatalf("profile = %q, want staging", got.Profile)
	}
}

func setupAuthTest(t *testing.T) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", home)
	t.Setenv("FIBE_API_KEY", "")
	t.Setenv("FIBE_DOMAIN", "")

	prevAPIKey, prevDomain, prevProfile := flagAPIKey, flagDomain, flagProfile
	prevOutput, prevOnly := flagOutput, flagOnly
	flagAPIKey = ""
	flagDomain = ""
	flagProfile = ""
	flagOutput = ""
	flagOnly = nil
	t.Cleanup(func() {
		flagAPIKey = prevAPIKey
		flagDomain = prevDomain
		flagProfile = prevProfile
		flagOutput = prevOutput
		flagOnly = prevOnly
	})
}

func (s *cliConfigStore) loadActiveForTest(t *testing.T) string {
	t.Helper()
	cfg, err := s.load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	return cfg.ActiveProfile
}
