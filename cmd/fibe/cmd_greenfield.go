package main

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/fibegg/sdk/fibe"
	"github.com/fibegg/sdk/internal/localplaygrounds"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

const greenfieldDefaultLinkDir = "/app/playground"

func greenfieldCmd() *cobra.Command {
	var (
		name                   string
		gitProvider            string
		private                bool
		templateID             string
		templateVersionID      int64
		templateIDTypoTempalte int64
		templateIDTypoTemlate  int64
		version                string
		templateBody           string
		repoFile               string
		repoRef                string
		githubAccount          string
		githubInstallationID   int64
		marqueeID              string
		marqueeIDTypoMarque    int64
		vars                   []string
		serviceSubdomains      []string
		waitTimeout            time.Duration
	)

	cmd := &cobra.Command{
		Use:   "greenfield [github-repo]",
		Short: "Create a new greenfield app from the platform template flow",
		Long: `Create a new app from a template, including one or more destination repositories and Props, an app-owned template version, and a deployed playground.

The command calls the Fibe greenfield API, waits for the playground to run,
and links the local playground checkout into /app/playground by default.
The target Marquee must be funded; unpaid Marquees fail with
MARQUEE_NOT_FUNDED before deployment starts.

Examples:
  fibe greenfield owner/repo --marquee 12
  fibe greenfield owner/repo@main --file fibe.yml --marquee 12
  fibe greenfield https://github.com/owner/repo --ref main --marquee 12
  fibe greenfield --name my-app --template "Rails 8 Starter Kit"
  fibe greenfield --name my-app --template-version 912
  fibe greenfield --name my-app --service-subdomain app=my-app --service-subdomain admin=my-app-admin
  fibe greenfield --name my-app -f my-template.yml
  fibe greenfield --name my-app --template-body 'services:\n  web:\n    image: nginx'`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			createProgress := newStatusLine(cmd.ErrOrStderr(), statusLineOptions{})
			createProgress.Start("creating greenfield app")
			defer createProgress.Stop()
			c := newClient(fibe.WithProgress(createProgress.Progress("creating greenfield app")))
			params := &fibe.GreenfieldCreateParams{}
			if err := applyGreenfieldFromFile(params); err != nil {
				return err
			}
			if cmd.Flags().Changed("name") {
				params.Name = name
			}
			if cmd.Flags().Changed("git-provider") {
				params.GitProvider = gitProvider
			} else if params.GitProvider == "" {
				params.GitProvider = "gitea"
			}
			if cmd.Flags().Changed("private") {
				params.Private = &private
			}
			selectedTemplateIdentifier, templateIDChanged := selectedGreenfieldTemplateIdentifier(cmd, templateID, templateIDTypoTempalte, templateIDTypoTemlate)
			if templateIDChanged && selectedTemplateIdentifier != "" {
				params.TemplateIdentifier = selectedTemplateIdentifier
			}
			if cmd.Flags().Changed("template-version") && templateVersionID > 0 {
				if params.TemplateIdentifier != "" || params.Version != "" {
					return fmt.Errorf("--template-version cannot be combined with --template or --version")
				}
				id := templateVersionID
				params.TemplateVersionID = &id
			}
			if cmd.Flags().Changed("version") {
				if params.TemplateVersionID != nil {
					return fmt.Errorf("--version cannot be combined with --template-version")
				}
				if params.TemplateIdentifier == "" {
					return fmt.Errorf("--version requires --template")
				}
				params.Version = version
			}
			if cmd.Flags().Changed("template-body") {
				params.TemplateBody = normalizeTemplateBodyValue(resolveStringValue(templateBody))
			}
			if cmd.Flags().Changed("file") {
				params.ConfigPath = repoFile
			}
			if cmd.Flags().Changed("ref") {
				params.GitHubRef = repoRef
			}
			if cmd.Flags().Changed("github-account") {
				params.GitHubAccount = githubAccount
			}
			if cmd.Flags().Changed("github-installation-id") && githubInstallationID > 0 {
				id := githubInstallationID
				params.GitHubInstallationID = &id
			}
			if cmd.Flags().Changed("marquee") && marqueeID != "" {
				params.MarqueeIdentifier = marqueeID
			} else if cmd.Flags().Changed("marque-id") && marqueeIDTypoMarque > 0 {
				id := marqueeIDTypoMarque
				params.MarqueeID = &id
			} else if params.MarqueeID == nil && params.MarqueeIdentifier == "" {
				identifier, err := resolveLaunchMarqueeIdentifier(c, "")
				if err != nil {
					return err
				}
				params.MarqueeIdentifier = identifier
			}
			if cmd.Flags().Changed("var") && len(vars) > 0 {
				parsed, err := parseGreenfieldVars(vars)
				if err != nil {
					return err
				}
				params.Variables = parsed
			}
			if cmd.Flags().Changed("service-subdomain") && len(serviceSubdomains) > 0 {
				parsed, err := parseGreenfieldStringMapFlags(serviceSubdomains, "--service-subdomain")
				if err != nil {
					return err
				}
				params.ServiceSubdomains = parsed
			}
			if (len(args) > 0 || params.RepositoryURL != "") && (params.TemplateBody != "" || params.TemplateIdentifier != "" || params.TemplateID != nil || params.TemplateVersionID != nil || params.Version != "") {
				return fmt.Errorf("GitHub repository argument cannot be combined with template body, template id, template version, or version")
			}

			repoRequest, err := resolveGitHubRepoRequest(cmd, c, args, githubRepoRequestOptions{
				ExistingURL:            params.RepositoryURL,
				ExistingName:           params.Name,
				ExistingRef:            params.GitHubRef,
				ExistingConfigPath:     params.ConfigPath,
				ExistingAccount:        params.GitHubAccount,
				ExistingInstallationID: params.GitHubInstallationID,
				FlagRef:                repoRef,
				FlagFile:               repoFile,
				FlagAccount:            githubAccount,
				FlagInstallationID:     githubInstallationID,
			})
			if err != nil {
				return err
			}
			if repoRequest != nil {
				params.RepositoryURL = repoRequest.URL
				params.Name = repoRequest.Name
				params.GitHubRef = repoRequest.Ref
				params.ConfigPath = repoRequest.ConfigPath
				params.GitHubAccount = repoRequest.Account
				params.GitHubInstallationID = repoRequest.GitHubInstallationID
			}
			if params.Name == "" {
				return fmt.Errorf("required field 'name' not set")
			}

			result, err := c.Greenfield.Create(ctx(), params)
			createProgress.Stop()
			if err != nil {
				return err
			}
			if result.Playground == nil || result.Playground.ID == 0 {
				return fmt.Errorf("greenfield create did not return a playground id")
			}

			waitProgress := newStatusLine(cmd.ErrOrStderr(), statusLineOptions{FallbackStart: true, FallbackUpdates: true})
			waitProgress.Start(fmt.Sprintf("waiting for playground %d to reach running...", result.Playground.ID))
			pg, err := waitForPlayground(ctx(), c, result.Playground.ID, "running", waitTimeout, 3*time.Second, func(status string) {
				waitProgress.Update(fmt.Sprintf("status: %s", status))
			})
			waitProgress.Stop()
			if err != nil {
				return err
			}
			result.Playground = pg

			target := greenfieldLocalTarget(result)
			link, err := localplaygrounds.Link(target, greenfieldDefaultLinkDir)
			if err != nil {
				return err
			}
			result.Link = link

			output(result)
			return nil
		},
	}

	cmd.Flags().SortFlags = false
	cmd.Flags().StringVar(&name, "name", "", "Repository/app name (optional with github-repo; must be unique)")
	cmd.Flags().StringVar(&gitProvider, "git-provider", "gitea", "Destination git provider: gitea or github (optional, default: gitea)")
	cmd.Flags().BoolVar(&private, "private", false, "Create destination repository as private")
	cmd.Flags().StringVar(&templateID, "template", "", "Template ID or name to use (optional, default: base template)")
	cmd.Flags().Int64Var(&templateVersionID, "template-version", 0, "Exact template version ID to use (optional)")
	cmd.Flags().StringVar(&version, "version", "", "Template version tag or number when --template is used (e.g. v1, optional, default: latest version)")
	cmd.Flags().StringVar(&templateBody, "template-body", "", "Template YAML body to use directly (optional)")
	cmd.Flags().StringVar(&repoFile, "file", "", "Config file path inside the GitHub repository (optional; defaults to fibe.yml, fibe.yaml, docker-compose.yml, docker-compose.yaml)")
	cmd.Flags().StringVar(&repoRef, "ref", "", "Git branch, tag, or commit for the config file (optional)")
	cmd.Flags().StringVar(&githubAccount, "github-account", "", "GitHub App installation account owner to use when multiple installations are connected")
	cmd.Flags().Int64Var(&githubInstallationID, "github-installation-id", 0, "GitHub App installation ID to use when multiple installations are connected")
	cmd.Flags().StringVar(&marqueeID, "marquee", "", "Target marquee ID or name (optional, default: current Marquee)")
	cmd.Flags().StringSliceVar(&vars, "var", nil, "Set template variables (e.g., --var app_name=Tower, optional)")
	cmd.Flags().StringSliceVar(&serviceSubdomains, "service-subdomain", nil, "Set an exposed service subdomain override (repeatable, e.g., --service-subdomain app=my-app)")
	cmd.Flags().DurationVar(&waitTimeout, "wait-timeout", 10*time.Minute, "Maximum time to wait for the playground to reach running (optional, default 10m0s)")
	cmd.Flags().Int64Var(&templateIDTypoTempalte, "tempalte-id", 0, "Alias for --template")
	cmd.Flags().Int64Var(&templateIDTypoTemlate, "temlate-id", 0, "Alias for --template")
	cmd.Flags().Int64Var(&marqueeIDTypoMarque, "marque-id", 0, "Alias for --marquee")
	_ = cmd.Flags().MarkHidden("tempalte-id")
	_ = cmd.Flags().MarkHidden("temlate-id")
	_ = cmd.Flags().MarkHidden("marque-id")
	return cmd
}

