package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/fibegg/sdk/fibe"
	"github.com/spf13/cobra"
)

func agentsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "agents",
		Aliases: []string{"ag"},
		Short:   "Manage AI agents",
		Long: `Manage Fibe agents — AI-powered assistants that work with your playgrounds.

Agents can be authenticated with GitHub, have mounted files, store messages
and activity logs, create artefacts, and interact via chat.

PROVIDERS:
  gemini, antigravity, claude-code, openai-codex, opencode, cursor

SUBCOMMANDS:
  list                  List all agents
  get <id-or-name>      Show agent details
  create                Create a new agent
  update <id-or-name>   Update agent settings
  delete <id-or-name>   Delete an agent
  duplicate <id-or-name> Clone an agent
  start-chat <id-or-name> Start an interactive chat session on a Marquee
  restart-chat <id-or-name> Restart the current chat container
  runtime-status <id-or-name> Show agent chat live status
  watch                 Watch agent resource events
  purge-chat <id-or-name> Tear down agent chat container and volumes
  chat <id-or-name>     Send a chat message
  upload-attachment <id-or-name> Upload a chat attachment
  download-attachment <id-or-name> <filename> Download a chat attachment
  authenticate <id-or-name> Authenticate agent with provider
  add-mounted-file <id-or-name> Attach a mounted file or Artefact snapshot
  update-mounted-file <id-or-name> Update mounted file metadata
  remove-mounted-file <id-or-name> Remove a mounted file
  pokes                 Manage scheduled agent pokes
  messages <id-or-name> Get agent messages
  set-messages <id-or-name> Replace agent messages content
  activity <id-or-name> Get agent activity
  set-activity <id-or-name> Replace agent activity content
  gitea-token <id-or-name> Get agent's Gitea token
  defaults              Manage player-level agent defaults`,
	}

	cmd.AddCommand(
		agListCmd(),
		agGetCmd(),
		agCreateCmd(),
		agUpdateCmd(),
		agDeleteCmd(),
		agDuplicateCmd(),
		agStartChatCmd(),
		agRestartChatCmd(),
		agRuntimeStatusCmd(),
		agWatchCmd(),
		agLiveStateCmd(),
		agCreateConversationCmd(),
		agDeleteConversationCmd(),
		agInterruptCmd(),
		agPurgeChatCmd(),
		agSendMessageCmd(),
		agUploadAttachmentCmd(),
		agDownloadAttachmentCmd(),
		agAuthCmd(),
		agAddMountedFileCmd(),
		agUpdateMountedFileCmd(),
		agRemoveMountedFileCmd(),
		agPokesCmd(),
		agMessagesCmd(),
		agSetMessagesCmd(),
		agActivityCmd(),
		agSetActivityCmd(),
		agGiteaTokenCmd(),
		agentDefaultsCmd(),
	)
	return cmd
}

func agListCmd() *cobra.Command {
	var query, provider, status, name, sort, createdAfter, createdBefore string
	var includeRuntimeStatus bool
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all agents",
		Long: `List all agents accessible to the authenticated user.

FILTERS:
  -q, --query           Search across name, description (substring match)
  --provider            Filter by provider. Values: gemini, antigravity, claude-code, openai-codex, opencode, cursor
  --status              Filter by exact status. Values: pending, authenticated, error
  --name                Filter by name (substring match)

DATE RANGE:
  --created-after       Show items created on or after this date (ISO 8601)
  --created-before      Show items created on or before this date (ISO 8601)

SORTING:
  --sort                Sort results. Format: {column}_{direction}
                        Columns: created_at, name
                        Direction: asc, desc
                        Default: created_at_desc

OUTPUT:
  Columns: ID, NAME, PROVIDER, STATUS, AUTH
  Add --include-runtime-status to include runtime reachability/queue columns.
  Use --output json for full details.

EXAMPLES:
  fibe agents list
  fibe ag list -q "my-agent" --status authenticated
  fibe ag list --provider gemini --sort name_asc
  fibe ag list --created-after 2026-01-01 -o json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			params := &fibe.AgentListParams{}
			if query != "" {
				params.Q = query
			}
			if provider != "" {
				params.Provider = provider
			}
			if status != "" {
				params.Status = status
			}
			if name != "" {
				params.Name = name
			}
			if createdAfter != "" {
				params.CreatedAfter = createdAfter
			}
			if createdBefore != "" {
				params.CreatedBefore = createdBefore
			}
			if sort != "" {
				params.Sort = sort
			}
			if includeRuntimeStatus {
				params.IncludeRuntimeStatus = &includeRuntimeStatus
			}
			if flagPage > 0 {
				params.Page = flagPage
			}
			if flagPerPage > 0 {
				params.PerPage = flagPerPage
			}
			agents, err := c.Agents.List(ctx(), params)
			if err != nil {
				return err
			}
			if effectiveOutput() != "table" {
				outputJSON(agents)
				return nil
			}
			headers := []string{"ID", "NAME", "PROVIDER", "STATUS", "AUTH"}
			if includeRuntimeStatus {
				headers = append(headers, "RUNTIME", "REACHABLE", "BUSY", "QUEUE")
			}
			rows := make([][]string, len(agents.Data))
			for i, a := range agents.Data {
				row := []string{
					fmtInt64(a.ID), a.Name, a.Provider, a.Status, fmtBool(a.Authenticated),
				}
				if includeRuntimeStatus {
					row = append(row, agentRuntimeStatusColumns(a.RuntimeStatus)...)
				}
				rows[i] = row
			}
			outputTable(headers, rows)
			return nil
		},
	}
	cmd.Flags().StringVarP(&query, "query", "q", "", "Search across name, description")
	cmd.Flags().StringVar(&provider, "provider", "", "Filter by provider (gemini, antigravity, claude-code, openai-codex, opencode, cursor)")
	cmd.Flags().StringVar(&status, "status", "", "Filter by status")
	cmd.Flags().StringVar(&name, "name", "", "Filter by name (substring)")
	cmd.Flags().StringVar(&createdAfter, "created-after", "", "Filter: created after date (ISO 8601)")
	cmd.Flags().StringVar(&createdBefore, "created-before", "", "Filter: created before date (ISO 8601)")
	cmd.Flags().StringVar(&sort, "sort", "", "Sort order (e.g. created_at_desc)")
	cmd.Flags().BoolVar(&includeRuntimeStatus, "include-runtime-status", false, "Include runtime status for listed agents (opt-in; may probe running runtimes)")
	return cmd
}

func agGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <id-or-name>",
		Short: "Show agent details",
		Long: `Get detailed information about a specific agent.

EXAMPLES:
  fibe agents get builder
  fibe ag get my-agent --output json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			agent, err := c.Agents.GetByIdentifier(ctx(), args[0])
			if err != nil {
				return err
			}
			if effectiveOutput() != "table" {
				outputJSON(agent)
				return nil
			}
			fmt.Printf("ID:            %d\n", agent.ID)
			fmt.Printf("Name:          %s\n", agent.Name)
			fmt.Printf("Provider:      %s (%s)\n", agent.Provider, agent.ProviderLabel)
			fmt.Printf("Status:        %s\n", agent.Status)
			fmt.Printf("Authenticated: %s\n", fmtBool(agent.Authenticated))
			fmt.Printf("Sync:          %s\n", fmtBool(agent.SyncEnabled))
			fmt.Printf("Syscheck:      %s\n", fmtBool(agent.SyscheckEnabled))
			fmt.Printf("Created:       %s\n", fmtTime(agent.CreatedAt))
			return nil
		},
	}
}

