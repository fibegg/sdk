package main

import (
	"fmt"
	"sort"

	"github.com/fibegg/sdk/fibe"
	"github.com/spf13/cobra"
)

func limitsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "limits",
		Short: "Show current quotas and API-key rate limits",
		Long: `Show the caller's current resource quotas, per-parent caps, and
API-key rate-limit usage (limit, remaining, seconds until reset).

Requires authentication via an API key (Authorization: Bearer ...).
The server only returns limits data to API-key-authenticated callers.

Examples:
  fibe limits
  fibe limits -o json
  fibe limits -o yaml --only resource_quotas`,
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			status, err := c.Status.Get(ctx())
			if err != nil {
				return err
			}

			if status.ResourceQuotas == nil && status.PerParentCaps == nil && status.RateLimits == nil {
				return fmt.Errorf("no limits data returned — ensure you're authenticated with an API key")
			}

			payload := struct {
				ResourceQuotas map[string]fibe.ResourceQuotaEntry `json:"resource_quotas,omitempty"`
				PerParentCaps  map[string]*int                    `json:"per_parent_caps,omitempty"`
				RateLimits     *fibe.RateLimitsSection            `json:"rate_limits,omitempty"`
			}{
				ResourceQuotas: status.ResourceQuotas,
				PerParentCaps:  status.PerParentCaps,
				RateLimits:     status.RateLimits,
			}

			switch effectiveOutput() {
			case "table":
				printLimitsTable(status)
			default:
				output(payload)
			}
			return nil
		},
	}
}

func printLimitsTable(status *fibe.Status) {
	fmt.Println("=== Resource Quotas ===")
	if len(status.ResourceQuotas) == 0 {
		fmt.Println("(none)")
	} else {
		rows := make([][]string, 0, len(status.ResourceQuotas))
		for _, key := range sortedKeys(status.ResourceQuotas) {
			entry := status.ResourceQuotas[key]
			rows = append(rows, []string{key, fmt.Sprintf("%d", entry.Used), formatLimit(entry.Limit), entry.Status})
		}
		outputTable([]string{"Resource", "Used", "Limit", "Status"}, rows)
	}

	fmt.Println()
	fmt.Println("=== Per-Parent Caps ===")
	if len(status.PerParentCaps) == 0 {
		fmt.Println("(none)")
	} else {
		rows := make([][]string, 0, len(status.PerParentCaps))
		for _, key := range sortedIntPtrKeys(status.PerParentCaps) {
			rows = append(rows, []string{key, formatLimit(status.PerParentCaps[key])})
		}
		outputTable([]string{"Cap", "Limit"}, rows)
	}

	fmt.Println()
	fmt.Println("=== API Key Rate Limits ===")
	if status.RateLimits == nil || status.RateLimits.APIKey == nil {
		fmt.Println("(not available)")
		return
	}
	rl := status.RateLimits.APIKey
	fmt.Printf("Limit:         %d req/hour\n", rl.Limit)
	fmt.Printf("Remaining:     %d\n", rl.Remaining)
	fmt.Printf("Resets in:     %d seconds\n", rl.ResetSeconds)
}

func sortedKeys(m map[string]fibe.ResourceQuotaEntry) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func sortedIntPtrKeys(m map[string]*int) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func formatLimit(limit *int) string {
	if limit == nil {
		return "unlimited"
	}
	return fmt.Sprintf("%d", *limit)
}
