package main

import (

	"github.com/fibegg/sdk/fibe"
	"github.com/spf13/cobra"
)
func auditLogsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "audit-logs",
		Short: "View audit logs",
		Long: `View Fibe audit logs — records of all API and UI actions.

Filter by resource type, channel (api/ui), or action prefix.

OPTIONAL FLAGS:
  --resource-type   Filter by resource (Playground, Agent, etc.)
  --channel         Filter by channel (api, ui)
  --action-prefix   Filter by action prefix

EXAMPLES:
  fibe audit-logs list
  fibe audit-logs list --channel api --resource-type Playground`,
	}
	cmd.AddCommand(alListCmd())
	return cmd
}

func alListCmd() *cobra.Command {
	var query, resType, channel, prefix, sort, createdAfter, createdBefore string
	cmd := &cobra.Command{
		Use: "list", Short: "List audit logs",
		Long: `List audit logs for the authenticated user.

FILTERS:
  -q, --query           Search across action (substring match)
  --resource-type       Filter by resource type (e.g. Playground, Agent, Prop)
  --channel             Filter by channel. Values: api, ui
  --action-prefix       Filter by action prefix (substring match)

DATE RANGE:
  --created-after       Show items created on or after this date (ISO 8601)
  --created-before      Show items created on or before this date (ISO 8601)

SORTING:
  --sort                Sort results. Format: {column}_{direction}
                        Columns: created_at
                        Direction: asc, desc
                        Default: created_at_desc

OUTPUT:
  Columns: ID, ACTION, RESOURCE, CHANNEL, CREATED
  Use --output json for full details.

EXAMPLES:
  fibe audit-logs list
  fibe audit-logs list --channel api --resource-type Playground
  fibe audit-logs list -q "create" --created-after 2026-01-01
  fibe audit-logs list --action-prefix playground. -o json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			params := &fibe.AuditLogListParams{}
			if query != "" { params.Q = query }
			if resType != "" { params.ResourceType = resType }
			if channel != "" { params.Channel = channel }
			if prefix != "" { params.ActionPrefix = prefix }
			if createdAfter != "" { params.CreatedAfter = createdAfter }
			if createdBefore != "" { params.CreatedBefore = createdBefore }
			if sort != "" { params.Sort = sort }
			if flagPage > 0 { params.Page = flagPage }
			if flagPerPage > 0 { params.PerPage = flagPerPage }
			logs, err := c.AuditLogs.List(ctx(), params)
			if err != nil { return err }
			if effectiveOutput() != "table" { outputJSON(logs); return nil }
			headers := []string{"ID", "ACTION", "RESOURCE", "CHANNEL", "CREATED"}
			rows := make([][]string, len(logs.Data))
			for i, l := range logs.Data { rows[i] = []string{fmtInt64(l.ID), l.Action, l.ResourceType, l.Channel, fmtTimeVal(l.CreatedAt)} }
			outputTable(headers, rows)
			return nil
		},
	}
	cmd.Flags().StringVarP(&query, "query", "q", "", "Search across action")
	cmd.Flags().StringVar(&resType, "resource-type", "", "Resource type filter")
	cmd.Flags().StringVar(&channel, "channel", "", "Channel filter (api, ui)")
	cmd.Flags().StringVar(&prefix, "action-prefix", "", "Action prefix filter")
	cmd.Flags().StringVar(&createdAfter, "created-after", "", "Filter: created after date (ISO 8601)")
	cmd.Flags().StringVar(&createdBefore, "created-before", "", "Filter: created before date (ISO 8601)")
	cmd.Flags().StringVar(&sort, "sort", "", "Sort order (e.g. created_at_desc)")
	return cmd
}

