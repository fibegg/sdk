package main

import (
	"fmt"
	"strings"
	"time"

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
Maintenance mode is an overlay: a playground can keep its runtime status while
exposed service URLs route to a 503 maintenance page.

LIFECYCLE SUMMARY:
  - pending:       Queued for deployment (Wait for start)
  - in_progress:   Building/Clone running (Monitor using 'fibe pg logs')
  - running:       Happy / Active
  - error:         Failed (Check error_message payload and logs)
  - has_changes:   Code drifted (Trigger 'fibe pg rollout')
  - completed:     Job watch finished
  - stopping:      Stop requested and waiting for containers to stop
  - stopped:       Containers stopped; can be started again
  - destroying:    Delete requested and cleanup is in progress

CORE TROUBLESHOOTING:
  - "Stuck in Pending": Valid marquee?
  - "Dirty Working Tree": Target source code repository drift (Requires commit/re-sync).

SUBCOMMANDS:
  list              List all playgrounds
  get <id-or-name> Show playground details
  create            Create a new playground
  update <id-or-name>       Update playground settings
  delete <id-or-name>       Delete a playground
  rollout <id-or-name>      Recreate with latest config
  hard-restart <id-or-name> Hard restart all services
  stop <id-or-name>         Stop playground containers
  start <id-or-name>        Start a stopped playground
  maintenance enable <id-or-name>   Enable maintenance routing
  maintenance disable <id-or-name>  Disable maintenance routing
  extend <id-or-name>       Extend expiration time
  status <id-or-name>       Check playground status
  compose <id-or-name>      Get docker-compose configuration
  logs <id-or-name>         Get service logs
  env <id-or-name>          Get environment metadata
  debug <id-or-name>        Get debug information`,
	}

	cmd.AddCommand(
		pgListCmd(),
		pgGetCmd(),
		pgCreateCmd(),
		pgUpdateCmd(),
		pgDeleteCmd(),
		pgRolloutCmd(),
		pgHardRestartCmd(),
		pgStopCmd(),
		pgStartCmd(),
		pgMaintenanceCmd(),
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
	var playspecID, marqueeID string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all playgrounds (excludes tricks)",
		Long: `List all playgrounds accessible to the authenticated user.
Tricks (job-mode workloads) are excluded — use 'fibe tricks list' instead.

FILTERS:
  -q, --query           Search across name (substring match)
  --status              Filter by exact status. Values: pending, in_progress, running, error, has_changes, completed, stopping, stopped, destroying
  --name                Filter by name (substring match)
  --playspec-id         Filter by playspec ID or name
  --marquee-id          Filter by marquee ID or name

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
  Columns: ID, NAME, STATUS, MAINT, PLAYSPEC, EXPIRES
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
			if playspecID != "" {
				params.PlayspecIdentifier = playspecID
			}
			if marqueeID != "" {
				params.MarqueeIdentifier = marqueeID
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
			headers := []string{"ID", "NAME", "STATUS", "MAINT", "PLAYSPEC", "EXPIRES"}
			rows := make([][]string, len(pgs.Data))
			for i, pg := range pgs.Data {
				rows[i] = []string{
					fmtInt64(pg.ID), pg.Name, pg.Status, fmtMaintenance(pg.MaintenanceEnabled),
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
	cmd.Flags().StringVar(&playspecID, "playspec-id", "", "Filter by playspec ID or name")
	cmd.Flags().StringVar(&marqueeID, "marquee-id", "", "Filter by marquee ID or name")
	cmd.Flags().StringVar(&createdAfter, "created-after", "", "Filter: created after date (ISO 8601)")
	cmd.Flags().StringVar(&createdBefore, "created-before", "", "Filter: created before date (ISO 8601)")
	cmd.Flags().StringVar(&sort, "sort", "", "Sort order (e.g. created_at_desc, name_asc)")
	return cmd
}

func pgGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <id-or-name>",
		Short: "Show detailed playground information",
		Long: `Get detailed information about a specific playground.

Includes all fields from list plus: compose project, internal password,
environment overrides, error messages, service status, and job results.

EXAMPLES:
  fibe playgrounds get 42
  fibe playgrounds get my-playground
  fibe pg get 42 --output json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			pg, err := c.Playgrounds.GetByIdentifier(ctx(), args[0])
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
			fmt.Printf("Maintenance: %s\n", fmtMaintenance(pg.MaintenanceEnabled))
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
	var playspecID string
	var marqueeID string

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
  --playspec-id   ID or name of the playspec to use

OPTIONAL FLAGS:
  --marquee-id    ID or name of the target marquee (server)

EXAMPLES:
  fibe playgrounds create --name my-app --playspec-id starter
  fibe pg create --name staging --playspec-id starter --marquee-id next
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
				params.PlayspecIdentifier = playspecID
			}
			if cmd.Flags().Changed("marquee-id") {
				params.MarqueeIdentifier = marqueeID
			}

			if params.Name == "" {
				return fmt.Errorf("required field 'name' not set")
			}
			if params.PlayspecID == 0 && params.PlayspecIdentifier == "" {
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
	cmd.Flags().StringVar(&playspecID, "playspec-id", "", "Playspec ID or name (required)")
	cmd.Flags().StringVar(&marqueeID, "marquee-id", "", "Marquee ID or name (optional)")
	return cmd
}

func pgUpdateCmd() *cobra.Command {
	var name string
	var playspecID, marqueeID string

	cmd := &cobra.Command{
		Use:   "update <id-or-name>",
		Short: "Update playground settings",
		Long: `Update an existing playground's configuration.

OPTIONAL FLAGS:
  --name           New playground name
  --playspec-id    Switch to a different playspec by ID or name
  --marquee-id     Move to a different marquee by ID or name

For complex updates (services, build_overrides_yaml), use --from-file:
  fibe playgrounds update 42 -f update.json

EXAMPLES:
  fibe playgrounds update 42 --name new-name
  fibe pg update 42 --marquee-id 7
  fibe pg update 42 --playspec-id 12 --marquee-id 7` + generateSchemaDoc(&fibe.PlaygroundUpdateParams{}),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			params := &fibe.PlaygroundUpdateParams{}
			if err := applyFromFile(params); err != nil {
				return err
			}
			if cmd.Flags().Changed("name") {
				params.Name = &name
			}
			if cmd.Flags().Changed("playspec-id") {
				params.PlayspecIdentifier = playspecID
			}
			if cmd.Flags().Changed("marquee-id") {
				params.MarqueeIdentifier = marqueeID
			}
			pg, err := c.Playgrounds.UpdateByIdentifier(ctx(), args[0], params)
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
	cmd.Flags().StringVar(&playspecID, "playspec-id", "", "Switch to a different playspec by ID or name")
	cmd.Flags().StringVar(&marqueeID, "marquee-id", "", "Move to a different marquee by ID or name")
	return cmd
}

func pgDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <id-or-name>",
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
			if err := c.Playgrounds.DeleteByIdentifier(ctx(), args[0]); err != nil {
				return err
			}
			fmt.Printf("Playground %s deletion initiated\n", args[0])
			return nil
		},
	}
}

func pgRolloutCmd() *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:   "rollout <id-or-name>",
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
			params := &fibe.PlaygroundActionParams{ActionType: fibe.PlaygroundActionRollout}
			if cmd.Flags().Changed("force") {
				params.Force = &force
			}
			pg, err := c.Playgrounds.ActionByIdentifier(ctx(), args[0], params)
			if err != nil {
				return err
			}
			fmt.Printf("Rollout initiated for playground %d — status: %s\n", pg.ID, pg.Status)
			return nil
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "Bypass eligible state protections when the server permits it")
	return cmd
}

func pgHardRestartCmd() *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:   "hard-restart <id-or-name>",
		Short: "Hard restart all playground services",
		Long: `Perform a hard restart of all services in the playground.

Unlike rollout, this does not rebuild containers — it stops and restarts them.
Use this when services are unresponsive but the configuration hasn't changed.

EXAMPLES:
  fibe playgrounds hard-restart 42`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			params := &fibe.PlaygroundActionParams{ActionType: fibe.PlaygroundActionHardRestart}
			if cmd.Flags().Changed("force") {
				params.Force = &force
			}
			pg, err := c.Playgrounds.ActionByIdentifier(ctx(), args[0], params)
			if err != nil {
				return err
			}
			fmt.Printf("Hard restart initiated for playground %d — status: %s\n", pg.ID, pg.Status)
			return nil
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "Bypass eligible state protections when the server permits it")
	return cmd
}

func pgStopCmd() *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:   "stop <id-or-name>",
		Short: "Stop playground containers",
		Long: `Stop a running playground and preserve its record and persistent volumes.

The server queues the normal playground stop path and moves the playground
through stopping to stopped when container teardown completes.

EXAMPLES:
  fibe playgrounds stop 42`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			params := &fibe.PlaygroundActionParams{ActionType: fibe.PlaygroundActionStop}
			if cmd.Flags().Changed("force") {
				params.Force = &force
			}
			pg, err := c.Playgrounds.ActionByIdentifier(ctx(), args[0], params)
			if err != nil {
				return err
			}
			fmt.Printf("Stop initiated for playground %d — status: %s\n", pg.ID, pg.Status)
			return nil
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "Bypass eligible state protections when the server permits it")
	return cmd
}

func pgStartCmd() *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:   "start <id-or-name>",
		Short: "Start a stopped playground",
		Long: `Start a stopped playground using its current playspec and service settings.

The server queues the normal deployment path and keeps persistent volumes unless
the playspec itself is configured otherwise.

EXAMPLES:
  fibe playgrounds start 42`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			params := &fibe.PlaygroundActionParams{ActionType: fibe.PlaygroundActionStart}
			if cmd.Flags().Changed("force") {
				params.Force = &force
			}
			pg, err := c.Playgrounds.ActionByIdentifier(ctx(), args[0], params)
			if err != nil {
				return err
			}
			fmt.Printf("Start initiated for playground %d — status: %s\n", pg.ID, pg.Status)
			return nil
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "Bypass eligible state protections when the server permits it")
	return cmd
}

func pgMaintenanceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "maintenance",
		Short: "Manage playground maintenance routing",
		Long: `Manage playground maintenance routing.

Maintenance mode does not start, stop, retry, or redeploy the playground.
It keeps the playground status unchanged while exposed service URLs route to
a static 503 page that says "maintenance is ongoing".`,
	}
	cmd.AddCommand(pgMaintenanceEnableCmd(), pgMaintenanceDisableCmd())
	return cmd
}

func pgMaintenanceEnableCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "enable <id-or-name>",
		Short: "Enable playground maintenance routing",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			pg, err := c.Playgrounds.ActionByIdentifier(ctx(), args[0], &fibe.PlaygroundActionParams{ActionType: fibe.PlaygroundActionEnableMaintenance})
			if err != nil {
				return err
			}
			fmt.Printf("Maintenance enabled for playground %d — status: %s\n", pg.ID, pg.Status)
			return nil
		},
	}
}

func pgMaintenanceDisableCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "disable <id-or-name>",
		Short: "Disable playground maintenance routing",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			pg, err := c.Playgrounds.ActionByIdentifier(ctx(), args[0], &fibe.PlaygroundActionParams{ActionType: fibe.PlaygroundActionDisableMaintenance})
			if err != nil {
				return err
			}
			fmt.Printf("Maintenance disabled for playground %d — status: %s\n", pg.ID, pg.Status)
			return nil
		},
	}
}

func pgExtendCmd() *cobra.Command {
	var hours int

	cmd := &cobra.Command{
		Use:   "extend <id-or-name>",
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
			var h *int
			if hours > 0 {
				h = &hours
			}
			result, err := c.Playgrounds.ExtendExpirationByIdentifier(ctx(), args[0], h)
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
		Use:   "status <id-or-name>",
		Short: "Check playground status",
		Long: `Get the current status of a playground, including job result if available.

Returns the playground status and, for completed jobs, the job result with
per-service outcomes.

EXAMPLES:
  fibe playgrounds status 42`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			status, err := c.Playgrounds.StatusByIdentifier(ctx(), args[0])
			if err != nil {
				return err
			}
			if effectiveOutput() != "table" {
				outputJSON(status)
				return nil
			}
			fmt.Printf("Playground %d: %s (maintenance: %s)\n", status.ID, status.Status, fmtMaintenance(status.MaintenanceEnabled))
			if reason := strings.TrimSpace(statusReasonText(status)); reason != "" {
				fmt.Printf("Reason: %s\n", reason)
			}
			for _, build := range status.BuildStatuses {
				fmt.Printf("Commit %s: %s\n", build.ServiceName, fmtPlaygroundBuildStatus(build))
			}
			return nil
		},
	}
}

func statusReasonText(status *fibe.PlaygroundStatus) string {
	if status == nil {
		return ""
	}
	if len(status.StateReasons) > 0 {
		return strings.Join(status.StateReasons, "; ")
	}
	if status.StateReason != nil {
		return *status.StateReason
	}
	return ""
}

func fmtPlaygroundBuildStatus(build fibe.PlaygroundBuildStatus) string {
	parts := []string{}
	if build.Branch != "" {
		parts = append(parts, build.Branch)
	}
	if build.Active != nil {
		parts = append(parts, fmt.Sprintf("active %s@%s", build.Active.Status, displayBuildSHA(build.Active)))
	}
	if build.Running != nil {
		parts = append(parts, fmt.Sprintf("running %s@%s", build.Running.Status, displayBuildSHA(build.Running)))
	}
	if build.Latest != nil && !sameBuildSnapshot(build.Latest, build.Active) && !sameBuildSnapshot(build.Latest, build.Running) {
		parts = append(parts, fmt.Sprintf("latest %s@%s", build.Latest.Status, displayBuildSHA(build.Latest)))
	}
	if len(parts) == 0 {
		return "no build record"
	}
	return strings.Join(parts, " | ")
}

func sameBuildSnapshot(a, b *fibe.PlaygroundBuildRecordSnapshot) bool {
	return a != nil && b != nil && a.ID == b.ID
}

func displayBuildSHA(build *fibe.PlaygroundBuildRecordSnapshot) string {
	if build == nil {
		return ""
	}
	if build.ShortCommitSHA != "" {
		return build.ShortCommitSHA
	}
	if len(build.CommitSHA) > 7 {
		return build.CommitSHA[:7]
	}
	return build.CommitSHA
}

func pgComposeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "compose <id-or-name>",
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
			compose, err := c.Playgrounds.ComposeByIdentifier(ctx(), args[0])
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
	var follow bool
	var maxLines int
	var duration time.Duration

	cmd := &cobra.Command{
		Use:   "logs <id-or-name>",
		Short: "Get logs from a playground",
		Long: `Retrieve logs from a playground.

Returns the most recent log lines from all services by default. Use --service
to focus on one service container.
By default returns the last 50 lines.

OPTIONAL FLAGS:
  --service   Optional service name to filter logs
  --tail      Number of lines to return (default: 50)
  --follow    Stream logs continuously

EXAMPLES:
  fibe playgrounds logs 42
  fibe pg logs 42 --service web --tail 100
  fibe pg logs 42 --follow
  fibe pg logs 42 --service web --follow --duration 10m`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if follow {
				return runLogMonitor(cmd, "playground", args[0], service, tail, maxLines, duration)
			}
			c := newClient()
			var t *int
			if tail > 0 {
				t = &tail
			}
			logs, err := c.Playgrounds.LogsByIdentifier(ctx(), args[0], service, t)
			if err != nil {
				return err
			}
			if effectiveOutput() != "table" {
				outputJSON(logs)
				return nil
			}
			printPlaygroundLogs(logs)
			return nil
		},
	}

	cmd.Flags().StringVar(&service, "service", "", "Optional service name")
	cmd.Flags().IntVar(&tail, "tail", 0, "Number of lines")
	cmd.Flags().BoolVar(&follow, "follow", false, "Stream logs continuously")
	cmd.Flags().IntVar(&maxLines, "max-lines", 0, "Follow mode: stop after N log lines (0 = unbounded)")
	cmd.Flags().DurationVar(&duration, "duration", 0, "Follow mode: stop after this duration (0 = until cancelled)")
	return cmd
}

func pgEnvCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "env <id-or-name>",
		Short: "Get playground environment metadata",
		Long: `Get the merged environment variables and metadata for a playground.

Returns the final merged environment, per-source metadata showing where
each variable comes from, and system-reserved keys.

EXAMPLES:
  fibe playgrounds env 42`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			env, err := c.Playgrounds.EnvMetadataByIdentifier(ctx(), args[0])
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
		Use:   "debug <id-or-name>",
		Short: "Get comprehensive debug information",
		Long: `Get comprehensive debug information for a playground.

Returns detailed internal state useful for troubleshooting issues.
Output is always JSON due to the complex nested structure.

EXAMPLES:
  fibe playgrounds debug 42`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			debug, err := c.Playgrounds.DebugWithParamsByIdentifier(ctx(), args[0], nil)
			if err != nil {
				return err
			}
			if effectiveOutput() == "table" {
				printPlaygroundDebugSummary(debug)
				return nil
			}
			outputJSON(debug)
			return nil
		},
	}
}

func printPlaygroundLogs(logs *fibe.PlaygroundLogs) {
	if logs.Source == "compose_up" || logs.Source == "none" {
		fmt.Printf("Source: %s\n", logs.Source)
		if logs.Startup != nil {
			fmt.Printf("Startup: %s", logs.Startup.State)
			if logs.Startup.ExitCode != nil {
				fmt.Printf(" exit=%d", *logs.Startup.ExitCode)
			}
			if !logs.Startup.Available {
				fmt.Print(" unavailable")
			}
			fmt.Println()
			if len(logs.Startup.MissingArtifacts) > 0 {
				fmt.Printf("Missing artifacts: %s\n", strings.Join(logs.Startup.MissingArtifacts, ", "))
			}
			if logs.Startup.Error != "" {
				fmt.Printf("Error: %s\n", logs.Startup.Error)
			}
		}
		if len(logs.Lines) > 0 {
			fmt.Println()
		}
	}
	if logs.Service == "" && len(logs.Entries) > 0 {
		for _, entry := range logs.Entries {
			if entry.Service == "" {
				fmt.Println(entry.Line)
				continue
			}
			fmt.Printf("[%s] %s\n", entry.Service, entry.Line)
		}
		return
	}
	for _, line := range logs.Lines {
		fmt.Println(line)
	}
}

func printPlaygroundDebugSummary(debug map[string]any) {
	if playground, ok := debug["playground"].(map[string]any); ok {
		fmt.Printf("Playground: %v", playground["id"])
		if name, ok := playground["name"].(string); ok && name != "" {
			fmt.Printf(" %s", name)
		}
		if status, ok := playground["status"].(string); ok && status != "" {
			fmt.Printf(" status=%s", status)
		}
		fmt.Println()
	}
	if startup, ok := debug["startup"].(map[string]any); ok {
		fmt.Printf("Startup: %v", startup["state"])
		if exitCode, ok := startup["exit_code"]; ok && exitCode != nil {
			fmt.Printf(" exit=%v", exitCode)
		}
		if available, ok := startup["available"].(bool); ok && !available {
			fmt.Print(" unavailable")
		}
		fmt.Println()
		if missing, ok := stringSliceFromAny(startup["missing_artifacts"]); ok && len(missing) > 0 {
			fmt.Printf("Missing artifacts: %s\n", strings.Join(missing, ", "))
		}
		if logTail, ok := stringSliceFromAny(startup["log_tail"]); ok && len(logTail) > 0 {
			fmt.Println("Compose-up log tail:")
			for _, line := range logTail {
				fmt.Println(line)
			}
		}
		return
	}
	outputJSON(debug)
}

func stringSliceFromAny(value any) ([]string, bool) {
	items, ok := value.([]any)
	if !ok {
		return nil, false
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		if text, ok := item.(string); ok {
			out = append(out, text)
		}
	}
	return out, true
}

func fmtMaintenance(enabled bool) string {
	if enabled {
		return "enabled"
	}
	return "disabled"
}
