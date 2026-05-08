package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

const (
	mcpServerName      = "fibe"
	mcpClientFlagHelp  = "claude-code|claude-desktop|cursor|vscode|antigravity|codex"
	mcpValidClientList = "claude-code, claude-desktop, cursor, vscode, antigravity, codex"
)

type mcpConfigFormat string

const (
	mcpConfigJSON mcpConfigFormat = "json"
	mcpConfigTOML mcpConfigFormat = "toml"
)

type mcpClientTarget struct {
	Path           string
	WrapperKey     string
	WrapperDefault map[string]any
	Format         mcpConfigFormat
}

// installOptions bundles the flags that shape the emitted MCP config entry.
// Each field is optional; empty means "use the default".
type installOptions struct {
	APIKey    string   // literal value to pass through --api-key
	Domain    string   // literal value to pass through --domain
	Env       []string // arbitrary KEY=VALUE pairs to inline
	Profile   string   // profile to pass to "fibe mcp serve"
	ToolSet   string   // full, core, or comma-separated named tiers (passed as FIBE_MCP_TOOLS)
	Yolo      bool     // sets FIBE_MCP_YOLO=1
	AuditLog  string   // sets FIBE_MCP_AUDIT_LOG
	Transport string   // stdio (default) | streamable-http (URL-backed clients)
	URL       string   // URL for URL-backed MCP clients
}

func resolveMCPProjectScope(client, project string, userScope bool) (string, error) {
	if userScope {
		if project != "" {
			return "", fmt.Errorf("--user cannot be combined with --project")
		}
		return "", nil
	}
	if project != "" {
		return project, nil
	}
	if client == "claude-code" {
		return ".", nil
	}
	return "", nil
}

// runMCPInstall writes the fibe MCP server entry into the target client's
// configuration file. Works with claude-code, claude-desktop, cursor, vscode,
// antigravity, and codex.
//
// Design notes:
//   - Never clobber unrelated entries: we read the existing config, merge
//     our "fibe" entry, and write the file back with the same shape.
//   - Dry-run mode prints the resolved path and the proposed delta so users
//     can review before overwriting.
//   - Each client has a slightly different schema (claude uses "mcpServers",
//     vscode uses "servers", Codex uses TOML "mcp_servers"), so we keep a
//     small per-client adapter.
//   - Default stdio installs pin a profile rather than forwarding shell
//     auth env vars. Explicit --api-key/--domain are passed to the launched
//     "fibe mcp serve" command as one-off overrides.
func runMCPInstall(client, project string, dryRun bool, opts installOptions) error {
	bin, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve fibe binary path: %w", err)
	}

	target, err := resolveMCPClientTarget(client, project)
	if err != nil {
		return err
	}

	entry, warnings, err := buildMCPInstallEntry(client, bin, opts)
	if err != nil {
		return err
	}

	existing := map[string]any{}
	if data, readErr := os.ReadFile(target.Path); readErr == nil && len(data) > 0 {
		existing, err = parseMCPConfig(data, target.Format)
		if err != nil {
			return fmt.Errorf("parse existing config %s: %w", target.Path, err)
		}
	}

	servers, _ := existing[target.WrapperKey].(map[string]any)
	if servers == nil {
		servers = target.WrapperDefault
	}
	servers[mcpServerName] = entry
	existing[target.WrapperKey] = servers

	out, err := marshalMCPConfig(existing, target.Format)
	if err != nil {
		return err
	}

	for _, w := range warnings {
		fmt.Fprintf(os.Stderr, "WARN: %s\n", w)
	}

	if dryRun {
		fmt.Printf("# target: %s\n\n%s", target.Path, string(out))
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(target.Path), 0o755); err != nil {
		return fmt.Errorf("create parent dir: %w", err)
	}
	if err := os.WriteFile(target.Path, out, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", target.Path, err)
	}
	fmt.Printf("Installed fibe MCP server into %s\n", target.Path)
	if notice := mcpInstallReloadNotice(client); notice != "" {
		fmt.Println(notice)
	}
	return nil
}

