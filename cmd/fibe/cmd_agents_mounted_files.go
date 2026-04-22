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

func agAddMountedFileCmd() *cobra.Command {
	var filePath, mountPath string
	var artefactID int64
	var targetServices []string
	var readOnly bool
	cmd := &cobra.Command{
		Use:   "add-mounted-file <id>",
		Short: "Attach a file or Artefact snapshot to an agent",
		Long: `Attach a mounted file to an agent.

Use --file to upload a local file, or --artefact-id to snapshot an existing Artefact.

EXAMPLES:
  fibe agents add-mounted-file 5 --file ./prod.env --mount-path '%{agent_data}/.env'
  fibe agents add-mounted-file 5 --artefact-id 123 --mount-path '%{workspace}/docs/context.md'`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, _ := strconv.ParseInt(args[0], 10, 64)
			if filePath == "" && artefactID == 0 {
				return fmt.Errorf("one of --file or --artefact-id is required")
			}
			if filePath != "" && artefactID != 0 {
				return fmt.Errorf("use only one of --file or --artefact-id")
			}
			readOnlyPtr := &readOnly
			p := &fibe.MountedFileParams{MountPath: mountPath, TargetServices: targetServices, ReadOnly: readOnlyPtr}
			c := newClient()
			var agent *fibe.Agent
			var err error
			if artefactID != 0 {
				agent, err = c.Agents.AddMountedFileFromArtefact(ctx(), id, artefactID, p)
			} else {
				data, readErr := os.ReadFile(filePath)
				if readErr != nil {
					return fmt.Errorf("read %s: %w", filePath, readErr)
				}
				agent, err = c.Agents.AddMountedFile(ctx(), id, bytes.NewReader(data), filepath.Base(filePath), p)
			}
			if err != nil {
				return err
			}
			outputJSON(agent)
			return nil
		},
	}
	cmd.Flags().StringVar(&filePath, "file", "", "Local file to upload")
	cmd.Flags().Int64Var(&artefactID, "artefact-id", 0, "Artefact ID to snapshot")
	cmd.Flags().StringVar(&mountPath, "mount-path", "", "Path inside the agent container")
	cmd.Flags().StringSliceVar(&targetServices, "services", nil, "Target services")
	cmd.Flags().BoolVar(&readOnly, "readonly", true, "Mount as read-only")
	return cmd
}

func agUpdateMountedFileCmd() *cobra.Command {
	var filename, mountPath string
	var targetServices []string
	var readOnly bool
	cmd := &cobra.Command{
		Use:   "update-mounted-file <id>",
		Short: "Update an agent mounted file",
		Long: `Update metadata on an existing agent mounted file.

EXAMPLES:
  fibe agents update-mounted-file 5 --filename prod.env --mount-path '%{agent_data}/config/prod.env'`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, _ := strconv.ParseInt(args[0], 10, 64)
			if filename == "" {
				return fmt.Errorf("required flag --filename not set")
			}
			p := &fibe.MountedFileUpdateParams{Filename: filename, MountPath: mountPath, TargetServices: targetServices}
			if cmd.Flags().Changed("readonly") {
				p.ReadOnly = &readOnly
			}
			agent, err := newClient().Agents.UpdateMountedFile(ctx(), id, p)
			if err != nil {
				return err
			}
			outputJSON(agent)
			return nil
		},
	}
	cmd.Flags().StringVar(&filename, "filename", "", "Existing filename")
	cmd.Flags().StringVar(&mountPath, "mount-path", "", "New path inside the agent container")
	cmd.Flags().StringSliceVar(&targetServices, "services", nil, "Target services")
	cmd.Flags().BoolVar(&readOnly, "readonly", true, "Mount as read-only")
	return cmd
}

func agRemoveMountedFileCmd() *cobra.Command {
	var filename string
	cmd := &cobra.Command{
		Use:   "remove-mounted-file <id>",
		Short: "Remove an agent mounted file",
		Long: `Remove a mounted file from an agent.

EXAMPLES:
  fibe agents remove-mounted-file 5 --filename prod.env`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, _ := strconv.ParseInt(args[0], 10, 64)
			if filename == "" {
				return fmt.Errorf("required flag --filename not set")
			}
			agent, err := newClient().Agents.RemoveMountedFile(ctx(), id, filename)
			if err != nil {
				return err
			}
			outputJSON(agent)
			return nil
		},
	}
	cmd.Flags().StringVar(&filename, "filename", "", "Filename to remove")
	return cmd
}
