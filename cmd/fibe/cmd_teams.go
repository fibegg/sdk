package main

import (
	"fmt"
	"strconv"

	"github.com/fibegg/sdk/fibe"
	"github.com/spf13/cobra"
)

func teamsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "teams",
		Short: "Manage teams and memberships",
		Long: `Manage Fibe teams — groups of users with shared resources.

Teams allow sharing playgrounds, playspecs, props, and other resources
with team members at different permission levels.

SUBCOMMANDS:
  list                              List all teams
  get <id>                          Show team details with members
  create                            Create a new team
  update <id>                       Update team name
  delete <id>                       Delete a team
  transfer <id>                     Transfer leadership
  invite <team-id>                  Invite a member
  accept <team-id> <id>             Accept an invitation
  decline <team-id> <id>            Decline an invitation
  update-member <team-id> <id>      Change a member's role
  remove-member <team-id> <id>      Remove a member
  leave <team-id>                   Leave a team
  resources <team-id>               List team resources
  contribute <team-id>              Share a resource with team
  remove-resource <team-id> <id>    Remove a contributed resource`,
	}
	cmd.AddCommand(
		teamListCmd(), teamGetCmd(), teamCreateCmd(), teamUpdateCmd(), teamDeleteCmd(),
		teamTransferCmd(), teamInviteCmd(), teamAcceptCmd(), teamDeclineCmd(),
		teamUpdateMemberCmd(), teamRemoveMemberCmd(),
		teamLeaveCmd(), teamResourcesCmd(), teamContributeCmd(), teamRemoveResourceCmd(),
	)
	return cmd
}

func teamListCmd() *cobra.Command {
	var query, name, sort string
	cmd := &cobra.Command{
		Use: "list", Short: "List all teams",
		Long: `List all teams accessible to the authenticated user.

FILTERS:
  -q, --query           Search across name (substring match)
  --name                Filter by name (substring match)

SORTING:
  --sort                Sort results. Format: {column}_{direction}
                        Columns: created_at, name
                        Direction: asc, desc
                        Default: created_at_desc

OUTPUT:
  Columns: ID, NAME, SLUG, MEMBERS
  Use --output json for full details.

EXAMPLES:
  fibe teams list
  fibe teams list -q "backend"
  fibe teams list --sort name_asc`,
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			params := &fibe.TeamListParams{}
			if query != "" {
				params.Q = query
			}
			if name != "" {
				params.Name = name
			}
			if sort != "" {
				params.Sort = sort
			}
			if flagPage > 0 {
				params.Page = flagPage
			}
			if flagPerPage > 0 {
				params.PerPage = flagPerPage
			}
			teams, err := c.Teams.List(ctx(), params)
			if err != nil {
				return err
			}
			if effectiveOutput() != "table" {
				outputJSON(teams)
				return nil
			}
			headers := []string{"ID", "NAME", "SLUG", "MEMBERS"}
			rows := make([][]string, len(teams.Data))
			for i, t := range teams.Data {
				rows[i] = []string{fmtInt64(t.ID), t.Name, t.Slug, fmtInt64(t.MembersCount)}
			}
			outputTable(headers, rows)
			return nil
		},
	}
	cmd.Flags().StringVarP(&query, "query", "q", "", "Search across name")
	cmd.Flags().StringVar(&name, "name", "", "Filter by name (substring)")
	cmd.Flags().StringVar(&sort, "sort", "", "Sort order (e.g. created_at_desc)")
	return cmd
}

func teamGetCmd() *cobra.Command {
	return &cobra.Command{
		Use: "get <id>", Short: "Show team details", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			id, _ := strconv.ParseInt(args[0], 10, 64)
			team, err := c.Teams.Get(ctx(), id)
			if err != nil {
				return err
			}
			outputJSON(team)
			return nil
		},
	}
}

