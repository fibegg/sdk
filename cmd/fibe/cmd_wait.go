package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/fibegg/sdk/fibe"
	"github.com/spf13/cobra"
)

func waitCmd() *cobra.Command {
	var (
		targetStatus string
		timeout      time.Duration
		interval     time.Duration
	)

	cmd := &cobra.Command{
		Use:   "wait <resource> <id-or-name>",
		Short: "Wait for a resource to reach a target status",
		Long: `Poll a resource until it reaches the desired status.

Eliminates retry loops in LLM agent code — delegates
polling to the CLI with built-in timeout and interval.

Supported resources: playground, trick

Examples:
  fibe wait playground next --status running
  fibe wait trick nightly-build --status completed
  fibe wait playground 42 --status running --timeout 5m --interval 5s`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			resource := strings.ToLower(args[0])
			identifier := args[1]

			c := newClient()
			deadline := time.After(timeout)

			switch resource {
			case "playground", "pg":
				for {
					status, err := c.Playgrounds.StatusByIdentifier(ctx(), identifier)
					if err != nil {
						return err
					}

					current := status.Status

					fmt.Fprintf(cmd.OutOrStderr(), "status: %s\n", current)

					if current == targetStatus {
						pg, err := c.Playgrounds.GetByIdentifier(ctx(), identifier)
						if err != nil {
							return err
						}
						output(pg)
						return nil
					}

					// Terminal failure states
					if current == "error" || current == "failed" || current == "destroyed" {
						return fibe.NewPlaygroundTerminalStateError(status)
					}

					select {
					case <-deadline:
						return fmt.Errorf("timeout after %s — last status: %s", timeout, current)
					case <-time.After(interval):
					}
				}
			case "trick", "tr":
				for {
					pg, err := c.Tricks.GetByIdentifier(ctx(), identifier)
					if err != nil {
						return err
					}

					current := pg.Status

					fmt.Fprintf(cmd.OutOrStderr(), "status: %s\n", current)

					if current == targetStatus {
						output(pg)
						return nil
					}

					// Terminal states for tricks
					if current == "completed" && targetStatus != "completed" {
						return fmt.Errorf("trick reached terminal state: %s", current)
					}
					if current == "error" || current == "failed" || current == "destroyed" {
						return fmt.Errorf("trick reached terminal state: %s", current)
					}

					select {
					case <-deadline:
						return fmt.Errorf("timeout after %s — last status: %s", timeout, current)
					case <-time.After(interval):
					}
				}
			default:
				return fmt.Errorf("unsupported resource %q — supported: playground, trick", resource)
			}
		},
	}

	cmd.Flags().StringVar(&targetStatus, "status", "running", "Target status to wait for")
	cmd.Flags().DurationVar(&timeout, "timeout", 10*time.Minute, "Maximum time to wait")
	cmd.Flags().DurationVar(&interval, "interval", 3*time.Second, "Polling interval")

	return cmd
}
