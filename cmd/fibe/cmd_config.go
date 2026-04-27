package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func configCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage CLI configuration",
		Long: `Manage Fibe CLI local configuration.

The CLI reads configuration from environment variables.
Use this command to view the active configuration.

Available settings:
  FIBE_API_KEY  - Authentication token
  FIBE_DOMAIN   - API domain (default: fibe.gg)
  FIBE_OUTPUT   - Output format (table, json, yaml)`,
		Run: func(cmd *cobra.Command, args []string) {
			apiKey := os.Getenv("FIBE_API_KEY")
			domain := os.Getenv("FIBE_DOMAIN")
			if domain == "" {
				domain = "fibe.gg"
			}
			outFormat := os.Getenv("FIBE_OUTPUT")
			if outFormat == "" {
				outFormat = "table"
			}
			maskedAPIKey := ""
			if apiKey != "" {
				mask := len(apiKey) - 4
				if mask < 0 {
					mask = 0
				}
				prefixLen := 8
				if len(apiKey) < prefixLen {
					prefixLen = len(apiKey)
				}
				maskedAPIKey = apiKey[:prefixLen] + "***" + apiKey[mask:]
			}

			if effectiveOutput() != "table" {
				outputJSON(map[string]string{
					"api_key_masked": maskedAPIKey,
					"domain":         domain,
					"output":         outFormat,
				})
				return
			}

			fmt.Println("=== Active Configuration ===")
			if apiKey == "" {
				fmt.Println("FIBE_API_KEY: not set")
			} else {
				fmt.Printf("FIBE_API_KEY: %s\n", maskedAPIKey)
			}
			fmt.Printf("FIBE_DOMAIN:  %s\n", domain)
			fmt.Printf("FIBE_OUTPUT:  %s\n", outFormat)
		},
	}
	return cmd
}