func applyGreenfieldFromFile(params *fibe.GreenfieldCreateParams) error {
	if err := applyFromFile(params); err != nil {
		return err
	}
	if params.TemplateBody == "" && len(rawPayload) > 0 && looksLikeTemplateBody(rawPayload) {
		params.TemplateBody = string(rawPayload)
	}
	return nil
}

func looksLikeTemplateBody(data []byte) bool {
	var doc map[string]any
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return false
	}
	if len(doc) == 0 {
		return false
	}
	_, hasServices := doc["services"]
	_, hasXFibe := doc["x-fibe.gg"]
	return hasServices || hasXFibe
}

func normalizeTemplateBodyValue(value string) string {
	return strings.ReplaceAll(value, `\n`, "\n")
}

func selectedGreenfieldTemplateIdentifier(cmd *cobra.Command, templateID string, tempalteID, temlateID int64) (string, bool) {
	switch {
	case cmd.Flags().Changed("template"):
		return strings.TrimSpace(templateID), true
	case cmd.Flags().Changed("tempalte-id"):
		if tempalteID > 0 {
			return strconv.FormatInt(tempalteID, 10), true
		}
		return "", true
	case cmd.Flags().Changed("temlate-id"):
		if temlateID > 0 {
			return strconv.FormatInt(temlateID, 10), true
		}
		return "", true
	default:
		return "", false
	}
}