func mcpInstallReloadNotice(client string) string {
	switch client {
	case "codex":
		return "Note: Codex loads MCP server config when a session starts; restart Codex or open a new session for this change to take effect."
	default:
		return ""
	}
}

func buildMCPInstallEntry(client, bin string, opts installOptions) (map[string]any, []string, error) {
	transport := opts.Transport
	if transport == "" {
		if opts.URL != "" {
			transport = "streamable-http"
		} else {
			transport = "stdio"
		}
	}
	if opts.URL != "" {
		return buildRemoteMCPInstallEntry(client, transport, opts)
	}
	if transport != "stdio" {
		return nil, nil, fmt.Errorf("transport %q requires --url; supported URL-backed clients: antigravity, cursor, vscode, codex", transport)
	}

	env, envVars, warnings := resolveInstallEnv(client, opts)
	profile := opts.Profile
	if profile == "" {
		profile = selectedProfileName()
	}
	args := []string{"mcp", "serve", "--profile", profile}
	if opts.APIKey != "" {
		args = append(args, "--api-key", opts.APIKey)
	}
	if opts.Domain != "" {
		args = append(args, "--domain", opts.Domain)
	}
	entry := map[string]any{
		"command": bin,
		"args":    args,
	}
	if len(env) > 0 {
		entry["env"] = env
	}
	if len(envVars) > 0 {
		entry["env_vars"] = envVars
	}
	if client == "claude-code" || client == "vscode" {
		// Claude Code and VS Code both emit "type":"stdio" in their current config schemas.
		entry["type"] = "stdio"
	}
	return entry, warnings, nil
}

func buildRemoteMCPInstallEntry(client, transport string, opts installOptions) (map[string]any, []string, error) {
	if transport != "streamable-http" {
		return nil, nil, fmt.Errorf("URL-backed MCP install only supports streamable-http transport, got %q", transport)
	}
	if err := validateRemoteInstallOptions(client, opts); err != nil {
		return nil, nil, err
	}

	switch client {
	case "antigravity":
		entry := map[string]any{"serverUrl": opts.URL}
		headers, warnings := remoteAuthHeaders(client, opts)
		if len(headers) > 0 {
			entry["headers"] = headers
		}
		return entry, warnings, nil
	case "cursor":
		entry := map[string]any{"url": opts.URL}
		headers, warnings := remoteAuthHeaders(client, opts)
		if len(headers) > 0 {
			entry["headers"] = headers
		}
		return entry, warnings, nil
	case "vscode":
		entry := map[string]any{
			"type": "http",
			"url":  opts.URL,
		}
		headers, warnings := remoteAuthHeaders(client, opts)
		if len(headers) > 0 {
			entry["headers"] = headers
		}
		return entry, warnings, nil
	case "codex":
		entry := map[string]any{
			"url":                  opts.URL,
			"bearer_token_env_var": "FIBE_API_KEY",
		}
		return entry, nil, nil
	default:
		return nil, nil, fmt.Errorf("client %q only supports stdio install today; URL-backed install is supported for antigravity, cursor, vscode, and codex", client)
	}
}

func validateRemoteInstallOptions(client string, opts installOptions) error {
	var unsupported []string
	if opts.Domain != "" {
		unsupported = append(unsupported, "--domain")
	}
	if opts.Profile != "" {
		unsupported = append(unsupported, "--profile")
	}
	if len(opts.Env) > 0 {
		unsupported = append(unsupported, "--env")
	}
	if opts.ToolSet != "" {
		unsupported = append(unsupported, "--tools")
	}
	if opts.Yolo {
		unsupported = append(unsupported, "--yolo")
	}
	if opts.AuditLog != "" {
		unsupported = append(unsupported, "--audit-log")
	}
	if len(unsupported) > 0 {
		return fmt.Errorf("URL-backed MCP install does not launch a local fibe process, so %s do not apply", strings.Join(unsupported, ", "))
	}
	if client == "codex" && opts.APIKey != "" {
		return fmt.Errorf("codex URL mode cannot inline API keys; export FIBE_API_KEY in your shell and use bearer_token_env_var instead")
	}
	return nil
}

