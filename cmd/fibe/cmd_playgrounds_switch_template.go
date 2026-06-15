package main

import (
	"fmt"
	"time"

	"github.com/fibegg/sdk/fibe"
	"github.com/spf13/cobra"
)

func pgSwitchTemplateCmd() *cobra.Command {
	var (
		template        string
		templateVersion int64
		templateBody    string
		templateName    string
		vars            []string
		regenerate      []string
		confirmWarnings bool
		preview         bool
		wait            bool
		waitTimeout     time.Duration
		responseMode    string
		yes             bool
	)
	cmd := &cobra.Command{
		Use:   "switch-template <id-or-name>",
		Short: "Switch a playground to another template version",
		Long: `Preview or apply a template switch for a deployed playground.

The playground must currently come from a template-backed playspec.

EXAMPLES:
  fibe playgrounds switch-template staging --template billing-app --preview
  fibe pg switch-template staging --template-version 912 --yes --wait
  fibe pg switch-template staging --template-body @template.yml --template-name billing-branch --yes`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			params := &fibe.PlaygroundTemplateSwitchParams{PlaygroundIdentifier: args[0]}
			if err := applyFromFile(params); err != nil {
				return err
			}
			if preview {
				params.Mode = "preview"
			} else if params.Mode == "" {
				params.Mode = "apply"
			}
			if template != "" {
				params.TemplateIdentifier = template
			}
			if templateVersion > 0 {
				params.TemplateVersionID = templateVersion
			}
			if cmd.Flags().Changed("template-body") {
				params.TemplateBody = normalizeTemplateBodyValue(resolveStringValue(templateBody))
			}
			if templateName != "" {
				params.TemplateName = templateName
			}
			if len(vars) > 0 {
				parsed, err := parseKeyValueFlags(vars)
				if err != nil {
					return err
				}
				params.Variables = parsed
			}
			if len(regenerate) > 0 {
				params.RegenerateVariables = regenerate
			}
			if cmd.Flags().Changed("confirm-warnings") {
				params.ConfirmWarnings = confirmWarnings
			}
			params.Wait = wait
			if cmd.Flags().Changed("wait-timeout") {
				params.WaitTimeoutSeconds = int64(waitTimeout / time.Second)
			}
			if responseMode != "" {
				params.ResponseMode = responseMode
			}
			if params.Mode != "preview" {
				if err := confirmDestructive(fmt.Sprintf("Switch playground %s to a different template", args[0]), yes); err != nil {
					return err
				}
			}
			var progress *statusLine
			c := newClient()
			if params.Mode != "preview" {
				action := "switching template for playground " + args[0]
				progress = newStatusLine(cmd.ErrOrStderr(), statusLineOptions{})
				progress.Start(action + "...")
				defer progress.Stop()
				c = newClient(fibe.WithProgress(progress.Progress(action)))
			}
			result, err := c.SwitchPlaygroundTemplate(ctx(), params)
			if progress != nil {
				progress.Stop()
			}
			if err != nil {
				return err
			}
			outputJSON(result)
			return nil
		},
	}
	cmd.Flags().StringVar(&template, "template", "", "Template ID or name; omitted with --template-version or --template-body")
	cmd.Flags().Int64Var(&templateVersion, "template-version", 0, "Exact target template version ID")
	cmd.Flags().StringVar(&templateBody, "template-body", "", "Template YAML body or @path")
	cmd.Flags().StringVar(&templateName, "template-name", "", "Name for a newly created template when --template-body creates one")
	cmd.Flags().StringArrayVar(&vars, "var", nil, "Template variable override as key=value (repeatable)")
	cmd.Flags().StringArrayVar(&regenerate, "regenerate", nil, "Random variable to regenerate (repeatable)")
	cmd.Flags().BoolVar(&confirmWarnings, "confirm-warnings", false, "Confirm risky switch warnings")
	cmd.Flags().BoolVar(&preview, "preview", false, "Preview without applying")
	cmd.Flags().BoolVar(&wait, "wait", true, "Wait for rollout in apply mode")
	cmd.Flags().DurationVar(&waitTimeout, "wait-timeout", 3*time.Minute, "Maximum rollout wait")
	cmd.Flags().StringVar(&responseMode, "response-mode", "", "Response mode passed to the server")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip interactive confirmation")
	return cmd
}
