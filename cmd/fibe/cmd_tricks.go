package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/fibegg/sdk/fibe"
	"github.com/spf13/cobra"
)

func tricksCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "tricks",
		Aliases: []string{"tr"},
		Short:   "Manage tricks (ad-hoc job workloads)",
		Long: `Manage Fibe tricks — ad-hoc job workloads that run to completion.

Unlike playgrounds (long-running environments), tricks are one-shot
executions created from job-mode playspecs. They start, run their
watched services, capture results, and complete.

LIFECYCLE:
  - pending:       Queued for deployment
  - in_progress:   Building and running
  - running:       Containers active, waiting for watched services
  - completed:     All watched services exited successfully
  - error:         One or more services failed

SUBCOMMANDS:
  list                List all tricks
  get <id-or-name>    Show trick details
  trigger             Run a new trick from a job-mode playspec
  rerun <id-or-name>  Re-run a completed/failed trick
  status <id-or-name> Check trick status and result
  logs <id-or-name>   Get service logs
  delete <id-or-name> Delete a trick`,
	}

	cmd.AddCommand(
		trListCmd(),
		trGetCmd(),
		trTriggerCmd(),
		trRerunCmd(),
		trStatusCmd(),
		trLogsCmd(),
		trDeleteCmd(),
	)
	return cmd
}

func trListCmd() *cobra.Command {
	var query, status, resultStatus, name, sort, createdAfter, createdBefore string
	var playspec string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all tricks",
		Long: `List all tricks (job-mode playgrounds) accessible to the authenticated user.

FILTERS:
  -q, --query           Search across name (substring match)
  --status              Filter by exact status. Values: pending, in_progress, running, completed, error
  --result-status       Filter job result. Values: succeeded, failed, unknown
  --name                Filter by name (substring match)
  --playspec            Filter by playspec ID or name

DATE RANGE:
  --created-after       Show items created on or after this date (ISO 8601)
  --created-before      Show items created on or before this date (ISO 8601)

SORTING:
  --sort                Sort results. Format: {column}_{direction}
                        Columns: created_at, name, status
                        Direction: asc, desc

RESULT COLUMN:
  ✓   All watched services exited successfully
  ✗   One or more services failed
  ?   Completed, but result details are not available in this response
  ⏳  Still running or pending

OUTPUT:
  Columns: ID, NAME, STATUS, RESULT, PLAYSPEC, CREATED
  Use --output json for full details.

EXAMPLES:
  fibe tricks list
  fibe tr list -q "ci-run"
  fibe tr list --status completed --sort created_at_desc
  fibe tr list --created-after 2026-01-01 -o json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			params := &fibe.PlaygroundListParams{}
			if query != "" {
				params.Q = query
			}
			if status != "" {
				params.Status = status
			}
			if resultStatus != "" {
				params.ResultStatus = resultStatus
			}
			if name != "" {
				params.Name = name
			}
			if playspec != "" {
				params.PlayspecIdentifier = playspec
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
			tricks, err := c.Tricks.List(ctx(), params)
			if err != nil {
				return err
			}
			if effectiveOutput() != "table" {
				outputJSON(tricks)
				return nil
			}
			headers := []string{"ID", "NAME", "STATUS", "RESULT", "PLAYSPEC", "CREATED"}
			rows := make([][]string, len(tricks.Data))
			for i, tr := range tricks.Data {
				rows[i] = []string{
					fmtInt64(tr.ID), tr.Name, tr.Status,
					trickResult(tr), fmtStr(tr.PlayspecName),
					fmtTimeVal(tr.CreatedAt),
				}
			}
			outputTable(headers, rows)
			return nil
		},
	}
	cmd.Flags().StringVarP(&query, "query", "q", "", "Search across name")
	cmd.Flags().StringVar(&status, "status", "", "Filter by status")
	cmd.Flags().StringVar(&resultStatus, "result-status", "", "Filter by job result status (succeeded, failed, unknown)")
	cmd.Flags().StringVar(&name, "name", "", "Filter by name (substring)")
	cmd.Flags().StringVar(&playspec, "playspec", "", "Filter by playspec ID or name")
	cmd.Flags().StringVar(&createdAfter, "created-after", "", "Filter: created after date (ISO 8601)")
	cmd.Flags().StringVar(&createdBefore, "created-before", "", "Filter: created before date (ISO 8601)")
	cmd.Flags().StringVar(&sort, "sort", "", "Sort order (e.g. created_at_desc)")
	return cmd
}

func trGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <id-or-name>",
		Short: "Show detailed trick information",
		Long: `Get detailed information about a specific trick.

Includes status, job result, error messages, and service info.

EXAMPLES:
  fibe tricks get 42
  fibe tr get 42 --output json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			tr, err := c.Tricks.GetByIdentifier(ctx(), args[0])
			if err != nil {
				return err
			}
			if effectiveOutput() != "table" {
				outputJSON(tr)
				return nil
			}
			fmt.Printf("ID:        %d\n", tr.ID)
			fmt.Printf("Name:      %s\n", tr.Name)
			fmt.Printf("Status:    %s\n", tr.Status)
			fmt.Printf("Result:    %s\n", trickResult(*tr))
			fmt.Printf("Playspec:  %s\n", fmtStr(tr.PlayspecName))
			fmt.Printf("Created:   %s\n", fmtTimeVal(tr.CreatedAt))
			if tr.ErrorMessage != nil {
				fmt.Printf("Error:     %s\n", *tr.ErrorMessage)
			}
			if tr.JobResult != nil && tr.JobResult.Success != nil {
				fmt.Printf("Success:   %s\n", fmtBool(*tr.JobResult.Success))
			}
			return nil
		},
	}
}