func remoteAuthHeaders(client string, opts installOptions) (map[string]string, []string) {
	if opts.APIKey != "" {
		return map[string]string{"Authorization": "Bearer " + opts.APIKey}, nil
	}
	switch client {
	case "cursor", "vscode":
		return map[string]string{"Authorization": "Bearer " + remoteAPIKeyPlaceholder(client)}, nil
	case "antigravity":
		if v := os.Getenv("FIBE_API_KEY"); v != "" {
			return map[string]string{"Authorization": "Bearer " + v}, nil
		}
		return nil, []string{
			"Authorization header omitted — antigravity does not expand ${VAR} placeholders in remote MCP config; rerun with --api-key <key> or export FIBE_API_KEY before install.",
		}
	default:
		return nil, nil
	}
}

func remoteAPIKeyPlaceholder(client string) string {
	switch client {
	case "cursor", "vscode":
		return "${env:FIBE_API_KEY}"
	default:
		return "${FIBE_API_KEY}"
	}
}

// resolveInstallEnv builds the "env" map for the MCP server entry based on
// the install flags and the target client's env-expansion quirks.
//
// Stdio installs pin a Fibe profile and do not forward FIBE_API_KEY or
// FIBE_DOMAIN by default. Explicit --api-key/--domain are represented as
// serve flags; this function only handles non-auth MCP process env.
func resolveInstallEnv(client string, opts installOptions) (map[string]string, []string, []string) {
	var warnings []string
	env := map[string]string{}

	if opts.ToolSet != "" {
		env["FIBE_MCP_TOOLS"] = opts.ToolSet
	}
	if opts.Yolo {
		env["FIBE_MCP_YOLO"] = "1"
	}
	if opts.AuditLog != "" {
		env["FIBE_MCP_AUDIT_LOG"] = opts.AuditLog
	}

	// Arbitrary --env KEY=VALUE pairs (override earlier values).
	warnings = append(warnings, applyInstallEnvPairs(env, opts.Env)...)

	return env, nil, warnings
}

func applyInstallEnvPairs(env map[string]string, pairs []string) []string {
	var warnings []string
	for _, pair := range pairs {
		k, v, ok := strings.Cut(pair, "=")
		if !ok {
			warnings = append(warnings, "ignoring malformed --env entry (expected KEY=VALUE): "+pair)
			continue
		}
		env[k] = v
	}
	return warnings
}

// runMCPUninstall removes the "fibe" entry from the target client's config
// without touching other servers. Mirrors runMCPInstall's resolution.
//
// project is unused for claude-desktop and antigravity because their configs
// are always user-scoped.
func runMCPUninstall(client, project string, dryRun bool) error {
	target, err := resolveMCPClientTarget(client, project)
	if err != nil {
		return err
	}

	data, readErr := os.ReadFile(target.Path)
	if readErr != nil {
		if os.IsNotExist(readErr) {
			fmt.Printf("No config file at %s — nothing to uninstall.\n", target.Path)
			return nil
		}
		return fmt.Errorf("read %s: %w", target.Path, readErr)
	}
	existing, err := parseMCPConfig(data, target.Format)
	if err != nil {
		return fmt.Errorf("parse existing config %s: %w", target.Path, err)
	}

	servers, _ := existing[target.WrapperKey].(map[string]any)
	if servers == nil {
		fmt.Printf("No %q entries in %s — nothing to uninstall.\n", target.WrapperKey, target.Path)
		return nil
	}
	if _, present := servers[mcpServerName]; !present {
		fmt.Printf("No 'fibe' entry in %s — nothing to uninstall.\n", target.Path)
		return nil
	}

	delete(servers, mcpServerName)
	if len(servers) == 0 {
		delete(existing, target.WrapperKey)
	} else {
		existing[target.WrapperKey] = servers
	}

	out, err := marshalMCPConfig(existing, target.Format)
	if err != nil {
		return err
	}
	if dryRun {
		fmt.Printf("# target: %s\n\n%s", target.Path, string(out))
		return nil
	}
	if err := os.WriteFile(target.Path, out, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", target.Path, err)
	}
	fmt.Printf("Removed fibe MCP server from %s\n", target.Path)
	return nil
}

