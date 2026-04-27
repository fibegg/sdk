package main

import (
	"fmt"
	"strconv"

	"github.com/fibegg/sdk/fibe"
	"github.com/spf13/cobra"
)

func playgroundsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "playgrounds",
		Aliases: []string{"pg"},
		Short:   "Manage playgrounds (running environments)",
		Long: `Manage Fibe playgrounds — running instances of your service compositions.

A playground is a live environment created from a playspec (service template).
Playgrounds can be started, stopped, restarted, and monitored.

LIFECYCLE SUMMARY:
  - pending:       Queued for deployment (Wait for start)
  - in_progress:   Building/Clone running (Monitor using 'fibe pg logs')
  - running:       Happy / Active
  - error:         Failed (Check error_message payload and logs)
  - has_changes:   Code drifted (Trigger 'fibe pg rollout')
  - completed:     Job watch finished

CORE TROUBLESHOOTING:
  - "Stuck in Pending": Valid marquee?
  - "Dirty Working Tree": Target source code repository drift (Requires commit/re-sync).

SUBCOMMANDS:
  list              List all playgrounds
  get <id>          Show playground details
  create            Create a new playground
  update <id>       Update playground settings
  delete <id>       Delete a playground
  rollout <id>      Recreate with latest config
  hard-restart <id> Hard restart all services
  extend <id>       Extend expiration time
  status <id>       Check playground status
  compose <id>      Get docker-compose configuration
  logs <id>         Get service logs
  env <id>          Get environment metadata
  debug <id>        Get debug information`,
	}

	cmd.AddCommand(
		pgListCmd(),
		pgGetCmd(),
		pgCreateCmd(),
		pgUpdateCmd(),
		pgDeleteCmd(),
		pgRolloutCmd(),
		pgHardRestartCmd(),
		pgExtendCmd(),
		pgStatusCmd(),
		pgComposeCmd(),
		pgLogsCmd(),
		pgEnvCmd(),
		pgDebugCmd(),
	)
	return cmd
}

func pgListCmd() *cobra.Command {
	var query, status, name, sort, createdAfter, createdBefore string
	var playspecID, marqueeID int64
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all playgrounds (excludes tricks)",
		Long: `List all playgrounds accessible to the authenticated user.
Tricks (job-mode workloads) are excluded — use 'fibe tricks list' instead.

FILTERS:
  -q, --query           Search across name (substring match)
  --status              Filter by exact status. Values: pending, in_progress, running, error, stopped, destroying
  --name                Filter by name (substring match)
  --playspec-id         Filter by playspec ID
  --marquee-id          Filter by marquee ID

DATE RANGE:
  --created-after       Show items created on or after this date (ISO 8601, e.g. 2026-01-15)
  --created-before      Show items created on or before this date (ISO 8601, e.g. 2026-06-30)

SORTING:
  --sort                Sort results. Format: {column}_{direction}
                        Columns: created_at, name, status
                        Direction: asc, desc
                        Default: created_at_desc
                        Example: --sort name_asc

OUTPUT:
  Columns: ID, NAME, STATUS, PLAYSPEC, EXPIRES
  Use --output json for full details.

EXAMPLES:
  fibe playgrounds list
  fibe pg list -q "my-app"
  fibe pg list --status running --sort name_asc
  fibe pg list --created-after 2026-01-01 --created-before 2026-06-01
  fibe pg list -q myapp --status running -o json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			f := false
			params := &fibe.PlaygroundListParams{JobMode: &f}
			if query != "" {
				params.Q = query
			}
			if status != "" {
				params.Status = status
			}
			if name != "" {
				params.Name = name
			}
			if playspecID > 0 {
				params.PlayspecID = playspecID
			}
			if marqueeID > 0 {
				params.MarqueeID = marqueeID
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
			pgs, err := c.Playgrounds.List(ctx(), params)
			if err != nil {
				return err
			}
			if effectiveOutput() != "table" {
				outputJSON(pgs)
				return nil
			}
			headers := []string{"ID", "NAME", "STATUS", "PLAYSPEC", "EXPIRES"}
			rows := make([][]string, len(pgs.Data))
			for i, pg := range pgs.Data {
				rows[i] = []string{
					fmtInt64(pg.ID), pg.Name, pg.Status,
					fmtStr(pg.PlayspecName), fmtTime(pg.ExpiresAt),
				}
			}
			outputTable(headers, rows)
			return nil
		},
	}
	cmd.Flags().StringVarP(&query, "query", "q", "", "Search across name")
	cmd.Flags().StringVar(&status, "status", "", "Filter by status")
	cmd.Flags().StringVar(&name, "name", "", "Filter by name (substring)")
	cmd.Flags().Int64Var(&playspecID, "playspec-id", 0, "Filter by playspec ID")
	cmd.Flags().Int64Var(&marqueeID, "marquee-id", 0, "Filter by marquee ID")
	cmd.Flags().StringVar(&createdAfter, "created-after", "", "Filter: created after date (ISO 8601)")
	cmd.Flags().StringVar(&createdBefore, "created-before", "", "Filter: created before date (ISO 8601)")
	cmd.Flags().StringVar(&sort, "sort", "", "Sort order (e.g. created_at_desc, name_asc)")
	return cmd
}

func pgGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <id>",
		Short: "Show detailed playground information",
		Long: `Get detailed information about a specific playground.

Includes all fields from list plus: compose project, internal password,
environment overrides, error messages, service status, and job results.

EXAMPLES:
  fibe playgrounds get 42
  fibe pg get 42 --output json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			id, _ := strconv.ParseInt(args[0], 10, 64)
			pg, err := c.Playgrounds.Get(ctx(), id)
			if err != nil {
				return err
			}
			if effectiveOutput() != "table" {
				outputJSON(pg)
				return nil
			}
			fmt.Printf("ID:        %d\n", pg.ID)
			fmt.Printf("Name:      %s\n", pg.Name)
			fmt.Printf("Status:    %s\n", pg.Status)
			fmt.Printf("Playspec:  %s\n", fmtStr(pg.PlayspecName))
			fmt.Printf("Expires:   %s\n", fmtTime(pg.ExpiresAt))
			fmt.Printf("Created:   %s\n", fmtTimeVal(pg.CreatedAt))
			if pg.ErrorMessage != nil {
				fmt.Printf("Error:     %s\n", *pg.ErrorMessage)
			}
			return nil
		},
	}
}

