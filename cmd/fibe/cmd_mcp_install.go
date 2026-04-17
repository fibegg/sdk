package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// installOptions bundles the flags that shape the emitted MCP config entry.
// Each field is optional; empty means "use the default".
type installOptions struct {
	APIKey    string   // literal value to inline for FIBE_API_KEY
	Domain    string   // literal value to inline for FIBE_DOMAIN
	Env       []string // arbitrary KEY=VALUE pairs to inline
	ToolSet   string   // "core" | "full" (passed as FIBE_MCP_TOOLS)
	Yolo      bool     // sets FIBE_MCP_YOLO=1
	AuditLog  string   // sets FIBE_MCP_AUDIT_LOG
}

// runMCPInstall writes the fibe MCP server entry into the target client's
// configuration file. Works with claude-code, claude-desktop, cursor, vscode,
// antigravity.
//
// Design notes:
//   - Never clobber unrelated entries: we read the existing config, merge
//     our "fibe" entry, and write the file back with the same shape.
//   - Dry-run mode prints the resolved path and the proposed delta so users
//     can review before overwriting.
//   - Each client has a slightly different schema (claude uses "mcpServers",
//     vscode uses "servers") so we keep a small per-client adapter.
//   - Some clients (e.g. Antigravity) don't expand ${VAR} placeholders in
//     env values. When --api-key is omitted for those, we auto-resolve
//     FIBE_API_KEY from the parent shell so the entry works out of the box.
func runMCPInstall(client, project string, dryRun bool, opts installOptions) error {
	bin, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve fibe binary path: %w", err)
	}

	env, warnings := resolveInstallEnv(client, opts)
	entry := map[string]any{
		"command": bin,
		"args":    []string{"mcp", "serve"},
		"env":     env,
	}

	var target string
	var wrapperKey string
	var wrapperDefault map[string]any
	switch client {
	case "claude-code":
		target, err = claudeCodeConfigPath(project)
		wrapperKey = "mcpServers"
		wrapperDefault = map[string]any{}
	case "claude-desktop":
		target, err = claudeDesktopConfigPath()
		wrapperKey = "mcpServers"
		wrapperDefault = map[string]any{}
	case "cursor":
		target, err = cursorConfigPath(project)
		wrapperKey = "mcpServers"
		wrapperDefault = map[string]any{}
	case "vscode":
		target, err = vscodeConfigPath(project)
		wrapperKey = "servers"
		wrapperDefault = map[string]any{}
		// VS Code's schema uses "type":"stdio" explicitly.
		entry["type"] = "stdio"
	case "antigravity":
		target, err = antigravityConfigPath()
		wrapperKey = "mcpServers"
		wrapperDefault = map[string]any{}
	default:
		return fmt.Errorf("unknown client %q — valid: claude-code, claude-desktop, cursor, vscode, antigravity", client)
	}
	if err != nil {
		return err
	}

	existing := map[string]any{}
	if data, readErr := os.ReadFile(target); readErr == nil && len(data) > 0 {
		if err := json.Unmarshal(data, &existing); err != nil {
			return fmt.Errorf("parse existing config %s: %w", target, err)
		}
	}

	servers, _ := existing[wrapperKey].(map[string]any)
	if servers == nil {
		servers = wrapperDefault
	}
	servers["fibe"] = entry
	existing[wrapperKey] = servers

	out, err := json.MarshalIndent(existing, "", "  ")
	if err != nil {
		return err
	}

	for _, w := range warnings {
		fmt.Fprintf(os.Stderr, "WARN: %s\n", w)
	}

	if dryRun {
		fmt.Printf("# target: %s\n\n%s\n", target, string(out))
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return fmt.Errorf("create parent dir: %w", err)
	}
	if err := os.WriteFile(target, append(out, '\n'), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", target, err)
	}
	fmt.Printf("Installed fibe MCP server into %s\n", target)
	return nil
}

