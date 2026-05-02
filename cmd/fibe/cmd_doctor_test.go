package main

import "testing"

func TestDoctorAPIKeyStatusUsesFlagBeforeEnv(t *testing.T) {
	configHome := t.TempDir()
	t.Setenv("HOME", configHome)
	t.Setenv("XDG_CONFIG_HOME", configHome)
	t.Setenv("FIBE_API_KEY", "fibe_test_env")

	prevAPIKey, prevDomain := flagAPIKey, flagDomain
	flagAPIKey = "fibe_test_flag"
	flagDomain = "next.fibe.live"
	defer func() {
		flagAPIKey = prevAPIKey
		flagDomain = prevDomain
	}()

	key, source := doctorAPIKeyStatus()
	if key != "fibe_test_flag" || source != "--api-key flag" {
		t.Fatalf("doctorAPIKeyStatus() = %q, %q; want flag key/source", key, source)
	}
}

func TestDoctorAPIKeyStatusUsesEnv(t *testing.T) {
	configHome := t.TempDir()
	t.Setenv("HOME", configHome)
	t.Setenv("XDG_CONFIG_HOME", configHome)
	t.Setenv("FIBE_API_KEY", "fibe_test_env")

	prevAPIKey, prevDomain := flagAPIKey, flagDomain
	flagAPIKey = ""
	flagDomain = "next.fibe.live"
	defer func() {
		flagAPIKey = prevAPIKey
		flagDomain = prevDomain
	}()

	key, source := doctorAPIKeyStatus()
	if key != "fibe_test_env" || source != "FIBE_API_KEY env" {
		t.Fatalf("doctorAPIKeyStatus() = %q, %q; want env key/source", key, source)
	}
}
