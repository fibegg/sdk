package main

import (
	"fmt"
	"strconv"

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
  get <id>            Show trick details
  trigger             Run a new trick from a job-mode playspec
  rerun <id>          Re-run a completed/failed trick
  status <id>         Check trick status and result
  logs <id>           Get service logs
  delete <id>         Delete a trick`,
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
	var query, status, name, sort, createdAfter, createdBefore string
	var playspecID int64
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all tricks",
		Long: `List all tricks (job-mode playgrounds) accessible to the authenticated user.

FILTERS:
  -q, --query           Search across name (substring match)
  --status              Filter by exact status. Values: pending, in_progress, running, completed, error
  --name                Filter by name (substring match)
  --playspec-id         Filter by playspec ID

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
			if query != "" { params.Q = query }
			if status != "" { params.Status = status }
			if name != "" { params.Name = name }
			if playspecID > 0 { params.PlayspecID = playspecID }
			if createdAfter != "" { params.CreatedAfter = createdAfter }
			if createdBefore != "" { params.CreatedBefore = createdBefore }
			if sort != "" { params.Sort = sort }
			if flagPage > 0 { params.Page = flagPage }
			if flagPerPage > 0 { params.PerPage = flagPerPage }
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
	cmd.Flags().StringVar(&name, "name", "", "Filter by name (substring)")
	cmd.Flags().Int64Var(&playspecID, "playspec-id", 0, "Filter by playspec ID")
	cmd.Flags().StringVar(&createdAfter, "created-after", "", "Filter: created after date (ISO 8601)")
	cmd.Flags().StringVar(&createdBefore, "created-before", "", "Filter: created before date (ISO 8601)")
	cmd.Flags().StringVar(&sort, "sort", "", "Sort order (e.g. created_at_desc)")
	return cmd
}

func trGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <id>",
		Short: "Show detailed trick information",
		Long: `Get detailed information about a specific trick.

Includes status, job result, error messages, and service info.

EXAMPLES:
  fibe tricks get 42
  fibe tr get 42 --output json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			id, _ := strconv.ParseInt(args[0], 10, 64)
			tr, err := c.Tricks.Get(ctx(), id)
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
	var playspecID int64
	var marqueeID int64
	var name string

	cmd := &cobra.Command{
		Use:   "trigger",
		Short: "Run a new trick from a job-mode playspec",
		Long: `Trigger a new trick run from a job-mode playspec.

A trick name is auto-generated as "{playspec-name}-{random}" unless
you provide one explicitly with --name.

REQUIRED FLAGS:
  --playspec-id   ID of the job-mode playspec

OPTIONAL FLAGS:
  --marquee-id    Target server
  --name          Custom trick name (auto-generated if omitted)

EXAMPLES:
  fibe tricks trigger --playspec-id 12
  fibe tr trigger --playspec-id 12 --marquee-id 3
  fibe tr trigger --playspec-id 12 --name "my-ci-run"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			params := &fibe.TrickTriggerParams{
				PlayspecID: playspecID,
			}
			if cmd.Flags().Changed("marquee-id") {
				params.MarqueeID = &marqueeID
			}
			if cmd.Flags().Changed("name") {
				params.Name = name
			}

			if params.PlayspecID == 0 {
				return fmt.Errorf("required field 'playspec-id' not set")
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

	cmd.Flags().Int64Var(&playspecID, "playspec-id", 0, "Job-mode playspec ID (required)")
	cmd.Flags().Int64Var(&marqueeID, "marquee-id", 0, "Target marquee ID (optional)")
	cmd.Flags().StringVar(&name, "name", "", "Custom trick name (auto-generated if omitted)")
	return cmd
}

func trRerunCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "rerun <id>",
		Short: "Re-run a completed or failed trick",
		Long: `Create a new trick run by copying the playspec and marquee settings
from an existing trick. The new trick gets a fresh auto-generated name.

EXAMPLES:
  fibe tricks rerun 42`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			id, _ := strconv.ParseInt(args[0], 10, 64)
			tr, err := c.Tricks.Rerun(ctx(), id)
			if err != nil {
				return err
			}
			if effectiveOutput() != "table" {
				outputJSON(tr)
				return nil
			}
			fmt.Printf("Re-triggered trick %d (%s) from source %d — status: %s\n", tr.ID, tr.Name, id, tr.Status)
			return nil
		},
	}
}

func trStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status <id>",
		Short: "Check trick status and job result",
		Long: `Get the current status of a trick, including the job result
with per-service outcomes when completed.

EXAMPLES:
  fibe tricks status 42`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			id, _ := strconv.ParseInt(args[0], 10, 64)
			status, err := c.Tricks.Status(ctx(), id)
			if err != nil {
				return err
			}
			if effectiveOutput() != "table" {
				outputJSON(status)
				return nil
			}
			fmt.Printf("Trick %d: %s\n", status.ID, status.Status)
			if status.JobResult != nil && status.JobResult.Success != nil {
				fmt.Printf("Result:  %s\n", trickResultFromJobResult(status.JobResult))
			}
			return nil
		},
	}
}

func trLogsCmd() *cobra.Command {
	var service string
	var tail int

	cmd := &cobra.Command{
		Use:   "logs <id>",
		Short: "Get service logs from a trick",
		Long: `Retrieve logs from a specific service in a trick.

For completed tricks, logs are served from cache. For running tricks,
logs are fetched live from the container.

REQUIRED FLAGS:
  --service   Name of the service to get logs from

OPTIONAL FLAGS:
  --tail      Number of lines to return (default: 50)

EXAMPLES:
  fibe tricks logs 42 --service worker
  fibe tr logs 42 --service app --tail 200`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			id, _ := strconv.ParseInt(args[0], 10, 64)
			var t *int
			if tail > 0 {
				t = &tail
			}
			logs, err := c.Tricks.Logs(ctx(), id, service, t)
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

func trDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a trick",
		Long: `Delete a trick and tear down its services.

EXAMPLES:
  fibe tricks delete 42`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			id, _ := strconv.ParseInt(args[0], 10, 64)
			if err := c.Tricks.Delete(ctx(), id); err != nil {
				return err
			}
			fmt.Printf("Trick %d deletion initiated\n", id)
			return nil
		},
	}
}

// trickResult returns a human-readable result indicator for a trick.
func trickResult(pg fibe.Playground) string {
	switch pg.Status {
	case "completed":
		if pg.JobResult != nil && pg.JobResult.Success != nil {
			if *pg.JobResult.Success {
				return "✓"
			}
			return "✗"
		}
		return "✓"
	case "error":
		return "✗"
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