func trTriggerCmd() *cobra.Command {
	var playspec string
	var marquee string
	var name string
	var envOverridesJSON string
	var onlyServices []string
	var exceptServices []string

	cmd := &cobra.Command{
		Use:   "trigger",
		Short: "Run a new trick from a job-mode playspec",
		Long: `Trigger a new trick run from a job-mode playspec.

A trick name is auto-generated as "{playspec-name}-{random}" unless
you provide one explicitly with --name.
The selected Marquee must be funded; unpaid Marquees fail
with MARQUEE_NOT_FUNDED.

REQUIRED FLAGS:
  --playspec      ID or name of the job-mode playspec

OPTIONAL FLAGS:
  --marquee       Target server ID or name
  --name          Custom trick name (auto-generated if omitted)
  --env-overrides JSON object of per-run environment overrides
  --only-service  Run only these service names (repeatable)
  --except-service Exclude these service names (repeatable)

EXAMPLES:
  fibe tricks trigger --playspec nightly-build
  fibe tr trigger --playspec nightly-build --marquee next
  fibe tr trigger --playspec nightly-build --name "my-ci-run"
  fibe tr trigger --playspec nightly-build --only-service tests --env-overrides '{"GITHUB_PAT":"..."}'`,
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			params := &fibe.TrickTriggerParams{
				PlayspecIdentifier: playspec,
			}
			if cmd.Flags().Changed("marquee") {
				params.MarqueeIdentifier = marquee
			}
			if cmd.Flags().Changed("name") {
				params.Name = name
			}
			if cmd.Flags().Changed("env-overrides") {
				parsed, err := parseStringMapJSONFlag(envOverridesJSON, "env-overrides")
				if err != nil {
					return err
				}
				params.EnvOverrides = parsed
			}
			if len(onlyServices) > 0 {
				params.OnlyServices = onlyServices
			}
			if len(exceptServices) > 0 {
				params.ExceptServices = exceptServices
			}

			if params.PlayspecID == 0 && params.PlayspecIdentifier == "" {
				return fmt.Errorf("required field 'playspec' not set")
			}

			tr, err := c.Tricks.Trigger(ctx(), params)
			if err != nil {
				return err
			}
			if effectiveOutput() != "table" {
				outputJSON(tr)
				return nil
			}
			fmt.Printf("Triggered trick %d (%s) — status: %s\n", tr.ID, tr.Name, tr.Status)
			return nil
		},
	}

	cmd.Flags().StringVar(&playspec, "playspec", "", "Job-mode playspec ID or name (required)")
	cmd.Flags().StringVar(&marquee, "marquee", "", "Target marquee ID or name (optional)")
	cmd.Flags().StringVar(&name, "name", "", "Custom trick name (auto-generated if omitted)")
	cmd.Flags().StringVar(&envOverridesJSON, "env-overrides", "", "JSON object of per-run environment overrides")
	cmd.Flags().StringSliceVar(&onlyServices, "only-service", nil, "Run only this service name (repeatable or comma-separated)")
	cmd.Flags().StringSliceVar(&exceptServices, "except-service", nil, "Exclude this service name (repeatable or comma-separated)")
	return cmd
}

func trRerunCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "rerun <id-or-name>",
		Short: "Re-run a completed or failed trick",
		Long: `Create a new trick run by copying the playspec and marquee settings
from an existing trick. The new trick gets a fresh auto-generated name.

EXAMPLES:
  fibe tricks rerun 42`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			tr, err := c.Tricks.RerunByIdentifier(ctx(), args[0])
			if err != nil {
				return err
			}
			if effectiveOutput() != "table" {
				outputJSON(tr)
				return nil
			}
			fmt.Printf("Re-triggered trick %d (%s) from source %s — status: %s\n", tr.ID, tr.Name, args[0], tr.Status)
			return nil
		},
	}
}

func trStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status <id-or-name>",
		Short: "Check trick status and job result",
		Long: `Get the current status of a trick, including the job result
with per-service outcomes when completed.

EXAMPLES:
  fibe tricks status 42`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			status, err := c.Tricks.StatusByIdentifier(ctx(), args[0])
			if err != nil {
				return err
			}
			if effectiveOutput() != "table" {
				outputJSON(status)
				return nil
			}
			fmt.Printf("Trick %d: %s\n", status.ID, status.Status)
			if outcome := fibe.TrickStatusOutcome(status); outcome != fibe.TrickResultRunning {
				fmt.Printf("Result:  %s\n", trickOutcomeLabel(outcome))
			}
			return nil
		},
	}
}

func trLogsCmd() *cobra.Command {
	var service string
	var tail int
	var all bool
	var follow bool
	var maxLines int
	var duration time.Duration

	cmd := &cobra.Command{
		Use:   "logs <id-or-name>",
		Short: "Get logs from a trick",
		Long: `Retrieve logs from a trick.

For completed tricks, logs are served from cache. For running tricks,
logs are fetched live from containers. All services are returned by default;
use --service to focus on one service.
Snapshot trick logs return all cached job logs by default. Use --tail to limit
the number of cached lines.

OPTIONAL FLAGS:
  --service   Optional service name to filter logs
  --tail      Number of lines to return; 0 means all cached job logs
  --all       Return all cached job logs
  --follow    Stream logs continuously

EXAMPLES:
  fibe tricks logs 42
  fibe tr logs 42 --service app --tail 200
  fibe tr logs 42 --service results --all
  fibe tr logs 42 --follow
  fibe tr logs 42 --service app --follow --duration 10m`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if all && cmd.Flags().Changed("tail") && tail > 0 {
				return fmt.Errorf("--all and --tail are mutually exclusive")
			}
			if all && follow {
				return fmt.Errorf("--all is only supported for snapshot logs; omit --follow")
			}
			if all {
				tail = 0
			}
			if follow {
				return runLogMonitor(cmd, "trick", args[0], service, tail, maxLines, duration)
			}
			progress := newStatusLine(cmd.ErrOrStderr(), statusLineOptions{})
			progress.Start("fetching logs for trick " + args[0] + "...")
			defer progress.Stop()
			c := newClient(fibe.WithProgress(progress.Progress("fetching logs for trick " + args[0])))
			t := &tail
			logs, err := c.Tricks.LogsByIdentifier(ctx(), args[0], service, t)
			progress.Stop()
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
	cmd.Flags().BoolVar(&all, "all", false, "Return all cached job logs")
	cmd.Flags().BoolVar(&follow, "follow", false, "Stream logs continuously")
	cmd.Flags().IntVar(&maxLines, "max-lines", 0, "Follow mode: stop after N log lines (0 = unbounded)")
	cmd.Flags().DurationVar(&duration, "duration", 0, "Follow mode: stop after this duration (0 = until cancelled)")
	return cmd
}

func trDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <id-or-name>",
		Short: "Delete a trick",
		Long: `Delete a trick and tear down its services.

EXAMPLES:
  fibe tricks delete 42`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			if err := c.Tricks.DeleteByIdentifier(ctx(), args[0]); err != nil {
				return err
			}
			fmt.Printf("Trick %s deletion initiated\n", args[0])
			return nil
		},
	}
}

// trickResult returns a human-readable result indicator for a trick.
func trickResult(pg fibe.Playground) string {
	switch fibe.TrickOutcome(pg) {
	case fibe.TrickResultSucceeded:
		return "✓"
	case fibe.TrickResultFailed:
		return "✗"
	case fibe.TrickResultUnknown:
		return "?"
	default:
		return "⏳"
	}
}

// trickResultFromJobResult returns a result indicator from a JobResult.
func trickResultFromJobResult(jr *fibe.JobResult) string {
	if jr == nil || jr.Success == nil {
		return "⏳"
	}
	if *jr.Success {
		return "✓ success"
	}
	return "✗ failed"
}

func trickOutcomeLabel(outcome string) string {
	switch outcome {
	case fibe.TrickResultSucceeded:
		return "✓ success"
	case fibe.TrickResultFailed:
		return "✗ failed"
	case fibe.TrickResultUnknown:
		return "? unknown"
	default:
		return "⏳ running"
	}
}

func parseStringMapJSONFlag(raw, flagName string) (map[string]string, error) {
	var out map[string]string
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil, fmt.Errorf("--%s must be a JSON object with string values: %w", flagName, err)
	}
	return out, nil
}
