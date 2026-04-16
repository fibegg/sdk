package main

import (
	"strconv"

	"github.com/fibegg/sdk/fibe"
	"github.com/spf13/cobra"
)
func muttersCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mutters",
		Short: "Manage agent mutters (observations)",
		Long: `Manage mutters — agent observation and monitoring data.

Mutters capture structured observations from agent runs, including
status updates, severity markers, and playground-scoped data.

SUBCOMMANDS:
  get <agent-id>        Get agent mutters
  create <agent-id>     Create a mutter item`,
	}
	cmd.AddCommand(mutterGetCmd(), mutterCreateCmd())
	return cmd
}

func mutterGetCmd() *cobra.Command {
	var playgroundID, query, status, severity string
	cmd := &cobra.Command{
		Use: "get <agent-id>", Short: "Get agent mutters", Args: cobra.ExactArgs(1),
		Long: `Get mutters for an agent with optional filters.

FILTERS:
  -q, --query           Search across mutter values (substring match)
  --playground-id       Filter by playground ID
  --status              Filter by status
  --severity            Filter by severity

EXAMPLES:
  fibe mutters get 5
  fibe mutters get 5 --playground-id 42
  fibe mutters get 5 --status error --severity high -o json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			agentID, _ := strconv.ParseInt(args[0], 10, 64)
			params := &fibe.MutterListParams{}
			if playgroundID != "" { params.PlaygroundID = playgroundID }
			if query != "" { params.Query = query }
			if status != "" { params.Status = status }
			if severity != "" { params.Severity = severity }
			if flagPage > 0 { params.Page = flagPage }
			if flagPerPage > 0 { params.PerPage = flagPerPage }
			mutter, err := c.Mutters.Get(ctx(), agentID, params)
			if err != nil { return err }
			outputJSON(mutter)
			return nil
		},
	}
	cmd.Flags().StringVarP(&query, "query", "q", "", "Search across mutter values")
	cmd.Flags().StringVar(&playgroundID, "playground-id", "", "Filter by playground ID")
	cmd.Flags().StringVar(&status, "status", "", "Filter by status")
	cmd.Flags().StringVar(&severity, "severity", "", "Filter by severity")
	return cmd
}

func mutterCreateCmd() *cobra.Command {
	var typ, body string
	var playgroundID int64
	cmd := &cobra.Command{
		Use: "create <agent-id>", Short: "Create a mutter item", Args: cobra.ExactArgs(1),
		Long: `Create a new mutter item for an agent.

REQUIRED FLAGS:
  --type    Item type
  --body    Item body content

OPTIONAL FLAGS:
  --playground-id   Associate with a playground

EXAMPLES:
  fibe mutters create 5 --type observation --body "Service restarted"
  fibe mutters create 5 --type alert --body "High CPU" --playground-id 42`,
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			agentID, _ := strconv.ParseInt(args[0], 10, 64)
			params := &fibe.MutterItemParams{Type: typ, Body: body}
			if cmd.Flags().Changed("playground-id") { params.PlaygroundID = &playgroundID }
			mutter, err := c.Mutters.CreateItem(ctx(), agentID, params)
			if err != nil { return err }
			outputJSON(mutter)
			return nil
		},
	}
	cmd.Flags().StringVar(&typ, "type", "", "Item type (required)")
	cmd.Flags().StringVar(&body, "body", "", "Item body (required)")
	cmd.Flags().Int64Var(&playgroundID, "playground-id", 0, "Associate with a playground")
	return cmd
}

