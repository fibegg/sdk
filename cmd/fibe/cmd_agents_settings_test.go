package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fibegg/sdk/fibe"
	"github.com/spf13/cobra"
)

func TestAgentCreateHelpExposesRuntimeSettingsFlags(t *testing.T) {
	help := commandHelp(t, agCreateCmd())

	for _, want := range []string{
		"--prompt",
		"--mcp-json",
		"--post-init-script",
		"--custom-env",
		"--cli-version",
		"--provider-args",
		"--skill-toggle",
	} {
		if !strings.Contains(help, want) {
			t.Fatalf("agents create help missing %s:\n%s", want, help)
		}
	}
}

func TestAgentDefaultsHelpExposesPlayerOverrideFlags(t *testing.T) {
	help := commandHelp(t, agentDefaultsUpdateCmd())

	for _, want := range []string{
		"--system-prompt",
		"--model-options",
		"--custom-env",
		"--mcp-json",
		"--post-init-script",
		"--provider-args",
		"--skill-toggle",
		"--provider",
	} {
		if !strings.Contains(help, want) {
			t.Fatalf("agent-defaults update help missing %s:\n%s", want, help)
		}
	}
}

func TestAgentDefaultsRootHelpExposesReset(t *testing.T) {
	help := commandHelp(t, agentDefaultsCmd())

	if !strings.Contains(help, "reset") {
		t.Fatalf("agent-defaults help missing reset:\n%s", help)
	}
}

func TestAgentDefaultsFromFileAcceptsEnvelope(t *testing.T) {
	oldFlag := flagFromFile
	oldRaw := rawPayload
	defer func() {
		flagFromFile = oldFlag
		rawPayload = oldRaw
	}()

	path := filepath.Join(t.TempDir(), "agent-defaults.json")
	if err := os.WriteFile(path, []byte(`{"agent_defaults":{"custom_env":"SDK_ENV=true"}}`), 0o600); err != nil {
		t.Fatalf("write defaults file: %v", err)
	}
	flagFromFile = path

	defaults, fromFile, err := agentDefaultsFromInput()
	if err != nil {
		t.Fatalf("agentDefaultsFromInput: %v", err)
	}
	if !fromFile {
		t.Fatalf("fromFile=false, want true")
	}
	if defaults["custom_env"] != "SDK_ENV=true" {
		t.Fatalf("defaults=%#v", defaults)
	}
}

func TestAgentDefaultsScopeWritesProviderOverrides(t *testing.T) {
	defaults := fibe.AgentDefaults{}
	scope := agentDefaultsScope(defaults, "gemini")
	scope["system_prompt"] = "Gemini prompt"

	overrides, ok := defaults["provider_overrides"].(map[string]any)
	if !ok {
		t.Fatalf("provider_overrides missing: %#v", defaults)
	}
	gemini, ok := overrides["gemini"].(map[string]any)
	if !ok || gemini["system_prompt"] != "Gemini prompt" {
		t.Fatalf("provider scope not written: %#v", overrides)
	}
}

func TestParseSkillToggleFlags(t *testing.T) {
	toggles, err := parseSkillToggleFlags([]string{"fibe-hunks.md=false", "search.md=true"})
	if err != nil {
		t.Fatalf("parseSkillToggleFlags: %v", err)
	}
	if toggles["fibe-hunks.md"] != false || toggles["search.md"] != true {
		t.Fatalf("toggles=%#v", toggles)
	}
}

func TestAgentUpdateHelpExposesRuntimeSettingsFlags(t *testing.T) {
	help := commandHelp(t, agUpdateCmd())

	for _, want := range []string{
		"--prompt",
		"--mcp-json",
		"--post-init-script",
		"--custom-env",
		"--cli-version",
		"--provider-args",
		"--skill-toggle",
	} {
		if !strings.Contains(help, want) {
			t.Fatalf("agents update help missing %s:\n%s", want, help)
		}
	}
}

func commandHelp(t *testing.T, cmd *cobra.Command) string {
	t.Helper()

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute help: %v", err)
	}
	return out.String()
}
