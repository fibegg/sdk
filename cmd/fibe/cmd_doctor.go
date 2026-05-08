package main

import (
	"fmt"
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
			c := newClient()
			resolved := resolveCLIAuth()

			result := map[string]any{
				"profile":       resolved.Profile,
				"domain":        c.BaseURL(),
				"version":       version,
				"auth_source":   resolved.AuthSource,
				"domain_source": resolved.DomainSource,
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

			apiKey, apiKeySource := doctorAPIKeyStatus()
			fmt.Printf("Profile: %s\n", resolved.Profile)
			if apiKey == "" {
				fmt.Println("API key: not set")
				fmt.Println("   Run: fibe login --api-key <key>")
			} else {
				fmt.Printf("API key: %s (%s)\n", maskKey(apiKey), apiKeySource)
			}

			fmt.Printf("Domain: %s (%s)\n", c.BaseURL(), resolved.DomainSource)
			fmt.Printf("Version: %s\n", version)
			if len(resolved.IgnoredEnv) > 0 {
				fmt.Printf("Ignored env: %v\n", resolved.IgnoredEnv)
			}
			fmt.Println()

			if err != nil {
				fmt.Printf("Connectivity: %v\n", err)
			} else {
				fmt.Printf("Connectivity: %dms\n", elapsed.Milliseconds())
				if me != nil {
					fmt.Printf("Authenticated as: %s (ID: %d)\n", me.Username, me.ID)
				}
				fmt.Printf("Last Request ID: %s\n", c.LastRequestID())
			}

			return nil
		},
	}
}

func doctorAPIKeyStatus() (string, string) {
	resolved := resolveCLIAuth()
	if resolved.APIKey != "" {
		return resolved.APIKey, resolved.AuthSource
	}
	return "", ""
}
