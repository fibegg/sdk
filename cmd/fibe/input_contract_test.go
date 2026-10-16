package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/spf13/cobra"
)

func TestCLIInvalidLocalInputsNeverRequest(t *testing.T) {
	setupAuthTest(t)
	var calls atomic.Int32
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.WriteHeader(500) }))
	defer api.Close()
	t.Setenv("FIBE_DOMAIN", api.URL)
	t.Setenv("FIBE_API_KEY", "synthetic")
	missing := "@" + filepath.Join(t.TempDir(), "missing-input")
	for _, tc := range []struct {
		name string
		make func() *cobra.Command
		args []string
		want string
	}{
		{"compose-file", launchCmd, []string{"--compose", missing}, "read text file"},
		{"spec-create-file", psCreateCmd, []string{"--name", "fixture", "--compose", missing}, "read text file"},
		{"spec-update-file", psUpdateCmd, []string{"fixture", "--base-compose", missing}, "read text file"},
		{"greenfield-file", greenfieldCmd, []string{"--name", "fixture", "--template-body", missing}, "read text file"},
		{"switch-file", pgSwitchTemplateCmd, []string{"fixture", "--template-body", missing}, "read text file"},
		{"trigger-file", psCreateCmd, []string{"--name", "fixture", "--trigger-prompt-template", missing}, "read text file"},
		{"muti-file", psUpdateCmd, []string{"fixture", "--muti-prompt-template", missing}, "read text file"},
		{"missing-map-separator", launchCmd, []string{"--compose", "services: {}", "--repository", "repo"}, "expects REPOSITORY"},
		{"empty-map-key", launchCmd, []string{"--compose", "services: {}", "--repository", "=repo"}, "expects REPOSITORY"},
		{"empty-map-target", launchCmd, []string{"--compose", "services: {}", "--repository", "repo="}, "expects REPOSITORY"},
		{"zero-map-target", launchCmd, []string{"--compose", "services: {}", "--repository", "repo=0"}, "positive ID"},
		{"negative-map-target", launchCmd, []string{"--compose", "services: {}", "--repository", "repo=-2"}, "positive ID"},
		{"overflow-map-target", launchCmd, []string{"--compose", "services: {}", "--repository", "repo=99999999999999999999999"}, "positive ID"},
		{"duplicate-map-target", launchCmd, []string{"--compose", "services: {}", "--repository", "repo=2", "--repository", "repo=other"}, "repeats mapping"},
		{"malformed-map-target", launchCmd, []string{"--compose", "services: {}", "--repository", "repo=other=third"}, "expects REPOSITORY"},
		{"invalid-system-filter", tplListCmd, []string{"--system=maybe"}, "must be true or false"},
		{"template-version", launchCmd, []string{"--template-version", "bad"}, "positive integer ID"},
		{"version", launchCmd, []string{"--template", "fixture", "--version=0"}, "positive integer"},
		{"installation-flag", launchCmd, []string{"--repo", "owner/repo", "--github-installation-id=0"}, "positive integer"},
		{"creation-conflict", launchCmd, []string{"--compose", "services: {}", "--create-playground", "--no-create-playground"}, "only one"},
		{"wait-without-create", launchCmd, []string{"--compose", "services: {}", "--wait", "--no-create-playground"}, "requires playground"},
		{"wait-timeout", launchCmd, []string{"--compose", "services: {}", "--wait-timeout=0"}, "must be positive"},
		{"unused-wait-timeout", launchCmd, []string{"--compose", "services: {}", "--wait-timeout=1s"}, "requires --wait"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cmd := tc.make()
			cmd.SilenceUsage = true
			cmd.SilenceErrors = true
			cmd.SetOut(&bytes.Buffer{})
			cmd.SetOut(&bytes.Buffer{})
			cmd.SetErr(&bytes.Buffer{})
			cmd.SetArgs(tc.args)
			if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error=%v want %s", err, tc.want)
			}
			if calls.Load() != 0 {
				t.Fatalf("local rejection sent %d requests", calls.Load())
			}
		})
	}
	for _, tc := range []struct {
		make   func() *cobra.Command
		prefix []string
	}{
		{jobEnvGetCmd, nil}, {jobEnvUpdateCmd, nil}, {jobEnvDeleteCmd, nil}, {instReposCmd, nil}, {instTokenCmd, nil},
		{tplVersionsDestroyCmd, []string{"fixture"}}, {tplVersionsTogglePublicCmd, []string{"fixture"}},
		{whGetCmd, nil}, {whUpdateCmd, nil}, {whDeleteCmd, nil}, {whTestCmd, nil}, {whDeliveriesCmd, nil}, {keyDeleteCmd, nil},
		{fbGetCmd, []string{"fixture"}}, {fbDeleteCmd, []string{"fixture"}},
	} {
		for _, id := range []string{"bad", "0", "-2", "999999999999999999999999"} {
			cmd := tc.make()
			cmd.SilenceUsage = true
			cmd.SilenceErrors = true
			cmd.SetOut(&bytes.Buffer{})
			cmd.SetErr(&bytes.Buffer{})
			cmd.SetArgs(append(append([]string{}, tc.prefix...), "--", id))
			if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), "positive integer ID") {
				t.Fatalf("%s id %q: %v", cmd.Use, id, err)
			}
			if calls.Load() != 0 {
				t.Fatal("invalid ID reached API")
			}
		}
	}
}

