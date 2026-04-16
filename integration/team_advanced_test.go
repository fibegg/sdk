package integration

import (
	"testing"

	"github.com/fibegg/sdk/fibe"
)

// Migrated from: 30-team-shared-resources.spec.js + 31-team-access-control.spec.js
func TestTeamSharedResources(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	team, err := c.Teams.Create(ctx(), &fibe.TeamCreateParams{
		Name: uniqueName("shared-team"),
	})
	requireNoError(t, err)
	t.Cleanup(func() { c.Teams.Delete(ctx(), team.ID) })

	agent, err := c.Agents.Create(ctx(), &fibe.AgentCreateParams{
		Name:     uniqueName("shared-agent"),
		Provider: fibe.ProviderGemini,
	})
	requireNoError(t, err)
	t.Cleanup(func() { c.Agents.Delete(ctx(), agent.ID) })

	spec, err := c.Playspecs.Create(ctx(), &fibe.PlayspecCreateParams{
		Name:            uniqueName("shared-spec"),
		BaseComposeYAML: "services:\n  web:\n    image: nginx:alpine\n",
		Services:        []fibe.PlayspecServiceDef{{Name: "web", Type: fibe.ServiceTypeStatic}},
	})
	requireNoError(t, err)
	t.Cleanup(func() { c.Playspecs.Delete(ctx(), *spec.ID) })

	var agentResID, specResID int64

	t.Run("contribute agent to team", func(t *testing.T) {
		res, err := c.Teams.ContributeResource(ctx(), team.ID, &fibe.TeamResourceParams{
			ResourceType:    "Agent",
			ResourceID:      agent.ID,
			PermissionLevel: "read",
		})
		requireNoError(t, err)
		agentResID = res.ID

		if res.ResourceType != "Agent" {
			t.Errorf("expected Agent, got %q", res.ResourceType)
		}
	})

	t.Run("contribute playspec to team", func(t *testing.T) {
		res, err := c.Teams.ContributeResource(ctx(), team.ID, &fibe.TeamResourceParams{
			ResourceType:    "Playspec",
			ResourceID:      *spec.ID,
			PermissionLevel: "read",
		})
		requireNoError(t, err)
		specResID = res.ID
		_ = specResID
	})

	t.Run("list shows both resources", func(t *testing.T) {
		result, err := c.Teams.ListResources(ctx(), team.ID, nil)
		requireNoError(t, err)

		if len(result.Data) < 2 {
			t.Errorf("expected at least 2 resources, got %d", len(result.Data))
		}

		types := map[string]bool{}
		for _, r := range result.Data {
			types[r.ResourceType] = true
		}
		if !types["Agent"] {
			t.Error("expected Agent in resources")
		}
		if !types["Playspec"] {
			t.Error("expected Playspec in resources")
		}
	})

	t.Run("remove contributed resource", func(t *testing.T) {
		if agentResID == 0 {
			t.Skip("no resource contributed")
		}
		err := c.Teams.RemoveResource(ctx(), team.ID, agentResID)
		requireNoError(t, err)
	})
}