func agentRuntimeStatusColumns(status *fibe.AgentRuntimeStatus) []string {
	if status == nil {
		return []string{"", "", "", ""}
	}
	return []string{
		status.Status,
		fmtBool(status.RuntimeReachable),
		fmtBool(status.IsProcessing),
		strconv.Itoa(status.QueueCount),
	}
}

func agCreateCmd() *cobra.Command {
	var name, provider, modelOptions, memoryLimit, cpuLimit string
	var prompt, systemPromptMode, mainMD, mainMDMode, mcpJSON, postInitScript, customEnv, cliVersion, providerArgs string
	var skillToggleFlags []string
	var mountFiles, mountArtefacts []string
	var playgroundCrumbsID string
	var syncEnabled, syscheckEnabled, providerAPIKeyMode, buildInPublic bool

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new agent",
		Long: `Create a new AI agent with the specified provider.

REQUIRED FLAGS:
  --name       Agent name
  --provider   Provider type: gemini, antigravity, claude-code, openai-codex, opencode, cursor

OPTIONAL FLAGS:
  --sync           Enable sync
  --syscheck       Enable system checks
  --provider-api-key-mode
                  Use provider API key authentication instead of subscription/OAuth mode
  --model-options Pin provider model option for this agent
  --memory-limit   Memory limit, for example 2G
  --cpu-limit      CPU limit, for example 1.5
  --prompt         Agent-specific system prompt override
  --system-prompt-mode default, append, or override
  --main-md        Agent-specific main.md override
  --main-md-mode   default, append, or override
  --mcp-json       MCP configuration JSON for the runtime
  --post-init-script
                  Shell script run after agent initialization
  --custom-env     KEY=VALUE lines injected into the agent runtime
  --cli-version    Fibe CLI version pin for this agent
  --provider-args  Provider CLI flags, for example "--bare --max-tokens 4096"
  --skill-toggle   Skill toggle as filename=true|false (repeatable)
  --build-in-public
                  Show the agent on the public profile when enabled
  --playground-crumbs-id
                  Playground ID or name for public timeline crumbs
  --mount-file     Local file mount as ./path:%{agent_data}/target.ext (repeatable)
  --mount-artefact Artefact snapshot mount as docs-bundle:%{workspace}/docs/file.md (repeatable)

EXAMPLES:
  fibe agents create --name my-agent --provider claude-code
  fibe ag create --name builder --provider opencode --provider-api-key-mode --model-options openai/gpt-4.1` + generateSchemaDoc(&fibe.AgentCreateParams{}),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			params := &fibe.AgentCreateParams{}
			if err := applyFromFile(params); err != nil {
				return err
			}

			if cmd.Flags().Changed("name") {
				params.Name = name
			}
			if cmd.Flags().Changed("provider") {
				params.Provider = provider
			}
			if cmd.Flags().Changed("sync") {
				params.SyncEnabled = &syncEnabled
			}
			if cmd.Flags().Changed("syscheck") {
				params.SyscheckEnabled = &syscheckEnabled
			}
			if cmd.Flags().Changed("build-in-public") {
				params.BuildInPublic = &buildInPublic
			}
			if cmd.Flags().Changed("playground-crumbs-id") {
				params.BuildInPublicPlaygroundIdentifier = playgroundCrumbsID
			}
			if cmd.Flags().Changed("provider-api-key-mode") {
				params.ProviderAPIKeyMode = &providerAPIKeyMode
			}
			if cmd.Flags().Changed("model-options") {
				params.ModelOptions = &modelOptions
			}
			if cmd.Flags().Changed("memory-limit") {
				params.MemoryLimit = &memoryLimit
			}
			if cmd.Flags().Changed("cpu-limit") {
				params.CpuLimit = &cpuLimit
			}
			if cmd.Flags().Changed("prompt") {
				params.Prompt = &prompt
			}
			if cmd.Flags().Changed("system-prompt-mode") {
				params.SystemPromptMode = &systemPromptMode
			}
			if cmd.Flags().Changed("main-md") {
				params.MainMD = &mainMD
			}
			if cmd.Flags().Changed("main-md-mode") {
				params.MainMDMode = &mainMDMode
			}
			if cmd.Flags().Changed("mcp-json") {
				params.MCPJSON = &mcpJSON
			}
			if cmd.Flags().Changed("post-init-script") {
				params.PostInitScript = &postInitScript
			}
			if cmd.Flags().Changed("custom-env") {
				params.CustomEnv = &customEnv
			}
			if cmd.Flags().Changed("cli-version") {
				params.CLIVersion = &cliVersion
			}
			if cmd.Flags().Changed("provider-args") {
				params.ProviderArgsCLI = &providerArgs
			}
			toggles, err := parseSkillToggleFlags(skillToggleFlags)
			if err != nil {
				return err
			}
			if len(toggles) > 0 {
				params.SkillToggles = toggles
			}
			mounts, err := parseAgentCreateMountFlags(mountFiles, mountArtefacts)
			if err != nil {
				return err
			}
			if len(mounts) > 0 {
				params.Mounts = mounts
			}

			if params.Name == "" {
				return fmt.Errorf("required field 'name' not set")
			}

			agent, err := c.Agents.Create(ctx(), params)
			if err != nil {
				return err
			}
			if effectiveOutput() != "table" {
				outputJSON(agent)
				return nil
			}
			fmt.Printf("Created agent %d (%s)\n", agent.ID, agent.Name)
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Agent name (required)")
	cmd.Flags().StringVar(&provider, "provider", "gemini", "Provider: gemini, antigravity, claude-code, openai-codex, opencode, cursor")
	cmd.Flags().BoolVar(&syncEnabled, "sync", false, "Enable sync")
	cmd.Flags().BoolVar(&syscheckEnabled, "syscheck", false, "Enable system checks")
	cmd.Flags().BoolVar(&buildInPublic, "build-in-public", false, "Show this agent on the public profile")
	cmd.Flags().StringVar(&playgroundCrumbsID, "playground-crumbs-id", "", "Playground ID or name for public timeline crumbs")
	cmd.Flags().BoolVar(&providerAPIKeyMode, "provider-api-key-mode", false, "Use provider API key auth mode")
	cmd.Flags().StringVar(&modelOptions, "model-options", "", "Provider model option")
	cmd.Flags().StringVar(&memoryLimit, "memory-limit", "", "Memory limit, for example 2G")
	cmd.Flags().StringVar(&cpuLimit, "cpu-limit", "", "CPU limit, for example 1.5")
	cmd.Flags().StringVar(&prompt, "prompt", "", "Agent-specific system prompt override")
	cmd.Flags().StringVar(&systemPromptMode, "system-prompt-mode", "", "System prompt mode: default, append, override")
	cmd.Flags().StringVar(&mainMD, "main-md", "", "Agent-specific main.md override")
	cmd.Flags().StringVar(&mainMDMode, "main-md-mode", "", "main.md mode: default, append, override")
	cmd.Flags().StringVar(&mcpJSON, "mcp-json", "", "MCP configuration JSON")
	cmd.Flags().StringVar(&postInitScript, "post-init-script", "", "Shell script run after agent initialization")
	cmd.Flags().StringVar(&customEnv, "custom-env", "", "KEY=VALUE lines injected into the agent runtime")
	cmd.Flags().StringVar(&cliVersion, "cli-version", "", "Fibe CLI version pin")
	cmd.Flags().StringVar(&providerArgs, "provider-args", "", "Provider CLI flags, for example \"--bare --max-tokens 4096\"")
	cmd.Flags().StringArrayVar(&skillToggleFlags, "skill-toggle", nil, "Skill toggle as filename=true|false (repeatable)")
	cmd.Flags().StringArrayVar(&mountFiles, "mount-file", nil, "Local file mount as ./path:%{agent_data}/target.ext (repeatable)")
	cmd.Flags().StringArrayVar(&mountArtefacts, "mount-artefact", nil, "Artefact snapshot mount as id-or-name:%{workspace}/docs/file.md (repeatable)")
	return cmd
}

