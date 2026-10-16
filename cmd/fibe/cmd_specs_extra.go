package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/fibegg/sdk/fibe"
	"github.com/spf13/cobra"
)

func init() {
	initSpecExtras = func(parent *cobra.Command) {
		parent.AddCommand(
			psAddMountedFileCmd(),
			psUpdateMountedFileCmd(),
			psRemoveMountedFileCmd(),
			psAddRegistryCredentialCmd(),
			psRemoveRegistryCredentialCmd(),
			psSwitchTemplateCmd(),
		)
	}
}

// psAddMountedFileCmd: fibe specs add-mounted-file <id-or-name> --file PATH [--mount-path P] [--services a,b] [--readonly]
func psAddMountedFileCmd() *cobra.Command {
	var filePath, mountPath string
	var targetServices []string
	var readOnly bool
	cmd := &cobra.Command{
		Use:   "add-mounted-file <id-or-name>",
		Short: "Attach a file to a spec",
		Long: `Attach a file to a spec so deployments mount it into the target services.

REQUIRED FLAGS:
  --file         Local file to upload

OPTIONAL FLAGS:
  --mount-path   Path inside the target container (default: /<filename>)
  --services     Comma-separated list of services to mount into (default: all)
  --readonly     Mount as read-only

EXAMPLES:
  fibe specs add-mounted-file 42 --file ./prod.env --mount-path /app/.env
  fibe specs add-mounted-file 42 --file ./secret.json --services api,worker --readonly`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if filePath == "" {
				return fmt.Errorf("required flag --file not set")
			}
			// #nosec G304 -- filePath is an explicit local input requested by the CLI user.
			data, err := os.ReadFile(filePath)
			if err != nil {
				return fmt.Errorf("read %s: %w", filePath, err)
			}
			filename := filepath.Base(filePath)
			p := &fibe.MountedFileParams{MountPath: mountPath, TargetServices: targetServices}
			if cmd.Flags().Changed("readonly") {
				p.ReadOnly = &readOnly
			}
			c := newClient()
			if err := c.Specs.AddMountedFileByIdentifier(ctx(), args[0], bytes.NewReader(data), filename, p); err != nil {
				return err
			}
			outputJSON(map[string]any{"id": args[0], "filename": filename, "ok": true})
			return nil
		},
	}
	cmd.Flags().StringVar(&filePath, "file", "", "Local file to upload (required)")
	cmd.Flags().StringVar(&mountPath, "mount-path", "", "Path inside the container")
	cmd.Flags().StringSliceVar(&targetServices, "services", nil, "Target services")
	cmd.Flags().BoolVar(&readOnly, "readonly", false, "Mount as read-only")
	return cmd
}

func psUpdateMountedFileCmd() *cobra.Command {
	var filename, mountPath string
	var targetServices []string
	var readOnly bool
	cmd := &cobra.Command{
		Use:   "update-mounted-file <id-or-name>",
		Short: "Update metadata on a spec mounted file",
		Long: `Update metadata (mount path, target services, readonly) on an existing mounted file.

REQUIRED FLAGS:
  --filename    Name of the existing mounted file

OPTIONAL FLAGS:
  --mount-path   New mount path
  --services     New target services list
  --readonly     Mount as read-only

EXAMPLES:
  fibe specs update-mounted-file 42 --filename prod.env --mount-path /etc/config.env`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if filename == "" {
				return fmt.Errorf("required flag --filename not set")
			}
			p := &fibe.MountedFileUpdateParams{
				Filename:       filename,
				MountPath:      mountPath,
				TargetServices: targetServices,
			}
			if cmd.Flags().Changed("readonly") {
				p.ReadOnly = &readOnly
			}
			c := newClient()
			if err := c.Specs.UpdateMountedFileByIdentifier(ctx(), args[0], p); err != nil {
				return err
			}
			outputJSON(map[string]any{"id": args[0], "filename": filename, "ok": true})
			return nil
		},
	}
	cmd.Flags().StringVar(&filename, "filename", "", "Existing filename (required)")
	cmd.Flags().StringVar(&mountPath, "mount-path", "", "New mount path")
	cmd.Flags().StringSliceVar(&targetServices, "services", nil, "New target services")
	cmd.Flags().BoolVar(&readOnly, "readonly", false, "Mount as read-only")
	return cmd
}

