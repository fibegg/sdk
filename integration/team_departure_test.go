package integration

import (
	"testing"

	"github.com/fibegg/sdk/fibe"
)

// Migrated from: 32-team-departure-resource-retention.spec.js
func TestTeamDeparture_ResourceRetention(t *testing.T) {
	t.Parallel()
	admin := adminClient(t)
	userB := userBClient(t)

	team, err := admin.Teams.Create(ctx(), &fibe.TeamCreateParams{
		Name: uniqueName("departure-team"),
	})
	requireNoError(t, err)
	t.Cleanup(func() { admin.Teams.Delete(ctx(), team.ID) })

	userBPlayer, err := userB.APIKeys.Me(ctx())
	requireNoError(t, err)

	t.Run("invite and accept user B", func(t *testing.T) {
		membership, err := admin.Teams.InviteMember(ctx(), team.ID, userBPlayer.Username)
		requireNoError(t, err)

		_, err = userB.Teams.AcceptInvite(ctx(), team.ID, membership.ID)
		requireNoError(t, err)
	})

	agent, err := admin.Agents.Create(ctx(), &fibe.AgentCreateParams{
		Name:     uniqueName("departure-agent"),
		Provider: fibe.ProviderGemini,
	})
	requireNoError(t, err)
	t.Cleanup(func() { admin.Agents.Delete(ctx(), agent.ID) })

	t.Run("contribute resource to team", func(t *testing.T) {
		_, err := admin.Teams.ContributeResource(ctx(), team.ID, &fibe.TeamResourceParams{
			ResourceType:    "Agent",
			ResourceID:      agent.ID,
			PermissionLevel: "read",
		})
		requireNoError(t, err)
	})

	t.Run("user B leaves team", func(t *testing.T) {
		err := userB.Teams.Leave(ctx(), team.ID)
		requireNoError(t, err)
	})

	t.Run("admin still owns resources after departure", func(t *testing.T) {
		got, err := admin.Agents.Get(ctx(), agent.ID)
		requireNoError(t, err)
		if got.ID != agent.ID {
			t.Error("admin should still own the agent")
		}
	})
}
