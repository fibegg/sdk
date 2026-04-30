package main

import "github.com/spf13/cobra"

func localCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "local",
		Short: "Explore local Fibe-related resources on this machine",
		Long: `Explore local Fibe-related resources without calling the Fibe API.

Examples:
  fibe local playgrounds list
  fibe local conversations list
  fibe local conversations get <uuid> --chat`,
	}

	cmd.AddCommand(
		localPlaygroundsCmd(),
		localConversationsCmd(),
	)

	return cmd
}
