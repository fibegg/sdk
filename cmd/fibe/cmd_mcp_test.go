package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestMCPClientHelpMentionsCodex(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		cmd  func() commandWithArgs
	}{
		{name: "install", cmd: func() commandWithArgs { return commandWithArgs{cmd: mcpInstallCmd(), args: []string{"--help"}} }},
		{name: "uninstall", cmd: func() commandWithArgs { return commandWithArgs{cmd: mcpUninstallCmd(), args: []string{"--help"}} }},
		{name: "config", cmd: func() commandWithArgs { return commandWithArgs{cmd: mcpConfigCmd(), args: []string{"--help"}} }},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			spec := tc.cmd()
			var buf bytes.Buffer
			spec.cmd.SetOut(&buf)
			spec.cmd.SetErr(&buf)
			spec.cmd.SetArgs(spec.args)

			if err := spec.cmd.Execute(); err != nil {
				t.Fatalf("execute help: %v", err)
			}
			if !strings.Contains(buf.String(), "codex") {
				t.Fatalf("expected help output to mention codex, got:\n%s", buf.String())
			}
		})
	}
}

func TestRunMCPInstallCodexWritesTOMLConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	opts := installOptions{
		ToolSet: "full",
		Yolo:    true,
		Env:     []string{"FOO=bar"},
	}
	if err := runMCPInstall("codex", "", false, opts); err != nil {
		t.Fatalf("install codex config: %v", err)
	}

	configPath := filepath.Join(home, ".codex", "config.toml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}

	cfg, err := parseMCPConfig(data, mcpConfigTOML)
	if err != nil {
		t.Fatalf("parse config: %v", err)
	}

	servers := nestedMap(t, cfg["mcp_servers"])
	entry := nestedMap(t, servers[mcpServerName])
	if entry["command"] == "" {
		t.Fatalf("expected command to be populated, got %#v", entry["command"])
	}
	if got := toStringSlice(t, entry["args"]); len(got) != 2 || got[0] != "mcp" || got[1] != "serve" {
		t.Fatalf("unexpected args: %#v", got)
	}

	env := nestedMap(t, entry["env"])
	if got := env["FIBE_MCP_TOOLS"]; got != "full" {
		t.Fatalf("expected FIBE_MCP_TOOLS=full, got %#v", got)
	}
	if got := env["FIBE_MCP_YOLO"]; got != "1" {
		t.Fatalf("expected FIBE_MCP_YOLO=1, got %#v", got)
	}
	if got := env["FOO"]; got != "bar" {
		t.Fatalf("expected FOO=bar, got %#v", got)
	}

	envVars := toStringSlice(t, entry["env_vars"])
	if !contains(envVars, "FIBE_API_KEY") || !contains(envVars, "FIBE_DOMAIN") {
		t.Fatalf("expected env_vars to forward FIBE_API_KEY and FIBE_DOMAIN, got %#v", envVars)
	}
}

func TestRunMCPUninstallCodexRemovesServer(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	if err := runMCPInstall("codex", "", false, installOptions{}); err != nil {
		t.Fatalf("install codex config: %v", err)
	}
	if err := runMCPUninstall("codex", "", false); err != nil {
		t.Fatalf("uninstall codex config: %v", err)
	}

	configPath := filepath.Join(home, ".codex", "config.toml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}

	cfg, err := parseMCPConfig(data, mcpConfigTOML)
	if err != nil {
		t.Fatalf("parse config: %v", err)
	}

	if serversRaw, ok := cfg["mcp_servers"]; ok {
		servers := nestedMap(t, serversRaw)
		if _, present := servers[mcpServerName]; present {
			t.Fatalf("expected %q to be removed, got %#v", mcpServerName, servers)
		}
	}
}

type commandWithArgs struct {
	cmd  *cobra.Command
	args []string
}

func nestedMap(t *testing.T, value any) map[string]any {
	t.Helper()
	out, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("expected map[string]any, got %T", value)
	}
	return out
}

func toStringSlice(t *testing.T, value any) []string {
	t.Helper()
	items, ok := value.([]any)
	if !ok {
		if direct, ok := value.([]string); ok {
			return direct
		}
		t.Fatalf("expected []any or []string, got %T", value)
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		s, ok := item.(string)
		if !ok {
			t.Fatalf("expected string slice item, got %T", item)
		}
		out = append(out, s)
	}
	return out
}

func contains(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}