func agUpdateCmd() *cobra.Command {
	var name, modelOptions, memoryLimit, cpuLimit string
	var prompt, systemPromptMode, mainMD, mainMDMode, mcpJSON, postInitScript, customEnv, cliVersion, providerArgs string
	var skillToggleFlags []string
	var syncEnabled, syscheckEnabled bool
	var buildInPublicPlaygroundID string

	cmd := &cobra.Command{
		Use:   "update <id-or-name>",
		Short: "Update agent settings",
		Long: `Update an existing agent's configuration.

OPTIONAL FLAGS:
  --name                          New agent name
  --sync                          Enable/disable sync
  --syscheck                      Enable/disable system checks
  --model-options                 Provider model option
  --memory-limit                  Memory limit, for example 2G
  --cpu-limit                     CPU limit, for example 1.5
  --prompt                        Agent-specific system prompt override
  --system-prompt-mode            default, append, or override
  --main-md                       Agent-specific main.md override
  --main-md-mode                  default, append, or override
  --mcp-json                      MCP configuration JSON for the runtime
  --post-init-script              Shell script run after agent initialization
  --custom-env                    KEY=VALUE lines injected into the agent runtime
  --cli-version                   Fibe CLI version pin for this agent
  --provider-args                 Provider CLI flags, for example "--bare --max-tokens 4096"
  --skill-toggle                  Skill toggle as filename=true|false (repeatable)
  --build-in-public-playground-id Playground ID or name for public builds

EXAMPLES:
  fibe agents update builder --name new-name
  fibe ag update my-agent --sync=false --memory-limit 1024` + generateSchemaDoc(&fibe.AgentUpdateParams{}),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			params := &fibe.AgentUpdateParams{}
			if err := applyFromFile(params); err != nil {
				return err
			}
			if cmd.Flags().Changed("name") {
				params.Name = &name
			}
			if cmd.Flags().Changed("sync") {
				params.SyncEnabled = &syncEnabled
			}
			if cmd.Flags().Changed("syscheck") {
				params.SyscheckEnabled = &syscheckEnabled
			}
			if cmd.Flags().Changed("model-options") {
				params.ModelOptions = &modelOptions
			}
			if cmd.Flags().Changed("memory-limit") {
				params.MemoryLimit = &memoryLimit
			}
			if cmd.Flags().Changed("cpu-limit") {
				params.CpuLimit = &cpuLimit
			}
			if cmd.Flags().Changed("build-in-public-playground-id") {
				params.BuildInPublicPlaygroundIdentifier = buildInPublicPlaygroundID
			}
			if cmd.Flags().Changed("prompt") {
				params.Prompt = &prompt
			}
			if cmd.Flags().Changed("system-prompt-mode") {
				params.SystemPromptMode = &systemPromptMode
			}
			if cmd.Flags().Changed("main-md") {
				params.MainMD = &mainMD
			}
			if cmd.Flags().Changed("main-md-mode") {
				params.MainMDMode = &mainMDMode
			}
			if cmd.Flags().Changed("mcp-json") {
				params.MCPJSON = &mcpJSON
			}
			if cmd.Flags().Changed("post-init-script") {
				params.PostInitScript = &postInitScript
			}
			if cmd.Flags().Changed("custom-env") {
				params.CustomEnv = &customEnv
			}
			if cmd.Flags().Changed("cli-version") {
				params.CLIVersion = &cliVersion
			}
			if cmd.Flags().Changed("provider-args") {
				params.ProviderArgsCLI = &providerArgs
			}
			toggles, err := parseSkillToggleFlags(skillToggleFlags)
			if err != nil {
				return err
			}
			if len(toggles) > 0 {
				params.SkillToggles = toggles
			}
			agent, err := c.Agents.UpdateByIdentifier(ctx(), args[0], params)
			if err != nil {
				return err
			}
			if effectiveOutput() != "table" {
				outputJSON(agent)
				return nil
			}
			fmt.Printf("Updated agent %d (%s)\n", agent.ID, agent.Name)
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "New agent name")
	cmd.Flags().BoolVar(&syncEnabled, "sync", false, "Enable sync")
	cmd.Flags().BoolVar(&syscheckEnabled, "syscheck", false, "Enable system checks")
	cmd.Flags().StringVar(&modelOptions, "model-options", "", "Provider model option")
	cmd.Flags().StringVar(&memoryLimit, "memory-limit", "", "Memory limit, for example 2G")
	cmd.Flags().StringVar(&cpuLimit, "cpu-limit", "", "CPU limit, for example 1.5")
	cmd.Flags().StringVar(&prompt, "prompt", "", "Agent-specific system prompt override")
	cmd.Flags().StringVar(&systemPromptMode, "system-prompt-mode", "", "System prompt mode: default, append, override")
	cmd.Flags().StringVar(&mainMD, "main-md", "", "Agent-specific main.md override")
	cmd.Flags().StringVar(&mainMDMode, "main-md-mode", "", "main.md mode: default, append, override")
	cmd.Flags().StringVar(&mcpJSON, "mcp-json", "", "MCP configuration JSON")
	cmd.Flags().StringVar(&postInitScript, "post-init-script", "", "Shell script run after agent initialization")
	cmd.Flags().StringVar(&customEnv, "custom-env", "", "KEY=VALUE lines injected into the agent runtime")
	cmd.Flags().StringVar(&cliVersion, "cli-version", "", "Fibe CLI version pin")
	cmd.Flags().StringVar(&providerArgs, "provider-args", "", "Provider CLI flags, for example \"--bare --max-tokens 4096\"")
	cmd.Flags().StringArrayVar(&skillToggleFlags, "skill-toggle", nil, "Skill toggle as filename=true|false (repeatable)")
	cmd.Flags().StringVar(&buildInPublicPlaygroundID, "build-in-public-playground-id", "", "Playground ID or name for public builds")
	return cmd
}