func TestLaunchSupportedSourcePayloads(t *testing.T) {
	for _, source := range []string{"spec", "template", "template-version", "repo"} {
		t.Run(source, func(t *testing.T) {
			setupAuthTest(t)
			var posted map[string]any
			api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodGet {
					if r.URL.Path == "/api/installations" {
						_, _ = w.Write([]byte(`{"data":[{"id":7,"installation_id":123,"installation_account":"owner"}]}`))
					} else {
						_, _ = w.Write([]byte(`{"id":7,"services":[{"name":"web"}]}`))
					}
					return
				}
				if err := json.NewDecoder(r.Body).Decode(&posted); err != nil {
					t.Error(err)
				}
				_, _ = w.Write([]byte(`{"id":8,"spec_id":7,"playground_id":8,"playground":{"id":8}}`))
			}))
			defer api.Close()
			t.Setenv("FIBE_DOMAIN", api.URL)
			t.Setenv("FIBE_API_KEY", "synthetic")
			value := "fixture"
			if source == "template-version" {
				value = "9"
			}
			if source == "repo" {
				value = "owner/repo@legacy-branch"
			}
			args := []string{"--" + source, value, "--name", "fixture", "--host", "21", "--service", "web.env_vars.EMPTY="}
			if source != "spec" {
				args = append(args, "--var", "EMPTY=", "--env", "EMPTY=", "--subdomain", "web=fixture", "--persist-volumes=false")
			}
			if source == "template" {
				args = append(args, "--version=2")
			}
			if source == "repo" {
				args = append(args, "--file", "compose.yml", "--github-installation-id=123", "--job-mode=false", "--no-create-playground=false")
			}
			cmd := launchCmd()
			cmd.SetArgs(args)
			cmd.SilenceUsage = true
			if _, err := captureStdout(cmd.Execute); err != nil {
				t.Fatal(err)
			}
			if posted == nil {
				t.Fatal("no mutation")
			}
			if source == "spec" {
				posted = posted["playground"].(map[string]any)
			}
			if posted["services"] == nil {
				t.Fatalf("service override omitted: %#v", posted)
			}
			if source != "spec" && posted["persist_volumes"] != false {
				t.Fatalf("explicit false omitted: %#v", posted)
			}
			if source == "repo" && (posted["github_ref"] != "legacy-branch" || posted["config_path"] != "compose.yml" || posted["create_playground"] != true || posted["job_mode"] != false) {
				t.Fatalf("repo payload=%#v", posted)
			}
		})
	}
}

