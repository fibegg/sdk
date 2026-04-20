package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

func serverInfoCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "server-info",
		Short: "Show Fibe server UTC clock and build identity",
		Long: `Show the Fibe server's current UTC time plus the build identity
(build time and git commit SHA) baked into the server image.

Hits the unauthenticated /up endpoint — works without an API key.
Useful for clock-drift checks and identifying which server build
you're talking to.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			info, err := c.ServerInfo.Get(ctx())
			if err != nil {
				return err
			}

			switch effectiveOutput() {
			case "table":
				fmt.Printf("Domain:            %s\n", orDash(info.Domain))
				fmt.Printf("Server time (UTC): %s\n", info.TimeUTC)
				fmt.Printf("Build time:        %s\n", orDash(info.BuildTime))
				fmt.Printf("Git commit SHA:    %s\n", orDash(info.GitCommitSHA))
			default:
				output(info)
			}
			return nil
		},
	}
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}