func resolveMCPClientTarget(client, project string) (mcpClientTarget, error) {
	switch client {
	case "claude-code":
		target, err := claudeCodeConfigPath(project)
		return mcpClientTarget{Path: target, WrapperKey: "mcpServers", WrapperDefault: map[string]any{}, Format: mcpConfigJSON}, err
	case "claude-desktop":
		target, err := claudeDesktopConfigPath()
		return mcpClientTarget{Path: target, WrapperKey: "mcpServers", WrapperDefault: map[string]any{}, Format: mcpConfigJSON}, err
	case "cursor":
		target, err := cursorConfigPath(project)
		return mcpClientTarget{Path: target, WrapperKey: "mcpServers", WrapperDefault: map[string]any{}, Format: mcpConfigJSON}, err
	case "vscode":
		target, err := vscodeConfigPath(project)
		return mcpClientTarget{Path: target, WrapperKey: "servers", WrapperDefault: map[string]any{}, Format: mcpConfigJSON}, err
	case "antigravity":
		target, err := antigravityConfigPath()
		return mcpClientTarget{Path: target, WrapperKey: "mcpServers", WrapperDefault: map[string]any{}, Format: mcpConfigJSON}, err
	case "codex":
		target, err := codexConfigPath(project)
		return mcpClientTarget{Path: target, WrapperKey: "mcp_servers", WrapperDefault: map[string]any{}, Format: mcpConfigTOML}, err
	default:
		return mcpClientTarget{}, fmt.Errorf("unknown client %q — valid: %s", client, mcpValidClientList)
	}
}

func parseMCPConfig(data []byte, format mcpConfigFormat) (map[string]any, error) {
	existing := map[string]any{}
	switch format {
	case mcpConfigJSON:
		if err := json.Unmarshal(data, &existing); err != nil {
			return nil, err
		}
	case mcpConfigTOML:
		if _, err := toml.Decode(string(data), &existing); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported config format %q", format)
	}
	return existing, nil
}

func marshalMCPConfig(existing map[string]any, format mcpConfigFormat) ([]byte, error) {
	switch format {
	case mcpConfigJSON:
		out, err := json.MarshalIndent(existing, "", "  ")
		if err != nil {
			return nil, err
		}
		return append(out, '\n'), nil
	case mcpConfigTOML:
		var buf bytes.Buffer
		if err := toml.NewEncoder(&buf).Encode(existing); err != nil {
			return nil, err
		}
		out := buf.Bytes()
		if len(out) == 0 || out[len(out)-1] != '\n' {
			out = append(out, '\n')
		}
		return out, nil
	default:
		return nil, fmt.Errorf("unsupported config format %q", format)
	}
}

func claudeCodeConfigPath(project string) (string, error) {
	if project != "" {
		abs, err := filepath.Abs(project)
		if err != nil {
			return "", err
		}
		return filepath.Join(abs, ".mcp.json"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".claude.json"), nil
}

func claudeDesktopConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "Library", "Application Support", "Claude", "claude_desktop_config.json"), nil
}

func cursorConfigPath(project string) (string, error) {
	if project != "" {
		abs, err := filepath.Abs(project)
		if err != nil {
			return "", err
		}
		return filepath.Join(abs, ".cursor", "mcp.json"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".cursor", "mcp.json"), nil
}

func codexConfigPath(project string) (string, error) {
	if project != "" {
		abs, err := filepath.Abs(project)
		if err != nil {
			return "", err
		}
		return filepath.Join(abs, ".codex", "config.toml"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".codex", "config.toml"), nil
}

// antigravityConfigPath returns the per-user Antigravity MCP config path.
// Antigravity is always user-scoped: ~/.gemini/antigravity/mcp_config.json.
func antigravityConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".gemini", "antigravity", "mcp_config.json"), nil
}

func vscodeConfigPath(project string) (string, error) {
	if project != "" {
		abs, err := filepath.Abs(project)
		if err != nil {
			return "", err
		}
		return filepath.Join(abs, ".vscode", "mcp.json"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".vscode", "mcp.json"), nil
}
