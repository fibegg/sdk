package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strings"
	"time"

	"github.com/fibegg/sdk/fibe"
	"github.com/spf13/cobra"
)

func fibeUserAgent() string { return "Fibe-CLI/" + version }

// deviceAuthResponse mirrors the JSON from POST /cli/device_codes.
type deviceAuthResponse struct {
	DeviceCode              string `json:"device_code"`
	UserCode                string `json:"user_code"`
	VerificationURI         string `json:"verification_uri"`
	VerificationURIComplete string `json:"verification_uri_complete"`
	ExpiresIn               int    `json:"expires_in"`
	Interval                int    `json:"interval"`
}

// devicePollResponse mirrors the JSON from POST /cli/device_codes/poll.
type devicePollResponse struct {
	Status   string   `json:"status"`
	Error    string   `json:"error"`
	APIKey   string   `json:"api_key"`
	APIKeyID int64    `json:"api_key_id"`
	Scopes   []string `json:"scopes"`
}

func authCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Manage CLI authentication",
		Long: `Authenticate the Fibe CLI with named profiles.

Credentials are stored in ~/.config/fibe/credentials.json. Non-secret profile
metadata is stored in ~/.config/fibe/config.json.

The default profile targets fibe.gg. FIBE_API_KEY/FIBE_DOMAIN are CI fallbacks
only when no active profile is configured.

Examples:
  fibe login --api-key fibe_live_...
  fibe auth login --profile staging --domain next.fibe.live --api-key fibe_test_...
  fibe auth use staging
  fibe --profile staging doctor`,
	}

	cmd.AddCommand(authLoginCmd())
	cmd.AddCommand(authListCmd())
	cmd.AddCommand(authUseCmd())
	cmd.AddCommand(authLogoutCmd())
	cmd.AddCommand(authStatusCmd())

	return cmd
}