func pgCreateCmd() *cobra.Command {
	var name string
	var playspecID int64
	var marqueeID int64

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Deploy a playspec blueprint as a running playground",
		Long: `Deploy a playspec blueprint onto a marquee host as a running playground.

Requires an existing playspec-id.
For an automated one-shot deployment directly from raw Docker Compose YAML 
(without pre-creating a playspec), use the 'fibe launch' command instead.

SUBDOMAIN BOUNDARIES:
  - Every exposed service reserves a subdomain prefix mapping to the Marquee root.
  - Subdomains MUST be strictly unique per server architecture. Fibe will reject conflicts.
  - You can manually map specific domain names over the automatic hashing by defining 'services[X].subdomain' in your payload payload.json

REQUIRED FLAGS:
  --name          Playground name
  --playspec-id   ID of the playspec to use

OPTIONAL FLAGS:
  --marquee-id    ID of the target marquee (server)

EXAMPLES:
  fibe playgrounds create --name my-app --playspec-id 5
  fibe pg create --name staging --playspec-id 5 --marquee-id 3
  echo '{"name": "test", "playspec_id": 5}' | fibe pg create
  fibe pg create -f payload.json` + generateSchemaDoc(&fibe.PlaygroundCreateParams{}),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			params := &fibe.PlaygroundCreateParams{}
			if err := applyFromFile(params); err != nil {
				return err
			}

			if cmd.Flags().Changed("name") {
				params.Name = name
			}
			if cmd.Flags().Changed("playspec-id") {
				params.PlayspecID = playspecID
			}
			if cmd.Flags().Changed("marquee-id") {
				params.MarqueeID = &marqueeID
			}

			if params.Name == "" {
				return fmt.Errorf("required field 'name' not set")
			}
			if params.PlayspecID == 0 {
				return fmt.Errorf("required field 'playspec-id' not set")
			}

			pg, err := c.Playgrounds.Create(ctx(), params)
			if err != nil {
				return err
			}
			if effectiveOutput() != "table" {
				outputJSON(pg)
				return nil
			}
			fmt.Printf("Created playground %d (%s) — status: %s\n", pg.ID, pg.Name, pg.Status)
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Playground name (required)")
	cmd.Flags().Int64Var(&playspecID, "playspec-id", 0, "Playspec ID (required)")
	cmd.Flags().Int64Var(&marqueeID, "marquee-id", 0, "Marquee ID (optional)")
	return cmd
}