func psRemoveMountedFileCmd() *cobra.Command {
	var filename string
	cmd := &cobra.Command{
		Use:   "remove-mounted-file <id-or-name>",
		Short: "Remove a spec mounted file",
		Long: `Remove a mounted file from a spec.

REQUIRED FLAGS:
  --filename    Name of the file to remove

EXAMPLES:
  fibe specs remove-mounted-file 42 --filename prod.env`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if filename == "" {
				return fmt.Errorf("required flag --filename not set")
			}
			c := newClient()
			if err := c.Specs.RemoveMountedFileByIdentifier(ctx(), args[0], filename); err != nil {
				return err
			}
			outputJSON(map[string]any{"id": args[0], "filename": filename, "removed": true})
			return nil
		},
	}
	cmd.Flags().StringVar(&filename, "filename", "", "Filename to remove (required)")
	return cmd
}

func psAddRegistryCredentialCmd() *cobra.Command {
	var registryType, registryURL, username, secret string
	cmd := &cobra.Command{
		Use:   "add-registry-credential <id-or-name>",
		Short: "Attach a docker registry credential to a spec",
		Long: `Attach a docker registry credential to a spec so image pulls can authenticate.

REQUIRED FLAGS:
  --registry-type   Registry type (docker, ghcr, ...)
  --registry-url    Registry URL
  --username        Registry username
  --secret          Registry password/token

EXAMPLES:
  fibe specs add-registry-credential 42 --registry-type docker --registry-url docker.io --username u --secret s`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			p := &fibe.RegistryCredentialParams{
				RegistryType: registryType, RegistryURL: registryURL,
				Username: username, Secret: secret,
			}
			if p.RegistryType == "" || p.RegistryURL == "" || p.Username == "" || p.Secret == "" {
				return fmt.Errorf("registry-type, registry-url, username, and secret are required")
			}
			c := newClient()
			result, err := c.Specs.AddRegistryCredentialByIdentifier(ctx(), args[0], p)
			if err != nil {
				return err
			}
			outputJSON(result)
			return nil
		},
	}
	cmd.Flags().StringVar(&registryType, "registry-type", "", "Registry type (required)")
	cmd.Flags().StringVar(&registryURL, "registry-url", "", "Registry URL (required)")
	cmd.Flags().StringVar(&username, "username", "", "Registry username (required)")
	cmd.Flags().StringVar(&secret, "secret", "", "Registry password/token (required)")
	return cmd
}

func psRemoveRegistryCredentialCmd() *cobra.Command {
	var credential string
	cmd := &cobra.Command{
		Use:   "remove-registry-credential <id-or-name>",
		Short: "Detach a docker registry credential from a spec",
		Long: `Detach a docker registry credential from a spec.

REQUIRED FLAGS:
  --credential   ID, registry URL, username, or registry_type/registry_url/username selector

		EXAMPLES:
	  fibe specs remove-registry-credential 42 --credential 7`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if credential == "" {
				return fmt.Errorf("required flag --credential not set")
			}
			c := newClient()
			credentialID, err := resolveCredentialID(c, args[0], credential)
			if err != nil {
				return err
			}
			if err := c.Specs.RemoveRegistryCredentialByIdentifier(ctx(), args[0], credentialID); err != nil {
				return err
			}
			outputJSON(map[string]any{"id": args[0], "credential_id": credentialID, "removed": true})
			return nil
		},
	}
	cmd.Flags().StringVar(&credential, "credential", "", "Credential ID or unique selector to remove (required)")
	return cmd
}

