package integration

import (
	"testing"

	"github.com/fibegg/sdk/fibe"
)

// Migrated from: 10-scope-enforcement.spec.js
func TestScopeEnforcement_ReadOnlyKey(t *testing.T) {
	t.Parallel()
	c := adminClient(t)
	readOnly := createScopedKey(t, c, "readonly-enforcement", []string{"marquees:read"})

	t.Run("can list marquees", func(t *testing.T) {
		t.Parallel()
		_, err := readOnly.Marquees.List(ctx(), nil)
		requireNoError(t, err)
	})

	t.Run("denied write to marquees", func(t *testing.T) {
		t.Parallel()
		_, err := readOnly.Marquees.Create(ctx(), &fibe.MarqueeCreateParams{
			Name: "nope", Host: "1.2.3.4", Port: 22, User: "root", SSHPrivateKey: "key",
		})
		requireAPIError(t, err, fibe.ErrCodeForbidden, 403)
	})

	t.Run("denied access to other resource types", func(t *testing.T) {
		endpoints := []struct {
			name string
			fn   func() error
		}{
			{"playspecs", func() error { _, e := readOnly.Playspecs.List(ctx(), nil); return e }},
			{"props", func() error { _, e := readOnly.Props.List(ctx(), nil); return e }},
			{"playgrounds", func() error { _, e := readOnly.Playgrounds.List(ctx(), nil); return e }},
			{"templates", func() error { _, e := readOnly.ImportTemplates.List(ctx(), nil); return e }},
			{"keys", func() error { _, e := readOnly.APIKeys.List(ctx(), nil); return e }},
		}

		for _, ep := range endpoints {
			ep := ep
			t.Run(ep.name, func(t *testing.T) {
				t.Parallel()
				requireAPIError(t, ep.fn(), fibe.ErrCodeForbidden, 403)
			})
		}
	})
}

// Migrated from: 10-scope-enforcement.spec.js
func TestScopeEnforcement_WildcardKey(t *testing.T) {
	t.Parallel()
	c := adminClient(t)
	wildcard := createScopedKey(t, c, "wildcard-enforcement", []string{"*"})

	endpoints := []struct {
		name string
		fn   func() error
	}{
		{"marquees", func() error { _, e := wildcard.Marquees.List(ctx(), nil); return e }},
		{"playspecs", func() error { _, e := wildcard.Playspecs.List(ctx(), nil); return e }},
		{"props", func() error { _, e := wildcard.Props.List(ctx(), nil); return e }},
		{"playgrounds", func() error { _, e := wildcard.Playgrounds.List(ctx(), nil); return e }},
		{"templates", func() error { _, e := wildcard.ImportTemplates.List(ctx(), nil); return e }},
		{"keys", func() error { _, e := wildcard.APIKeys.List(ctx(), nil); return e }},
		{"secrets", func() error { _, e := wildcard.Secrets.List(ctx(), nil); return e }},
		{"agents", func() error { _, e := wildcard.Agents.List(ctx(), nil); return e }},
		{"webhooks", func() error { _, e := wildcard.WebhookEndpoints.List(ctx(), nil); return e }},
		{"audit_logs", func() error { _, e := wildcard.AuditLogs.List(ctx(), nil); return e }},
	}

	for _, ep := range endpoints {
		ep := ep
		t.Run(ep.name+" accessible with wildcard", func(t *testing.T) {
			t.Parallel()
			requireNoError(t, ep.fn())
		})
	}
}

// Migrated from: 10-scope-enforcement.spec.js
func TestScopeEnforcement_InvalidScopes(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	t.Run("reject nonexistent scope", func(t *testing.T) {
		t.Parallel()
		_, err := c.APIKeys.Create(ctx(), &fibe.APIKeyCreateParams{
			Label:  uniqueName("bad-scope"),
			Scopes: []string{"nonexistent:scope"},
		})
		requireAPIError(t, err, fibe.ErrCodeValidationFailed, 422)
	})

	t.Run("reject empty label", func(t *testing.T) {
		t.Parallel()
		_, err := c.APIKeys.Create(ctx(), &fibe.APIKeyCreateParams{
			Label:  "",
			Scopes: []string{"agents:read"},
		})
		if err == nil {
			t.Error("expected error for empty label")
		}
	})

	t.Run("reject empty scopes", func(t *testing.T) {
		t.Parallel()
		_, err := c.APIKeys.Create(ctx(), &fibe.APIKeyCreateParams{
			Label:  uniqueName("empty-scopes"),
			Scopes: []string{},
		})
		if err == nil {
			t.Error("expected error for empty scopes")
		}
	})
}

// Migrated from: 25-agent-scopes.spec.js
func TestScopeEnforcement_AgentSubResources(t *testing.T) {
	c := adminClient(t)

	agent, err := c.Agents.Create(ctx(), &fibe.AgentCreateParams{
		Name:     uniqueName("scope-sub-agent"),
		Provider: fibe.ProviderGemini,
	})
	requireNoError(t, err)
	t.Cleanup(func() { c.Agents.Delete(ctx(), agent.ID) })

	t.Run("agents:read can read feedbacks", func(t *testing.T) {
		readOnly := createScopedKey(t, c, "agent-read-fb", []string{"agents:read", "feedbacks:read"})
		_, err := readOnly.Feedbacks.List(ctx(), agent.ID, nil)
		requireNoError(t, err)
	})

	t.Run("no feedbacks scope gets 403", func(t *testing.T) {
		noFeedback := createScopedKey(t, c, "no-fb", []string{"agents:read"})
		_, err := noFeedback.Feedbacks.List(ctx(), agent.ID, nil)
		requireAPIError(t, err, fibe.ErrCodeForbidden, 403)
	})
}

// Migrated from: 28-granular-scopes.spec.js
func TestScopeEnforcement_GranularScopes(t *testing.T) {
	c := adminClient(t)

	agent, err := c.Agents.Create(ctx(), &fibe.AgentCreateParams{
		Name:     uniqueName("granular-agent"),
		Provider: fibe.ProviderGemini,
	})
	requireNoError(t, err)
	t.Cleanup(func() { c.Agents.Delete(ctx(), agent.ID) })

	t.Run("agent-accessible key", func(t *testing.T) {
		key, err := c.APIKeys.Create(ctx(), &fibe.APIKeyCreateParams{
			Label:           uniqueName("agent-key"),
			Scopes:          []string{"agents:read"},
			AgentAccessible: ptr(true),
		})
		requireNoError(t, err)
		t.Cleanup(func() { c.APIKeys.Delete(ctx(), *key.ID) })

		if !key.AgentAccessible {
			t.Error("expected agent_accessible=true")
		}
	})

	t.Run("granular scope restricts access", func(t *testing.T) {
		key, err := c.APIKeys.Create(ctx(), &fibe.APIKeyCreateParams{
			Label:  uniqueName("granular-key"),
			Scopes: []string{"agents:read"},
			GranularScopes: map[string][]int64{
				"agents:read": {agent.ID},
			},
		})
		requireNoError(t, err)
		t.Cleanup(func() { c.APIKeys.Delete(ctx(), *key.ID) })

		granular := c.WithKey(*key.Token)

		got, err := granular.Agents.Get(ctx(), agent.ID)
		requireNoError(t, err)
		if got.ID != agent.ID {
			t.Error("should access the allowed agent")
		}
	})
}
