package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

func statusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show account status dashboard",
		Long: `Show a summary of all your resources in a single request.

Returns counts for playgrounds (total/active/stopped), agents,
props, playspecs, marquees, secrets, API keys, and subscription info.

Designed for LLM agents to gather full context efficiently:
  fibe status -o yaml --only playgrounds,agents`,
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			status, err := c.Status.Get(ctx())
			if err != nil {
				return err
			}

			switch effectiveOutput() {
			case "table":
				fmt.Println("=== Account Status ===")
				fmt.Printf("Playgrounds:  %d total, %d active, %d stopped\n",
					status.Playgrounds.Total, status.Playgrounds.Active, status.Playgrounds.Stopped)
				fmt.Printf("Agents:       %d total, %d authenticated\n",
					status.Agents.Total, status.Agents.Authenticated)
				fmt.Printf("Props:        %d\n", status.Props)
				fmt.Printf("Playspecs:    %d\n", status.Playspecs)
				fmt.Printf("Marquees:     %d\n", status.Marquees)
				fmt.Printf("Secrets:      %d\n", status.Secrets)
				fmt.Printf("API Keys:     %d\n", status.APIKeys)
				fmt.Printf("Plan:         %s (playground limit: %d)\n",
					status.Subscription.Plan, status.Subscription.PlaygroundLimit)
			default:
				output(status)
			}
			return nil
		},
	}
}
