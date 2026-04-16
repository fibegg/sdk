package integration

import (
	"testing"

	"github.com/fibegg/sdk/fibe"
)

func TestAgents_CRUD(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	var agentID int64

	t.Run("create agent", func(t *testing.T) {
		// Parallelism disabled for this subtest: state flows sequentially to below tests using agentID
		agent, err := c.Agents.Create(ctx(), &fibe.AgentCreateParams{
			Name:     uniqueName("test-agent"),
			Provider: fibe.ProviderGemini,
		})
		requireNoError(t, err)

		agentID = agent.ID
		if agent.Name == "" {
			t.Error("expected agent name")
		}
		if agent.Provider != fibe.ProviderGemini {
			t.Errorf("expected provider %q, got %q", fibe.ProviderGemini, agent.Provider)
		}
	})
	t.Cleanup(func() {
		if agentID > 0 {
			c.Agents.Delete(ctx(), agentID)
		}
	})

	t.Run("list agents", func(t *testing.T) {
		result, err := c.Agents.List(ctx(), nil)
		requireNoError(t, err)

		if result.Meta.Total == 0 {
			t.Error("expected at least one agent")
		}

		found := false
		for _, a := range result.Data {
			if a.ID == agentID {
				found = true
				break
			}
		}
		if agentID > 0 && !found {
			t.Error("created agent not found in list")
		}
	})

	t.Run("get agent", func(t *testing.T) {
		if agentID == 0 {
			t.Skip("no agent created")
		}
		agent, err := c.Agents.Get(ctx(), agentID)
		requireNoError(t, err)

		if agent.ID != agentID {
			t.Errorf("expected ID %d, got %d", agentID, agent.ID)
		}
		if agent.ProviderLabel == "" {
			t.Error("expected provider_label")
		}
	})

	t.Run("update agent", func(t *testing.T) {
		if agentID == 0 {
			t.Skip("no agent created")
		}
		newName := uniqueName("updated-agent")
		agent, err := c.Agents.Update(ctx(), agentID, &fibe.AgentUpdateParams{
			Name: &newName,
		})
		requireNoError(t, err)

		if agent.Name != newName {
			t.Errorf("expected name %q, got %q", newName, agent.Name)
		}
	})

	t.Run("duplicate agent", func(t *testing.T) {
		if agentID == 0 {
			t.Skip("no agent created")
		}
		dup, err := c.Agents.Duplicate(ctx(), agentID)
		requireNoError(t, err)

		if dup.ID == agentID {
			t.Error("duplicate should have different ID")
		}
		t.Cleanup(func() { c.Agents.Delete(ctx(), dup.ID) })
	})

	t.Run("delete agent", func(t *testing.T) {
		agent, err := c.Agents.Create(ctx(), &fibe.AgentCreateParams{
			Name:     uniqueName("delete-me"),
			Provider: fibe.ProviderGemini,
		})
		requireNoError(t, err)

		err = c.Agents.Delete(ctx(), agent.ID)
		requireNoError(t, err)

		_, err = c.Agents.Get(ctx(), agent.ID)
		requireAPIError(t, err, fibe.ErrCodeNotFound, 404)
	})
}

func TestAgents_Messages(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	agent, err := c.Agents.Create(ctx(), &fibe.AgentCreateParams{
		Name:     uniqueName("msg-agent"),
		Provider: fibe.ProviderGemini,
	})
	requireNoError(t, err)
	t.Cleanup(func() { c.Agents.Delete(ctx(), agent.ID) })

	t.Run("get empty messages", func(t *testing.T) {
		t.Parallel()
		data, err := c.Agents.GetMessages(ctx(), agent.ID)
		requireNoError(t, err)
		if data == nil {
			t.Error("expected non-nil response")
		}
	})

	t.Run("update messages", func(t *testing.T) {
		t.Parallel()
		content := []map[string]string{
			{"role": "user", "body": "hello"},
			{"role": "assistant", "body": "hi there"},
		}
		err := c.Agents.UpdateMessages(ctx(), agent.ID, content)
		requireNoError(t, err)
	})

	t.Run("get activity", func(t *testing.T) {
		t.Parallel()
		data, err := c.Agents.GetActivity(ctx(), agent.ID)
		requireNoError(t, err)
		if data == nil {
			t.Error("expected non-nil response")
		}
	})
}

func TestAgents_ScopeEnforcement(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	agent, err := c.Agents.Create(ctx(), &fibe.AgentCreateParams{
		Name:     uniqueName("scope-agent"),
		Provider: fibe.ProviderGemini,
	})
	requireNoError(t, err)
	t.Cleanup(func() { c.Agents.Delete(ctx(), agent.ID) })

	readOnly := createScopedKey(t, c, "agents-read", []string{"agents:read"})

	t.Run("read key can list", func(t *testing.T) {
		t.Parallel()
		_, err := readOnly.Agents.List(ctx(), nil)
		requireNoError(t, err)
	})

	t.Run("read key can get", func(t *testing.T) {
		t.Parallel()
		_, err := readOnly.Agents.Get(ctx(), agent.ID)
		requireNoError(t, err)
	})

	t.Run("read key cannot create", func(t *testing.T) {
		t.Parallel()
		_, err := readOnly.Agents.Create(ctx(), &fibe.AgentCreateParams{
			Name:     "nope",
			Provider: fibe.ProviderGemini,
		})
		requireAPIError(t, err, fibe.ErrCodeForbidden, 403)
	})

	t.Run("read key cannot delete", func(t *testing.T) {
		t.Parallel()
		err := readOnly.Agents.Delete(ctx(), agent.ID)
		requireAPIError(t, err, fibe.ErrCodeForbidden, 403)
	})
}

func TestAgents_GetGitHubTokenForRepo(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	agent, err := c.Agents.Create(ctx(), &fibe.AgentCreateParams{
		Name:     uniqueName("gh-token-agent"),
		Provider: fibe.ProviderGemini,
	})
	requireNoError(t, err)
	t.Cleanup(func() { c.Agents.Delete(ctx(), agent.ID) })

	t.Run("get token", func(t *testing.T) {
		t.Parallel()
		token, err := c.Agents.GetGitHubTokenForRepo(ctx(), agent.ID, "some/repo")
		if err != nil {
			if apiErr, ok := err.(*fibe.APIError); ok {
				if apiErr.Code == "GITHUB_CONNECTION_REQUIRED" || apiErr.Message != "" {
					return
				}
			}
			t.Errorf("unexpected error: %v", err)
		} else {
			if token.Token == "" {
				t.Error("expected token")
			}
		}
	})
}

func TestAgents_GetGiteaToken(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	agent, err := c.Agents.Create(ctx(), &fibe.AgentCreateParams{
		Name:     uniqueName("gitea-token-agent"),
		Provider: fibe.ProviderGemini,
	})
	requireNoError(t, err)
	t.Cleanup(func() { c.Agents.Delete(ctx(), agent.ID) })

	t.Run("get gitea token", func(t *testing.T) {
		t.Parallel()
		token, err := c.Agents.GetGiteaToken(ctx(), agent.ID)
		if err != nil {
			if apiErr, ok := err.(*fibe.APIError); ok && apiErr.Code == "GITEA_CONNECTION_REQUIRED" {
				return
			}
			t.Errorf("unexpected error: %v", err)
		} else {
			if token.Token == "" {
				t.Error("expected token")
			}
		}
	})
}