func parseSkillToggleFlags(flags []string) (map[string]bool, error) {
	toggles := map[string]bool{}
	for _, raw := range flags {
		name, value, ok := strings.Cut(raw, "=")
		name = strings.TrimSpace(name)
		value = strings.TrimSpace(value)
		if !ok || name == "" || value == "" {
			return nil, fmt.Errorf("--skill-toggle must be filename=true|false")
		}
		enabled, err := strconv.ParseBool(value)
		if err != nil {
			return nil, fmt.Errorf("--skill-toggle %q has invalid boolean value: %w", raw, err)
		}
		toggles[name] = enabled
	}
	return toggles, nil
}

func parseAgentCreateMountFlags(fileSpecs, artefactSpecs []string) ([]fibe.AgentMountSpec, error) {
	mounts := make([]fibe.AgentMountSpec, 0, len(fileSpecs)+len(artefactSpecs))
	for _, spec := range fileSpecs {
		source, target, err := splitMountSpec(spec, "--mount-file")
		if err != nil {
			return nil, err
		}
		mounts = append(mounts, fibe.AgentMountSpec{
			SourceType:  "upload",
			Filename:    filepath.Base(source),
			ContentPath: source,
			MountPath:   target,
		})
	}
	for _, spec := range artefactSpecs {
		source, target, err := splitMountSpec(spec, "--mount-artefact")
		if err != nil {
			return nil, err
		}
		identifier := strings.TrimSpace(source)
		if identifier == "" {
			return nil, fmt.Errorf("--mount-artefact source must be an artefact ID or name")
		}
		mounts = append(mounts, fibe.AgentMountSpec{
			SourceType:         "artefact",
			ArtefactIdentifier: identifier,
			MountPath:          target,
		})
	}
	return mounts, nil
}

