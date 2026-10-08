package main

import (
	"fmt"
	"github.com/fibegg/sdk/fibe"
	"github.com/spf13/cobra"
	"strconv"
)

func envPackID(raw string) (int64, error) {
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("expected a positive numeric ID")
	}
	return id, nil
}
func envPacksCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "env-packs", Aliases: []string{"env-pack"}, Short: "Manage readable ENV packs and ordered target attachments", Long: "ENV packs contain ordinary readable values. Use --from-file JSON/YAML for values and attachment lists. Omitted/null attachments preserve existing references; [] detaches every pack. Changes apply on the next explicit runtime operation."}
	cmd.AddCommand(envPackListCmd(), envPackGetCmd(), envPackCreateCmd(), envPackUpdateCmd(), envPackDeleteCmd(), envPackAttachmentsCmd())
	return cmd
}
func envPackListCmd() *cobra.Command {
	return &cobra.Command{Use: "list", Short: "List packs in the authenticated owner context", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, args []string) error {
		out, err := newClient().EnvPacks.List(ctx(), &fibe.EnvPackListParams{Page: flagPage, PerPage: flagPerPage})
		if err != nil {
			return err
		}
		outputJSON(out)
		return nil
	}}
}
func envPackGetCmd() *cobra.Command {
	return &cobra.Command{Use: "get <id>", Short: "Read a pack and its ordinary values", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		id, err := envPackID(args[0])
		if err != nil {
			return err
		}
		out, err := newClient().EnvPacks.Get(ctx(), id)
		if err != nil {
			return err
		}
		outputJSON(out)
		return nil
	}}
}
func envPackCreateCmd() *cobra.Command {
	var name string
	cmd := &cobra.Command{Use: "create", Short: "Create a pack; values come from --from-file", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, args []string) error {
		p := &fibe.EnvPackCreateParams{Env: map[string]string{}}
		if err := applyFromFile(p); err != nil {
			return err
		}
		if cmd.Flags().Changed("name") {
			p.Name = name
		}
		out, err := newClient().EnvPacks.Create(ctx(), p)
		if err != nil {
			return err
		}
		outputJSON(out)
		return nil
	}}
	cmd.Flags().StringVar(&name, "name", "", "Pack name")
	return cmd
}
func envPackUpdateCmd() *cobra.Command {
	var name string
	var version int64
	cmd := &cobra.Command{Use: "update <id>", Short: "Update a pack using --from-file; empty values remain empty", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		id, err := envPackID(args[0])
		if err != nil {
			return err
		}
		p := &fibe.EnvPackUpdateParams{}
		if err := applyFromFile(p); err != nil {
			return err
		}
		if cmd.Flags().Changed("name") {
			p.Name = &name
		}
		if cmd.Flags().Changed("version") {
			p.Version = &version
		}
		out, err := newClient().EnvPacks.Update(ctx(), id, p)
		if err != nil {
			return err
		}
		outputJSON(out)
		return nil
	}}
	cmd.Flags().StringVar(&name, "name", "", "Pack name")
	cmd.Flags().Int64Var(&version, "version", 0, "Expected pack version")
	return cmd
}
func envPackDeleteCmd() *cobra.Command {
	var yes bool
	cmd := &cobra.Command{Use: "delete <id>", Short: "Soft-delete a pack; applied runtime snapshots remain recoverable", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		id, err := envPackID(args[0])
		if err != nil {
			return err
		}
		if err := confirmDestructive("Delete ENV pack", yes); err != nil {
			return err
		}
		return newClient().EnvPacks.Delete(ctx(), id)
	}}
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip the deletion prompt")
	return cmd
}
func envPackAttachmentsCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "attachments", Short: "Manage attachments on a Playground or Spec"}
	cmd.AddCommand(&cobra.Command{Use: "get <playgrounds|specs> <target-id>", Args: cobra.ExactArgs(2), RunE: func(cmd *cobra.Command, args []string) error {
		id, err := envPackID(args[1])
		if err != nil {
			return err
		}
		out, err := newClient().EnvPacks.Attachments(ctx(), args[0], id)
		if err != nil {
			return err
		}
		outputJSON(out)
		return nil
	}})
	cmd.AddCommand(&cobra.Command{Use: "set <playgrounds|specs> <target-id>", Short: "Replace the ordered list from --from-file {env_pack_attachments:[...]}; [] detaches all", Args: cobra.ExactArgs(2), RunE: func(cmd *cobra.Command, args []string) error {
		id, err := envPackID(args[1])
		if err != nil {
			return err
		}
		p := &fibe.EnvPackAttachmentsParams{}
		if err := applyFromFile(p); err != nil {
			return err
		}
		out, err := newClient().EnvPacks.ReplaceAttachments(ctx(), args[0], id, p)
		if err != nil {
			return err
		}
		outputJSON(out)
		return nil
	}})
	for _, action := range []string{"reorder", "detach", "retarget", "renew"} {
		cmd.AddCommand(envPackAttachmentActionCmd(action))
	}
	return cmd
}
func envPackAttachmentActionCmd(action string) *cobra.Command {
	var order []int64
	var services []string
	var all bool
	use := action + " <playgrounds|specs> <target-id>"
	argc := 2
	if action != "reorder" {
		use += " <pack-id>"
		argc = 3
	}
	cmd := &cobra.Command{Use: use, Short: action + " selected attachments while preserving other grants", Args: cobra.ExactArgs(argc), RunE: func(cmd *cobra.Command, args []string) error {
		targetID, err := envPackID(args[1])
		if err != nil {
			return err
		}
		c := newClient()
		current, err := c.EnvPacks.Attachments(ctx(), args[0], targetID)
		if err != nil {
			return err
		}
		selected := int64(0)
		if argc == 3 {
			selected, err = envPackID(args[2])
			if err != nil {
				return err
			}
		}
		rows := make([]fibe.EnvPackAttachmentInput, 0, len(current.EnvPackAttachments))
		found := false
		for _, row := range current.EnvPackAttachments {
			var names *[]string
			if row.ServiceNames != nil {
				copyNames := append([]string{}, row.ServiceNames...)
				names = &copyNames
			}
			input := fibe.EnvPackAttachmentInput{EnvPackID: row.EnvPackID, ServiceNames: names}
			if row.EnvPackID == selected {
				found = true
				switch action {
				case "renew":
					out, err := c.EnvPacks.RenewGrant(ctx(), args[0], targetID, row.ID)
					if err != nil {
						return err
					}
					outputJSON(out)
					return nil
				case "detach":
					continue
				case "retarget":
					if all == cmd.Flags().Changed("services") {
						return fmt.Errorf("retarget requires exactly one of --all-services or --services")
					}
					if all {
						input.ServiceNames = nil
					} else {
						input.ServiceNames = &services
					}
				}
			}
			rows = append(rows, input)
		}
		if argc == 3 && !found {
			return fmt.Errorf("pack is not attached to this target")
		}
		if action == "reorder" {
			if len(order) != len(rows) {
				return fmt.Errorf("--order must contain each attached pack ID exactly once")
			}
			ordered := make([]fibe.EnvPackAttachmentInput, 0, len(rows))
			seen := map[int64]bool{}
			for _, id := range order {
				if seen[id] {
					return fmt.Errorf("duplicate pack in --order")
				}
				seen[id] = true
				found = false
				for _, row := range rows {
					if row.EnvPackID == id {
						ordered = append(ordered, row)
						found = true
						break
					}
				}
				if !found {
					return fmt.Errorf("--order contains an unattached pack")
				}
			}
			rows = ordered
		}
		out, err := c.EnvPacks.ReplaceAttachments(ctx(), args[0], targetID, &fibe.EnvPackAttachmentsParams{Attachments: &rows})
		if err != nil {
			return err
		}
		outputJSON(out)
		return nil
	}}
	if action == "reorder" {
		cmd.Flags().Int64SliceVar(&order, "order", nil, "Every attached pack ID in the desired order")
	}
	if action == "retarget" {
		cmd.Flags().StringSliceVar(&services, "services", nil, "Selected service names (nonempty)")
		cmd.Flags().BoolVar(&all, "all-services", false, "Apply to all current and future services")
	}
	return cmd
}