func pgUpdateCmd() *cobra.Command {
	var name string
	var playspecID, marqueeID int64

	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update playground settings",
		Long: `Update an existing playground's configuration.

OPTIONAL FLAGS:
  --name           New playground name
  --playspec-id    Switch to a different playspec
  --marquee-id     Move to a different marquee

For complex updates (services, build_overrides_yaml), use --from-file:
  fibe playgrounds update 42 -f update.json

EXAMPLES:
  fibe playgrounds update 42 --name new-name
  fibe pg update 42 --marquee-id 7
  fibe pg update 42 --playspec-id 12 --marquee-id 7` + generateSchemaDoc(&fibe.PlaygroundUpdateParams{}),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			id, _ := strconv.ParseInt(args[0], 10, 64)
			params := &fibe.PlaygroundUpdateParams{}
			if err := applyFromFile(params); err != nil {
				return err
			}
			if cmd.Flags().Changed("name") {
				params.Name = &name
			}
			if cmd.Flags().Changed("playspec-id") {
				params.PlayspecID = &playspecID
			}
			if cmd.Flags().Changed("marquee-id") {
				params.MarqueeID = &marqueeID
			}
			pg, err := c.Playgrounds.Update(ctx(), id, params)
			if err != nil {
				return err
			}
			if effectiveOutput() != "table" {
				outputJSON(pg)
				return nil
			}
			fmt.Printf("Updated playground %d (%s)\n", pg.ID, pg.Name)
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "New playground name")
	cmd.Flags().Int64Var(&playspecID, "playspec-id", 0, "Switch to a different playspec")
	cmd.Flags().Int64Var(&marqueeID, "marquee-id", 0, "Move to a different marquee")
	return cmd
}

func pgDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a playground",
		Long: `Delete a playground and tear down all its services.

This is an asynchronous operation — the playground will be marked for deletion
and its containers will be stopped and removed.

WARNING: This action is irreversible. All data in non-persistent volumes will be lost.

EXAMPLES:
  fibe playgrounds delete 42`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			id, _ := strconv.ParseInt(args[0], 10, 64)
			if err := c.Playgrounds.Delete(ctx(), id); err != nil {
				return err
			}
			fmt.Printf("Playground %d deletion initiated\n", id)
			return nil
		},
	}
}

func pgRolloutCmd() *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:   "rollout <id>",
		Short: "Recreate playground with latest configuration",
		Long: `Trigger a rollout to recreate the playground with the latest playspec configuration.

This tears down the existing containers and rebuilds them from scratch.
Equivalent to a fresh deployment with current settings.

Use this when you've updated the playspec and want the changes applied.

EXAMPLES:
  fibe playgrounds rollout 42`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			id, _ := strconv.ParseInt(args[0], 10, 64)
			params := &fibe.PlaygroundActionParams{ActionType: fibe.PlaygroundActionRollout}
			if cmd.Flags().Changed("force") {
				params.Force = &force
			}
			pg, err := c.Playgrounds.Action(ctx(), id, params)
			if err != nil {
				return err
			}
			fmt.Printf("Rollout initiated for playground %d — status: %s\n", pg.ID, pg.Status)
			return nil
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "Bypass state protections when the server permits it")
	return cmd
}

func pgHardRestartCmd() *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:   "hard-restart <id>",
		Short: "Hard restart all playground services",
		Long: `Perform a hard restart of all services in the playground.

Unlike rollout, this does not rebuild containers — it stops and restarts them.
Use this when services are unresponsive but the configuration hasn't changed.

EXAMPLES:
  fibe playgrounds hard-restart 42`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			id, _ := strconv.ParseInt(args[0], 10, 64)
			params := &fibe.PlaygroundActionParams{ActionType: fibe.PlaygroundActionHardRestart}
			if cmd.Flags().Changed("force") {
				params.Force = &force
			}
			pg, err := c.Playgrounds.Action(ctx(), id, params)
			if err != nil {
				return err
			}
			fmt.Printf("Hard restart initiated for playground %d — status: %s\n", pg.ID, pg.Status)
			return nil
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "Bypass state protections when the server permits it")
	return cmd
}