func psSwitchTemplateCmd() *cobra.Command {
	var targetID int64
	var vars []string
	var regenerate []string
	var confirmWarnings bool
	var preview bool

	cmd := &cobra.Command{
		Use:   "switch-template <id-or-name>",
		Short: "Preview or apply a template switch for a spec",
		Long: `Switch a template-backed spec to any readable template version in the same fork tree.

REQUIRED FLAGS:
  --template-version   Target template version ID

OPTIONAL FLAGS:
  --var key=value                Override a template variable (repeatable)
  --regenerate name              Regenerate a random template variable (repeatable)
  --confirm-warnings             Required when the preview reports risky changes
  --preview                      Only preview the switch

EXAMPLES:
  fibe specs switch-template 42 --template-version 123 --preview
  fibe ps switch-template 42 --template-version 123 --var app_name=demo --confirm-warnings`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if targetID == 0 {
				return fmt.Errorf("required flag --template-version not set")
			}
			parsedVars, err := parseKeyValueFlags(vars)
			if err != nil {
				return err
			}

			params := &fibe.SpecTemplateVersionSwitchParams{
				TargetTemplateVersionID: targetID,
				Variables:               parsedVars,
				RegenerateVariables:     regenerate,
				ConfirmWarnings:         confirmWarnings,
			}

			c := newClient()
			var result *fibe.SpecTemplateVersionSwitchResult
			if preview {
				result, err = c.Specs.PreviewTemplateVersionSwitchByIdentifier(ctx(), args[0], params)
			} else {
				result, err = c.Specs.SwitchTemplateVersionByIdentifier(ctx(), args[0], params)
			}
			if err != nil {
				return err
			}

			if effectiveOutput() != "table" {
				outputJSON(result)
				return nil
			}
			outputTemplateVersionSwitchTable(result)
			return nil
		},
	}

	cmd.Flags().Int64Var(&targetID, "template-version", 0, "Target template version ID")
	cmd.Flags().StringArrayVar(&vars, "var", nil, "Template variable override as key=value (repeatable)")
	cmd.Flags().StringArrayVar(&regenerate, "regenerate", nil, "Random variable to regenerate (repeatable)")
	cmd.Flags().BoolVar(&confirmWarnings, "confirm-warnings", false, "Confirm risky switch warnings")
	cmd.Flags().BoolVar(&preview, "preview", false, "Preview without applying")
	return cmd
}

func outputTemplateVersionSwitchTable(result *fibe.SpecTemplateVersionSwitchResult) {
	headers := []string{"FIELD", "VALUE"}
	rows := [][]string{
		{"From", formatTemplateVersionRef(result.FromTemplateVersion)},
		{"Target", formatTemplateVersionRef(result.TargetTemplateVersion)},
		{"Suggested Upgrade", strconv.FormatBool(result.SuggestedUpgrade)},
		{"Required Variables", strings.Join(templateSwitchVariableNames(result.RequiredVariables), ", ")},
		{"Warnings", strings.Join(templateSwitchWarningCodes(result.Warnings), ", ")},
		{"Rollout Playgrounds", strconv.Itoa(len(result.PlaygroundRolloutPlan.Rollout))},
		{"Unchanged Playgrounds", strconv.Itoa(len(result.PlaygroundRolloutPlan.Unchanged))},
		{"Blocked Playgrounds", strconv.Itoa(len(result.PlaygroundRolloutPlan.Blocked))},
		{"No-op", strconv.FormatBool(result.NoOp)},
	}
	outputTable(headers, rows)
}

func formatTemplateVersionRef(ref *fibe.TemplateVersionRef) string {
	if ref == nil {
		return ""
	}
	template := ""
	if ref.Template != nil {
		template = ref.Template.Name + " "
	}
	return fmt.Sprintf("%sv%s (%s)", template, fmtInt64Ptr(ref.Version), fmtInt64Ptr(ref.ID))
}

func templateSwitchVariableNames(vars []fibe.TemplateSwitchVariable) []string {
	names := make([]string, 0, len(vars))
	for _, variable := range vars {
		names = append(names, variable.Name)
	}
	return names
}

func templateSwitchWarningCodes(warnings []fibe.TemplateSwitchWarning) []string {
	codes := make([]string, 0, len(warnings))
	for _, warning := range warnings {
		codes = append(codes, warning.Code)
	}
	return codes
}