func splitMountSpec(spec, flag string) (string, string, error) {
	parts := strings.SplitN(spec, ":", 2)
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
		return "", "", fmt.Errorf("%s must be SOURCE:TARGET_PATH", flag)
	}
	return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]), nil
}

func agDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <id-or-name>",
		Short: "Delete an agent",
		Long: `Delete an agent and all its associated data (messages, artefacts, etc.).

WARNING: This action is irreversible.

EXAMPLES:
  fibe agents delete my-agent`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			if err := c.Agents.DeleteByIdentifier(ctx(), args[0]); err != nil {
				return err
			}
			fmt.Printf("Agent %s deleted\n", args[0])
			return nil
		},
	}
}

func agDuplicateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "duplicate <id-or-name>",
		Short: "Clone an existing agent",
		Long: `Create a copy of an existing agent with all its configuration.

The new agent will have a different ID but identical settings.

EXAMPLES:
  fibe agents duplicate my-agent`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			agent, err := c.Agents.DuplicateByIdentifier(ctx(), args[0])
			if err != nil {
				return err
			}
			if effectiveOutput() != "table" {
				outputJSON(agent)
				return nil
			}
			fmt.Printf("Duplicated agent %s -> new agent %d (%s)\n", args[0], agent.ID, agent.Name)
			return nil
		},
	}
}

func agStartChatCmd() *cobra.Command {
	var marqueeID string
	cmd := &cobra.Command{
		Use:   "start-chat <id-or-name>",
		Short: "Start an interactive chat session for an agent",
		Long: `Start the agent chat runtime on a target Marquee.

The target Marquee must be funded. The server returns
MARQUEE_NOT_FUNDED when billing is expired or missing.

REQUIRED FLAGS:
  --marquee-id   Target Marquee ID or name

EXAMPLES:
  fibe agents start-chat builder --marquee-id next
  fibe ag start-chat my-agent --marquee-id my-marquee`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if marqueeID == "" {
				return fmt.Errorf("required field 'marquee-id' not set")
			}
			session, err := newClient().Agents.StartChatByAgentIdentifier(ctx(), args[0], marqueeID)
			if err != nil {
				return err
			}
			outputJSON(session)
			return nil
		},
	}
	cmd.Flags().StringVar(&marqueeID, "marquee-id", "", "Target Marquee ID or name (required)")
	return cmd
}

func agRuntimeStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "runtime-status <id-or-name>",
		Short: "Show agent chat live status",
		Long: `Show the latest chat status for an agent and its authenticated/processing/queue state.

EXAMPLES:
  fibe agents runtime-status builder
  fibe ag runtime-status my-agent -o json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			status, err := newClient().Agents.RuntimeStatusByIdentifier(ctx(), args[0])
			if err != nil {
				return err
			}
			if effectiveOutput() != "table" {
				outputJSON(status)
				return nil
			}
			fmt.Printf("ID:                %d\n", status.ID)
			fmt.Printf("Status:            %s\n", status.Status)
			fmt.Printf("Chat URL:          %s\n", fmtStr(status.ChatURL))
			fmt.Printf("Runtime reachable: %s\n", fmtBool(status.RuntimeReachable))
			fmt.Printf("Authenticated:     %s\n", fmtBool(status.Authenticated))
			fmt.Printf("Processing:        %s\n", fmtBool(status.IsProcessing))
			fmt.Printf("Queue count:       %d\n", status.QueueCount)
			return nil
		},
	}
}

func agWatchCmd() *cobra.Command {
	var maxEvents int
	var duration time.Duration
	cmd := &cobra.Command{
		Use:   "watch",
		Short: "Watch agent resource events",
		Long: `Watch agent resource events through Fibe's live event stream.

Events are emitted as NDJSON and include the raw resource event payload.

OPTIONAL FLAGS:
  --max-events   Stop after this many events
  --duration     Stop after this duration (0 = until cancelled)

EXAMPLES:
  fibe agents watch
  fibe ag watch --max-events 5 --duration 1m`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			watchCtx := ctx()
			cancel := func() {}
			if duration > 0 {
				watchCtx, cancel = context.WithTimeout(watchCtx, duration)
			}
			defer cancel()

			events, errs := newClient().Cable.SubscribeResource(watchCtx, "Agent")
			enc := json.NewEncoder(cmd.OutOrStdout())
			count := 0
			for events != nil || errs != nil {
				select {
				case ev, ok := <-events:
					if !ok {
						events = nil
						continue
					}
					if err := enc.Encode(ev); err != nil {
						return err
					}
					count++
					if maxEvents > 0 && count >= maxEvents {
						cancel()
					}
				case err, ok := <-errs:
					if !ok {
						errs = nil
						continue
					}
					if err != nil && watchCtx.Err() == nil {
						return err
					}
				case <-watchCtx.Done():
					return nil
				}
			}
			return nil
		},
	}
	cmd.Flags().IntVar(&maxEvents, "max-events", 0, "Stop after this many events (0 = unbounded)")
	cmd.Flags().DurationVar(&duration, "duration", 0, "Stop after this duration (0 = until cancelled)")
	return cmd
}

func agLiveStateCmd() *cobra.Command {
	var conversationID string
	cmd := &cobra.Command{
		Use:   "live-state <id-or-name>",
		Short: "Show agent live state",
		Long: `Show the current live state for an agent conversation.

