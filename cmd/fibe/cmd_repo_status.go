package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func repoStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "repo-status",
		Short: "Check accessibility of GitHub repositories",
	}
	cmd.AddCommand(repoStatusCheckCmd())
	return cmd
}

func repoStatusCheckCmd() *cobra.Command {
	var urls []string
	cmd := &cobra.Command{
		Use:   "check",
		Short: "Check the status of GitHub repositories",
		Long: `Check whether GitHub repositories are accessible to your installation(s).

Accepts up to 50 URLs.

REQUIRED FLAGS:
  --url    GitHub repo URL (repeatable)

EXAMPLES:
  fibe repo-status check --url https://github.com/owner/repo1
  fibe repo-status check --url owner/repo1 --url owner/repo2`,
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			normalized := make([]string, 0, len(urls))
			for _, u := range urls {
				u = strings.TrimSpace(u)
				if u == "" {
					continue
				}
				normalized = append(normalized, u)
			}
			if len(normalized) == 0 {
				return fmt.Errorf("at least one --url is required")
			}
			result, err := c.RepoStatus.Check(ctx(), normalized)
			if err != nil {
				return err
			}
			outputJSON(result)
			return nil
		},
	}
	cmd.Flags().StringSliceVar(&urls, "url", nil, "GitHub repo URL (repeatable, required)")
	return cmd
}
