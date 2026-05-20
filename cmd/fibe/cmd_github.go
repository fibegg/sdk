package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

func githubCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "github",
		Short: "Manage GitHub connections",
	}
	cmd.AddCommand(githubAppsCmd())
	return cmd
}

func githubAppsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "apps",
		Short: "Manage the Fibe GitHub App connection",
	}
	cmd.AddCommand(githubAppsConnectCmd())
	return cmd
}

func githubAppsConnectCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "connect",
		Short: "Print the GitHub App installation URL",
		Long: `Print the GitHub App installation URL for this Fibe server.

Open the URL in a browser, sign in to the same Fibe account if prompted,
and finish the GitHub App setup on GitHub.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			info, err := c.GitHubApps.ConnectInfo(ctx())
			if err != nil {
				return err
			}
			if effectiveOutput() != "table" {
				outputJSON(info)
				return nil
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Open this URL to connect GitHub:\n%s\n", info.InstallURL)
			fmt.Fprintln(cmd.OutOrStdout(), "Finish setup in the browser, then rerun your fibe command.")
			return nil
		},
	}
}