func parseGreenfieldVars(values []string) (map[string]any, error) {
	out := make(map[string]any, len(values))
	for _, value := range values {
		parts := strings.SplitN(value, "=", 2)
		key := normalizeVariableFlagKey(parts[0])
		if len(parts) != 2 || key == "" {
			return nil, fmt.Errorf("invalid --var %q, expected key=value", value)
		}
		out[key] = parts[1]
	}
	return out, nil
}

func parseGreenfieldStringMapFlags(values []string, flagName string) (map[string]string, error) {
	out := make(map[string]string, len(values))
	for _, value := range values {
		parts := strings.SplitN(value, "=", 2)
		key := normalizeVariableFlagKey(parts[0])
		if len(parts) != 2 || key == "" {
			return nil, fmt.Errorf("invalid %s %q, expected key=value", flagName, value)
		}
		item := strings.TrimSpace(parts[1])
		if item == "" {
			return nil, fmt.Errorf("invalid %s %q, value cannot be blank", flagName, value)
		}
		out[key] = item
	}
	return out, nil
}

func normalizeVariableFlagKey(key string) string {
	return strings.TrimLeft(strings.TrimSpace(key), "-")
}

func greenfieldLocalTarget(result *fibe.GreenfieldResult) string {
	if result.Playground != nil && result.Playground.PlayspecName != nil && *result.Playground.PlayspecName != "" {
		return *result.Playground.PlayspecName
	}
	if result.Playspec != nil && result.Playspec.Name != "" {
		return result.Playspec.Name
	}
	if result.Playground != nil && result.Playground.Name != "" {
		return result.Playground.Name
	}
	return result.Name
}

func waitForPlayground(ctx context.Context, c *fibe.Client, id int64, target string, timeout time.Duration, interval time.Duration, progress func(string)) (*fibe.Playground, error) {
	if target == "" {
		target = "running"
	}
	readiness, err := fibe.NormalizePlaygroundWaitReadiness("", target)
	if err != nil {
		return nil, err
	}
	deadline := time.After(timeout)
	for {
		status, err := c.Playgrounds.Status(ctx, id)
		if err != nil {
			return nil, err
		}
		ready, pendingReason := fibe.PlaygroundStatusMatchesWaitTarget(status, target, readiness)
		if progress != nil {
			progress(playgroundWaitProgressText(status, target, pendingReason))
		}
		if ready {
			return c.Playgrounds.Get(ctx, id)
		}
		if status.Status == "error" || status.Status == "failed" || status.Status == "destroyed" {
			return nil, fibe.NewPlaygroundTerminalStateError(status)
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-deadline:
			if pendingReason == "" {
				pendingReason = fmt.Sprintf("last status: %s", status.Status)
			}
			return nil, fmt.Errorf("timeout after %s — %s", timeout, pendingReason)
		case <-time.After(interval):
		}
	}
}

func playgroundWaitProgressText(status *fibe.PlaygroundStatus, target string, pendingReason string) string {
	if status == nil {
		return "unknown"
	}
	text := status.Status
	if status.Status == target && pendingReason != "" {
		text += " (" + pendingReason + ")"
	}
	return text
}