func authLoginCmd() *cobra.Command {
	var flagHostname string

	cmd := &cobra.Command{
		Use:   "login",
		Short: "Authenticate CLI and save a named profile",
		Long: `Authenticate the CLI and save credentials to a named profile.

If --api-key is provided, the key is validated with /api/me and stored.
If --api-key is omitted, a browser device flow is started.

Defaults:
  --profile default
  --domain  fibe.gg

Examples:
  fibe login --api-key fibe_live_...
  fibe auth login --profile staging --domain next.fibe.live --api-key fibe_test_...`,
		RunE: func(cmd *cobra.Command, args []string) error {
			profile := loginProfileName()
			if err := validateProfileName(profile); err != nil {
				return err
			}
			domain := normalizeDomainInput(flagDomain)
			baseURL := effectiveBaseURL(domain)

			if flagHostname == "" {
				h, err := os.Hostname()
				if err == nil {
					flagHostname = h
				} else {
					flagHostname = "unknown"
				}
			}

			store := fibe.NewCredentialStore(fibe.DefaultCredentialPath())
			if entry, err := store.GetProfile(profile); err == nil && entry != nil {
				fmt.Fprintf(os.Stderr, "Warning: overwriting existing credentials for profile %s (%s)\n", profile, entry.Domain)
			}

			if flagAPIKey != "" {
				me, err := validateAPIKey(domain, flagAPIKey)
				if err != nil {
					return fmt.Errorf("API key validation failed for %s: %w", baseURL, err)
				}
				if err := saveAuthProfile(profile, domain, flagAPIKey, 0); err != nil {
					return fmt.Errorf("authenticated but failed to save credentials: %w", err)
				}
				fmt.Fprintf(os.Stderr, "Authenticated profile %s with %s\n", profile, baseURL)
				if me != nil && me.Username != "" {
					fmt.Fprintf(os.Stderr, "User: %s (ID: %d)\n", me.Username, me.ID)
				}
				fmt.Fprintf(os.Stderr, "Credentials saved to %s\n", fibe.DefaultCredentialPath())
				fmt.Fprintf(os.Stderr, "Active profile: %s\n", profile)
				return nil
			}

			// Step 1: Initiate device authorization
			fmt.Fprintf(os.Stderr, "Initiating authentication for profile %s with %s...\n", profile, baseURL)

			initResp, err := initiateDeviceAuth(baseURL, flagHostname)
			if err != nil {
				return fmt.Errorf("failed to initiate device auth: %w", err)
			}

			// Step 2: Show verification info
			fmt.Fprintln(os.Stderr)
			fmt.Fprintf(os.Stderr, "  Your code: %s\n", initResp.UserCode)
			fmt.Fprintln(os.Stderr)
			fmt.Fprintf(os.Stderr, "  Open this URL to authorize:\n")
			fmt.Fprintf(os.Stderr, "  %s\n", initResp.VerificationURIComplete)
			fmt.Fprintln(os.Stderr)

			// Try to open browser
			if err := openBrowser(initResp.VerificationURIComplete); err != nil {
				fmt.Fprintf(os.Stderr, "  (Could not open browser automatically: %v)\n\n", err)
			} else {
				fmt.Fprintln(os.Stderr, "  Browser opened. Waiting for approval...")
			}

			// Step 3: Poll with Ctrl-C support
			interval := time.Duration(initResp.Interval) * time.Second
			if interval < 3*time.Second {
				interval = 5 * time.Second
			}
			deadline := time.Now().Add(time.Duration(initResp.ExpiresIn) * time.Second)

			ctx, cancel := context.WithDeadline(context.Background(), deadline)
			defer cancel()

			// Handle Ctrl-C gracefully
			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, os.Interrupt)
			defer signal.Stop(sigCh)

			ticker := time.NewTicker(interval)
			defer ticker.Stop()

			for {
				select {
				case <-sigCh:
					fmt.Fprintln(os.Stderr, "\n\nAborted. No credentials were saved.")
					return nil
				case <-ctx.Done():
					fmt.Fprintln(os.Stderr)
					return fmt.Errorf("timed out waiting for authorization — run `fibe auth login` again")
				case <-ticker.C:
					pollResp, httpStatus, err := pollDeviceAuth(baseURL, initResp.DeviceCode)
					if err != nil {
						fmt.Fprintf(os.Stderr, "  Poll error: %v (retrying...)\n", err)
						continue
					}

					switch {
					case pollResp.Status == "authorization_pending":
						fmt.Fprint(os.Stderr, ".")
						continue

					case pollResp.Status == "authorized" && pollResp.APIKey != "":
						fmt.Fprintln(os.Stderr)
						if err := saveAuthProfile(profile, domain, pollResp.APIKey, pollResp.APIKeyID); err != nil {
							return fmt.Errorf("authenticated but failed to save credentials: %w", err)
						}
						fmt.Fprintf(os.Stderr, "\nAuthenticated profile %s with %s\n", profile, baseURL)
						fmt.Fprintf(os.Stderr, "  Credentials saved to %s\n", fibe.DefaultCredentialPath())
						fmt.Fprintf(os.Stderr, "  Active profile: %s\n", profile)
						return nil

					case pollResp.Error == "access_denied":
						fmt.Fprintln(os.Stderr)
						return fmt.Errorf("authorization denied by user")

					case pollResp.Error == "already_consumed":
						fmt.Fprintln(os.Stderr)
						return fmt.Errorf("API key was already retrieved — run `fibe auth login` again")

					case httpStatus == http.StatusGone || pollResp.Error == "expired_token":
						fmt.Fprintln(os.Stderr)
						return fmt.Errorf("device code expired — run `fibe auth login` again")

					default:
						fmt.Fprintln(os.Stderr)
						return fmt.Errorf("unexpected response: %s", pollResp.Error)
					}
				}
			}
		},
	}

	cmd.Flags().StringVar(&flagHostname, "hostname", "", "Override hostname sent to server (defaults to os.Hostname)")

	return cmd
}

func authListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List auth profiles",
		Long:  "List configured Fibe auth profiles, their domains, and masked credential status.",
		RunE: func(cmd *cobra.Command, args []string) error {
			rows, err := listAuthProfiles()
			if err != nil {
				return err
			}
			if effectiveOutput() != "table" {
				outputJSON(rows)
				return nil
			}
			if len(rows) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No auth profiles configured. Run `fibe login --api-key <key>` to create the default profile.")
				return nil
			}
			tableRows := make([][]string, 0, len(rows))
			for _, row := range rows {
				active := ""
				if row.Active {
					active = "*"
				}
				auth := "missing"
				if row.HasKey {
					auth = row.MaskedKey
				}
				tableRows = append(tableRows, []string{active, row.Name, effectiveBaseURL(row.Domain), auth})
			}
			outputTable([]string{"ACTIVE", "PROFILE", "DOMAIN", "KEY"}, tableRows)
			return nil
		},
	}
}

func authUseCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "use <profile>",
		Short: "Switch active auth profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			profile := args[0]
			if err := validateProfileName(profile); err != nil {
				return err
			}
			if !profileExists(profile) {
				return fmt.Errorf("auth profile %q does not exist; run `fibe auth list`", profile)
			}
			if err := newCLIConfigStore(defaultCLIConfigPath()).setActive(profile); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Active profile: %s\n", profile)
			return nil
		},
	}
}

func authLogoutCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Remove stored CLI credentials",
		Long: `Remove stored credentials for a profile.

Uses --profile when provided, otherwise the active profile. Also attempts to
revoke the API key on the server when the stored API key ID is available.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			profile := selectedProfileName()
			store := fibe.NewCredentialStore(fibe.DefaultCredentialPath())

			entry, err := store.GetProfile(profile)
			if err != nil || entry == nil {
				fmt.Fprintf(cmd.OutOrStdout(), "No stored credentials for profile %s\n", profile)
				return nil
			}

			// Best-effort: revoke the key on the server
			if entry.APIKeyID > 0 {
				client := fibe.NewClient(
					fibe.WithDisableAutoConfig(),
					fibe.WithAPIKey(entry.APIKey),
					fibe.WithDomain(entry.Domain),
				)
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()
				if err := client.APIKeys.Delete(ctx, entry.APIKeyID); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: could not revoke key on server: %v\n", err)
				} else {
					fmt.Fprintf(os.Stderr, "API key revoked on %s\n", effectiveBaseURL(entry.Domain))
				}
			}

			if err := deleteAuthProfile(profile); err != nil {
				return fmt.Errorf("failed to remove local credentials: %w", err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Logged out profile %s\n", profile)
			return nil
		},
	}
}

func authStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show current authentication status",
		Long: `Show which auth profile, domain, and credential source are active.

Switch between environments using profiles:
  fibe auth use staging
  fibe --profile staging auth status`,
		RunE: func(cmd *cobra.Command, args []string) error {
			resolved := resolveCLIAuth()
			if effectiveOutput() != "table" {
				outputJSON(map[string]any{
					"profile":             resolved.Profile,
					"domain":              effectiveBaseURL(resolved.Domain),
					"auth_source":         resolved.AuthSource,
					"domain_source":       resolved.DomainSource,
					"authenticated_local": resolved.APIKey != "",
					"ignored_env":         resolved.IgnoredEnv,
				})
				return nil
			}
			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "Profile: %s\n", resolved.Profile)
			fmt.Fprintf(out, "Domain:  %s (%s)\n", effectiveBaseURL(resolved.Domain), resolved.DomainSource)
			fmt.Fprintf(out, "Source:  %s\n", resolved.AuthSource)
			if resolved.APIKey != "" {
				fmt.Fprintf(out, "Key:     %s\n", maskKey(resolved.APIKey))
				if resolved.APIKeyID > 0 {
					fmt.Fprintf(out, "Key ID:  %d\n", resolved.APIKeyID)
				}
			} else {
				fmt.Fprintln(out, "Status:  not authenticated")
				fmt.Fprintln(out, "Run `fibe login --api-key <key>` or `fibe auth login`.")
			}
			if len(resolved.IgnoredEnv) > 0 {
				fmt.Fprintf(out, "Ignored env: %s\n", strings.Join(resolved.IgnoredEnv, ", "))
			}

			return nil
		},
	}
}

func maskKey(key string) string {
	if len(key) <= 12 {
		return "***"
	}
	return key[:8] + "***" + key[len(key)-4:]
}

func openBrowser(url string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", url).Start()
	case "linux":
		return exec.Command("xdg-open", url).Start()
	case "windows":
		return exec.Command("cmd", "/c", "start", url).Start()
	}
	return fmt.Errorf("unsupported platform %s", runtime.GOOS)
}

func deviceFlowHTTPClient() *http.Client {
	return &http.Client{Timeout: 15 * time.Second}
}

func validateAPIKey(domain, apiKey string) (*fibe.Player, error) {
	client := fibe.NewClient(
		fibe.WithDisableAutoConfig(),
		fibe.WithDomain(domain),
		fibe.WithAPIKey(apiKey),
		fibe.WithMaxRetries(0),
	)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	return client.APIKeys.Me(ctx)
}

func initiateDeviceAuth(baseURL, hostname string) (*deviceAuthResponse, error) {
	body := strings.NewReader(fmt.Sprintf(`{"hostname":%q}`, hostname))
	req, err := http.NewRequest(http.MethodPost, baseURL+"/cli/device_codes", body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", fibeUserAgent())

	resp, err := deviceFlowHTTPClient().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read error: %w", err)
	}

	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("server returned %d: %s", resp.StatusCode, string(data))
	}

	var result deviceAuthResponse
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("decode error: %v (body: %s)", err, string(data))
	}
	return &result, nil
}

func pollDeviceAuth(baseURL, deviceCode string) (*devicePollResponse, int, error) {
	body := strings.NewReader(fmt.Sprintf(`{"device_code":%q}`, deviceCode))
	req, err := http.NewRequest(http.MethodPost, baseURL+"/cli/device_codes/poll", body)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", fibeUserAgent())

	resp, err := deviceFlowHTTPClient().Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("read error: %w", err)
	}

	var result devicePollResponse
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, resp.StatusCode, fmt.Errorf("decode error: %v (body: %s)", err, string(data))
	}
	return &result, resp.StatusCode, nil
}