func pgExtendCmd() *cobra.Command {
	var hours int

	cmd := &cobra.Command{
		Use:   "extend <id>",
		Short: "Extend playground expiration time",
		Long: `Extend the expiration time of a playground.

Playgrounds expire after a set duration. Use this to keep them alive longer.

OPTIONAL FLAGS:
  --hours   Number of hours to extend (default: platform setting)

EXAMPLES:
  fibe playgrounds extend 42
  fibe pg extend 42 --hours 24`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			id, _ := strconv.ParseInt(args[0], 10, 64)
			var h *int
			if hours > 0 {
				h = &hours
			}
			result, err := c.Playgrounds.ExtendExpiration(ctx(), id, h)
			if err != nil {
				return err
			}
			fmt.Printf("Playground %d extended — expires: %s\n", result.ID, fmtTimeVal(result.ExpiresAt))
			return nil
		},
	}

	cmd.Flags().IntVar(&hours, "hours", 0, "Hours to extend")
	return cmd
}

func pgStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status <id>",
		Short: "Check playground status",
		Long: `Get the current status of a playground, including job result if available.

Returns the playground status and, for completed jobs, the job result with
per-service outcomes.

EXAMPLES:
  fibe playgrounds status 42`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			id, _ := strconv.ParseInt(args[0], 10, 64)
			status, err := c.Playgrounds.Status(ctx(), id)
			if err != nil {
				return err
			}
			if effectiveOutput() != "table" {
				outputJSON(status)
				return nil
			}
			fmt.Printf("Playground %d: %s\n", status.ID, status.Status)
			return nil
		},
	}
}

func pgComposeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "compose <id>",
		Short: "Get playground docker-compose configuration",
		Long: `Retrieve the generated docker-compose YAML for a playground.

Returns the compose YAML and the compose project name.
Useful for debugging or replicating the environment locally.

EXAMPLES:
  fibe playgrounds compose 42
  fibe pg compose 42 --output json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			id, _ := strconv.ParseInt(args[0], 10, 64)
			compose, err := c.Playgrounds.Compose(ctx(), id)
			if err != nil {
				return err
			}
			if effectiveOutput() != "table" {
				outputJSON(compose)
				return nil
			}
			fmt.Printf("Project: %s\n\n%s", compose.ComposeProject, compose.ComposeYAML)
			return nil
		},
	}
}

func pgLogsCmd() *cobra.Command {
	var service string
	var tail int

	cmd := &cobra.Command{
		Use:   "logs <id>",
		Short: "Get service logs from a playground",
		Long: `Retrieve logs from a specific service in a playground.

Returns the most recent log lines from the specified service container.
By default returns the last 50 lines.

REQUIRED FLAGS:
  --service   Name of the service to get logs from

OPTIONAL FLAGS:
  --tail      Number of lines to return (default: 50)

EXAMPLES:
  fibe playgrounds logs 42 --service web
  fibe pg logs 42 --service web --tail 100`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			id, _ := strconv.ParseInt(args[0], 10, 64)
			var t *int
			if tail > 0 {
				t = &tail
			}
			logs, err := c.Playgrounds.Logs(ctx(), id, service, t)
			if err != nil {
				return err
			}
			if effectiveOutput() != "table" {
				outputJSON(logs)
				return nil
			}
			for _, line := range logs.Lines {
				fmt.Println(line)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&service, "service", "", "Service name (required)")
	cmd.Flags().IntVar(&tail, "tail", 0, "Number of lines")
	cmd.MarkFlagRequired("service")
	return cmd
}

func pgEnvCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "env <id>",
		Short: "Get playground environment metadata",
		Long: `Get the merged environment variables and metadata for a playground.

Returns the final merged environment, per-source metadata showing where
each variable comes from, and system-reserved keys.

EXAMPLES:
  fibe playgrounds env 42`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			id, _ := strconv.ParseInt(args[0], 10, 64)
			env, err := c.Playgrounds.EnvMetadata(ctx(), id)
			if err != nil {
				return err
			}
			outputJSON(env)
			return nil
		},
	}
}

func pgDebugCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "debug <id>",
		Short: "Get comprehensive debug information",
		Long: `Get comprehensive debug information for a playground.

Returns detailed internal state useful for troubleshooting issues.
Output is always JSON due to the complex nested structure.

EXAMPLES:
  fibe playgrounds debug 42`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			id, _ := strconv.ParseInt(args[0], 10, 64)
			debug, err := c.Playgrounds.Debug(ctx(), id)
			if err != nil {
				return err
			}
			outputJSON(debug)
			return nil
		},
	}
}
