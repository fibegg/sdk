package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func meCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "me",
		Short: "Show the authenticated user's profile",
		Long: `Display information about the currently authenticated user.

Returns the player ID, username, GitHub handle, email, avatar URL
and any API key scopes associated with the API key being used.

This is useful for verifying which account your API key belongs to.

EXAMPLES:
  fibe me
  fibe me --output json`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			player, err := c.APIKeys.Me(ctx())
			if err != nil {
				return err
			}
			if effectiveOutput() != "table" {
				outputJSON(player)
				return nil
			}
			fmt.Printf("ID:       %d\n", player.ID)
			fmt.Printf("Username: %s\n", player.Username)
			fmt.Printf("GitHub:   %s\n", fmtStr(player.GithubHandle))
			fmt.Printf("Email:    %s\n", fmtStr(player.Email))
			if len(player.APIKeyScopes) > 0 {
				fmt.Printf("API Key Scopes: %s\n", strings.Join(player.APIKeyScopes, ", "))
			}
			return nil
		},
	}
}
