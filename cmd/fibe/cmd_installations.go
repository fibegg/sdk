package main

import (
	"fmt"
	"strconv"

	"github.com/fibegg/sdk/fibe"
	"github.com/spf13/cobra"
)

func installationsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "installations",
		Aliases: []string{"inst"},
		Short:   "Manage GitHub App installations and tokens",
		Long: `Manage GitHub App installations linked to the authenticated player.

An installation is the link between a Fibe player and a GitHub account/org
where the Fibe GitHub App has been installed. Each installation grants
access to a set of repositories.

This is the canonical way to obtain GitHub installation tokens — the
older 'fibe agents github-token' is a backwards-compatibility alias.

SUBCOMMANDS:
  list                List your installations
  repos <id>          List repositories accessible through an installation
  token <id>          Get an installation access token`,
	}
	cmd.AddCommand(instListCmd(), instReposCmd(), instTokenCmd())
	return cmd
}

func instListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List GitHub App installations",
		Long: `List all GitHub App installations linked to the authenticated player.

OUTPUT:
  Columns: ID, PROVIDER, INSTALLATION_ID, ACCOUNT, CREATED
  Use --output json for full details.

EXAMPLES:
  fibe installations list
  fibe inst list -o json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			result, err := c.Installations.List(ctx())
			if err != nil {
				return err
			}
			if effectiveOutput() != "table" {
				outputJSON(result)
				return nil
			}
			headers := []string{"ID", "PROVIDER", "INSTALLATION_ID", "ACCOUNT", "CREATED"}
			rows := make([][]string, len(result.Data))
			for i, inst := range result.Data {
				account := ""
				if inst.InstallationAccount != nil {
					account = *inst.InstallationAccount
				}
				rows[i] = []string{
					fmtInt64(inst.ID),
					inst.Provider,
					fmtInt64(inst.InstallationID),
					account,
					inst.CreatedAt,
				}
			}
			outputTable(headers, rows)
			return nil
		},
	}
}

func instReposCmd() *cobra.Command {
	var query string
	cmd := &cobra.Command{
		Use:   "repos <id>",
		Short: "List repos accessible through an installation",
		Long: `List repositories accessible through a GitHub App installation.

When --query is provided, performs a GitHub repo search scoped to the
installation. Otherwise lists all repos that the installation has access to.

FILTERS:
  -q, --query           Search query (repo name substring)

PAGINATION:
  --page                Page number (default: 1)
  --per-page            Items per page (default: 30, max: 100)

OUTPUT:
  Columns: ID, FULL_NAME, PRIVATE, DEFAULT_BRANCH
  Use --output json for full details (URLs, description, etc).

EXAMPLES:
  fibe installations repos 12
  fibe installations repos 12 -q myproject
  fibe inst repos 12 --per-page 50 -o json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			id, _ := strconv.ParseInt(args[0], 10, 64)
			params := &fibe.InstallationReposParams{}
			if query != "" {
				params.Q = query
			}
			if flagPage > 0 {
				params.Page = flagPage
			}
			if flagPerPage > 0 {
				params.PerPage = flagPerPage
			}
			result, err := c.Installations.Repos(ctx(), id, params)
			if err != nil {
				return err
			}
			if effectiveOutput() != "table" {
				outputJSON(result)
				return nil
			}
			headers := []string{"ID", "FULL_NAME", "PRIVATE", "DEFAULT_BRANCH"}
			rows := make([][]string, len(result.Data))
			for i, r := range result.Data {
				rows[i] = []string{
					fmtInt64(r.ID),
					r.FullName,
					fmt.Sprintf("%t", r.Private),
					r.DefaultBranch,
				}
			}
			outputTable(headers, rows)
			return nil
		},
	}
	cmd.Flags().StringVarP(&query, "query", "q", "", "Search query (repo name substring)")
	return cmd
}

func instTokenCmd() *cobra.Command {
	var repo string
	cmd := &cobra.Command{
		Use:   "token <id>",
		Short: "Get an installation access token",
		Long: `Get a fresh installation access token.

When --repo is provided (in owner/name format), a repository-scoped token
is returned (the installation must have access to that repo). Otherwise
returns an installation-wide token.

Tokens are short-lived (typically 1 hour). Cache the value carefully.

OPTIONAL FLAGS:
  --repo    Scope token to a specific repository (owner/name format)

EXAMPLES:
  fibe installations token 12
  fibe installations token 12 --repo myorg/myrepo
  fibe inst token 12 -o json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			id, _ := strconv.ParseInt(args[0], 10, 64)
			token, err := c.Installations.Token(ctx(), id, repo)
			if err != nil {
				return err
			}
			if effectiveOutput() != "table" {
				outputJSON(token)
				return nil
			}
			fmt.Printf("Token:      %s\n", token.Token)
			fmt.Printf("Expires in: %d seconds\n", token.ExpiresIn)
			return nil
		},
	}
	cmd.Flags().StringVar(&repo, "repo", "", "Scope token to a specific repository (owner/name)")
	return cmd
}