This includes transient processing state and streamed text when available.

OPTIONAL FLAGS:
  --conversation-id   Specific conversation/thread ID

EXAMPLES:
  fibe agents live-state builder --conversation-id conv-123
  fibe ag live-state my-agent -o json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			state, err := newClient().Agents.LiveStateByIdentifier(ctx(), args[0], &fibe.AgentDataParams{ConversationID: conversationID})
			if err != nil {
				return err
			}
			outputJSON(state)
			return nil
		},
	}
	cmd.Flags().StringVar(&conversationID, "conversation-id", "", "Specific conversation/thread ID")
	return cmd
}

func agCreateConversationCmd() *cobra.Command {
	var conversationID, title string
	cmd := &cobra.Command{
		Use:   "create-conversation <id-or-name>",
		Short: "Create or upsert an agent conversation",
		Long: `Create or upsert a deterministic conversation for an agent.

REQUIRED FLAGS:
  --conversation-id   Conversation/thread ID

OPTIONAL FLAGS:
  --title             Human-readable title

EXAMPLES:
  fibe agents create-conversation builder --conversation-id conv-123 --title "Landing page"`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(conversationID) == "" {
				return fmt.Errorf("required field 'conversation-id' not set")
			}
			result, err := newClient().Agents.CreateConversationByIdentifier(ctx(), args[0], &fibe.AgentConversationParams{
				ConversationID: conversationID,
				Title:          title,
			})
			if err != nil {
				return err
			}
			outputJSON(result)
			return nil
		},
	}
	cmd.Flags().StringVar(&conversationID, "conversation-id", "", "Conversation/thread ID (required)")
	cmd.Flags().StringVar(&title, "title", "", "Human-readable title")
	return cmd
}

func agDeleteConversationCmd() *cobra.Command {
	var conversationID string
	cmd := &cobra.Command{
		Use:   "delete-conversation <id-or-name>",
		Short: "Delete an agent conversation",
		Long: `Delete a conversation for an agent.

REQUIRED FLAGS:
  --conversation-id   Conversation/thread ID

EXAMPLES:
  fibe agents delete-conversation builder --conversation-id conv-123`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(conversationID) == "" {
				return fmt.Errorf("required field 'conversation-id' not set")
			}
			if err := newClient().Agents.DeleteConversationByIdentifier(ctx(), args[0], conversationID); err != nil {
				return err
			}
			if effectiveOutput() != "table" {
				outputJSON(map[string]any{"deleted": true, "conversation_id": conversationID})
				return nil
			}
			fmt.Printf("Conversation %s deleted for agent %s\n", conversationID, args[0])
			return nil
		},
	}
	cmd.Flags().StringVar(&conversationID, "conversation-id", "", "Conversation/thread ID (required)")
	return cmd
}

func agInterruptCmd() *cobra.Command {
	var conversationID string
	cmd := &cobra.Command{
		Use:   "interrupt <id-or-name>",
		Short: "Interrupt a running agent turn",
		Long: `Interrupt the current running turn for an agent.

OPTIONAL FLAGS:
  --conversation-id   Specific conversation/thread ID

EXAMPLES:
  fibe agents interrupt builder --conversation-id conv-123
  fibe ag interrupt my-agent`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := newClient().Agents.InterruptByIdentifier(ctx(), args[0], &fibe.AgentConversationParams{ConversationID: conversationID})
			if err != nil {
				return err
			}
			outputJSON(result)
			return nil
		},
	}
	cmd.Flags().StringVar(&conversationID, "conversation-id", "", "Specific conversation/thread ID")
	return cmd
}

func agRestartChatCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "restart-chat <id-or-name>",
		Short: "Restart an agent chat",
		Long: `Restart the current agent chat in place.

This preserves chat volumes and queues the normal start/deploy path, so the
chat uses the current configured image before coming back up.

EXAMPLES:
  fibe agents restart-chat builder
  fibe ag restart-chat my-agent -o json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			session, err := newClient().Agents.RestartChatByIdentifier(ctx(), args[0])
			if err != nil {
				return err
			}
			if effectiveOutput() != "table" {
				outputJSON(session)
				return nil
			}
			fmt.Printf("Restart queued for chat %d for agent %s (status: %s)\n", session.ID, args[0], session.Status)
			return nil
		},
	}
}

func agPurgeChatCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "purge-chat <id-or-name>",
		Short: "Queue teardown of an agent chat container and volumes",
		Long: `Queue teardown of the latest agent chat environment and persistent volumes.

WARNING: This removes persistent volumes for the agent chat.

EXAMPLES:
  fibe agents purge-chat builder
  fibe ag purge-chat my-agent -o json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			session, err := newClient().Agents.PurgeChatByIdentifier(ctx(), args[0])
			if err != nil {
				return err
			}
			if effectiveOutput() != "table" {
				outputJSON(session)
				return nil
			}
			status := session.RequestStatus
			if status == "" {
				status = session.Status
			}
			fmt.Printf("Purge queued for chat %d for agent %s (status: %s)\n", session.ID, args[0], status)
			return nil
		},
	}
}