func TestTeamMemberships_InviteEdgeCases(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	team, err := c.Teams.Create(ctx(), &fibe.TeamCreateParams{
		Name: uniqueName("invite-edge-team"),
	})
	requireNoError(t, err)
	t.Cleanup(func() { c.Teams.Delete(ctx(), team.ID) })

	t.Run("invite with blank username returns 400", func(t *testing.T) {
		t.Parallel()
		_, err := c.Teams.InviteMember(ctx(), team.ID, "")
		requireAPIError(t, err, fibe.ErrCodeBadRequest, 400)
	})

	t.Run("invite nonexistent user", func(t *testing.T) {
		t.Parallel()
		result, err := c.Teams.InviteMember(ctx(), team.ID, "nonexistent-user-xyz-99999")
		if err != nil {
			apiErr, ok := err.(*fibe.APIError)
			if ok && (apiErr.StatusCode == 422 || apiErr.StatusCode == 404) {
				return
			}
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Status != "pending" {
			t.Errorf("expected pending status for invite, got %q", result.Status)
		}
	})
}

func TestTeamMemberships_DeclineInvite(t *testing.T) {
	t.Parallel()
	admin := adminClient(t)
	userB := userBClient(t)

	team, err := admin.Teams.Create(ctx(), &fibe.TeamCreateParams{
		Name: uniqueName("decline-team"),
	})
	requireNoError(t, err)
	t.Cleanup(func() { admin.Teams.Delete(ctx(), team.ID) })

	userBPlayer, err := userB.APIKeys.Me(ctx())
	requireNoError(t, err)

	membership, err := admin.Teams.InviteMember(ctx(), team.ID, userBPlayer.Username)
	requireNoError(t, err)

	t.Run("user B can decline the invite", func(t *testing.T) {
		declined, err := userB.Teams.DeclineInvite(ctx(), team.ID, membership.ID)
		requireNoError(t, err)

		if declined.Status != "declined" {
			t.Errorf("expected status 'declined', got %q", declined.Status)
		}
	})
}

func TestTeamMemberships_OwnerProtection(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	team, err := c.Teams.Create(ctx(), &fibe.TeamCreateParams{
		Name: uniqueName("owner-protect-team"),
	})
	requireNoError(t, err)
	t.Cleanup(func() { c.Teams.Delete(ctx(), team.ID) })

	detail, err := c.Teams.Get(ctx(), team.ID)
	requireNoError(t, err)

	var ownerMembershipID int64
	for _, m := range detail.Memberships {
		if m.Role == "owner" {
			ownerMembershipID = m.ID
			break
		}
	}

	if ownerMembershipID == 0 {
		t.Skip("could not find owner membership")
	}

	t.Run("cannot change owner role to member", func(t *testing.T) {
		t.Parallel()
		_, err := c.Teams.UpdateMember(ctx(), team.ID, ownerMembershipID, "member")
		requireAPIError(t, err, fibe.ErrCodeConflict, 409)
	})

	t.Run("cannot remove the owner", func(t *testing.T) {
		t.Parallel()
		err := c.Teams.RemoveMember(ctx(), team.ID, ownerMembershipID)
		requireAPIError(t, err, fibe.ErrCodeConflict, 409)
	})

	t.Run("owner cannot leave without transferring leadership", func(t *testing.T) {
		t.Parallel()
		err := c.Teams.Leave(ctx(), team.ID)
		requireAPIError(t, err, fibe.ErrCodeConflict, 409)
	})
}

func TestTeamMemberships_UpdateRole(t *testing.T) {
	t.Parallel()
	admin := adminClient(t)
	userB := userBClient(t)

	team, err := admin.Teams.Create(ctx(), &fibe.TeamCreateParams{
		Name: uniqueName("role-update-team"),
	})
	requireNoError(t, err)
	t.Cleanup(func() { admin.Teams.Delete(ctx(), team.ID) })

	userBPlayer, err := userB.APIKeys.Me(ctx())
	requireNoError(t, err)

	membership, err := admin.Teams.InviteMember(ctx(), team.ID, userBPlayer.Username)
	requireNoError(t, err)

	_, err = userB.Teams.AcceptInvite(ctx(), team.ID, membership.ID)
	requireNoError(t, err)

	t.Run("admin updates member role to admin", func(t *testing.T) {
		updated, err := admin.Teams.UpdateMember(ctx(), team.ID, membership.ID, "admin")
		requireNoError(t, err)

		if updated.Role != "admin" {
			t.Errorf("expected role 'admin', got %q", updated.Role)
		}
	})

	t.Run("admin removes member", func(t *testing.T) {
		err := admin.Teams.RemoveMember(ctx(), team.ID, membership.ID)
		requireNoError(t, err)
	})
}

func TestTeamMemberships_IDOR(t *testing.T) {
	t.Parallel()
	admin := adminClient(t)
	userB := userBClient(t)

	team, err := admin.Teams.Create(ctx(), &fibe.TeamCreateParams{
		Name: uniqueName("membership-idor-team"),
	})
	requireNoError(t, err)
	t.Cleanup(func() { admin.Teams.Delete(ctx(), team.ID) })

	t.Run("user B cannot invite to admin team", func(t *testing.T) {
		t.Parallel()
		_, err := userB.Teams.InviteMember(ctx(), team.ID, "someone")
		requireAPIError(t, err, fibe.ErrCodeNotFound, 404)
	})

	t.Run("user B cannot update members in admin team", func(t *testing.T) {
		t.Parallel()
		_, err := userB.Teams.UpdateMember(ctx(), team.ID, 1, "admin")
		requireAPIError(t, err, fibe.ErrCodeNotFound, 404)
	})
}

// Migrated from: 31-team-access-control.spec.js (permission levels)
func TestTeamAccessControl(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	team, err := c.Teams.Create(ctx(), &fibe.TeamCreateParams{
		Name: uniqueName("acl-team"),
	})
	requireNoError(t, err)
	t.Cleanup(func() { c.Teams.Delete(ctx(), team.ID) })

	t.Run("get team shows memberships", func(t *testing.T) {
		t.Parallel()
		detail, err := c.Teams.Get(ctx(), team.ID)
		requireNoError(t, err)

		if len(detail.Memberships) == 0 {
			t.Error("expected creator as owner membership")
		}
		hasOwner := false
		for _, m := range detail.Memberships {
			if m.Role == "owner" {
				hasOwner = true
			}
		}
		if !hasOwner {
			t.Error("expected at least one owner membership")
		}
	})

	t.Run("update team name", func(t *testing.T) {
		t.Parallel()
		newName := uniqueName("acl-updated")
		team, err := c.Teams.Update(ctx(), team.ID, &fibe.TeamUpdateParams{Name: newName})
		requireNoError(t, err)
		if team.Name != newName {
			t.Error("name should be updated")
		}
	})
}
