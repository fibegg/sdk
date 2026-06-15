package main

import (
	"errors"
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
		full         bool
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
  fibe wait playground 42 --status running --timeout 5m --interval 5s
  fibe wait trick nightly-build --status completed --full`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			resource := strings.ToLower(args[0])
			identifier := args[1]
			auth := resolveCLIAuth()

			c := newClient()
			progress := newStatusLine(cmd.ErrOrStderr(), statusLineOptions{FallbackUpdates: true})
			progress.Start(fmt.Sprintf("waiting for %s %s to reach %s", resource, identifier, targetStatus))
			defer progress.Stop()
			deadline := time.After(timeout)

			switch resource {
			case "playground", "pg":
				for {
					status, err := c.Playgrounds.StatusByIdentifier(ctx(), identifier)
					if err != nil {
						return waitResourceError(resource, identifier, auth, err)
					}

					current := status.Status

					progress.Update(fmt.Sprintf("status: %s", current))

					if current == targetStatus {
						pg, err := c.Playgrounds.GetByIdentifier(ctx(), identifier)
						if err != nil {
							return waitResourceError(resource, identifier, auth, err)
						}
						progress.Stop()
						if full {
							output(pg)
						} else {
							output(waitResultFromPlayground(pg))
						}
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
					status, err := c.Tricks.StatusByIdentifier(ctx(), identifier)
					if err != nil {
						return waitResourceError(resource, identifier, auth, err)
					}

					current := status.Status

					progress.Update(fmt.Sprintf("status: %s", current))

					if current == targetStatus {
						if fibe.TrickStatusResultFailed(status) {
							return fmt.Errorf("trick reached %s with failed result", current)
						}
						if full {
							pg, err := c.Tricks.GetByIdentifier(ctx(), identifier)
							if err != nil {
								return waitResourceError(resource, identifier, auth, err)
							}
							progress.Stop()
							output(pg)
						} else {
							progress.Stop()
							output(waitResultFromStatus(status))
						}
						return nil
					}

					// Terminal states for tricks
					if current == "completed" && targetStatus != "completed" {
						return fmt.Errorf("trick reached terminal state: %s", current)
					}
					if current == "completed" && fibe.TrickStatusResultFailed(status) {
						return fmt.Errorf("trick reached completed with failed result")
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
	cmd.Flags().BoolVar(&full, "full", false, "Print the full resource payload after success")

	return cmd
}

type waitNotFoundError struct {
	resource   string
	identifier string
	profile    string
	domain     string
	err        error
}

func (e *waitNotFoundError) Error() string {
	listCommand := "fibe playgrounds list --only id,name,status"
	if e.resource == "trick" || e.resource == "tr" {
		listCommand = "fibe tricks list --only id,name,status"
	}
	return fmt.Sprintf("%s %q was not found in profile %q (%s). Use --profile/--domain to select another environment, or run `%s` to find the correct name or ID",
		e.resource, e.identifier, e.profile, effectiveBaseURL(e.domain), listCommand)
}

func (e *waitNotFoundError) Unwrap() error {
	return e.err
}

func waitResourceError(resource string, identifier string, auth resolvedAuth, err error) error {
	var apiErr *fibe.APIError
	if errors.As(err, &apiErr) && apiErr.IsNotFound() {
		return &waitNotFoundError{
			resource:   resource,
			identifier: identifier,
			profile:    auth.Profile,
			domain:     auth.Domain,
			err:        err,
		}
	}
	return err
}

func waitResultFromPlayground(pg *fibe.Playground) map[string]any {
	if pg == nil {
		return map[string]any{"ok": true}
	}
	out := map[string]any{
		"ok":     true,
		"id":     pg.ID,
		"name":   pg.Name,
		"status": pg.Status,
	}
	if pg.ResultStatus != nil {
		out["result_status"] = *pg.ResultStatus
	}
	if pg.JobResult != nil && pg.JobResult.Summary != nil {
		out["job_summary"] = pg.JobResult.Summary
	}
	return out
}

func waitResultFromStatus(status *fibe.PlaygroundStatus) map[string]any {
	if status == nil {
		return map[string]any{"ok": true}
	}
	out := map[string]any{
		"ok":     true,
		"id":     status.ID,
		"status": status.Status,
	}
	if status.ResultStatus != nil {
		out["result_status"] = *status.ResultStatus
	}
	if status.JobResult != nil && status.JobResult.Summary != nil {
		out["job_summary"] = status.JobResult.Summary
	}
	return out
}
