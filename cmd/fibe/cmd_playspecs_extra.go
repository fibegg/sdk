package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/fibegg/sdk/fibe"
	"github.com/spf13/cobra"
)

// cmd_playspecs_extra.go registers subcommands that close the
// cmd/fibe ↔ svc_playspecs.go parity gap: mounted files + registry credentials.
// These are wired into the playspecs parent command via initPlayspecExtras in
// init.

func init() {
	initPlayspecExtras = func(parent *cobra.Command) {
		parent.AddCommand(
			psAddMountedFileCmd(),
			psUpdateMountedFileCmd(),
			psRemoveMountedFileCmd(),
			psAddRegistryCredentialCmd(),
			psRemoveRegistryCredentialCmd(),
		)
	}
}

// psAddMountedFileCmd: fibe playspecs add-mounted-file <id> --file PATH [--mount-path P] [--services a,b] [--readonly]
func psAddMountedFileCmd() *cobra.Command {
	var filePath, mountPath string
	var targetServices []string
	var readOnly bool
	cmd := &cobra.Command{
		Use:   "add-mounted-file <id>",
		Short: "Attach a file to a playspec",
		Long: `Attach a file to a playspec so deployments mount it into the target services.

REQUIRED FLAGS:
  --file         Local file to upload

OPTIONAL FLAGS:
  --mount-path   Path inside the target container (default: /<filename>)
  --services     Comma-separated list of services to mount into (default: all)
  --readonly     Mount as read-only

EXAMPLES:
  fibe playspecs add-mounted-file 42 --file ./prod.env --mount-path /app/.env
  fibe playspecs add-mounted-file 42 --file ./secret.json --services api,worker --readonly`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, _ := strconv.ParseInt(args[0], 10, 64)
			if filePath == "" {
				return fmt.Errorf("required flag --file not set")
			}
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
			if err := c.Playspecs.AddMountedFile(ctx(), id, bytes.NewReader(data), filename, p); err != nil {
				return err
			}
			outputJSON(map[string]any{"id": id, "filename": filename, "ok": true})
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
		Use:   "update-mounted-file <id>",
		Short: "Update metadata on a playspec mounted file",
		Long: `Update metadata (mount path, target services, readonly) on an existing mounted file.

REQUIRED FLAGS:
  --filename    Name of the existing mounted file

OPTIONAL FLAGS:
  --mount-path   New mount path
  --services     New target services list
  --readonly     Mount as read-only

EXAMPLES:
  fibe playspecs update-mounted-file 42 --filename prod.env --mount-path /etc/config.env`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, _ := strconv.ParseInt(args[0], 10, 64)
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
			if err := c.Playspecs.UpdateMountedFile(ctx(), id, p); err != nil {
				return err
			}
			outputJSON(map[string]any{"id": id, "filename": filename, "ok": true})
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
		Use:   "remove-mounted-file <id>",
		Short: "Remove a playspec mounted file",
		Long: `Remove a mounted file from a playspec.

REQUIRED FLAGS:
  --filename    Name of the file to remove

EXAMPLES:
  fibe playspecs remove-mounted-file 42 --filename prod.env`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, _ := strconv.ParseInt(args[0], 10, 64)
			if filename == "" {
				return fmt.Errorf("required flag --filename not set")
			}
			c := newClient()
			if err := c.Playspecs.RemoveMountedFile(ctx(), id, filename); err != nil {
				return err
			}
			outputJSON(map[string]any{"id": id, "filename": filename, "removed": true})
			return nil
		},
	}
	cmd.Flags().StringVar(&filename, "filename", "", "Filename to remove (required)")
	return cmd
}

func psAddRegistryCredentialCmd() *cobra.Command {
	var registryType, registryURL, username, secret string
	cmd := &cobra.Command{
		Use:   "add-registry-credential <id>",
		Short: "Attach a docker registry credential to a playspec",
		Long: `Attach a docker registry credential to a playspec so image pulls can authenticate.

REQUIRED FLAGS:
  --registry-type   Registry type (docker, ghcr, ...)
  --registry-url    Registry URL
  --username        Registry username
  --secret          Registry password/token

EXAMPLES:
  fibe playspecs add-registry-credential 42 --registry-type docker --registry-url docker.io --username u --secret s`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, _ := strconv.ParseInt(args[0], 10, 64)
			p := &fibe.RegistryCredentialParams{
				RegistryType: registryType, RegistryURL: registryURL,
				Username: username, Secret: secret,
			}
			if p.RegistryType == "" || p.RegistryURL == "" || p.Username == "" || p.Secret == "" {
				return fmt.Errorf("registry-type, registry-url, username, and secret are required")
			}
			c := newClient()
			result, err := c.Playspecs.AddRegistryCredential(ctx(), id, p)
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
	var credentialID string
	cmd := &cobra.Command{
		Use:   "remove-registry-credential <id>",
		Short: "Detach a docker registry credential from a playspec",
		Long: `Detach a docker registry credential from a playspec.

REQUIRED FLAGS:
  --credential-id   ID of the credential to remove

		EXAMPLES:
	  fibe playspecs remove-registry-credential 42 --credential-id 7`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, _ := strconv.ParseInt(args[0], 10, 64)
			if credentialID == "" {
				return fmt.Errorf("required flag --credential-id not set")
			}
			c := newClient()
			if err := c.Playspecs.RemoveRegistryCredential(ctx(), id, credentialID); err != nil {
				return err
			}
			outputJSON(map[string]any{"id": id, "credential_id": credentialID, "removed": true})
			return nil
		},
	}
	cmd.Flags().StringVar(&credentialID, "credential-id", "", "Credential ID to remove (required)")
	return cmd
}
