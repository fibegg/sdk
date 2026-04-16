package main

import (
	"fmt"
	"strconv"

	"github.com/fibegg/sdk/fibe"
	"github.com/spf13/cobra"
)
func mutationsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mutations",
		Short: "Manage code mutations for a prop",
		Long: `Manage mutations — code changes detected in repositories (mutation testing).

Mutations track diffs, their cure status, and associated metadata.
Used primarily for CI/CD mutation testing workflows.

SUBCOMMANDS:
  list <prop-id>            List mutations
  create <prop-id>          Create a mutation
  update <prop-id> <id>     Update mutation status`,
	}
	cmd.AddCommand(mutListCmd(), mutCreateCmd(), mutUpdateCmd())
	return cmd
}

func mutListCmd() *cobra.Command {
	var status, curedAfter, curedBefore, createdAfter, createdBefore, sort string
	cmd := &cobra.Command{
		Use: "list <prop-id>", Short: "List mutations for a prop", Args: cobra.ExactArgs(1),
		Long: `List mutations for a prop.

FILTERS:
  --status              Filter by status (e.g. alive, cured, killed)

DATE RANGE:
  --cured-after         Show mutations cured on or after this date (ISO 8601)
  --cured-before        Show mutations cured on or before this date (ISO 8601)
  --created-after       Show items created on or after this date (ISO 8601)
  --created-before      Show items created on or before this date (ISO 8601)

SORTING:
  --sort                Sort results. Format: {column}_{direction}
                        Columns: created_at, cured_at
                        Direction: asc, desc
                        Default: created_at_desc

EXAMPLES:
  fibe mutations list 7
  fibe mutations list 7 --status alive
  fibe mutations list 7 --status cured --sort cured_at_desc -o json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			propID, _ := strconv.ParseInt(args[0], 10, 64)
			params := &fibe.MutationListParams{}
			if status != "" { params.Status = status }
			if curedAfter != "" { params.CuredAfter = curedAfter }
			if curedBefore != "" { params.CuredBefore = curedBefore }
			if createdAfter != "" { params.CreatedAfter = createdAfter }
			if createdBefore != "" { params.CreatedBefore = createdBefore }
			if sort != "" { params.Sort = sort }
			if flagPage > 0 { params.Page = flagPage }
			if flagPerPage > 0 { params.PerPage = flagPerPage }
			muts, err := c.Mutations.List(ctx(), propID, params)
			if err != nil { return err }
			outputJSON(muts)
			return nil
		},
	}
	cmd.Flags().StringVar(&status, "status", "", "Filter by status (alive, cured, killed)")
	cmd.Flags().StringVar(&curedAfter, "cured-after", "", "Filter: cured after date (ISO 8601)")
	cmd.Flags().StringVar(&curedBefore, "cured-before", "", "Filter: cured before date (ISO 8601)")
	cmd.Flags().StringVar(&createdAfter, "created-after", "", "Filter: created after date (ISO 8601)")
	cmd.Flags().StringVar(&createdBefore, "created-before", "", "Filter: created before date (ISO 8601)")
	cmd.Flags().StringVar(&sort, "sort", "", "Sort order (e.g. created_at_desc)")
	return cmd
}

func mutCreateCmd() *cobra.Command {
	var diff, sha, branch string
	cmd := &cobra.Command{
		Use: "create <prop-id>", Short: "Create a new mutation", Args: cobra.ExactArgs(1),
		Long: "Record a new code mutation.\n\nREQUIRED FLAGS:\n  --diff     Git diff content\n  --sha      Commit SHA where mutation was found\n  --branch   Branch name",
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			propID, _ := strconv.ParseInt(args[0], 10, 64)
			params := &fibe.MutationCreateParams{}
			if err := applyFromFile(params); err != nil { return err }
			if cmd.Flags().Changed("diff") { params.GitDiff = diff }
			if cmd.Flags().Changed("sha") { params.FoundCommitSHA = sha }
			if cmd.Flags().Changed("branch") { params.Branch = branch }
			
			if params.GitDiff == "" { return fmt.Errorf("required field 'diff' not set") }
			if params.FoundCommitSHA == "" { return fmt.Errorf("required field 'sha' not set") }
			if params.Branch == "" { return fmt.Errorf("required field 'branch' not set") }
			
			mut, err := c.Mutations.Create(ctx(), propID, params)
			if err != nil { return err }
			fmt.Printf("Created mutation %s\n", fmtInt64Ptr(mut.ID))
			return nil
		},
	}
	cmd.Flags().StringVar(&diff, "diff", "", "Git diff (required)")
	cmd.Flags().StringVar(&sha, "sha", "", "Commit SHA (required)")
	cmd.Flags().StringVar(&branch, "branch", "", "Branch (required)")
	return cmd
}

func mutUpdateCmd() *cobra.Command {
	var status string
	cmd := &cobra.Command{
		Use: "update <prop-id> <id>", Short: "Update mutation status", Args: cobra.ExactArgs(2),
		Long: "Update the status of a mutation.\n\nFLAGS:\n  --status   New status (cured, killed, etc.)\n\nEXAMPLES:\n  fibe mutations update 7 123 --status cured",
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			propID, _ := strconv.ParseInt(args[0], 10, 64)
			id, _ := strconv.ParseInt(args[1], 10, 64)
			params := &fibe.MutationUpdateParams{}
			if err := applyFromFile(params); err != nil { return err }
			if cmd.Flags().Changed("status") { params.Status = &status }
			
			if params.Status == nil || *params.Status == "" {
				return fmt.Errorf("required field 'status' not set")
			}
			
			_, err := c.Mutations.Update(ctx(), propID, id, params)
			if err != nil { return err }
			fmt.Printf("Updated mutation %d — status: %s\n", id, status)
			return nil
		},
	}
	cmd.Flags().StringVar(&status, "status", "", "New status (required)")
	return cmd
}

