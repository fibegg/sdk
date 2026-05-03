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

func TestMCPInstallReloadNoticeMentionsCodexSessions(t *testing.T) {
	notice := mcpInstallReloadNotice("codex")
	if !strings.Contains(notice, "new session") {
		t.Fatalf("expected codex reload notice to mention new sessions, got %q", notice)
	}
	if got := mcpInstallReloadNotice("cursor"); got != "" {
		t.Fatalf("expected no cursor reload notice, got %q", got)
	}
}

func TestRunMCPInstallClaudeCodeProjectWritesMCPJSON(t *testing.T) {
	home := t.TempDir()
	project := t.TempDir()
	t.Setenv("HOME", home)

	opts := installOptions{
		APIKey: "pk_test_claude",
		Domain: "next.fibe.live",
	}
	if err := runMCPInstall("claude-code", project, false, opts); err != nil {
		t.Fatalf("install claude-code project config: %v", err)
	}

	configPath := filepath.Join(project, ".mcp.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	if _, err := os.Stat(filepath.Join(project, ".claude", "settings.json")); !os.IsNotExist(err) {
		t.Fatalf("expected project install not to write .claude/settings.json, stat err: %v", err)
	}

	cfg, err := parseMCPConfig(data, mcpConfigJSON)
	if err != nil {
		t.Fatalf("parse config: %v", err)
	}

	servers := nestedMap(t, cfg["mcpServers"])
	entry := nestedMap(t, servers[mcpServerName])
	if got := entry["type"]; got != "stdio" {
		t.Fatalf("expected type=stdio, got %#v", got)
	}
	if entry["command"] == "" {
		t.Fatalf("expected command to be populated, got %#v", entry["command"])
	}
	if got := toStringSlice(t, entry["args"]); len(got) != 2 || got[0] != "mcp" || got[1] != "serve" {
		t.Fatalf("unexpected args: %#v", got)
	}

	env := nestedMap(t, entry["env"])
	if got := env["FIBE_API_KEY"]; got != "pk_test_claude" {
		t.Fatalf("expected FIBE_API_KEY=pk_test_claude, got %#v", got)
	}
	if got := env["FIBE_DOMAIN"]; got != "next.fibe.live" {
		t.Fatalf("expected FIBE_DOMAIN=next.fibe.live, got %#v", got)
	}
}

func TestResolveMCPProjectScopeDefaultsClaudeCodeToCurrentProject(t *testing.T) {
	project, err := resolveMCPProjectScope("claude-code", "", false)
	if err != nil {
		t.Fatalf("resolve scope: %v", err)
	}
	if project != "." {
		t.Fatalf("project=%q want current directory", project)
	}

	project, err = resolveMCPProjectScope("claude-code", "", true)
	if err != nil {
		t.Fatalf("resolve user scope: %v", err)
	}
	if project != "" {
		t.Fatalf("user scope project=%q want empty", project)
	}

	if _, err := resolveMCPProjectScope("claude-code", ".", true); err == nil {
		t.Fatal("expected --user and --project conflict")
	}
}

func TestRunMCPInstallCodexURLModeWritesURLConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	opts := installOptions{
		Transport: "streamable-http",
		URL:       "http://127.0.0.1:7797/mcp",
	}
	if err := runMCPInstall("codex", "", false, opts); err != nil {
		t.Fatalf("install codex url config: %v", err)
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
	if got := entry["url"]; got != "http://127.0.0.1:7797/mcp" {
		t.Fatalf("expected url entry, got %#v", got)
	}
	if got := entry["bearer_token_env_var"]; got != "FIBE_API_KEY" {
		t.Fatalf("expected bearer_token_env_var=FIBE_API_KEY, got %#v", got)
	}
	if _, present := entry["command"]; present {
		t.Fatalf("did not expect stdio command entry in url mode, got %#v", entry["command"])
	}
}

