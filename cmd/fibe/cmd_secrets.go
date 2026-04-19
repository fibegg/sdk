package main

import (
	"fmt"
	"strconv"

	"github.com/fibegg/sdk/fibe"
	"github.com/spf13/cobra"
)

func secretsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "secrets",
		Short: "Manage encrypted secrets",
		Long: `Manage Fibe secrets — encrypted key-value pairs for environment variables.

Secrets are injected into playgrounds as environment variables.
Values are encrypted at rest and only revealed with an explicit --reveal flag.

SUBCOMMANDS:
  list              List all secrets (keys only)
  get <id>          Show secret metadata; pass --reveal for plaintext value
  create            Create a new secret
  update <id>       Update secret value
  delete <id>       Delete a secret`,
	}
	cmd.AddCommand(secListCmd(), secGetCmd(), secCreateCmd(), secUpdateCmd(), secDeleteCmd())
	return cmd
}

func secListCmd() *cobra.Command {
	var query, key, sort, createdAfter, createdBefore string
	cmd := &cobra.Command{
		Use: "list", Short: "List all secrets",
		Long: `List all secrets. Values are NOT shown — use 'get --reveal' to reveal.

FILTERS:
  -q, --query           Search across key, description (substring match)
  --key                 Filter by key name (substring match)

DATE RANGE:
  --created-after       Show items created on or after this date (ISO 8601)
  --created-before      Show items created on or before this date (ISO 8601)

SORTING:
  --sort                Sort results. Format: {column}_{direction}
                        Columns: created_at, key
                        Direction: asc, desc
                        Default: created_at_desc

OUTPUT:
  Columns: ID, KEY, DESCRIPTION, CREATED
  Use --output json for full details.

EXAMPLES:
  fibe secrets list
  fibe secrets list -q "DATABASE"
  fibe secrets list --key API_TOKEN --sort key_asc`,
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			params := &fibe.SecretListParams{}
			if query != "" {
				params.Q = query
			}
			if key != "" {
				params.Key = key
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
			result, err := c.Secrets.List(ctx(), params)
			if err != nil {
				return err
			}
			if effectiveOutput() != "table" {
				outputJSON(result)
				return nil
			}
			headers := []string{"ID", "KEY", "DESCRIPTION", "CREATED"}
			rows := make([][]string, len(result.Data))
			for i, s := range result.Data {
				rows[i] = []string{fmtInt64Ptr(s.ID), s.Key, fmtStr(s.Description), fmtTime(s.CreatedAt)}
			}
			outputTable(headers, rows)
			return nil
		},
	}
	cmd.Flags().StringVarP(&query, "query", "q", "", "Search across key, description")
	cmd.Flags().StringVar(&key, "key", "", "Filter by key (substring)")
	cmd.Flags().StringVar(&createdAfter, "created-after", "", "Filter: created after date (ISO 8601)")
	cmd.Flags().StringVar(&createdBefore, "created-before", "", "Filter: created before date (ISO 8601)")
	cmd.Flags().StringVar(&sort, "sort", "", "Sort order (e.g. created_at_desc)")
	return cmd
}

func secGetCmd() *cobra.Command {
	var reveal bool
	cmd := &cobra.Command{
		Use: "get <id>", Short: "Show secret metadata", Args: cobra.ExactArgs(1),
		Long: "Retrieve a secret. Values are omitted unless --reveal is set.\n\nWARNING: --reveal shows the value in plaintext.\n\nEXAMPLES:\n  fibe secrets get 10\n  fibe secrets get 10 --reveal",
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			id, _ := strconv.ParseInt(args[0], 10, 64)
			s, err := c.Secrets.Get(ctx(), id, reveal)
			if err != nil {
				return err
			}
			if effectiveOutput() != "table" {
				outputJSON(s)
				return nil
			}
			fmt.Printf("ID:    %s\nKey:   %s\nDesc:  %s\n", fmtInt64Ptr(s.ID), s.Key, fmtStr(s.Description))
			if reveal {
				fmt.Printf("Value: %s\n", fmtStr(s.Value))
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&reveal, "reveal", false, "Include the plaintext secret value")
	return cmd
}

func secCreateCmd() *cobra.Command {
	var key, value, desc string
	cmd := &cobra.Command{
		Use: "create", Short: "Create a new secret",
		Long: "Create a new encrypted secret.\n\nSECRET CONSTRAINTS:\n  - Keys MUST only contain alphanumeric characters, dashes, and underscores.\n  - Secrets inject into Playground environments automatically.\n  - 'fibe secrets list' and 'fibe secrets get' intentionally omit values. Use 'get --reveal' to decrypt securely.\n\nREQUIRED FLAGS:\n  --key     Secret key name\n  --value   Secret value\n\nOPTIONAL FLAGS:\n  --description   Description\n\nEXAMPLES:\n  fibe secrets create --key DATABASE_URL --value postgres://...\n  fibe secrets create --key API_TOKEN --value xxx --description \"Third-party API\"",
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			params := &fibe.SecretCreateParams{}
			if err := applyFromFile(params); err != nil {
				return err
			}
			if cmd.Flags().Changed("key") {
				params.Key = key
			}
			if cmd.Flags().Changed("value") {
				params.Value = value
			}
			if cmd.Flags().Changed("description") {
				params.Description = &desc
			}

			if params.Key == "" {
				return fmt.Errorf("required field 'key' not set")
			}
			if params.Value == "" {
				return fmt.Errorf("required field 'value' not set")
			}

			s, err := c.Secrets.Create(ctx(), params)
			if err != nil {
				return err
			}
			fmt.Printf("Created secret %s (%s)\n", fmtInt64Ptr(s.ID), s.Key)
			return nil
		},
	}
	cmd.Flags().StringVar(&key, "key", "", "Key name (required)")
	cmd.Flags().StringVar(&value, "value", "", "Value (required)")
	cmd.Flags().StringVar(&desc, "description", "", "Description")
	return cmd
}

func secUpdateCmd() *cobra.Command {
	var value, desc string
	cmd := &cobra.Command{
		Use: "update <id>", Short: "Update a secret", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			id, _ := strconv.ParseInt(args[0], 10, 64)
			params := &fibe.SecretUpdateParams{}
			if err := applyFromFile(params); err != nil {
				return err
			}
			if cmd.Flags().Changed("value") {
				params.Value = &value
			}
			if cmd.Flags().Changed("description") {
				params.Description = &desc
			}
			s, err := c.Secrets.Update(ctx(), id, params)
			if err != nil {
				return err
			}
			fmt.Printf("Updated secret %s\n", fmtInt64Ptr(s.ID))
			return nil
		},
	}
	cmd.Flags().StringVar(&value, "value", "", "New value")
	cmd.Flags().StringVar(&desc, "description", "", "New description")
	return cmd
}

func secDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use: "delete <id>", Short: "Delete a secret", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			id, _ := strconv.ParseInt(args[0], 10, 64)
			if err := c.Secrets.Delete(ctx(), id); err != nil {
				return err
			}
			fmt.Printf("Secret %d deleted\n", id)
			return nil
		},
	}
}
