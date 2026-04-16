package main

import (
	"fmt"
	"strconv"

	"github.com/fibegg/sdk/fibe"
	"github.com/spf13/cobra"
)
func hunksCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "hunks",
		Short: "Manage code hunks for a prop",
		Long: `Manage hunks — individual diff chunks from repository commits.

Hunks are atomic units of code change used for code review workflows.
Each hunk tracks file, change type, author, and processing status.

SUBCOMMANDS:
  list <prop-id>          List hunks
  get <prop-id> <id>      Show hunk with diff
  update <prop-id> <id>   Update hunk status
  ingest <prop-id>        Schedule hunk ingestion
  next <prop-id>          Get next unprocessed hunk`,
	}
	cmd.AddCommand(hunkListCmd(), hunkGetCmd(), hunkUpdateCmd(), hunkIngestCmd(), hunkNextCmd())
	return cmd
}

func hunkListCmd() *cobra.Command {
	var filePath, changeType, authorEmail, authorName, commitSHA, status, processor string
	var committedAfter, committedBefore, createdAfter, createdBefore, sort string
	cmd := &cobra.Command{
		Use: "list <prop-id>", Short: "List hunks", Args: cobra.ExactArgs(1),
		Long: `List hunks for a prop.

FILTERS:
  --file-path           Filter by file path (exact match)
  --change-type         Filter by change type (e.g. modified, added, deleted)
  --author-email        Filter by author email (exact match)
  --author-name         Filter by author name (substring match)
  --commit-sha          Filter by commit SHA (exact match)
  --status              Filter by status (e.g. pending, processed, skipped)
  --processor           Filter by processor name (exact match)

DATE RANGE:
  --committed-after     Show hunks committed on or after this date (ISO 8601)
  --committed-before    Show hunks committed on or before this date (ISO 8601)
  --created-after       Show hunks created on or after this date (ISO 8601)
  --created-before      Show hunks created on or before this date (ISO 8601)

SORTING:
  --sort                Sort results. Format: {column}_{direction}
                        Columns: ordinal, committed_at
                        Direction: asc, desc
                        Default: ordinal_desc

EXAMPLES:
  fibe hunks list 7
  fibe hunks list 7 --status pending --sort committed_at_desc
  fibe hunks list 7 --author-email dev@example.com --file-path src/main.go -o json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			propID, _ := strconv.ParseInt(args[0], 10, 64)
			params := &fibe.HunkListParams{}
			if filePath != "" { params.FilePath = filePath }
			if changeType != "" { params.ChangeType = changeType }
			if authorEmail != "" { params.AuthorEmail = authorEmail }
			if authorName != "" { params.AuthorName = authorName }
			if commitSHA != "" { params.CommitSHA = commitSHA }
			if status != "" { params.Status = status }
			if processor != "" { params.ProcessorName = processor }
			if committedAfter != "" { params.CommittedAfter = committedAfter }
			if committedBefore != "" { params.CommittedBefore = committedBefore }
			if createdAfter != "" { params.CreatedAfter = createdAfter }
			if createdBefore != "" { params.CreatedBefore = createdBefore }
			if sort != "" { params.Sort = sort }
			if flagPage > 0 { params.Page = flagPage }
			if flagPerPage > 0 { params.PerPage = flagPerPage }
			hunks, err := c.Hunks.List(ctx(), propID, params)
			if err != nil { return err }
			outputJSON(hunks)
			return nil
		},
	}
	cmd.Flags().StringVar(&filePath, "file-path", "", "Filter by file path")
	cmd.Flags().StringVar(&changeType, "change-type", "", "Filter by change type")
	cmd.Flags().StringVar(&authorEmail, "author-email", "", "Filter by author email")
	cmd.Flags().StringVar(&authorName, "author-name", "", "Filter by author name (substring)")
	cmd.Flags().StringVar(&commitSHA, "commit-sha", "", "Filter by commit SHA")
	cmd.Flags().StringVar(&status, "status", "", "Filter by status (pending, processed, skipped)")
	cmd.Flags().StringVar(&processor, "processor", "", "Filter by processor name")
	cmd.Flags().StringVar(&committedAfter, "committed-after", "", "Filter: committed after date (ISO 8601)")
	cmd.Flags().StringVar(&committedBefore, "committed-before", "", "Filter: committed before date (ISO 8601)")
	cmd.Flags().StringVar(&createdAfter, "created-after", "", "Filter: created after date (ISO 8601)")
	cmd.Flags().StringVar(&createdBefore, "created-before", "", "Filter: created before date (ISO 8601)")
	cmd.Flags().StringVar(&sort, "sort", "", "Sort order (e.g. ordinal_desc, committed_at_asc)")
	return cmd
}

func hunkGetCmd() *cobra.Command {
	return &cobra.Command{
		Use: "get <prop-id> <id>", Short: "Show hunk with diff content", Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			propID, _ := strconv.ParseInt(args[0], 10, 64)
			id, _ := strconv.ParseInt(args[1], 10, 64)
			hunk, err := c.Hunks.Get(ctx(), propID, id)
			if err != nil { return err }
			outputJSON(hunk)
			return nil
		},
	}
}

func hunkUpdateCmd() *cobra.Command {
	var status, processor string
	cmd := &cobra.Command{
		Use: "update <prop-id> <id>", Short: "Update hunk status", Args: cobra.ExactArgs(2),
		Long: "Mark a hunk as processed or skipped.\n\nFLAGS:\n  --status      New status (processed, skipped)\n  --processor   Processor name\n\nEXAMPLES:\n  fibe hunks update 7 100 --status processed --processor my-processor",
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			propID, _ := strconv.ParseInt(args[0], 10, 64)
			id, _ := strconv.ParseInt(args[1], 10, 64)
			params := &fibe.HunkUpdateParams{}
			if err := applyFromFile(params); err != nil { return err }
			if cmd.Flags().Changed("status") { params.Status = &status }
			if cmd.Flags().Changed("processor") { params.ProcessorName = &processor }
			_, err := c.Hunks.Update(ctx(), propID, id, params)
			if err != nil { return err }
			fmt.Printf("Updated hunk %d\n", id)
			return nil
		},
	}
	cmd.Flags().StringVar(&status, "status", "", "New status")
	cmd.Flags().StringVar(&processor, "processor", "", "Processor name")
	return cmd
}

func hunkIngestCmd() *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use: "ingest <prop-id>", Short: "Schedule hunk ingestion", Args: cobra.ExactArgs(1),
		Long: "Schedule background processing to ingest hunks from new commits.\n\nFLAGS:\n  --force   Force re-ingestion of all commits\n\nEXAMPLES:\n  fibe hunks ingest 7\n  fibe hunks ingest 7 --force",
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			propID, _ := strconv.ParseInt(args[0], 10, 64)
			must(c.Hunks.Ingest(ctx(), propID, force))
			fmt.Println("Hunk ingestion scheduled")
			return nil
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "Force re-ingestion")
	return cmd
}

func hunkNextCmd() *cobra.Command {
	var processor string
	cmd := &cobra.Command{
		Use: "next <prop-id>", Short: "Get next unprocessed hunk", Args: cobra.ExactArgs(1),
		Long: "Get the next pending hunk for processing.\n\nREQUIRED FLAGS:\n  --processor   Your processor name\n\nEXAMPLES:\n  fibe hunks next 7 --processor code-review-bot",
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			propID, _ := strconv.ParseInt(args[0], 10, 64)
			hunk, err := c.Hunks.Next(ctx(), propID, processor)
			if err != nil { return err }
			outputJSON(hunk)
			return nil
		},
	}
	cmd.Flags().StringVar(&processor, "processor", "", "Processor name (required)")
	cmd.MarkFlagRequired("processor")
	return cmd
}

