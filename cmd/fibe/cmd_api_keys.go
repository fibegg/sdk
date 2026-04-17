package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/fibegg/sdk/fibe"
	"github.com/spf13/cobra"
)

func apiKeysCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "api-keys",
		Aliases: []string{"keys"},
		Short:   "Manage API keys",
		Long: `Manage Fibe API keys for programmatic access.

API keys support granular scopes, expiration dates, and per-key rate limits.
The token is only shown once at creation time.

AVAILABLE SCOPES:
  marquees:read, marquees:write, marquees:delete, marquees:manage
  props:read, props:write, props:delete
  playspecs:read, playspecs:write, playspecs:delete
  playgrounds:read, playgrounds:write, playgrounds:delete
  import_templates:read, import_templates:write
  agents:read, agents:write, agents:delete
  artefacts:read, artefacts:write, artefacts:delete
  mutters:read, mutters:write
  feedbacks:read, feedbacks:write, feedbacks:delete
  mutations:read, mutations:write
  hunks:read, hunks:write
  launch:write
  keys:manage
  webhooks:read, webhooks:write, webhooks:delete
  teams:read, teams:write, teams:delete
  mcp:access
  secrets:read, secrets:write, secrets:delete, secrets:manage
  * (full access)

SUBCOMMANDS:
  list         List all API keys
  create       Create a new API key
  delete <id>  Delete an API key`,
	}
	cmd.AddCommand(keyListCmd(), keyCreateCmd(), keyDeleteCmd())
	return cmd
}

func keyListCmd() *cobra.Command {
	var query, label, sort string
	cmd := &cobra.Command{
		Use: "list", Short: "List all API keys",
		Long: `List all API keys for the authenticated user. Token values are masked.

FILTERS:
  -q, --query           Search across label (substring match)
  --label               Filter by label (substring match)

SORTING:
  --sort                Sort results. Format: {column}_{direction}
                        Columns: created_at
                        Direction: asc, desc
                        Default: created_at_desc

OUTPUT:
  Columns: ID, LABEL, MASKED_TOKEN, EXPIRES, CREATED
  Use --output json for full details.

EXAMPLES:
  fibe api-keys list
  fibe keys list -q "CI"
  fibe keys list --label production --sort created_at_asc`,
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			params := &fibe.APIKeyListParams{}
			if query != "" {
				params.Q = query
			}
			if label != "" {
				params.Label = label
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
			keys, err := c.APIKeys.List(ctx(), params)
			if err != nil {
				return err
			}
			if effectiveOutput() != "table" {
				outputJSON(keys)
				return nil
			}
			headers := []string{"ID", "LABEL", "MASKED_TOKEN", "EXPIRES", "CREATED"}
			rows := make([][]string, len(keys.Data))
			for i, k := range keys.Data {
				rows[i] = []string{fmtInt64Ptr(k.ID), k.Label, k.MaskedToken, fmtTime(k.ExpiresAt), fmtTime(k.CreatedAt)}
			}
			outputTable(headers, rows)
			return nil
		},
	}
	cmd.Flags().StringVarP(&query, "query", "q", "", "Search across label")
	cmd.Flags().StringVar(&label, "label", "", "Filter by label (substring)")
	cmd.Flags().StringVar(&sort, "sort", "", "Sort order (e.g. created_at_desc)")
	return cmd
}

func keyCreateCmd() *cobra.Command {
	var label, expiresAt string
	var scopes, granularScopes []string
	var agentAccessible bool
	cmd := &cobra.Command{
		Use: "create", Short: "Create a new API key",
		Long: `Create a new API key. The full token is only shown ONCE — save it.

AUTH USAGE:
  Send as 'Authorization: Bearer fibe_...' header.

REQUIRED FLAGS:
  --label                Key label/description

OPTIONAL FLAGS:
  --scope                Access scopes (repeatable, default: *). Format: resource:action,
                         e.g. playgrounds:read, agents:write, mcp:access
  --granular-scope       Granular scope binding to specific IDs (repeatable).
                         Format: scope_name=id1,id2,...
                         e.g. --granular-scope playspecs:read=12,15
  --expires-at           Expiration time (RFC3339, e.g. 2026-12-31T23:59:59Z)
  --agent-accessible     Allow this key to be used by agents (default: false)

EXAMPLES:
  fibe api-keys create --label "CI/CD"
  fibe keys create --label "Read only" --scope playgrounds:read --scope agents:read
  fibe keys create --label "Bot key" --agent-accessible --expires-at 2026-12-31T23:59:59Z
  fibe keys create --label "Scoped" --granular-scope playspecs:read=12,15` + generateSchemaDoc(&fibe.APIKeyCreateParams{}),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			params := &fibe.APIKeyCreateParams{}
			if err := applyFromFile(params); err != nil {
				return err
			}
			if cmd.Flags().Changed("label") {
				params.Label = label
			}
			if len(scopes) > 0 {
				params.Scopes = scopes
			}
			if cmd.Flags().Changed("agent-accessible") {
				params.AgentAccessible = &agentAccessible
			}
			if cmd.Flags().Changed("expires-at") {
				t, err := time.Parse(time.RFC3339, expiresAt)
				if err != nil {
					return fmt.Errorf("invalid --expires-at (must be RFC3339): %w", err)
				}
				params.ExpiresAt = &t
			}
			if len(granularScopes) > 0 {
				if params.GranularScopes == nil {
					params.GranularScopes = map[string][]int64{}
				}
				for _, gs := range granularScopes {
					eq := strings.Index(gs, "=")
					if eq < 1 || eq == len(gs)-1 {
						return fmt.Errorf("invalid --granular-scope %q: expected scope=id1,id2,...", gs)
					}
					name := gs[:eq]
					ids := []int64{}
					for _, idStr := range strings.Split(gs[eq+1:], ",") {
						id, err := strconv.ParseInt(strings.TrimSpace(idStr), 10, 64)
						if err != nil {
							return fmt.Errorf("invalid id %q in --granular-scope %q: %w", idStr, gs, err)
						}
						ids = append(ids, id)
					}
					params.GranularScopes[name] = ids
				}
			}

			if params.Label == "" {
				return fmt.Errorf("required field 'label' not set")
			}

			key, err := c.APIKeys.Create(ctx(), params)
			if err != nil {
				return err
			}
			if effectiveOutput() != "table" {
				outputJSON(key)
				return nil
			}
			fmt.Printf("Created API key %s (%s)\n", fmtInt64Ptr(key.ID), key.Label)
			if key.Token != nil {
				fmt.Printf("\nToken: %s\n", *key.Token)
				fmt.Println("\nSave this token — it will not be shown again!")
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&label, "label", "", "Key label (required)")
	cmd.Flags().StringSliceVar(&scopes, "scope", nil, "Access scope (repeatable)")
	cmd.Flags().StringSliceVar(&granularScopes, "granular-scope", nil, "Granular scope (repeatable, format: scope=id1,id2)")
	cmd.Flags().StringVar(&expiresAt, "expires-at", "", "Expiration time (RFC3339)")
	cmd.Flags().BoolVar(&agentAccessible, "agent-accessible", false, "Allow agent access")
	return cmd
}

func keyDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use: "delete <id>", Short: "Delete an API key", Args: cobra.ExactArgs(1),
		Long: "Delete an API key. The key will immediately stop working.\n\nEXAMPLES:\n  fibe api-keys delete 15",
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			id, _ := strconv.ParseInt(args[0], 10, 64)
			if err := c.APIKeys.Delete(ctx(), id); err != nil {
				return err
			}
			fmt.Printf("API key %d deleted\n", id)
			return nil
		},
	}
}
