package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/fibegg/sdk/fibe"
	"github.com/spf13/cobra"
)

func jobEnvCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "job-env",
		Short: "Manage job-mode environment variables and secrets",
		Long: `Manage global and Prop-scoped ENV entries injected into job-mode Tricks.

Keys must be uppercase letters, numbers, and underscores, and may not use the reserved FIBE_ prefix.`,
	}
	cmd.AddCommand(jobEnvListCmd(), jobEnvGetCmd(), jobEnvSetCmd(), jobEnvUpdateCmd(), jobEnvDeleteCmd())
	return cmd
}

func jobEnvListCmd() *cobra.Command {
	var propID int64
	var query string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List job ENV entries",
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			params := &fibe.JobEnvListParams{}
			if propID > 0 {
				params.PropID = propID
			}
			if query != "" {
				params.Q = query
			}
			if flagPage > 0 {
				params.Page = flagPage
			}
			if flagPerPage > 0 {
				params.PerPage = flagPerPage
			}
			result, err := c.JobEnv.List(ctx(), params)
			if err != nil {
				return err
			}
			if effectiveOutput() != "table" {
				outputJSON(result)
				return nil
			}
			headers := []string{"ID", "SCOPE", "KEY", "KIND", "ENABLED"}
			rows := make([][]string, len(result.Data))
			for i, entry := range result.Data {
				scope := "global"
				if entry.PropID != nil {
					scope = fmt.Sprintf("prop:%d", *entry.PropID)
				}
				kind := "variable"
				if entry.Secret {
					kind = "secret"
				}
				rows[i] = []string{fmtInt64Ptr(entry.ID), scope, entry.Key, kind, fmt.Sprintf("%t", entry.Enabled)}
			}
			outputTable(headers, rows)
			return nil
		},
	}
	cmd.Flags().Int64Var(&propID, "prop-id", 0, "Filter by Prop ID")
	cmd.Flags().StringVarP(&query, "query", "q", "", "Search key/description")
	return cmd
}

func jobEnvGetCmd() *cobra.Command {
	var reveal bool
	cmd := &cobra.Command{
		Use:   "get <id>",
		Short: "Show a job ENV entry",
		Long:  "Retrieve a job ENV entry. Secret values are omitted unless --reveal is set.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, _ := strconv.ParseInt(args[0], 10, 64)
			entry, err := newClient().JobEnv.Get(ctx(), id, reveal)
			if err != nil {
				return err
			}
			if effectiveOutput() != "table" {
				outputJSON(entry)
				return nil
			}
			scope := "global"
			if entry.PropID != nil {
				scope = fmt.Sprintf("prop:%d", *entry.PropID)
			}
			kind := "variable"
			if entry.Secret {
				kind = "secret"
			}
			fmt.Printf("ID:      %s\nScope:   %s\nKey:     %s\nKind:    %s\nEnabled: %t\nDesc:    %s\n",
				fmtInt64Ptr(entry.ID), scope, entry.Key, kind, entry.Enabled, fmtStr(entry.Description))
			if !entry.Secret || reveal {
				fmt.Printf("Value:   %s\n", fmtStr(entry.Value))
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&reveal, "reveal", false, "Include the plaintext value for secret entries")
	return cmd
}

func jobEnvSetCmd() *cobra.Command {
	var propID int64
	var secret bool
	var desc string
	cmd := &cobra.Command{
		Use:   "set KEY=VALUE",
		Short: "Create a job ENV entry",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			key, value, ok := strings.Cut(args[0], "=")
			if !ok || key == "" {
				return fmt.Errorf("expected KEY=VALUE")
			}
			c := newClient()
			params := &fibe.JobEnvSetParams{Key: key, Value: value, Secret: secret}
			if propID > 0 {
				params.PropID = &propID
			}
			if desc != "" {
				params.Description = &desc
			}
			entry, err := c.JobEnv.Set(ctx(), params)
			if err != nil {
				return err
			}
			outputJSON(entry)
			return nil
		},
	}
	cmd.Flags().Int64Var(&propID, "prop-id", 0, "Scope to a Prop ID instead of global")
	cmd.Flags().BoolVar(&secret, "secret", false, "Store as secret and mask in list responses")
	cmd.Flags().StringVar(&desc, "description", "", "Optional description")
	return cmd
}

func jobEnvUpdateCmd() *cobra.Command {
	var value string
	var secret, enabled bool
	var desc string
	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update a job ENV entry",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, _ := strconv.ParseInt(args[0], 10, 64)
			params := &fibe.JobEnvUpdateParams{}
			if cmd.Flags().Changed("value") {
				params.Value = &value
			}
			if cmd.Flags().Changed("secret") {
				params.Secret = &secret
			}
			if cmd.Flags().Changed("enabled") {
				params.Enabled = &enabled
			}
			if cmd.Flags().Changed("description") {
				params.Description = &desc
			}
			entry, err := newClient().JobEnv.Update(ctx(), id, params)
			if err != nil {
				return err
			}
			outputJSON(entry)
			return nil
		},
	}
	cmd.Flags().StringVar(&value, "value", "", "New value")
	cmd.Flags().BoolVar(&secret, "secret", false, "Whether entry is a secret")
	cmd.Flags().BoolVar(&enabled, "enabled", true, "Whether entry is enabled")
	cmd.Flags().StringVar(&desc, "description", "", "New description")
	return cmd
}

func jobEnvDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a job ENV entry",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, _ := strconv.ParseInt(args[0], 10, 64)
			if err := newClient().JobEnv.Delete(ctx(), id); err != nil {
				return err
			}
			fmt.Printf("Job ENV entry %d deleted\n", id)
			return nil
		},
	}
}
