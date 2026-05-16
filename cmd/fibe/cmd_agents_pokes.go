package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/fibegg/sdk/fibe"
	"github.com/spf13/cobra"
)

func agPokesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "pokes",
		Aliases: []string{"poke"},
		Short:   "Manage scheduled agent pokes",
	}
	cmd.AddCommand(
		agPokesListCmd(),
		agPokesGetCmd(),
		agPokesCreateCmd(),
		agPokesUpdateCmd(),
		agPokesDeleteCmd(),
	)
	return cmd
}

func agPokesListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list <agent>",
		Short: "List scheduled pokes for an agent",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := newClient().Agents.ListPokesByIdentifier(ctx(), args[0], &fibe.AgentPokeListParams{Page: flagPage, PerPage: flagPerPage})
			if err != nil {
				return err
			}
			if effectiveOutput() != "table" {
				outputJSON(result)
				return nil
			}
			rows := make([][]string, len(result.Data))
			for i, poke := range result.Data {
				rows[i] = []string{
					fmtInt64(poke.ID),
					fmtBool(poke.Enabled),
					poke.Schedule,
					fmtStringPtr(poke.ConversationID),
					fmtTime(poke.NextRunAt),
					fmtStringPtr(poke.LastStatus),
					strconv.FormatInt(poke.SentCount, 10),
				}
			}
			outputTable([]string{"ID", "ENABLED", "SCHEDULE", "CONVERSATION", "NEXT_RUN", "LAST", "SENT"}, rows)
			return nil
		},
	}
}

func agPokesGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <agent> <poke-id>",
		Short: "Show a scheduled agent poke",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			pokeID, err := parsePokeID(args[1])
			if err != nil {
				return err
			}
			poke, err := newClient().Agents.GetPokeByIdentifier(ctx(), args[0], pokeID)
			if err != nil {
				return err
			}
			if effectiveOutput() != "table" {
				outputJSON(poke)
				return nil
			}
			fmt.Printf("ID:           %d\n", poke.ID)
			fmt.Printf("Agent:        %d\n", poke.AgentID)
			fmt.Printf("Enabled:      %s\n", fmtBool(poke.Enabled))
			fmt.Printf("Schedule:     %s\n", poke.Schedule)
			fmt.Printf("Conversation: %s\n", fmtStringPtr(poke.ConversationID))
			fmt.Printf("Next run:     %s\n", fmtTime(poke.NextRunAt))
			fmt.Printf("Last run:     %s\n", fmtTime(poke.LastRunAt))
			fmt.Printf("Last status:  %s\n", fmtStringPtr(poke.LastStatus))
			fmt.Printf("Sent count:   %d\n", poke.SentCount)
			fmt.Printf("Prompt:\n%s\n", poke.Prompt)
			return nil
		},
	}
}

func agPokesCreateCmd() *cobra.Command {
	var schedule, prompt, conversationID string
	var disabled bool

	cmd := &cobra.Command{
		Use:   "create <agent>",
		Short: "Create a scheduled agent poke",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			params := &fibe.AgentPokeCreateParams{}
			if err := applyFromFile(params); err != nil {
				return err
			}
			if cmd.Flags().Changed("schedule") {
				params.Schedule = schedule
			}
			if cmd.Flags().Changed("prompt") {
				params.Prompt = prompt
			}
			if cmd.Flags().Changed("conversation-id") {
				params.ConversationID = conversationID
			}
			if disabled {
				enabled := false
				params.Enabled = &enabled
			}
			poke, err := newClient().Agents.CreatePokeByIdentifier(ctx(), args[0], params)
			if err != nil {
				return err
			}
			if effectiveOutput() != "table" {
				outputJSON(poke)
				return nil
			}
			fmt.Printf("Created poke %d for agent %s\n", poke.ID, args[0])
			return nil
		},
	}
	cmd.Flags().StringVar(&schedule, "schedule", "", "Cron-like schedule, for example '*/5 * * * *'")
	cmd.Flags().StringVar(&prompt, "prompt", "", "Prompt to send on schedule")
	cmd.Flags().StringVar(&conversationID, "conversation-id", "", "Specific conversation ID to target")
	cmd.Flags().BoolVar(&disabled, "disabled", false, "Create the poke disabled")
	return cmd
}

func agPokesUpdateCmd() *cobra.Command {
	var schedule, prompt, conversationID string
	var clearConversation, enabled, disabled bool

	cmd := &cobra.Command{
		Use:   "update <agent> <poke-id>",
		Short: "Update a scheduled agent poke",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if enabled && disabled {
				return fmt.Errorf("--enabled and --disabled are mutually exclusive")
			}
			if clearConversation && cmd.Flags().Changed("conversation-id") {
				return fmt.Errorf("--clear-conversation and --conversation-id are mutually exclusive")
			}
			pokeID, err := parsePokeID(args[1])
			if err != nil {
				return err
			}
			params := &fibe.AgentPokeUpdateParams{}
			if err := applyFromFile(params); err != nil {
				return err
			}
			if cmd.Flags().Changed("schedule") {
				params.Schedule = &schedule
			}
			if cmd.Flags().Changed("prompt") {
				params.Prompt = &prompt
			}
			if cmd.Flags().Changed("conversation-id") {
				params.ConversationID = &conversationID
			}
			if clearConversation {
				blank := ""
				params.ConversationID = &blank
			}
			if enabled || disabled {
				value := enabled
				params.Enabled = &value
			}
			poke, err := newClient().Agents.UpdatePokeByIdentifier(ctx(), args[0], pokeID, params)
			if err != nil {
				return err
			}
			if effectiveOutput() != "table" {
				outputJSON(poke)
				return nil
			}
			fmt.Printf("Updated poke %d for agent %s\n", poke.ID, args[0])
			return nil
		},
	}
	cmd.Flags().StringVar(&schedule, "schedule", "", "Cron-like schedule")
	cmd.Flags().StringVar(&prompt, "prompt", "", "Prompt to send on schedule")
	cmd.Flags().StringVar(&conversationID, "conversation-id", "", "Specific conversation ID to target")
	cmd.Flags().BoolVar(&clearConversation, "clear-conversation", false, "Clear the target conversation")
	cmd.Flags().BoolVar(&enabled, "enabled", false, "Enable the poke")
	cmd.Flags().BoolVar(&disabled, "disabled", false, "Disable the poke")
	return cmd
}

func agPokesDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <agent> <poke-id>",
		Short: "Delete a scheduled agent poke",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			pokeID, err := parsePokeID(args[1])
			if err != nil {
				return err
			}
			if err := newClient().Agents.DeletePokeByIdentifier(ctx(), args[0], pokeID); err != nil {
				return err
			}
			if effectiveOutput() != "table" {
				outputJSON(map[string]any{"deleted": true, "agent": args[0], "id": pokeID})
				return nil
			}
			fmt.Printf("Deleted poke %d for agent %s\n", pokeID, args[0])
			return nil
		},
	}
}

func parsePokeID(raw string) (int64, error) {
	id, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("poke-id must be a positive integer")
	}
	return id, nil
}

func fmtStringPtr(value *string) string {
	if value == nil || *value == "" {
		return "-"
	}
	return *value
}
