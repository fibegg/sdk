package main

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/fibegg/sdk/fibe"
	"github.com/spf13/cobra"
)

func monitorCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "monitor",
		Aliases: []string{"mon"},
		Short:   "List and follow agent messages, activity, mutters, and artefacts",
		Long: `Monitor agent-produced events with a single paginated list API.

Two modes:
  list    One-shot paginated query.
  follow  Poll for newly produced events.

Monitorable types (see --type):
  message       Chat messages
  activity      Activity story entries
  mutter        Agent mutters
  artefact      Generated artefacts

Tokens need scope "monitor:read". Per-agent granular allowlist is supported
via granular_scopes on the API key. Use monitor list -q ... as the canonical
agent-content search surface.`,
	}
	cmd.AddCommand(monitorListCmd(), monitorFollowCmd())
	return cmd
}

func monitorListCmd() *cobra.Command {
	var agentIDs, types, since, query string
	var contentLimit int
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List monitor events (one-shot, paginated)",
		Long: `List monitor events matching the given filters, newest first.

FILTERS:
  --agent           Comma-separated agent IDs. Empty = all accessible.
  --type            Comma-separated types (message, activity, mutter,
                    artefact). Default: all.
  --since           ISO 8601 lower bound on occurred_at.
  -q, --query       Full-text search across content.
  --content-limit   Advanced: truncate each payload to N bytes.

PAGINATION:
  Use the global --page and --per-page flags (default 1 / 25).

EXAMPLES:
  fibe monitor list --agent 42
  fibe mon list --type message,artefact -q error
  fibe mon list --since 2026-04-16T00:00:00Z --page 2 --per-page 50 -o json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			params := &fibe.MonitorListParams{
				AgentIDs:     agentIDs,
				Types:        types,
				Since:        since,
				Q:            query,
				ContentLimit: contentLimit,
			}
			if flagPage > 0 {
				params.Page = flagPage
			}
			if flagPerPage > 0 {
				params.PerPage = flagPerPage
			}
			res, err := c.Monitor.List(ctx(), params)
			if err != nil {
				return err
			}
			if effectiveOutput() != "table" {
				outputJSON(res)
				return nil
			}
			headers := []string{"OCCURRED_AT", "AGENT", "TYPE", "ITEM_ID"}
			rows := make([][]string, len(res.Data))
			for i, ev := range res.Data {
				rows[i] = []string{ev.OccurredAt, fmtInt64(ev.AgentID), ev.Type, truncateString(ev.ItemID, 48)}
			}
			outputTable(headers, rows)
			return nil
		},
	}
	cmd.Flags().StringVar(&agentIDs, "agent", "", "Comma-separated agent IDs")
	cmd.Flags().StringVar(&types, "type", "", "Comma-separated types")
	cmd.Flags().StringVar(&since, "since", "", "Lower bound ISO 8601")
	cmd.Flags().StringVarP(&query, "query", "q", "", "Full-text search")
	cmd.Flags().IntVar(&contentLimit, "content-limit", 0, "Advanced: truncate each payload to N bytes")
	return cmd
}

func monitorFollowCmd() *cobra.Command {
	var agentIDs, types, since, query string
	var contentLimit, maxEvents int
	var pollInterval, duration time.Duration
	cmd := &cobra.Command{
		Use:   "follow",
		Short: "Stream monitor events as they happen",
		Long: `Follow monitor events, emitting each one as NDJSON on stdout
in oldest-first order within each poll window.

STREAMING:
  Follows by polling the list endpoint with a rolling since watermark. The stream
  closes when --duration elapses, --max-events is hit, the process receives
  a signal, or an unrecoverable error occurs.

EXAMPLES:
  fibe monitor follow --agent 42
  fibe mon follow --type message,artefact -q error --duration 10m
  fibe mon follow --max-events 50 --poll-interval 5s`,
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			params := &fibe.MonitorListParams{
				AgentIDs:     agentIDs,
				Types:        types,
				Since:        since,
				Q:            query,
				ContentLimit: contentLimit,
			}
			opts := &fibe.MonitorFollowOptions{
				PollInterval: pollInterval,
				Duration:     duration,
				MaxEvents:    maxEvents,
			}
			events, errs := c.Monitor.Follow(ctx(), params, opts)

			enc := json.NewEncoder(cmd.OutOrStdout())
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
				case err, ok := <-errs:
					if !ok {
						errs = nil
						continue
					}
					if err != nil {
						return err
					}
				}
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&agentIDs, "agent", "", "Comma-separated agent IDs")
	cmd.Flags().StringVar(&types, "type", "", "Comma-separated types")
	cmd.Flags().StringVar(&since, "since", "", "Start ISO 8601 (default: now)")
	cmd.Flags().StringVarP(&query, "query", "q", "", "Full-text search")
	cmd.Flags().IntVar(&contentLimit, "content-limit", 0, "Advanced: truncate each payload to N bytes")
	cmd.Flags().IntVar(&maxEvents, "max-events", 0, "Stop after N events (0 = unbounded)")
	cmd.Flags().DurationVar(&pollInterval, "poll-interval", 2*time.Second, "Polling interval")
	cmd.Flags().DurationVar(&duration, "duration", 0, "Stop after this duration (0 = until cancelled)")
	return cmd
}

func truncateString(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-1] + "…"
}

func monitorTypesDocHint() string {
	return strings.Join(fibe.MonitorValidTypes, ", ")
}

var _ = monitorTypesDocHint