func agSendMessageCmd() *cobra.Command {
	var text string
	var conversationID string
	var busyPolicy string
	var attachmentPaths []string
	var attachmentFilenames []string

	cmd := &cobra.Command{
		Use:     "send-message <id-or-name>",
		Aliases: []string{"chat"},
		Short:   "Send a message to an agent",
		Long: `Send a text message to an agent and receive a response.

The agent processes the message asynchronously (status: 202 Accepted).

REQUIRED FLAGS:
  --text                  Message text to send

OPTIONAL FLAGS:
  --conversation-id       Specific conversation/thread ID
  --busy-policy           Agent busy behavior, e.g. queue
  --attach                Local file path to upload before sending. Repeatable
  --attachment-filename   Already-uploaded filename to include. Repeatable

EXAMPLES:
  fibe agents send-message builder --text "Fix the failing tests"
  fibe ag send-message my-agent --text "Deploy to staging"
  fibe ag send-message my-agent --conversation-id conv-123 --text "Use this log" --attach ./log.txt
  echo '{"text": "Debug the build output"}' | fibe agents send-message my-agent -f -
  fibe agents send-message my-agent -f instructions.json` + generateSchemaDoc(&fibe.AgentChatParams{}),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			params := &fibe.AgentChatParams{}
			if err := applyFromFile(params); err != nil {
				return err
			}
			if cmd.Flags().Changed("text") {
				params.Text = text
			}
			if cmd.Flags().Changed("conversation-id") {
				params.ConversationID = conversationID
			}
			if cmd.Flags().Changed("busy-policy") {
				params.BusyPolicy = busyPolicy
			}
			if len(attachmentFilenames) > 0 {
				params.AttachmentFilenames = append(params.AttachmentFilenames, attachmentFilenames...)
			}
			if params.Text == "" {
				return fmt.Errorf("required field 'text' not set")
			}
			for _, path := range attachmentPaths {
				upload, err := c.Agents.UploadByIdentifier(ctx(), args[0], &fibe.AgentUploadParams{
					FilePath:       path,
					ConversationID: params.ConversationID,
				})
				if err != nil {
					return err
				}
				if upload.Filename != "" {
					params.AttachmentFilenames = append(params.AttachmentFilenames, upload.Filename)
				}
			}
			result, err := c.Agents.ChatByIdentifier(ctx(), args[0], params)
			if err != nil {
				return err
			}
			outputJSON(result)
			return nil
		},
	}

	cmd.Flags().StringVar(&text, "text", "", "Chat message text (required)")
	cmd.Flags().StringVar(&conversationID, "conversation-id", "", "Specific conversation/thread ID")
	cmd.Flags().StringVar(&busyPolicy, "busy-policy", "", "Agent busy behavior, e.g. queue")
	cmd.Flags().StringArrayVar(&attachmentPaths, "attach", nil, "Local file path to upload before sending (repeatable)")
	cmd.Flags().StringArrayVar(&attachmentFilenames, "attachment-filename", nil, "Already-uploaded filename to include (repeatable)")
	return cmd
}