func teamCreateCmd() *cobra.Command {
	var name string
	cmd := &cobra.Command{
		Use: "create", Short: "Create a new team",
		Long: "Create a new team.\n\nREQUIRED FLAGS:\n  --name   Team name\n\nEXAMPLES:\n  fibe teams create --name \"Backend Team\"",
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			params := &fibe.TeamCreateParams{}
			if err := applyFromFile(params); err != nil {
				return err
			}
			if cmd.Flags().Changed("name") {
				params.Name = name
			}

			if params.Name == "" {
				return fmt.Errorf("required field 'name' not set")
			}

			team, err := c.Teams.Create(ctx(), params)
			if err != nil {
				return err
			}
			fmt.Printf("Created team %d (%s)\n", team.ID, team.Name)
			return nil
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Team name (required)")
	return cmd
}

func teamUpdateCmd() *cobra.Command {
	var name string
	cmd := &cobra.Command{
		Use: "update <id>", Short: "Update team name", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			id, _ := strconv.ParseInt(args[0], 10, 64)
			params := &fibe.TeamUpdateParams{}
			if err := applyFromFile(params); err != nil {
				return err
			}
			if cmd.Flags().Changed("name") {
				params.Name = name
			}

			team, err := c.Teams.Update(ctx(), id, params)
			if err != nil {
				return err
			}
			fmt.Printf("Updated team %d (%s)\n", team.ID, team.Name)
			return nil
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "New name (required)")
	return cmd
}

func teamDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use: "delete <id>", Short: "Delete a team", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			id, _ := strconv.ParseInt(args[0], 10, 64)
			if err := c.Teams.Delete(ctx(), id); err != nil {
				return err
			}
			fmt.Printf("Team %d deleted\n", id)
			return nil
		},
	}
}

func teamTransferCmd() *cobra.Command {
	var newLeaderID int64
	cmd := &cobra.Command{
		Use: "transfer <id>", Short: "Transfer team leadership", Args: cobra.ExactArgs(1),
		Long: "Transfer team leadership to another member.\n\nREQUIRED FLAGS:\n  --to   Player ID of the new leader",
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			id, _ := strconv.ParseInt(args[0], 10, 64)
			_, err := c.Teams.TransferLeadership(ctx(), id, newLeaderID)
			if err != nil {
				return err
			}
			fmt.Printf("Leadership transferred to player %d\n", newLeaderID)
			return nil
		},
	}
	cmd.Flags().Int64Var(&newLeaderID, "to", 0, "New leader player ID (required)")
	cmd.MarkFlagRequired("to")
	return cmd
}

func teamInviteCmd() *cobra.Command {
	var username string
	cmd := &cobra.Command{
		Use: "invite <team-id>", Short: "Invite a member", Args: cobra.ExactArgs(1),
		Long: "Invite a user to join a team.\n\nREQUIRED FLAGS:\n  --username   Username to invite\n\nEXAMPLES:\n  fibe teams invite 1 --username johndoe",
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			teamID, _ := strconv.ParseInt(args[0], 10, 64)
			m, err := c.Teams.InviteMember(ctx(), teamID, username)
			if err != nil {
				return err
			}
			fmt.Printf("Invited %s (membership %d) — status: %s\n", username, m.ID, m.Status)
			return nil
		},
	}
	cmd.Flags().StringVar(&username, "username", "", "Username (required)")
	cmd.MarkFlagRequired("username")
	return cmd
}

func teamAcceptCmd() *cobra.Command {
	return &cobra.Command{
		Use: "accept <team-id> <membership-id>", Short: "Accept team invitation", Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			teamID, _ := strconv.ParseInt(args[0], 10, 64)
			mID, _ := strconv.ParseInt(args[1], 10, 64)
			_, err := c.Teams.AcceptInvite(ctx(), teamID, mID)
			if err != nil {
				return err
			}
			fmt.Println("Invitation accepted")
			return nil
		},
	}
}

func teamDeclineCmd() *cobra.Command {
	return &cobra.Command{
		Use: "decline <team-id> <membership-id>", Short: "Decline team invitation", Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			teamID, _ := strconv.ParseInt(args[0], 10, 64)
			mID, _ := strconv.ParseInt(args[1], 10, 64)
			_, err := c.Teams.DeclineInvite(ctx(), teamID, mID)
			if err != nil {
				return err
			}
			fmt.Println("Invitation declined")
			return nil
		},
	}
}

func teamLeaveCmd() *cobra.Command {
	return &cobra.Command{
		Use: "leave <team-id>", Short: "Leave a team", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			teamID, _ := strconv.ParseInt(args[0], 10, 64)
			if err := c.Teams.Leave(ctx(), teamID); err != nil {
				return err
			}
			fmt.Println("Left team")
			return nil
		},
	}
}

func teamResourcesCmd() *cobra.Command {
	return &cobra.Command{
		Use: "resources <team-id>", Short: "List team resources", Args: cobra.ExactArgs(1),
		Long: "List all resources shared with a team.\n\nEXAMPLES:\n  fibe teams resources 1",
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			teamID, _ := strconv.ParseInt(args[0], 10, 64)
			resources, err := c.Teams.ListResources(ctx(), teamID, nil)
			if err != nil {
				return err
			}
			if effectiveOutput() != "table" {
				outputJSON(resources)
				return nil
			}
			headers := []string{"ID", "TYPE", "RESOURCE_ID", "NAME", "PERMISSION"}
			rows := make([][]string, len(resources.Data))
			for i, r := range resources.Data {
				rows[i] = []string{fmtInt64(r.ID), r.ResourceType, fmtInt64(r.ResourceID), r.ResourceName, r.PermissionLevel}
			}
			outputTable(headers, rows)
			return nil
		},
	}
}

