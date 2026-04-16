package integration

import (
	"testing"

	"github.com/fibegg/sdk/fibe"
)

func TestTeams_CRUD(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	var teamID int64

	t.Run("create team", func(t *testing.T) {
		// Parallel disabled: dependent sequence
		team, err := c.Teams.Create(ctx(), &fibe.TeamCreateParams{
			Name: uniqueName("test-team"),
		})
		requireNoError(t, err)

		teamID = team.ID
		if team.Name == "" {
			t.Error("expected team name")
		}
		if team.Slug == "" {
			t.Error("expected team slug")
		}
	})
	t.Cleanup(func() {
		if teamID > 0 {
			c.Teams.Delete(ctx(), teamID)
		}
	})

	t.Run("list teams", func(t *testing.T) {
		t.Parallel()
		result, err := c.Teams.List(ctx(), nil)
		requireNoError(t, err)

		if result.Meta.Total == 0 {
			t.Error("expected at least one team")
		}
	})

	t.Run("get team detail", func(t *testing.T) {
		// Parallel disabled: dependent sequence with update
		if teamID == 0 {
			t.Skip("no team created")
		}
		team, err := c.Teams.Get(ctx(), teamID)
		requireNoError(t, err)

		if team.ID != teamID {
			t.Errorf("expected ID %d", teamID)
		}
		if team.Memberships == nil {
			t.Error("expected memberships in detail response")
		}
	})

	t.Run("update team", func(t *testing.T) {
		// Parallel disabled: dependent sequence with get
		if teamID == 0 {
			t.Skip("no team created")
		}
		newName := uniqueName("updated-team")
		team, err := c.Teams.Update(ctx(), teamID, &fibe.TeamUpdateParams{
			Name: newName,
		})
		requireNoError(t, err)

		if team.Name != newName {
			t.Errorf("expected name %q", newName)
		}
	})

	t.Run("list resources (empty)", func(t *testing.T) {
		t.Parallel()
		if teamID == 0 {
			t.Skip("no team created")
		}
		result, err := c.Teams.ListResources(ctx(), teamID, nil)
		requireNoError(t, err)

		if result.Data == nil {
			t.Error("expected data to be non-nil")
		}
	})

	t.Run("delete team", func(t *testing.T) {
		t.Parallel()
		team, err := c.Teams.Create(ctx(), &fibe.TeamCreateParams{
			Name: uniqueName("delete-team"),
		})
		requireNoError(t, err)

		err = c.Teams.Delete(ctx(), team.ID)
		requireNoError(t, err)

		_, err = c.Teams.Get(ctx(), team.ID)
		requireAPIError(t, err, fibe.ErrCodeNotFound, 404)
	})
}

func TestTeams_ResourceSharing(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	team, err := c.Teams.Create(ctx(), &fibe.TeamCreateParams{
		Name: uniqueName("share-team"),
	})
	requireNoError(t, err)
	t.Cleanup(func() { c.Teams.Delete(ctx(), team.ID) })

	spec, err := c.Playspecs.Create(ctx(), &fibe.PlayspecCreateParams{
		Name:            uniqueName("share-spec"),
		BaseComposeYAML: "services:\n  web:\n    image: nginx:alpine\n",
		Services:        []fibe.PlayspecServiceDef{{Name: "web", Type: fibe.ServiceTypeStatic}},
	})
	requireNoError(t, err)
	t.Cleanup(func() { c.Playspecs.Delete(ctx(), *spec.ID) })

	t.Run("contribute resource", func(t *testing.T) {
		// Parallel disabled: dependent sequence
		res, err := c.Teams.ContributeResource(ctx(), team.ID, &fibe.TeamResourceParams{
			ResourceType:    "Playspec",
			ResourceID:      *spec.ID,
			PermissionLevel: "read",
		})
		requireNoError(t, err)

		if res.ResourceType != "Playspec" {
			t.Errorf("expected resource type Playspec, got %q", res.ResourceType)
		}

		t.Run("list resources shows contribution", func(t *testing.T) {
			// Subtests do not need Parallel() here, run in order
			result, err := c.Teams.ListResources(ctx(), team.ID, nil)
			requireNoError(t, err)

			found := false
			for _, r := range result.Data {
				if r.ResourceID == *spec.ID {
					found = true
					break
				}
			}
			if !found {
				t.Error("contributed resource not found in list")
			}
		})

		t.Run("remove resource", func(t *testing.T) {
			err := c.Teams.RemoveResource(ctx(), team.ID, res.ID)
			requireNoError(t, err)
		})
	})
}

func TestTeams_ScopeEnforcement(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	team, err := c.Teams.Create(ctx(), &fibe.TeamCreateParams{
		Name: uniqueName("scope-team"),
	})
	requireNoError(t, err)
	t.Cleanup(func() { c.Teams.Delete(ctx(), team.ID) })

	t.Run("read-only can list", func(t *testing.T) {
		t.Parallel()
		readOnly := createScopedKey(t, c, "teams-read", []string{"teams:read"})
		_, err := readOnly.Teams.List(ctx(), nil)
		requireNoError(t, err)
	})

	t.Run("read-only cannot update", func(t *testing.T) {
		t.Parallel()
		readOnly := createScopedKey(t, c, "teams-read2", []string{"teams:read"})
		newName := "nope"
		_, err := readOnly.Teams.Update(ctx(), team.ID, &fibe.TeamUpdateParams{
			Name: newName,
		})
		requireAPIError(t, err, fibe.ErrCodeForbidden, 403)
	})
}