func agUploadAttachmentCmd() *cobra.Command {
	var filePath, filename, conversationID string
	cmd := &cobra.Command{
		Use:     "upload-attachment <id-or-name>",
		Aliases: []string{"upload"},
		Short:   "Upload a chat attachment",
		Long: `Upload a file for an agent chat.

The returned filename can be passed to send-message with --attachment-filename
or later downloaded with download-attachment.

REQUIRED FLAGS:
  --file              Local file path to upload

OPTIONAL FLAGS:
  --filename          Override uploaded filename
  --conversation-id   Specific conversation/thread ID

EXAMPLES:
  fibe agents upload-attachment my-agent --file ./context.zip
  fibe ag upload my-agent --conversation-id conv-123 --file ./screenshot.png -o json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(filePath) == "" {
				return fmt.Errorf("required field 'file' not set")
			}
			result, err := newClient().Agents.UploadByIdentifier(ctx(), args[0], &fibe.AgentUploadParams{
				FilePath:       filePath,
				FileName:       filename,
				ConversationID: conversationID,
			})
			if err != nil {
				return err
			}
			if effectiveOutput() != "table" {
				outputJSON(result)
				return nil
			}
			fmt.Printf("Uploaded: %s\n", result.Filename)
			return nil
		},
	}
	cmd.Flags().StringVar(&filePath, "file", "", "Local file path to upload (required)")
	cmd.Flags().StringVar(&filename, "filename", "", "Override uploaded filename")
	cmd.Flags().StringVar(&conversationID, "conversation-id", "", "Specific conversation/thread ID")
	return cmd
}

func agDownloadAttachmentCmd() *cobra.Command {
	var to, conversationID string
	cmd := &cobra.Command{
		Use:     "download-attachment <id-or-name> <filename>",
		Aliases: []string{"download-upload"},
		Short:   "Download a chat attachment",
		Long: `Download an agent chat attachment.

REQUIRED FLAGS:
  --to                Output file path (use - for stdout)

OPTIONAL FLAGS:
  --conversation-id   Specific conversation/thread ID

EXAMPLES:
  fibe agents download-attachment my-agent runtime-file.zip --to ./runtime-file.zip
  fibe ag download-upload my-agent screenshot.png --conversation-id conv-123 --to - > screenshot.png`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(to) == "" {
				return fmt.Errorf("required field 'to' not set")
			}
			body, filename, contentType, err := newClient().Agents.DownloadAttachmentByIdentifier(ctx(), args[0], args[1], &fibe.AgentDataParams{ConversationID: conversationID})
			if err != nil {
				return err
			}
			defer body.Close()

			var dst io.Writer
			if to == "-" {
				dst = os.Stdout
			} else {
				f, err := os.Create(to)
				if err != nil {
					return fmt.Errorf("create output file: %w", err)
				}
				defer f.Close()
				dst = f
			}
			n, err := io.Copy(dst, body)
			if err != nil {
				return fmt.Errorf("write content: %w", err)
			}
			if to != "-" {
				if filename == "" {
					filename = args[1]
				}
				fmt.Fprintf(os.Stderr, "Wrote %d bytes (filename: %s, content-type: %s) to %s\n", n, filename, contentType, to)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&to, "to", "", "Output file path (required, use - for stdout)")
	cmd.Flags().StringVar(&conversationID, "conversation-id", "", "Specific conversation/thread ID")
	return cmd
}

func agAuthCmd() *cobra.Command {
	var code, token string

	cmd := &cobra.Command{
		Use:   "authenticate <id-or-name>",
		Short: "Authenticate agent with its provider",
		Long: `Authenticate an agent with GitHub or Gitea.

Provide either a code (for OAuth flow) or a token (for direct authentication).

OPTIONAL FLAGS:
  --code    OAuth authorization code
  --token   Direct access token

EXAMPLES:
  fibe agents authenticate builder --token ghp_xxxx
  fibe ag authenticate my-agent --code abc123`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			var codePtr, tokenPtr *string
			if code != "" {
				codePtr = &code
			}
			if token != "" {
				tokenPtr = &token
			}
			agent, err := c.Agents.AuthenticateByIdentifier(ctx(), args[0], codePtr, tokenPtr)
			if err != nil {
				return err
			}
			fmt.Printf("Agent %d authenticated: %s\n", agent.ID, fmtBool(agent.Authenticated))
			return nil
		},
	}

	cmd.Flags().StringVar(&code, "code", "", "OAuth code")
	cmd.Flags().StringVar(&token, "token", "", "Access token")
	return cmd
}

func agMessagesCmd() *cobra.Command {
	var conversationID string
	cmd := &cobra.Command{
		Use:   "messages <id-or-name>",
		Short: "Get agent messages",
		Long: `Retrieve the stored messages for an agent.

Messages are agent conversation history stored as JSON.

OPTIONAL FLAGS:
  --conversation-id   Specific conversation/thread ID

EXAMPLES:
  fibe agents messages my-agent
  fibe agents messages my-agent --conversation-id conv-123`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			data, err := c.Agents.GetMessagesByIdentifierWithParams(ctx(), args[0], &fibe.AgentDataParams{ConversationID: conversationID})
			if err != nil {
				return err
			}
			outputJSON(data)
			return nil
		},
	}
	cmd.Flags().StringVar(&conversationID, "conversation-id", "", "Specific conversation/thread ID")
	return cmd
}

func agActivityCmd() *cobra.Command {
	var conversationID string
	cmd := &cobra.Command{
		Use:   "activity <id-or-name>",
		Short: "Get agent activity log",
		Long: `Retrieve the activity log for an agent.

Activity logs track what the agent has done, including actions taken
and their outcomes.

OPTIONAL FLAGS:
  --conversation-id   Specific conversation/thread ID

EXAMPLES:
  fibe agents activity my-agent
  fibe agents activity my-agent --conversation-id conv-123`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			data, err := c.Agents.GetActivityByIdentifierWithParams(ctx(), args[0], &fibe.AgentDataParams{ConversationID: conversationID})
			if err != nil {
				return err
			}
			outputJSON(data)
			return nil
		},
	}
	cmd.Flags().StringVar(&conversationID, "conversation-id", "", "Specific conversation/thread ID")
	return cmd
}

func agGiteaTokenCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "gitea-token <id-or-name>",
		Short: "Get agent's Gitea access token",
		Long: `Get a Gitea access token for an agent.

Returns the token, Gitea host, and username.

EXAMPLES:
  fibe agents gitea-token my-agent`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			token, err := c.Agents.GetGiteaTokenByIdentifier(ctx(), args[0])
			if err != nil {
				return err
			}
			if effectiveOutput() != "table" {
				outputJSON(token)
				return nil
			}
			fmt.Printf("Token:     %s\n", token.Token)
			fmt.Printf("Host:      %s\n", token.GiteaHost)
			fmt.Printf("Username:  %s\n", token.Username)
			return nil
		},
	}
}

func agSetMessagesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set-messages <id-or-name>",
		Short: "Replace agent messages content",
		Long: `Replace the messages content for an agent.

Reads JSON content from STDIN or the --from-file flag.

EXAMPLES:
  echo '[{"role":"user","content":"hi"}]' | fibe agents set-messages my-agent -f -
  fibe agents set-messages my-agent -f messages.json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			if len(rawPayload) == 0 {
				return fmt.Errorf("provide content via --from-file or STDIN")
			}
			err := c.Agents.UpdateMessagesByIdentifier(ctx(), args[0], string(rawPayload))
			if err != nil {
				return err
			}
			fmt.Println("Messages updated")
			return nil
		},
	}
}

func agSetActivityCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set-activity <id-or-name>",
		Short: "Replace agent activity content",
		Long: `Replace the activity log content for an agent.

Reads JSON content from STDIN or the --from-file flag.

EXAMPLES:
  cat activity.json | fibe agents set-activity my-agent -f -
  fibe agents set-activity my-agent -f activity.json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			if len(rawPayload) == 0 {
				return fmt.Errorf("provide content via --from-file or STDIN")
			}
			err := c.Agents.UpdateActivityByIdentifier(ctx(), args[0], string(rawPayload))
			if err != nil {
				return err
			}
			fmt.Println("Activity updated")
			return nil
		},
	}
}

// =============================================================================
// Artefacts: download
// =============================================================================
