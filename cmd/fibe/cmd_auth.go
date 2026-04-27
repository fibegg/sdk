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
		Long: `Authenticate the Fibe CLI using a browser-based device flow.

Credentials are stored in ~/.config/fibe/credentials.json, keyed by FIBE_DOMAIN.
This supports multiple environments (fibe.gg, next.fibe.live, rails.test:3000).

Resolution order: --api-key > FIBE_API_KEY > credentials.json

Use --domain to target a specific environment:
  fibe --domain next.fibe.live auth login
  fibe --domain rails.test:3000 auth status`,
	}

	cmd.AddCommand(authLoginCmd())
	cmd.AddCommand(authLogoutCmd())
	cmd.AddCommand(authStatusCmd())

	return cmd
}

func authLoginCmd() *cobra.Command {
	var flagHostname string

	cmd := &cobra.Command{
		Use:   "login",
		Short: "Authenticate CLI via browser",
		Long: `Start a device authorization flow to authenticate the CLI.

1. Opens a browser link on FIBE_DOMAIN for you to approve
2. Polls the server until you approve or the code expires (15 min)
3. Stores the resulting API key in ~/.config/fibe/credentials.json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			domain := resolveDomain()
			scheme := resolveScheme(domain)
			baseURL := scheme + "://" + domain

			if flagHostname == "" {
				h, err := os.Hostname()
				if err == nil {
					flagHostname = h
				} else {
					flagHostname = "unknown"
				}
			}

			// Check if already authenticated for this domain
			store := fibe.NewCredentialStore(fibe.DefaultCredentialPath())
			if entry, err := store.Get(domain); err == nil && entry != nil {
				fmt.Fprintf(os.Stderr, "Warning: overwriting existing credentials for %s\n", domain)
			}

			// Step 1: Initiate device authorization
			fmt.Fprintf(os.Stderr, "Initiating authentication with %s...\n", domain)

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
						// Store credential
						err := store.Set(&fibe.CredentialEntry{
							APIKey:   pollResp.APIKey,
							APIKeyID: pollResp.APIKeyID,
							Domain:   domain,
						})
						if err != nil {
							return fmt.Errorf("authenticated but failed to save credentials: %w", err)
						}
						fmt.Fprintf(os.Stderr, "\n✓ Authenticated with %s\n", domain)
						fmt.Fprintf(os.Stderr, "  Credentials saved to %s\n", fibe.DefaultCredentialPath())
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

func authLogoutCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Remove stored CLI credentials",
		Long: `Remove the stored API key for the current FIBE_DOMAIN.

Also attempts to revoke the API key on the server (best-effort).`,
		RunE: func(cmd *cobra.Command, args []string) error {
			domain := resolveDomain()
			store := fibe.NewCredentialStore(fibe.DefaultCredentialPath())

			entry, err := store.Get(domain)
			if err != nil || entry == nil {
				fmt.Fprintf(os.Stderr, "No stored credentials for %s\n", domain)
				return nil
			}

			// Best-effort: revoke the key on the server
			if entry.APIKeyID > 0 {
				client := fibe.NewClient(
					fibe.WithAPIKey(entry.APIKey),
					fibe.WithDomain(domain),
				)
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()
				if err := client.APIKeys.Delete(ctx, entry.APIKeyID); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: could not revoke key on server: %v\n", err)
				} else {
					fmt.Fprintf(os.Stderr, "API key revoked on %s\n", domain)
				}
			}

			if err := store.Delete(domain); err != nil {
				return fmt.Errorf("failed to remove local credentials: %w", err)
			}
			fmt.Fprintf(os.Stderr, "✓ Logged out from %s\n", domain)
			return nil
		},
	}
}

func authStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show current authentication status",
		Long: `Show which credential source is active and list all stored domains.

Switch between environments using --domain:
  fibe --domain fibe.gg auth status
  fibe --domain next.fibe.live auth status`,
		RunE: func(cmd *cobra.Command, args []string) error {
			domain := resolveDomain()

			// Check all sources in priority order
			if flagAPIKey != "" {
				fmt.Fprintf(os.Stderr, "Domain:  %s\n", domain)
				fmt.Fprintf(os.Stderr, "Source:  --api-key flag\n")
				fmt.Fprintf(os.Stderr, "Key:     %s\n", maskKey(flagAPIKey))
				return nil
			}
			if envKey := os.Getenv("FIBE_API_KEY"); envKey != "" {
				fmt.Fprintf(os.Stderr, "Domain:  %s\n", domain)
				fmt.Fprintf(os.Stderr, "Source:  FIBE_API_KEY env\n")
				fmt.Fprintf(os.Stderr, "Key:     %s\n", maskKey(envKey))
				return nil
			}

			store := fibe.NewCredentialStore(fibe.DefaultCredentialPath())
			entry, err := store.Get(domain)
			if err == nil && entry != nil {
				fmt.Fprintf(os.Stderr, "Domain:  %s\n", domain)
				fmt.Fprintf(os.Stderr, "Source:  credentials.json (fibe auth login)\n")
				fmt.Fprintf(os.Stderr, "Key:     %s\n", maskKey(entry.APIKey))
				if entry.APIKeyID > 0 {
					fmt.Fprintf(os.Stderr, "Key ID:  %d\n", entry.APIKeyID)
				}
			} else {
				fmt.Fprintf(os.Stderr, "Domain:  %s\n", domain)
				fmt.Fprintln(os.Stderr, "Status:  not authenticated")
				fmt.Fprintln(os.Stderr, "Run `fibe auth login` to authenticate.")
			}

			// Always show all stored domains
			all, _ := store.List()
			if len(all) > 0 {
				fmt.Fprintln(os.Stderr)
				fmt.Fprintln(os.Stderr, "All stored domains:")
				for d := range all {
					marker := "  "
					if d == domain {
						marker = "→ "
					}
					fmt.Fprintf(os.Stderr, "  %s%s\n", marker, d)
				}
				fmt.Fprintln(os.Stderr)
				fmt.Fprintln(os.Stderr, "Switch with: fibe --domain <domain> ...")
			}

			return nil
		},
	}
}

// --- helpers ---

func resolveDomain() string {
	if flagDomain != "" {
		return flagDomain
	}
	if d := os.Getenv("FIBE_DOMAIN"); d != "" {
		return d
	}
	return "fibe.gg"
}

func resolveScheme(domain string) string {
	if strings.HasSuffix(domain, ".test") ||
		strings.Contains(domain, ".test:") ||
		strings.Contains(domain, "localhost") ||
		strings.Contains(domain, "127.0.0.1") {
		return "http"
	}
	return "https"
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