func TestRunMCPInstallCursorURLModeWritesHeaderConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	opts := installOptions{
		Transport: "streamable-http",
		URL:       "https://fibe.example.com/mcp",
	}
	if err := runMCPInstall("cursor", "", false, opts); err != nil {
		t.Fatalf("install cursor url config: %v", err)
	}

	configPath := filepath.Join(home, ".cursor", "mcp.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}

	cfg, err := parseMCPConfig(data, mcpConfigJSON)
	if err != nil {
		t.Fatalf("parse config: %v", err)
	}

	servers := nestedMap(t, cfg["mcpServers"])
	entry := nestedMap(t, servers[mcpServerName])
	if got := entry["url"]; got != "https://fibe.example.com/mcp" {
		t.Fatalf("expected url entry, got %#v", got)
	}
	headers := nestedMap(t, entry["headers"])
	if got := headers["Authorization"]; got != "Bearer ${env:FIBE_API_KEY}" {
		t.Fatalf("expected env-backed auth header, got %#v", got)
	}
}

func TestRunMCPInstallAntigravityURLModeWritesHeaderConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("FIBE_API_KEY", "pk_live_antigravity")

	opts := installOptions{
		Transport: "streamable-http",
		URL:       "https://fibe.example.com/mcp",
	}
	if err := runMCPInstall("antigravity", "", false, opts); err != nil {
		t.Fatalf("install antigravity url config: %v", err)
	}

	configPath := filepath.Join(home, ".gemini", "antigravity", "mcp_config.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}

	cfg, err := parseMCPConfig(data, mcpConfigJSON)
	if err != nil {
		t.Fatalf("parse config: %v", err)
	}

	servers := nestedMap(t, cfg["mcpServers"])
	entry := nestedMap(t, servers[mcpServerName])
	if got := entry["serverUrl"]; got != "https://fibe.example.com/mcp" {
		t.Fatalf("expected serverUrl entry, got %#v", got)
	}
	headers := nestedMap(t, entry["headers"])
	if got := headers["Authorization"]; got != "Bearer pk_live_antigravity" {
		t.Fatalf("expected literal auth header, got %#v", got)
	}
}

func TestRunMCPInstallVSCodeURLModeWritesHTTPConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	opts := installOptions{
		Transport: "streamable-http",
		URL:       "https://fibe.example.com/mcp",
	}
	if err := runMCPInstall("vscode", "", false, opts); err != nil {
		t.Fatalf("install vscode url config: %v", err)
	}

	configPath := filepath.Join(home, ".vscode", "mcp.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}

	cfg, err := parseMCPConfig(data, mcpConfigJSON)
	if err != nil {
		t.Fatalf("parse config: %v", err)
	}

	servers := nestedMap(t, cfg["servers"])
	entry := nestedMap(t, servers[mcpServerName])
	if got := entry["type"]; got != "http" {
		t.Fatalf("expected type=http, got %#v", got)
	}
	if got := entry["url"]; got != "https://fibe.example.com/mcp" {
		t.Fatalf("expected url entry, got %#v", got)
	}
	headers := nestedMap(t, entry["headers"])
	if got := headers["Authorization"]; got != "Bearer ${env:FIBE_API_KEY}" {
		t.Fatalf("expected env-backed auth header, got %#v", got)
	}
}

func TestRunMCPInstallRejectsNonStdioTransportWithoutURL(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	err := runMCPInstall("codex", "", false, installOptions{Transport: "streamable-http"})
	if err == nil {
		t.Fatal("expected transport/url validation error")
	}
	if !strings.Contains(err.Error(), "requires --url") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunMCPInstallRejectsStdioOnlyFlagsInURLMode(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	err := runMCPInstall("cursor", "", false, installOptions{
		Transport: "streamable-http",
		URL:       "https://fibe.example.com/mcp",
		ToolSet:   "full",
	})
	if err == nil {
		t.Fatal("expected url/stdio flag validation error")
	}
	if !strings.Contains(err.Error(), "--tools") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunMCPInstallRejectsCodexURLInlineAPIKey(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	err := runMCPInstall("codex", "", false, installOptions{
		Transport: "streamable-http",
		URL:       "https://fibe.example.com/mcp",
		APIKey:    "pk_test_123",
	})
	if err == nil {
		t.Fatal("expected codex url api key validation error")
	}
	if !strings.Contains(err.Error(), "cannot inline API keys") {
		t.Fatalf("unexpected error: %v", err)
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
