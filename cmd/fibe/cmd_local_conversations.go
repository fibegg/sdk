package main

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/fibegg/sdk/internal/localconversations"
	"github.com/spf13/cobra"
)

func localConversationsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "conversations",
		Aliases: []string{"lc"},
		Short:   "Explore local AI conversations on this machine",
		Long: `Explore locally stored AI conversations without calling the Fibe API.

The first provider strategies cover Codex, Claude Code, and Claude Desktop
local agent-mode transcripts. Default locations are discovered from the current
user's home directory.

Environment Variables:
  FIBE_LOCAL_CONVERSATION_PATHS   Additional search roots, separated by the OS path-list separator

Examples:
  fibe local conversations list
  fibe local conversations list --provider codex
  fibe local conversations list --provider claude-code --path ~/.claude/projects -o json`,
	}

	cmd.AddCommand(lcListCmd())
	cmd.AddCommand(lcGetCmd())
	return cmd
}

func lcListCmd() *cobra.Command {
	var providers []string
	var paths []string
	var includeMetadataOnly bool
	var limit int

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List local Codex and Claude conversations",
		RunE: func(cmd *cobra.Command, args []string) error {
			conversations, err := localconversations.List(cmd.Context(), localconversations.ListOptions{
				Providers:           providers,
				Paths:               paths,
				IncludeMetadataOnly: includeMetadataOnly,
				Limit:               limit,
			})
			if err != nil {
				return err
			}

			switch effectiveOutput() {
			case "json", "yaml":
				output(conversations)
			default:
				columns := selectedLocalConversationColumns()
				headers := make([]string, 0, len(columns))
				for _, column := range columns {
					headers = append(headers, column.header)
				}
				rows := make([][]string, 0, len(conversations))
				for _, conversation := range conversations {
					row := make([]string, 0, len(columns))
					for _, column := range columns {
						row = append(row, column.value(conversation))
					}
					rows = append(rows, row)
				}
				outputTable(headers, rows)
			}
			return nil
		},
	}

	cmd.Flags().StringSliceVar(&providers, "provider", nil, "Provider filter (repeatable or comma-separated): codex, claude, claude-code, claude-desktop")
	cmd.Flags().StringArrayVar(&paths, "path", nil, "Additional file or directory to scan (repeatable)")
	cmd.Flags().BoolVar(&includeMetadataOnly, "include-metadata-only", false, "Include Claude Desktop metadata records that do not contain transcript text")
	cmd.Flags().IntVar(&limit, "limit", 0, "Maximum number of conversations to print (default: all)")
	return cmd
}

func lcGetCmd() *cobra.Command {
	var providers []string
	var paths []string
	var includeMetadataOnly bool
	var chat bool

	cmd := &cobra.Command{
		Use:   "get <uuid>",
		Short: "Show full local conversation data",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			conversation, err := localconversations.Get(cmd.Context(), args[0], localconversations.ListOptions{
				Providers:           providers,
				Paths:               paths,
				IncludeMetadataOnly: includeMetadataOnly,
			})
			if err != nil {
				return err
			}
			if chat {
				output(localconversations.ChatTranscript(conversation))
				return nil
			}
			output(conversation)
			return nil
		},
	}

	cmd.Flags().StringSliceVar(&providers, "provider", nil, "Provider filter (repeatable or comma-separated): codex, claude, claude-code, claude-desktop")
	cmd.Flags().StringArrayVar(&paths, "path", nil, "Additional file or directory to scan (repeatable)")
	cmd.Flags().BoolVar(&includeMetadataOnly, "include-metadata-only", false, "Include Claude Desktop metadata records that do not contain transcript text")
	cmd.Flags().BoolVar(&chat, "chat", false, "Output compact user/provider chat turns only")
	return cmd
}

type localConversationColumn struct {
	field  string
	header string
	value  func(localconversations.Conversation) string
}

func selectedLocalConversationColumns() []localConversationColumn {
	columns := []localConversationColumn{
		{field: "provider", header: "PROVIDER", value: func(c localconversations.Conversation) string { return c.Provider }},
		{field: "uuid", header: "UUID", value: func(c localconversations.Conversation) string { return c.UUID }},
		{field: "last_message_date", header: "LAST_MESSAGE", value: func(c localconversations.Conversation) string { return fmtTime(c.LastMessageDate) }},
		{field: "user_message_count", header: "USER_MSGS", value: func(c localconversations.Conversation) string { return strconv.Itoa(c.UserMessageCount) }},
		{field: "total_token_count", header: "TOKENS", value: func(c localconversations.Conversation) string { return fmtInt64(c.TotalTokenCount) }},
		{field: "first_user_message_sentence", header: "FIRST_USER_MESSAGE", value: func(c localconversations.Conversation) string {
			return truncateTableCell(c.FirstUserMessageSentence, 120)
		}},
		{field: "path", header: "PATH", value: func(c localconversations.Conversation) string { return c.Path }},
		{field: "metadata", header: "METADATA", value: func(c localconversations.Conversation) string {
			return truncateTableCell(compactJSON(c.Metadata), 120)
		}},
	}
	if len(flagOnly) == 0 {
		return columns[:7]
	}

	byField := make(map[string]localConversationColumn, len(columns))
	for _, column := range columns {
		byField[column.field] = column
	}

	var selected []localConversationColumn
	for _, raw := range flagOnly {
		for _, part := range strings.Split(raw, ",") {
			field := strings.TrimSpace(part)
			if field == "" {
				continue
			}
			if column, ok := byField[field]; ok {
				selected = append(selected, column)
			}
		}
	}
	return selected
}

func compactJSON(v any) string {
	if v == nil {
		return ""
	}
	data, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(data)
}

func truncateTableCell(value string, maxRunes int) string {
	if maxRunes <= 0 {
		return value
	}
	runes := []rune(value)
	if len(runes) <= maxRunes {
		return value
	}
	if maxRunes <= 3 {
		return string(runes[:maxRunes])
	}
	return string(runes[:maxRunes-3]) + "..."
}