func TestLaunchSourceSpecificFlagsFailBeforeLookup(t *testing.T) {
	setupAuthTest(t)
	var calls atomic.Int32
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.WriteHeader(500) }))
	defer api.Close()
	t.Setenv("FIBE_DOMAIN", api.URL)
	t.Setenv("FIBE_API_KEY", "synthetic")
	for _, tc := range []struct {
		source []string
		flags  [][]string
	}{
		{[]string{"--spec", "fixture"}, [][]string{{"--var", "A=x"}, {"--env", "A=x"}, {"--subdomain", "web=x"}, {"--persist-volumes=false"}, {"--version=2"}, {"--repository", "repo=2"}, {"--job-mode=false"}}},
		{[]string{"--template", "fixture"}, [][]string{{"--file", "x"}, {"--ref", "main"}, {"--github-account", "owner"}, {"--github-installation-id=2"}, {"--repository", "repo=2"}, {"--create-playground=false"}, {"--no-create-playground=false"}, {"--job-mode=false"}}},
		{[]string{"--template-version", "2"}, [][]string{{"--version=2"}, {"--file", "x"}, {"--job-mode"}}},
		{[]string{"--compose", "services: {}"}, [][]string{{"--file", "x"}, {"--ref", "main"}, {"--github-account", "owner"}, {"--github-installation-id=2"}, {"--version=2"}}},
		{[]string{"--repo", "owner/repo"}, [][]string{{"--version=2"}}},
	} {
		for _, flags := range tc.flags {
			cmd := launchCmd()
			cmd.SilenceUsage = true
			cmd.SilenceErrors = true
			cmd.SetOut(&bytes.Buffer{})
			cmd.SetErr(&bytes.Buffer{})
			cmd.SetArgs(append(append([]string{}, tc.source...), flags...))
			if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), "not supported") {
				t.Fatalf("%v %v: %v", tc.source, flags, err)
			}
		}
	}
	if calls.Load() != 0 {
		t.Fatalf("sent %d requests", calls.Load())
	}
}

func TestLaunchAndJobEnvPreserveExplicitFalseAndEmpty(t *testing.T) {
	setupAuthTest(t)
	var bodies []map[string]any
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		bodies = append(bodies, body)
		_, _ = w.Write([]byte(`{"spec_id":7,"playground_id":8,"id":8}`))
	}))
	defer api.Close()
	t.Setenv("FIBE_DOMAIN", api.URL)
	t.Setenv("FIBE_API_KEY", "synthetic")
	file := filepath.Join(t.TempDir(), "compose.yml")
	if err := os.WriteFile(file, []byte("services:\n  web:\n    image: alpine\n"), 0600); err != nil {
		t.Fatal(err)
	}
	cmd := launchCmd()
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--compose", "@" + file, "--name", "fixture", "--host", "21", "--job-mode=false", "--persist-volumes=false", "--create-playground=false", "--env", "EMPTY=", "--repository", "origin=42", "--repository", "other=fixture-repository"})
	if _, err := captureStdout(cmd.Execute); err != nil {
		t.Fatal(err)
	}
	body := bodies[0]
	for _, key := range []string{"job_mode", "persist_volumes", "create_playground"} {
		if body[key] != false {
			t.Fatalf("%s=%#v", key, body[key])
		}
	}
	if body["compose_yaml"] != "services:\n  web:\n    image: alpine\n" {
		t.Fatalf("compose=%#v", body)
	}
	if body["env_overrides"].(map[string]any)["EMPTY"] != "" || body["repository_mappings"].(map[string]any)["origin"] != float64(42) || body["repository_mappings"].(map[string]any)["other"] != "fixture-repository" {
		t.Fatalf("payload=%#v", body)
	}
	cmd = jobEnvUpdateCmd()
	cmd.SetArgs([]string{"8", "--value=", "--secret=false", "--enabled=false", "--description="})
	if _, err := captureStdout(cmd.Execute); err != nil {
		t.Fatal(err)
	}
	body = bodies[1]["job_env"].(map[string]any)
	for key, want := range map[string]any{"value": "", "secret": false, "enabled": false, "description": ""} {
		if body[key] != want {
			t.Fatalf("%s=%#v", key, body[key])
		}
	}
}