// resolveInstallEnv builds the "env" map for the MCP server entry based on
// the install flags and the target client's env-expansion quirks.
//
// For clients that support ${VAR} expansion (claude-code, claude-desktop,
// cursor, vscode): we keep "${FIBE_API_KEY}" as the placeholder unless the
// caller supplied --api-key.
//
// For clients that don't (antigravity): we auto-resolve FIBE_API_KEY from
// the parent shell at install time. If the shell doesn't have it, we emit a
// warning and leave an empty string so the user can edit the file.
func resolveInstallEnv(client string, opts installOptions) (map[string]string, []string) {
	var warnings []string
	env := map[string]string{}
	expandsPlaceholders := clientExpandsEnvPlaceholders(client)

	// API key.
	switch {
	case opts.APIKey != "":
		env["FIBE_API_KEY"] = opts.APIKey
	case expandsPlaceholders:
		env["FIBE_API_KEY"] = "${FIBE_API_KEY}"
	default:
		if v := os.Getenv("FIBE_API_KEY"); v != "" {
			env["FIBE_API_KEY"] = v
		} else {
			env["FIBE_API_KEY"] = ""
			warnings = append(warnings,
				"FIBE_API_KEY left empty — "+client+" does not expand ${VAR} placeholders; "+
					"rerun with --api-key <key> or edit the config manually.")
		}
	}

	// Domain override.
	switch {
	case opts.Domain != "":
		env["FIBE_DOMAIN"] = opts.Domain
	case expandsPlaceholders:
		env["FIBE_DOMAIN"] = "${FIBE_DOMAIN}"
	default:
		if v := os.Getenv("FIBE_DOMAIN"); v != "" {
			env["FIBE_DOMAIN"] = v
		} else {
			env["FIBE_DOMAIN"] = "fibe.gg"
		}
	}

	// Tool set + yolo + audit log.
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
	for _, pair := range opts.Env {
		k, v, ok := strings.Cut(pair, "=")
		if !ok {
			warnings = append(warnings, "ignoring malformed --env entry (expected KEY=VALUE): "+pair)
			continue
		}
		env[k] = v
	}

	return env, warnings
}

// clientExpandsEnvPlaceholders returns true for MCP clients known to expand
// ${VAR} syntax inside env values. Antigravity does not; Claude Code,
// Claude Desktop, Cursor, and VS Code do (at least as of this writing).
func clientExpandsEnvPlaceholders(client string) bool {
	switch client {
	case "antigravity":
		return false
	default:
		return true
	}
}

// runMCPUninstall removes the "fibe" entry from the target client's config
// without touching other servers. Mirrors runMCPInstall's resolution.
//
// project is unused for claude-desktop and antigravity because their configs
// are always user-scoped.
func runMCPUninstall(client, project string, dryRun bool) error {
	var target string
	var wrapperKey string
	var err error
	switch client {
	case "claude-code":
		target, err = claudeCodeConfigPath(project)
		wrapperKey = "mcpServers"
	case "claude-desktop":
		target, err = claudeDesktopConfigPath()
		wrapperKey = "mcpServers"
	case "cursor":
		target, err = cursorConfigPath(project)
		wrapperKey = "mcpServers"
	case "vscode":
		target, err = vscodeConfigPath(project)
		wrapperKey = "servers"
	case "antigravity":
		target, err = antigravityConfigPath()
		wrapperKey = "mcpServers"
	default:
		return fmt.Errorf("unknown client %q — valid: claude-code, claude-desktop, cursor, vscode, antigravity", client)
	}
	if err != nil {
		return err
	}

	data, readErr := os.ReadFile(target)
	if readErr != nil {
		if os.IsNotExist(readErr) {
			fmt.Printf("No config file at %s — nothing to uninstall.\n", target)
			return nil
		}
		return fmt.Errorf("read %s: %w", target, readErr)
	}
	existing := map[string]any{}
	if len(data) > 0 {
		if err := json.Unmarshal(data, &existing); err != nil {
			return fmt.Errorf("parse existing config %s: %w", target, err)
		}
	}

	servers, _ := existing[wrapperKey].(map[string]any)
	if servers == nil {
		fmt.Printf("No %q entries in %s — nothing to uninstall.\n", wrapperKey, target)
		return nil
	}
	if _, present := servers["fibe"]; !present {
		fmt.Printf("No 'fibe' entry in %s — nothing to uninstall.\n", target)
		return nil
	}

	delete(servers, "fibe")
	if len(servers) == 0 {
		delete(existing, wrapperKey)
	} else {
		existing[wrapperKey] = servers
	}

	out, err := json.MarshalIndent(existing, "", "  ")
	if err != nil {
		return err
	}
	if dryRun {
		fmt.Printf("# target: %s\n\n%s\n", target, string(out))
		return nil
	}
	if err := os.WriteFile(target, append(out, '\n'), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", target, err)
	}
	fmt.Printf("Removed fibe MCP server from %s\n", target)
	return nil
}

func claudeCodeConfigPath(project string) (string, error) {
	if project != "" {
		abs, err := filepath.Abs(project)
		if err != nil {
			return "", err
		}
		return filepath.Join(abs, ".claude", "settings.json"), nil
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
