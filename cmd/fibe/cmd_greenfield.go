package main

import (
	"context"
	"fmt"
	"os"
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
		templateID             int64
		templateIDTypoTempalte int64
		templateIDTypoTemlate  int64
		version                string
		templateBody           string
		marqueeID              string
		marqueeIDTypoMarque    int64
		vars                   []string
		waitTimeout            time.Duration
	)

	cmd := &cobra.Command{
		Use:   "greenfield",
		Short: "Create a new greenfield app from the platform template flow",
		Long: `Create a new app repository, Prop, app-owned template version, and deployed playground.

The command calls the Rails greenfield endpoint, waits for the playground to run,
and links the local playground checkout into /app/playground by default.

Examples:
  fibe greenfield --name my-app --template-id 347
  fibe greenfield --name my-app -f my-template.yml
  fibe greenfield --name my-app --template-body 'services:\n  web:\n    image: nginx'`,
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
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
			selectedTemplateID, templateIDChanged := selectedGreenfieldTemplateID(cmd, templateID, templateIDTypoTempalte, templateIDTypoTemlate)
			if templateIDChanged && selectedTemplateID > 0 {
				id := selectedTemplateID
				params.TemplateID = &id
			}
			if cmd.Flags().Changed("version") {
				if params.TemplateID == nil {
					return fmt.Errorf("--version requires --template-id")
				}
				params.Version = version
			}
			if cmd.Flags().Changed("template-body") {
				params.TemplateBody = normalizeTemplateBodyValue(resolveStringValue(templateBody))
			}
			if cmd.Flags().Changed("marquee-id") && marqueeID != "" {
				params.MarqueeIdentifier = marqueeID
			} else if cmd.Flags().Changed("marque-id") && marqueeIDTypoMarque > 0 {
				id := marqueeIDTypoMarque
				params.MarqueeID = &id
			} else if params.MarqueeID == nil && params.MarqueeIdentifier == "" {
				id, err := marqueeIDFromEnv()
				if err != nil {
					return err
				}
				params.MarqueeID = &id
			}
			if cmd.Flags().Changed("var") && len(vars) > 0 {
				parsed, err := parseGreenfieldVars(vars)
				if err != nil {
					return err
				}
				params.Variables = parsed
			}
			if params.Name == "" {
				return fmt.Errorf("required field 'name' not set")
			}

			result, err := c.Greenfield.Create(ctx(), params)
			if err != nil {
				return err
			}
			if result.Playground == nil || result.Playground.ID == 0 {
				return fmt.Errorf("greenfield create did not return a playground id")
			}

			fmt.Fprintf(cmd.ErrOrStderr(), "waiting for playground %d to reach running...\n", result.Playground.ID)
			pg, err := waitForPlayground(ctx(), c, result.Playground.ID, "running", waitTimeout, 3*time.Second, func(status string) {
				fmt.Fprintf(cmd.ErrOrStderr(), "status: %s\n", status)
			})
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
	cmd.Flags().StringVar(&name, "name", "", "Repository/app name (required, must be unique)")
	cmd.Flags().StringVar(&gitProvider, "git-provider", "gitea", "Destination git provider: gitea or github (optional, default: gitea)")
	cmd.Flags().Int64Var(&templateID, "template-id", 0, "Template to use (optional, default: base template)")
	cmd.Flags().StringVar(&version, "version", "", "Template version tag or number when --template-id is used (e.g. v1, optional, default: latest version)")
	cmd.Flags().StringVar(&templateBody, "template-body", "", "Template YAML body to use directly (optional)")
	cmd.Flags().StringVar(&marqueeID, "marquee-id", "", "Target marquee ID or name (optional, default: current Marquee)")
	cmd.Flags().StringSliceVar(&vars, "var", nil, "Set template variables (e.g., --var app_name=Tower, optional)")
	cmd.Flags().DurationVar(&waitTimeout, "wait-timeout", 10*time.Minute, "Maximum time to wait for the playground to reach running (optional, default 10m0s)")
	cmd.Flags().Int64Var(&templateIDTypoTempalte, "tempalte-id", 0, "Alias for --template-id")
	cmd.Flags().Int64Var(&templateIDTypoTemlate, "temlate-id", 0, "Alias for --template-id")
	cmd.Flags().Int64Var(&marqueeIDTypoMarque, "marque-id", 0, "Alias for --marquee-id")
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

func selectedGreenfieldTemplateID(cmd *cobra.Command, templateID, tempalteID, temlateID int64) (int64, bool) {
	switch {
	case cmd.Flags().Changed("template-id"):
		return templateID, true
	case cmd.Flags().Changed("tempalte-id"):
		return tempalteID, true
	case cmd.Flags().Changed("temlate-id"):
		return temlateID, true
	default:
		return 0, false
	}
}

func marqueeIDFromEnv() (int64, error) {
	raw := strings.TrimSpace(os.Getenv("FIBE_MARQUEE_ID"))
	if raw == "" {
		return 0, fmt.Errorf("--marquee-id is required when the current Marquee is not available (FIBE_MARQUEE_ID is not set)")
	}
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("FIBE_MARQUEE_ID must be a positive integer")
	}
	return id, nil
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
	deadline := time.After(timeout)
	for {
		status, err := c.Playgrounds.Status(ctx, id)
		if err != nil {
			return nil, err
		}
		if progress != nil {
			progress(status.Status)
		}
		if status.Status == target {
			return c.Playgrounds.Get(ctx, id)
		}
		if status.Status == "error" || status.Status == "failed" || status.Status == "destroyed" {
			return nil, fmt.Errorf("%s", fibe.PlaygroundTerminalStateError(status))
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-deadline:
			return nil, fmt.Errorf("timeout after %s — last status: %s", timeout, status.Status)
		case <-time.After(interval):
		}
	}
}
