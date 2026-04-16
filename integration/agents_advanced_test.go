package integration

import (
	"strconv"
	"testing"

	"github.com/fibegg/sdk/fibe"
)

func registryCredentialID(raw any) (string, bool) {
	switch v := raw.(type) {
	case string:
		return v, v != ""
	case float64:
		return strconv.FormatInt(int64(v), 10), true
	case int:
		return strconv.Itoa(v), true
	case int64:
		return strconv.FormatInt(v, 10), true
	default:
		return "", false
	}
}

// Migrated from: 21-agents-crud.spec.js (full lifecycle)
func TestAgents_FullLifecycle(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	t.Run("create and get detail", func(t *testing.T) {
		t.Parallel()
		agent, err := c.Agents.Create(ctx(), &fibe.AgentCreateParams{
			Name:        uniqueName("lifecycle-agent"),
			Provider:    fibe.ProviderClaudeCode,
			Description: ptr("integration test agent"),
		})
		requireNoError(t, err)
		t.Cleanup(func() { c.Agents.Delete(ctx(), agent.ID) })

		if agent.Provider != fibe.ProviderClaudeCode {
			t.Errorf("expected provider %q, got %q", fibe.ProviderClaudeCode, agent.Provider)
		}

		detail, err := c.Agents.Get(ctx(), agent.ID)
		requireNoError(t, err)

		if detail.Description == nil || *detail.Description != "integration test agent" {
			t.Error("expected description to persist")
		}
		if detail.ProviderLabel == "" {
			t.Error("expected provider_label")
		}
	})

	t.Run("update name and sync settings", func(t *testing.T) {
		t.Parallel()
		agent, err := c.Agents.Create(ctx(), &fibe.AgentCreateParams{
			Name:     uniqueName("update-agent"),
			Provider: fibe.ProviderGemini,
		})
		requireNoError(t, err)
		t.Cleanup(func() { c.Agents.Delete(ctx(), agent.ID) })

		newName := uniqueName("renamed")
		updated, err := c.Agents.Update(ctx(), agent.ID, &fibe.AgentUpdateParams{
			Name:        &newName,
			SyncEnabled: ptr(true),
		})
		requireNoError(t, err)

		if updated.Name != newName {
			t.Errorf("expected name %q", newName)
		}
		if !updated.SyncEnabled {
			t.Error("expected sync_enabled=true")
		}
	})

	t.Run("duplicate creates independent copy", func(t *testing.T) {
		t.Parallel()
		original, err := c.Agents.Create(ctx(), &fibe.AgentCreateParams{
			Name:        uniqueName("original"),
			Provider:    fibe.ProviderGemini,
			Description: ptr("original agent"),
		})
		requireNoError(t, err)
		t.Cleanup(func() { c.Agents.Delete(ctx(), original.ID) })

		dup, err := c.Agents.Duplicate(ctx(), original.ID)
		requireNoError(t, err)
		t.Cleanup(func() { c.Agents.Delete(ctx(), dup.ID) })

		if dup.ID == original.ID {
			t.Error("duplicate should have different ID")
		}
		if dup.Provider != original.Provider {
			t.Error("duplicate should have same provider")
		}
	})
}

// Migrated from: 22-artefacts.spec.js (listing only — upload requires multipart)
func TestArtefacts_List(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	agent, err := c.Agents.Create(ctx(), &fibe.AgentCreateParams{
		Name:     uniqueName("artefact-agent"),
		Provider: fibe.ProviderGemini,
	})
	requireNoError(t, err)
	t.Cleanup(func() { c.Agents.Delete(ctx(), agent.ID) })

	t.Run("list returns empty for new agent", func(t *testing.T) {
		t.Parallel()
		result, err := c.Artefacts.List(ctx(), agent.ID, nil)
		requireNoError(t, err)

		if result.Meta.Total != 0 {
			t.Errorf("expected 0 artefacts for new agent, got %d", result.Meta.Total)
		}
	})

	t.Run("filtering params accepted", func(t *testing.T) {
		t.Parallel()
		_, err := c.Artefacts.List(ctx(), agent.ID, &fibe.ArtefactListParams{
			Query:   "test",
			Sort:    "created_at_asc",
			PerPage: 10,
		})
		requireNoError(t, err)
	})
}

// Migrated from: 27-playspec-registry-credentials.spec.js
func TestPlayspecRegistryCredentials(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	spec, err := c.Playspecs.Create(ctx(), &fibe.PlayspecCreateParams{
		Name:            uniqueName("registry-spec"),
		BaseComposeYAML: "services:\n  web:\n    image: nginx:alpine\n",
		Services:        []fibe.PlayspecServiceDef{{Name: "web", Type: fibe.ServiceTypeStatic}},
	})
	requireNoError(t, err)
	t.Cleanup(func() { c.Playspecs.Delete(ctx(), *spec.ID) })

	t.Run("add registry credential", func(t *testing.T) {
		_, err := c.Playspecs.AddRegistryCredential(ctx(), *spec.ID, &fibe.RegistryCredentialParams{
			RegistryType: "ghcr",
			RegistryURL:  "ghcr.io",
			Username:     "testuser",
			Secret:       "ghp_test_token_123",
		})
		requireNoError(t, err)
	})

	t.Run("credential visible in detail", func(t *testing.T) {
		detail, err := c.Playspecs.Get(ctx(), *spec.ID)
		requireNoError(t, err)

		if detail.Credentials == nil {
			t.Error("expected credentials in playspec detail")
		}
	})

	t.Run("remove registry credential", func(t *testing.T) {
		detail, err := c.Playspecs.Get(ctx(), *spec.ID)
		requireNoError(t, err)

		creds, ok := detail.Credentials.([]any)
		if !ok || len(creds) == 0 {
			t.Skip("no credentials to remove")
		}

		credMap, ok := creds[0].(map[string]any)
		if !ok {
			t.Skip("unexpected credential format")
		}
		credID, ok := registryCredentialID(credMap["id"])
		if !ok {
			t.Skip("no credential ID")
		}

		err = c.Playspecs.RemoveRegistryCredential(ctx(), *spec.ID, credID)
		requireNoError(t, err)
	})
}
