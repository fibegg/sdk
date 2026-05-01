package main

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

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
  gemini, claude-code, openai-codex, opencode, cursor

SUBCOMMANDS:
  list                  List all agents
  get <id-or-name>      Show agent details
  create                Create a new agent
  update <id-or-name>   Update agent settings
  delete <id-or-name>   Delete an agent
  duplicate <id-or-name> Clone an agent
  start-chat <id-or-name> Start an interactive chat session on a Marquee
  runtime-status <id-or-name> Show agent chat runtime status
  purge-chat <id-or-name> Tear down agent chat container and volumes
  chat <id-or-name>     Send a chat message
  authenticate <id-or-name> Authenticate agent with provider
  add-mounted-file <id-or-name> Attach a mounted file or Artefact snapshot
  update-mounted-file <id-or-name> Update mounted file metadata
  remove-mounted-file <id-or-name> Remove a mounted file
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
		agRuntimeStatusCmd(),
		agPurgeChatCmd(),
		agSendMessageCmd(),
		agAuthCmd(),
		agAddMountedFileCmd(),
		agUpdateMountedFileCmd(),
		agRemoveMountedFileCmd(),
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
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all agents",
		Long: `List all agents accessible to the authenticated user.

FILTERS:
  -q, --query           Search across name, description (substring match)
  --provider            Filter by provider. Values: gemini, claude-code, openai-codex, opencode, cursor
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
			rows := make([][]string, len(agents.Data))
			for i, a := range agents.Data {
				rows[i] = []string{
					fmtInt64(a.ID), a.Name, a.Provider, a.Status, fmtBool(a.Authenticated),
				}
			}
			outputTable(headers, rows)
			return nil
		},
	}
	cmd.Flags().StringVarP(&query, "query", "q", "", "Search across name, description")
	cmd.Flags().StringVar(&provider, "provider", "", "Filter by provider (gemini, claude-code, openai-codex, opencode, cursor)")
	cmd.Flags().StringVar(&status, "status", "", "Filter by status")
	cmd.Flags().StringVar(&name, "name", "", "Filter by name (substring)")
	cmd.Flags().StringVar(&createdAfter, "created-after", "", "Filter: created after date (ISO 8601)")
	cmd.Flags().StringVar(&createdBefore, "created-before", "", "Filter: created before date (ISO 8601)")
	cmd.Flags().StringVar(&sort, "sort", "", "Sort order (e.g. created_at_desc)")
	return cmd
}

func agGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <id-or-name>",
		Short: "Show agent details",
		Long: `Get detailed information about a specific agent.

EXAMPLES:
  fibe agents get 5
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

