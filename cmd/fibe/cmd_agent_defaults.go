package main

import (
	"fmt"
	"strings"

	"github.com/fibegg/sdk/fibe"
	"github.com/spf13/cobra"
)

func agentDefaultsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "defaults",
		Aliases: []string{"agent-defaults"},
		Short:   "Manage player-level agent defaults",
		Long: `Manage player-level agent defaults used as overrides for newly created and running agents.

These are the same defaults exposed in the profile UI: prompts, model options,
runtime limits, custom environment, MCP JSON, post-init scripts, provider args,
skill toggles, default mounts, and per-provider overrides.`,
	}
	cmd.AddCommand(agentDefaultsGetCmd(), agentDefaultsUpdateCmd(), agentDefaultsResetCmd())
	return cmd
}

func agentDefaultsGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get",
		Short: "Show player-level agent defaults",
		RunE: func(cmd *cobra.Command, args []string) error {
			defaults, err := newClient().AgentDefaults.Get(ctx())
			if err != nil {
				return err
			}
			output(defaults)
			return nil
		},
	}
}

func agentDefaultsResetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "reset",
		Short: "Clear player overrides and use admin agent defaults",
		Long:  "Clear all player-level agent default overrides. After reset, agents inherit admin global and provider defaults unless they have per-agent overrides.",
		RunE: func(cmd *cobra.Command, args []string) error {
			defaults, err := newClient().AgentDefaults.Reset(ctx())
			if err != nil {
				return err
			}
			output(defaults)
			return nil
		},
	}
}

func agentDefaultsUpdateCmd() *cobra.Command {
	var provider string
	var systemPrompt, modelOptions, memoryLimit, cpuLimit, cliVersion, providerCLIVersion string
	var customEnv, mcpJSON, postInitScript, providerArgs string
	var syscheckEnabled bool
	var skillToggleFlags []string

	cmd := &cobra.Command{
		Use:   "update",
		Short: "Replace player-level agent defaults",
		Long: `Replace player-level agent defaults.

Pass a JSON/YAML object with -f/--from-file. The file may either be the defaults
object itself or an envelope like {"agent_defaults": {...}}.

Flags build a defaults object directly. Pass --provider to write the flags under
provider_overrides.<provider>; otherwise they write global defaults.

EXAMPLES:
  fibe agents defaults get -o json
  fibe agents defaults update -f defaults.json
  fibe agents defaults update --system-prompt "You are Fibe" --custom-env "DEBUG=true"
  fibe agents defaults update --provider gemini --model-options gemini-pro --syscheck=false
  fibe agents defaults update --provider-args "--bare --max-tokens 4096"
  fibe agents defaults update --skill-toggle fibe-hunks.md=false`,
		RunE: func(cmd *cobra.Command, args []string) error {
			defaults, fromFile, err := agentDefaultsFromInput()
			if err != nil {
				return err
			}
			if defaults == nil {
				defaults = fibe.AgentDefaults{}
			}

			scope := agentDefaultsScope(defaults, provider)
			changed := fromFile
			setStringDefault(cmd, scope, "system-prompt", "system_prompt", systemPrompt, &changed)
			setStringDefault(cmd, scope, "model-options", "model_options", modelOptions, &changed)
			setStringDefault(cmd, scope, "memory-limit", "memory_limit", memoryLimit, &changed)
			setStringDefault(cmd, scope, "cpu-limit", "cpu_limit", cpuLimit, &changed)
			setStringDefault(cmd, scope, "cli-version", "cli_version", cliVersion, &changed)
			setStringDefault(cmd, scope, "provider-cli-version", "provider_cli_version", providerCLIVersion, &changed)
			setStringDefault(cmd, scope, "custom-env", "custom_env", customEnv, &changed)
			setStringDefault(cmd, scope, "mcp-json", "mcp_json", mcpJSON, &changed)
			setStringDefault(cmd, scope, "post-init-script", "post_init_script", postInitScript, &changed)
			setStringDefault(cmd, scope, "provider-args", "provider_args_cli", providerArgs, &changed)
			if cmd.Flags().Changed("syscheck") {
				scope["syscheck_enabled"] = syscheckEnabled
				changed = true
			}
			toggles, err := parseSkillToggleFlags(skillToggleFlags)
			if err != nil {
				return err
			}
			if len(toggles) > 0 {
				scope["skill_toggles"] = toggles
				changed = true
			}
			if !changed {
				return fmt.Errorf("provide defaults with --from-file or at least one override flag")
			}

			result, err := newClient().AgentDefaults.Update(ctx(), defaults)
			if err != nil {
				return err
			}
			output(result)
			return nil
		},
	}

	cmd.Flags().StringVar(&provider, "provider", "", "Provider override key, for example gemini or claude-code")
	cmd.Flags().StringVar(&systemPrompt, "system-prompt", "", "Default system prompt")
	cmd.Flags().StringVar(&modelOptions, "model-options", "", "Default provider model option")
	cmd.Flags().StringVar(&memoryLimit, "memory-limit", "", "Default memory limit, for example 2G")
	cmd.Flags().StringVar(&cpuLimit, "cpu-limit", "", "Default CPU limit, for example 1.5")
	cmd.Flags().StringVar(&cliVersion, "cli-version", "", "Default Fibe CLI version pin")
	cmd.Flags().StringVar(&providerCLIVersion, "provider-cli-version", "", "Default provider CLI version pin")
	cmd.Flags().StringVar(&customEnv, "custom-env", "", "Default KEY=VALUE lines injected into the agent runtime")
	cmd.Flags().StringVar(&mcpJSON, "mcp-json", "", "Default MCP configuration JSON")
	cmd.Flags().StringVar(&postInitScript, "post-init-script", "", "Default post-init shell script")
	cmd.Flags().StringVar(&providerArgs, "provider-args", "", "Default provider CLI flags, for example \"--bare --max-tokens 4096\"")
	cmd.Flags().BoolVar(&syscheckEnabled, "syscheck", false, "Default system-check setting")
	cmd.Flags().StringArrayVar(&skillToggleFlags, "skill-toggle", nil, "Default skill toggle as filename=true|false (repeatable)")
	return cmd
}

func agentDefaultsFromInput() (fibe.AgentDefaults, bool, error) {
	raw := map[string]any{}
	if err := applyFromFile(&raw); err != nil {
		return nil, false, err
	}
	if rawPayload == nil {
		return nil, false, nil
	}
	if nested, ok := raw["agent_defaults"].(map[string]any); ok {
		return fibe.AgentDefaults(nested), true, nil
	}
	return fibe.AgentDefaults(raw), true, nil
}

func agentDefaultsScope(defaults fibe.AgentDefaults, provider string) map[string]any {
	provider = strings.TrimSpace(provider)
	if provider == "" {
		return defaults
	}
	overrides, ok := defaults["provider_overrides"].(map[string]any)
	if !ok {
		overrides = map[string]any{}
		defaults["provider_overrides"] = overrides
	}
	scope, ok := overrides[provider].(map[string]any)
	if !ok {
		scope = map[string]any{}
		overrides[provider] = scope
	}
	return scope
}

func setStringDefault(cmd *cobra.Command, scope map[string]any, flagName, key, value string, changed *bool) {
	if !cmd.Flags().Changed(flagName) {
		return
	}
	scope[key] = value
	*changed = true
}
