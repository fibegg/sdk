package main

import (
	"fmt"
	"os"
	"time"

	"github.com/fibegg/sdk/fibe"
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
			c := newClient()

			result := map[string]any{
				"domain":  c.BaseURL(),
				"version": version,
			}

			start := time.Now()
			me, err := c.APIKeys.Me(ctx())
			elapsed := time.Since(start)

			if err != nil {
				result["authenticated"] = false
				result["error"] = err.Error()
			} else {
				result["authenticated"] = true
				if me != nil {
					result["user_id"] = me.ID
					result["username"] = me.Username
					result["github_handle"] = me.GithubHandle
					result["email"] = me.Email
					result["avatar_url"] = me.AvatarURL
					if len(me.APIKeyScopes) > 0 {
						result["api_key_scopes"] = me.APIKeyScopes
					}
				}
			}

			if effectiveOutput() != "table" {
				output(result)
				return nil
			}

			fmt.Println("=== Fibe Doctor ===")
			fmt.Println()

			// Check API key presence
			apiKey, apiKeySource := doctorAPIKeyStatus()
			if apiKey == "" {
				fmt.Println("❌ API key: not set")
				fmt.Println("   Set it with: export FIBE_API_KEY=pk_live_... or pass --api-key")
			} else {
				fmt.Printf("✅ API key: %s (%s)\n", maskKey(apiKey), apiKeySource)
			}

			fmt.Printf("✅ Domain: %s\n", c.BaseURL())
			fmt.Printf("✅ Version: %s\n", version)
			fmt.Println()

			if err != nil {
				fmt.Printf("❌ Connectivity: %v\n", err)
			} else {
				fmt.Printf("✅ Connectivity: %dms\n", elapsed.Milliseconds())
				if me != nil {
					fmt.Printf("✅ Authenticated as: %s (ID: %d)\n", me.Username, me.ID)
				}
				fmt.Printf("✅ Last Request ID: %s\n", c.LastRequestID())
			}

			return nil
		},
	}
}

func doctorAPIKeyStatus() (string, string) {
	if flagAPIKey != "" {
		return flagAPIKey, "--api-key flag"
	}
	if apiKey := os.Getenv("FIBE_API_KEY"); apiKey != "" {
		return apiKey, "FIBE_API_KEY env"
	}

	store := fibe.NewCredentialStore(fibe.DefaultCredentialPath())
	if entry, err := store.Get(resolveDomain()); err == nil && entry != nil && entry.APIKey != "" {
		return entry.APIKey, "credentials.json"
	}

	return "", ""
}