func agCreateCmd() *cobra.Command {
	var name, provider, modelOptions, memoryLimit, cpuLimit string
	var prompt, mcpJSON, postInitScript, customEnv, cliVersion, providerArgs string
	var skillToggleFlags []string
	var mountFiles, mountArtefacts []string
	var playgroundCrumbsID string
	var syncEnabled, syncSkillsEnabled, syscheckEnabled, providerAPIKeyMode, buildInPublic bool

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new agent",
		Long: `Create a new AI agent with the specified provider.

REQUIRED FLAGS:
  --name       Agent name
  --provider   Provider type: gemini, claude-code, openai-codex, opencode, cursor

OPTIONAL FLAGS:
  --sync           Enable sync
  --sync-skills    Enable Fibe system skill sync
  --syscheck       Enable system checks
  --provider-api-key-mode
                  Use provider API key authentication instead of subscription/OAuth mode
  --model-options Pin provider model option for this agent
  --memory-limit   Memory limit, for example 2G
  --cpu-limit      CPU limit, for example 1.5
  --prompt         Agent-specific system prompt override
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
  --mount-artefact Artefact snapshot mount as 123:%{workspace}/docs/file.md (repeatable)

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
			if cmd.Flags().Changed("sync-skills") {
				params.SyncSkillsEnabled = &syncSkillsEnabled
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
	cmd.Flags().StringVar(&provider, "provider", "gemini", "Provider: gemini, claude-code, openai-codex, opencode, cursor")
	cmd.Flags().BoolVar(&syncEnabled, "sync", false, "Enable sync")
	cmd.Flags().BoolVar(&syncSkillsEnabled, "sync-skills", false, "Enable Fibe system skill sync")
	cmd.Flags().BoolVar(&syscheckEnabled, "syscheck", false, "Enable system checks")
	cmd.Flags().BoolVar(&buildInPublic, "build-in-public", false, "Show this agent on the public profile")
	cmd.Flags().StringVar(&playgroundCrumbsID, "playground-crumbs-id", "", "Playground ID or name for public timeline crumbs")
	cmd.Flags().BoolVar(&providerAPIKeyMode, "provider-api-key-mode", false, "Use provider API key auth mode")
	cmd.Flags().StringVar(&modelOptions, "model-options", "", "Provider model option")
	cmd.Flags().StringVar(&memoryLimit, "memory-limit", "", "Memory limit, for example 2G")
	cmd.Flags().StringVar(&cpuLimit, "cpu-limit", "", "CPU limit, for example 1.5")
	cmd.Flags().StringVar(&prompt, "prompt", "", "Agent-specific system prompt override")
	cmd.Flags().StringVar(&mcpJSON, "mcp-json", "", "MCP configuration JSON")
	cmd.Flags().StringVar(&postInitScript, "post-init-script", "", "Shell script run after agent initialization")
	cmd.Flags().StringVar(&customEnv, "custom-env", "", "KEY=VALUE lines injected into the agent runtime")
	cmd.Flags().StringVar(&cliVersion, "cli-version", "", "Fibe CLI version pin")
	cmd.Flags().StringVar(&providerArgs, "provider-args", "", "Provider CLI flags, for example \"--bare --max-tokens 4096\"")
	cmd.Flags().StringArrayVar(&skillToggleFlags, "skill-toggle", nil, "Skill toggle as filename=true|false (repeatable)")
	cmd.Flags().StringArrayVar(&mountFiles, "mount-file", nil, "Local file mount as ./path:%{agent_data}/target.ext (repeatable)")
	cmd.Flags().StringArrayVar(&mountArtefacts, "mount-artefact", nil, "Artefact snapshot mount as 123:%{workspace}/docs/file.md (repeatable)")
	return cmd
}

func agUpdateCmd() *cobra.Command {
	var name, modelOptions, memoryLimit, cpuLimit string
	var prompt, mcpJSON, postInitScript, customEnv, cliVersion, providerArgs string
	var skillToggleFlags []string
	var syncEnabled, syncSkillsEnabled, syscheckEnabled, providerAPIKeyMode bool
	var buildInPublicPlaygroundID string

	cmd := &cobra.Command{
		Use:   "update <id-or-name>",
		Short: "Update agent settings",
		Long: `Update an existing agent's configuration.

OPTIONAL FLAGS:
  --name                          New agent name
  --sync                          Enable/disable sync
  --sync-skills                   Enable/disable Fibe system skill sync
  --syscheck                      Enable/disable system checks
  --provider-api-key-mode         Enable/disable provider API key auth mode
  --model-options                 Provider model option
  --memory-limit                  Memory limit, for example 2G
  --cpu-limit                     CPU limit, for example 1.5
  --prompt                        Agent-specific system prompt override
  --mcp-json                      MCP configuration JSON for the runtime
  --post-init-script              Shell script run after agent initialization
  --custom-env                    KEY=VALUE lines injected into the agent runtime
  --cli-version                   Fibe CLI version pin for this agent
  --provider-args                 Provider CLI flags, for example "--bare --max-tokens 4096"
  --skill-toggle                  Skill toggle as filename=true|false (repeatable)
  --build-in-public-playground-id Playground ID or name for public builds

EXAMPLES:
  fibe agents update 5 --name new-name
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
			if cmd.Flags().Changed("sync-skills") {
				params.SyncSkillsEnabled = &syncSkillsEnabled
			}
			if cmd.Flags().Changed("syscheck") {
				params.SyscheckEnabled = &syscheckEnabled
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
			if cmd.Flags().Changed("build-in-public-playground-id") {
				params.BuildInPublicPlaygroundIdentifier = buildInPublicPlaygroundID
			}
			if cmd.Flags().Changed("prompt") {
				params.Prompt = &prompt
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
	cmd.Flags().BoolVar(&syncSkillsEnabled, "sync-skills", false, "Enable Fibe system skill sync")
	cmd.Flags().BoolVar(&syscheckEnabled, "syscheck", false, "Enable system checks")
	cmd.Flags().BoolVar(&providerAPIKeyMode, "provider-api-key-mode", false, "Use provider API key auth mode")
	cmd.Flags().StringVar(&modelOptions, "model-options", "", "Provider model option")
	cmd.Flags().StringVar(&memoryLimit, "memory-limit", "", "Memory limit, for example 2G")
	cmd.Flags().StringVar(&cpuLimit, "cpu-limit", "", "CPU limit, for example 1.5")
	cmd.Flags().StringVar(&prompt, "prompt", "", "Agent-specific system prompt override")
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
		id, err := strconv.ParseInt(source, 10, 64)
		if err != nil || id <= 0 {
			return nil, fmt.Errorf("--mount-artefact source must be a positive artefact id: %s", source)
		}
		mounts = append(mounts, fibe.AgentMountSpec{
			SourceType: "artefact",
			ArtefactID: &id,
			MountPath:  target,
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

REQUIRED FLAGS:
  --marquee-id   Target Marquee ID or name

EXAMPLES:
  fibe agents start-chat 5 --marquee-id 2
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
		Short: "Show agent chat runtime status",
		Long: `Show the latest chat status for an agent and, when the runtime is running,
query its authenticated/processing/queue state.

EXAMPLES:
  fibe agents runtime-status 5
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

func agPurgeChatCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "purge-chat <id-or-name>",
		Short: "Tear down an agent chat container and volumes",
		Long: `Synchronously purge the latest agent chat runtime container and persistent volumes.

WARNING: This removes runtime volumes for the agent chat.

EXAMPLES:
  fibe agents purge-chat 5
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
			fmt.Printf("Purged chat %d for agent %s (status: %s)\n", session.ID, args[0], session.Status)
			return nil
		},
	}
}

func agSendMessageCmd() *cobra.Command {
	var text string

	cmd := &cobra.Command{
		Use:     "send-message <id-or-name>",
		Aliases: []string{"chat"},
		Short:   "Send a message to an agent",
		Long: `Send a text message to an agent and receive a response.

The agent processes the message asynchronously (status: 202 Accepted).

REQUIRED FLAGS:
  --text                  Message text to send

EXAMPLES:
  fibe agents send-message 5 --text "Fix the failing tests"
  fibe ag send-message my-agent --text "Deploy to staging"
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
			if params.Text == "" {
				return fmt.Errorf("required field 'text' not set")
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
  fibe agents authenticate 5 --token ghp_xxxx
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
	return &cobra.Command{
		Use:   "messages <id-or-name>",
		Short: "Get agent messages",
		Long: `Retrieve the stored messages for an agent.

Messages are agent conversation history stored as JSON.

EXAMPLES:
  fibe agents messages my-agent`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			data, err := c.Agents.GetMessagesByIdentifier(ctx(), args[0])
			if err != nil {
				return err
			}
			outputJSON(data)
			return nil
		},
	}
}

func agActivityCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "activity <id-or-name>",
		Short: "Get agent activity log",
		Long: `Retrieve the activity log for an agent.

Activity logs track what the agent has done, including actions taken
and their outcomes.

EXAMPLES:
  fibe agents activity my-agent`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			data, err := c.Agents.GetActivityByIdentifier(ctx(), args[0])
			if err != nil {
				return err
			}
			outputJSON(data)
			return nil
		},
	}
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