func teamContributeCmd() *cobra.Command {
	var resType string
	var resID int64
	var perm string
	cmd := &cobra.Command{
		Use: "contribute <team-id>", Short: "Share a resource with the team", Args: cobra.ExactArgs(1),
		Long: "Share a resource with a team at a specific permission level.\n\nREQUIRED FLAGS:\n  --type          Resource type (Playspec, Prop, Marquee, etc.)\n  --resource-id   Resource ID\n\nOPTIONAL FLAGS:\n  --permission    Permission level (read, write, admin — default: read)\n\nEXAMPLES:\n  fibe teams contribute 1 --type Playspec --resource-id 5 --permission write",
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			teamID, _ := strconv.ParseInt(args[0], 10, 64)
			params := &fibe.TeamResourceParams{}
			if err := applyFromFile(params); err != nil {
				return err
			}
			if cmd.Flags().Changed("type") {
				params.ResourceType = resType
			}
			if cmd.Flags().Changed("resource-id") {
				params.ResourceID = resID
			}
			if cmd.Flags().Changed("permission") {
				params.PermissionLevel = perm
			}

			if params.ResourceType == "" {
				return fmt.Errorf("required field 'type' not set")
			}
			if params.ResourceID == 0 {
				return fmt.Errorf("required field 'resource-id' not set")
			}

			r, err := c.Teams.ContributeResource(ctx(), teamID, params)
			if err != nil {
				return err
			}
			fmt.Printf("Contributed %s %d to team\n", r.ResourceType, r.ResourceID)
			return nil
		},
	}
	cmd.Flags().StringVar(&resType, "type", "", "Resource type (required)")
	cmd.Flags().Int64Var(&resID, "resource-id", 0, "Resource ID (required)")
	cmd.Flags().StringVar(&perm, "permission", "read", "Permission level")
	return cmd
}

func teamUpdateMemberCmd() *cobra.Command {
	var role string
	cmd := &cobra.Command{
		Use:   "update-member <team-id> <membership-id>",
		Short: "Change a team member's role",
		Long: `Update a team member's role.

The team owner cannot have their role changed — transfer leadership first.

REQUIRED FLAGS:
  --role    New role (e.g. member, admin)

EXAMPLES:
  fibe teams update-member 1 42 --role admin`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			teamID, _ := strconv.ParseInt(args[0], 10, 64)
			membershipID, _ := strconv.ParseInt(args[1], 10, 64)
			if role == "" {
				return fmt.Errorf("required field 'role' not set")
			}
			result, err := c.Teams.UpdateMember(ctx(), teamID, membershipID, role)
			if err != nil {
				return err
			}
			outputJSON(result)
			return nil
		},
	}
	cmd.Flags().StringVar(&role, "role", "", "New role (required)")
	return cmd
}

func teamRemoveMemberCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "remove-member <team-id> <membership-id>",
		Short: "Remove a member from a team",
		Long: `Remove a member from a team.

The team owner cannot be removed — transfer leadership first.

EXAMPLES:
  fibe teams remove-member 1 42`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			teamID, _ := strconv.ParseInt(args[0], 10, 64)
			membershipID, _ := strconv.ParseInt(args[1], 10, 64)
			err := c.Teams.RemoveMember(ctx(), teamID, membershipID)
			if err != nil {
				return err
			}
			fmt.Printf("Member %d removed from team %d\n", membershipID, teamID)
			return nil
		},
	}
}

func teamRemoveResourceCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "remove-resource <team-id> <resource-id>",
		Short: "Remove a contributed resource from a team",
		Long: `Stop sharing a resource with a team.

The resource ID is the team_resource record ID (from 'fibe teams resources <team-id>'),
not the underlying playspec/prop/marquee ID.

EXAMPLES:
  fibe teams remove-resource 1 7`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := newClient()
			teamID, _ := strconv.ParseInt(args[0], 10, 64)
			resourceID, _ := strconv.ParseInt(args[1], 10, 64)
			err := c.Teams.RemoveResource(ctx(), teamID, resourceID)
			if err != nil {
				return err
			}
			fmt.Printf("Resource %d removed from team %d\n", resourceID, teamID)
			return nil
		},
	}
}

// =============================================================================
// Agents: PUT messages/activity (set-* commands)
// =============================================================================
