package main

import (
	"fmt"
	"strconv"

	"github.com/fibegg/sdk/fibe"
	"github.com/spf13/cobra"
)

func feedbacksCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "feedbacks",
		Short: "Manage agent feedback",
		Long: `Manage feedback attached to agent interactions.

Feedback captures user annotations on agent output: text selections,
comments, and contextual information.

SUBCOMMANDS:
  list <agent-id>          List feedbacks
  get <agent-id> <id>      Show feedback details
  create <agent-id>        Create feedback
  delete <agent-id> <id>   Delete feedback`,
	}
	cmd.AddCommand(fbListCmd(), fbGetCmd(), fbCreateCmd(), fbDeleteCmd())
	return cmd
}

func fbListCmd() *cobra.Command {
	var query, sourceType, sourceID, playgroundID, createdAfter, createdBefore, sort string
	cmd := &cobra.Command{
		Use: "list <agent-id>", Short: "List agent feedbacks", Args: cobra.ExactArgs(1),
		Long: `List feedbacks for an agent.

FILTERS:
  -q, --query           Search across comment, selected_text, context (substring match)
  --source-type         Filter by source type (exact match)
  --source-id           Filter by source ID (exact match)
  --playground-id       Filter by playground ID

DATE RANGE:
  --created-after       Show items created on or after this date (ISO 8601)
  --created-before      Show items created on or before this date (ISO 8601)

SORTING:
  --sort                Sort results. Format: {column}_{direction}
                        Columns: created_at
                        Direction: asc, desc
                        Default: created_at_desc

EXAMPLES:
  fibe feedbacks list 5
  fibe feedbacks list 5 -q "bug" --source-type messages
  fibe feedbacks list 5 --playground-id 42 -o json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			agentID, _ := strconv.ParseInt(args[0], 10, 64)
			params := &fibe.FeedbackListParams{}
			if query != "" {
				params.Query = query
			}
			if sourceType != "" {
				params.SourceType = sourceType
			}
			if sourceID != "" {
				params.SourceID = sourceID
			}
			if playgroundID != "" {
				params.PlaygroundID = playgroundID
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
			fbs, err := c.Feedbacks.List(ctx(), agentID, params)
			if err != nil {
				return err
			}
			outputJSON(fbs)
			return nil
		},
	}
	cmd.Flags().StringVarP(&query, "query", "q", "", "Search across comment, selected_text, context")
	cmd.Flags().StringVar(&sourceType, "source-type", "", "Filter by source type")
	cmd.Flags().StringVar(&sourceID, "source-id", "", "Filter by source ID")
	cmd.Flags().StringVar(&playgroundID, "playground-id", "", "Filter by playground ID")
	cmd.Flags().StringVar(&createdAfter, "created-after", "", "Filter: created after date (ISO 8601)")
	cmd.Flags().StringVar(&createdBefore, "created-before", "", "Filter: created before date (ISO 8601)")
	cmd.Flags().StringVar(&sort, "sort", "", "Sort order (e.g. created_at_desc)")
	return cmd
}

func fbGetCmd() *cobra.Command {
	return &cobra.Command{
		Use: "get <agent-id> <id>", Short: "Show feedback", Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			agentID, _ := strconv.ParseInt(args[0], 10, 64)
			id, _ := strconv.ParseInt(args[1], 10, 64)
			fb, err := c.Feedbacks.Get(ctx(), agentID, id)
			if err != nil {
				return err
			}
			outputJSON(fb)
			return nil
		},
	}
}

func fbCreateCmd() *cobra.Command {
	var sourceType, comment string
	cmd := &cobra.Command{
		Use: "create <agent-id>", Short: "Create feedback", Args: cobra.ExactArgs(1),
		Long: "Create new feedback for an agent's completion capability.\n\nFEEDBACK CONSTRAINTS:\n  - The rating must be a strict integer grading the response (typically 1 to 5 points).\n  - Do not hallucinate scores. Ask the human explicitly for a numeric rating.\n\nREQUIRED FLAGS:\n  --source-type  Feedback source\n  --comment      Feedback explanation text",
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			agentID, _ := strconv.ParseInt(args[0], 10, 64)
			params := &fibe.FeedbackCreateParams{}
			if err := applyFromFile(params); err != nil {
				return err
			}
			if cmd.Flags().Changed("source-type") {
				params.SourceType = sourceType
			}
			if cmd.Flags().Changed("comment") {
				params.Comment = &comment
			}

			if params.SourceType == "" {
				return fmt.Errorf("required field 'source-type' not set")
			}

			fb, err := c.Feedbacks.Create(ctx(), agentID, params)
			if err != nil {
				return err
			}
			fmt.Printf("Created feedback %s\n", fmtInt64Ptr(fb.ID))
			return nil
		},
	}
	cmd.Flags().StringVar(&sourceType, "source-type", "", "Source type (required)")
	cmd.Flags().StringVar(&comment, "comment", "", "Comment")
	return cmd
}

func fbDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use: "delete <agent-id> <id>", Short: "Delete feedback", Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			agentID, _ := strconv.ParseInt(args[0], 10, 64)
			id, _ := strconv.ParseInt(args[1], 10, 64)
			if err := c.Feedbacks.Delete(ctx(), agentID, id); err != nil {
				return err
			}
			fmt.Printf("Feedback %d deleted\n", id)
			return nil
		},
	}
}
