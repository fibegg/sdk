package main

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
)

func doctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Run self-diagnostic checks",
		Long: `Check API key validity, server connectivity, and SDK version.

Useful for debugging authentication issues and verifying
the CLI is configured correctly.

Output includes:
  - API key status (valid/expired/invalid)
  - Server connectivity and response time
  - SDK version and domain`,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("=== Fibe Doctor ===")
			fmt.Println()

			// Check API key presence
			apiKey := os.Getenv("FIBE_API_KEY")
			if apiKey == "" {
				fmt.Println("❌ FIBE_API_KEY: not set")
				fmt.Println("   Set it with: export FIBE_API_KEY=pk_live_...")
				return nil
			}
			masked := apiKey[:10] + "..." + apiKey[len(apiKey)-4:]
			fmt.Printf("✅ FIBE_API_KEY: %s\n", masked)

			// Check domain
			domain := os.Getenv("FIBE_DOMAIN")
			if domain == "" {
				domain = "fibe.gg"
			}
			fmt.Printf("✅ Domain: %s\n", domain)
			fmt.Printf("✅ Version: %s\n", version)
			fmt.Println()

			// Test connectivity
			c := newClient()
			start := time.Now()
			me, err := c.APIKeys.Me(ctx())
			elapsed := time.Since(start)

			if err != nil {
				fmt.Printf("❌ Connectivity: %v\n", err)
				return nil
			}

			fmt.Printf("✅ Connectivity: %dms\n", elapsed.Milliseconds())
			if me != nil {
				fmt.Printf("✅ Authenticated as: %s (ID: %d)\n", me.Username, me.ID)
			}
			fmt.Printf("✅ Last Request ID: %s\n", c.LastRequestID())

			return nil
		},
	}
}
