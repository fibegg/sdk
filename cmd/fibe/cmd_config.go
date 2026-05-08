package main

import (
	"fmt"

	"github.com/fibegg/sdk/fibe"
	"github.com/spf13/cobra"
)

func configCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage CLI configuration",
		Long: `Show Fibe CLI local configuration.

Auth profiles are stored in ~/.config/fibe/config.json and credentials are
stored separately in ~/.config/fibe/credentials.json.

FIBE_API_KEY/FIBE_DOMAIN are used only as fallbacks when no profile is
configured.`,
		Run: func(cmd *cobra.Command, args []string) {
			resolved := resolveCLIAuth()

			if effectiveOutput() != "table" {
				outputJSON(map[string]any{
					"profile":            resolved.Profile,
					"domain":             effectiveBaseURL(resolved.Domain),
					"auth_source":        resolved.AuthSource,
					"domain_source":      resolved.DomainSource,
					"config_path":        defaultCLIConfigPath(),
					"credentials_path":   fibeCredentialPath(),
					"output":             effectiveOutput(),
					"ignored_env":        resolved.IgnoredEnv,
					"profile_configured": resolved.ProfileConfigured,
				})
				return
			}

			fmt.Println("=== Active Configuration ===")
			fmt.Printf("Profile:          %s\n", resolved.Profile)
			fmt.Printf("Domain:           %s (%s)\n", effectiveBaseURL(resolved.Domain), resolved.DomainSource)
			fmt.Printf("Auth source:      %s\n", resolved.AuthSource)
			fmt.Printf("Config path:      %s\n", defaultCLIConfigPath())
			fmt.Printf("Credentials path: %s\n", fibeCredentialPath())
			fmt.Printf("Output:           %s\n", effectiveOutput())
			if len(resolved.IgnoredEnv) > 0 {
				fmt.Printf("Ignored env:      %v\n", resolved.IgnoredEnv)
			}
		},
	}
	return cmd
}

func fibeCredentialPath() string {
	return fibe.DefaultCredentialPath()
}
