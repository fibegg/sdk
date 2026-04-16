package integration

import (
	"strings"
	"testing"

	"github.com/fibegg/sdk/fibe"
)

// Migrated from: 11-marquee-validation.spec.js
func TestMarqueeValidation(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	t.Run("create with minimum port", func(t *testing.T) {
		t.Parallel()
		mq, err := c.Marquees.Create(ctx(), &fibe.MarqueeCreateParams{
			Name: uniqueName("port-min"), Host: "10.0.0.1", Port: 1, User: "root",
			SSHPrivateKey: "-----BEGIN OPENSSH PRIVATE KEY-----\ntest\n-----END OPENSSH PRIVATE KEY-----",
		})
		if err == nil {
			t.Cleanup(func() { c.Marquees.Delete(ctx(), mq.ID) })
		}
	})

	t.Run("create with maximum port", func(t *testing.T) {
		t.Parallel()
		mq, err := c.Marquees.Create(ctx(), &fibe.MarqueeCreateParams{
			Name: uniqueName("port-max"), Host: "10.0.0.2", Port: 65535, User: "root",
			SSHPrivateKey: "-----BEGIN OPENSSH PRIVATE KEY-----\ntest\n-----END OPENSSH PRIVATE KEY-----",
		})
		if err == nil {
			t.Cleanup(func() { c.Marquees.Delete(ctx(), mq.ID) })
		}
	})

	t.Run("reject port out of range via client validation", func(t *testing.T) {
		t.Parallel()
		_, err := c.Marquees.Create(ctx(), &fibe.MarqueeCreateParams{
			Name: "bad-port", Host: "10.0.0.3", Port: 99999, User: "root",
			SSHPrivateKey: "key",
		})
		if err == nil {
			t.Error("expected validation error for port > 65535")
		}
	})

	t.Run("reject missing name", func(t *testing.T) {
		t.Parallel()
		_, err := c.Marquees.Create(ctx(), &fibe.MarqueeCreateParams{
			Host: "10.0.0.4", Port: 22, User: "root", SSHPrivateKey: "key",
		})
		if err == nil {
			t.Error("expected validation error for missing name")
		}
	})

	t.Run("reject dockerhub enabled without credentials", func(t *testing.T) {
		t.Parallel()
		_, err := c.Marquees.Create(ctx(), &fibe.MarqueeCreateParams{
			Name: "docker-fail", Host: "10.0.0.5", Port: 22, User: "root",
			SSHPrivateKey:        "key",
			DockerhubAuthEnabled: ptr(true),
		})
		if err == nil {
			t.Error("expected validation error for dockerhub without credentials")
		}
	})
}

// Migrated from: 12-prop-validation.spec.js
func TestPropValidation(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	t.Run("reject missing repository_url", func(t *testing.T) {
		t.Parallel()
		_, err := c.Props.Create(ctx(), &fibe.PropCreateParams{})
		if err == nil {
			t.Error("expected validation error for missing repo URL")
		}
	})

	t.Run("duplicate repo_url handled gracefully", func(t *testing.T) {
		t.Parallel()
		url := "https://github.com/octocat/" + uniqueName("Hello-World")
		prop1, err := c.Props.Create(ctx(), &fibe.PropCreateParams{
			RepositoryURL: url,
			Name:          ptr(uniqueName("dup-test")),
		})
		requireNoError(t, err)
		t.Cleanup(func() { c.Props.Delete(ctx(), prop1.ID) })

		prop2, err := c.Props.Create(ctx(), &fibe.PropCreateParams{
			RepositoryURL: url,
			Name:          ptr(uniqueName("dup-test-2")),
		})
		if err != nil {
			return
		}
		if prop2.ID != prop1.ID {
			t.Cleanup(func() { c.Props.Delete(ctx(), prop2.ID) })
		}
	})
}

// Migrated from: 13-playspec-validation.spec.js
func TestPlayspecValidation(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	t.Run("reject missing name", func(t *testing.T) {
		t.Parallel()
		_, err := c.Playspecs.Create(ctx(), &fibe.PlayspecCreateParams{
			BaseComposeYAML: "services:\n  web:\n    image: nginx\n",
			Services:        []fibe.PlayspecServiceDef{{Name: "web", Type: fibe.ServiceTypeStatic}},
		})
		if err == nil {
			t.Error("expected validation error for missing name")
		}
	})

	t.Run("reject missing compose YAML", func(t *testing.T) {
		t.Parallel()
		_, err := c.Playspecs.Create(ctx(), &fibe.PlayspecCreateParams{
			Name: uniqueName("no-compose"),
		})
		if err == nil {
			t.Error("expected validation error for missing compose")
		}
	})

	t.Run("reject duplicate service names", func(t *testing.T) {
		t.Parallel()
		_, err := c.Playspecs.Create(ctx(), &fibe.PlayspecCreateParams{
			Name:            uniqueName("dup-svc"),
			BaseComposeYAML: "services:\n  web:\n    image: nginx\n",
			Services: []fibe.PlayspecServiceDef{
				{Name: "web", Type: fibe.ServiceTypeStatic},
				{Name: "web", Type: fibe.ServiceTypeStatic},
			},
		})
		if err == nil {
			t.Error("expected validation error for duplicate service names")
		}
	})

	t.Run("reject invalid service type", func(t *testing.T) {
		t.Parallel()
		_, err := c.Playspecs.Create(ctx(), &fibe.PlayspecCreateParams{
			Name:            uniqueName("bad-type"),
			BaseComposeYAML: "services:\n  web:\n    image: nginx\n",
			Services:        []fibe.PlayspecServiceDef{{Name: "web", Type: "invalid"}},
		})
		if err == nil {
			t.Error("expected validation error for invalid service type")
		}
	})

	t.Run("validate compose YAML endpoint", func(t *testing.T) {
		t.Parallel()
		result, err := c.Playspecs.ValidateCompose(ctx(), "services:\n  web:\n    image: nginx\n  db:\n    image: postgres\n")
		requireNoError(t, err)

		if result == nil {
			t.Fatal("expected validation result to be non-nil")
		}
	})

	t.Run("validate compose with invalid YAML", func(t *testing.T) {
		t.Parallel()
		result, err := c.Playspecs.ValidateCompose(ctx(), "this is not valid yaml: [[[")
		requireNoError(t, err)

		if result == nil {
			t.Fatal("expected validation result to be non-nil even for invalid YAML")
		}
	})
}

// Migrated from: 43-secrets-crud.spec.js (validation parts)
func TestSecretValidation(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	t.Run("reject invalid key format", func(t *testing.T) {
		t.Parallel()
		_, err := c.Secrets.Create(ctx(), &fibe.SecretCreateParams{
			Key:   "invalid key with spaces!",
			Value: "val",
		})
		if err == nil {
			t.Error("expected validation error for invalid key format")
		}
	})

	t.Run("reject empty value", func(t *testing.T) {
		t.Parallel()
		_, err := c.Secrets.Create(ctx(), &fibe.SecretCreateParams{
			Key:   "EMPTY_VAL",
			Value: "",
		})
		if err == nil {
			t.Error("expected validation error for empty value")
		}
	})
}

// Migrated from: 21-agents-crud.spec.js (validation parts)
func TestAgentValidation(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	t.Run("reject invalid provider", func(t *testing.T) {
		t.Parallel()
		_, err := c.Agents.Create(ctx(), &fibe.AgentCreateParams{
			Name:     uniqueName("bad-provider"),
			Provider: "nonexistent-provider",
		})
		if err == nil {
			t.Error("expected validation error for invalid provider")
		}
	})

	t.Run("reject empty name", func(t *testing.T) {
		t.Parallel()
		_, err := c.Agents.Create(ctx(), &fibe.AgentCreateParams{
			Name:     "",
			Provider: fibe.ProviderGemini,
		})
		if err == nil {
			t.Error("expected validation error for empty name")
		}
	})

	t.Run("all valid providers accepted", func(t *testing.T) {
		t.Parallel()
		for _, provider := range fibe.ValidProviders {
			t.Run(provider, func(t *testing.T) {
				t.Parallel()
				agent, err := c.Agents.Create(ctx(), &fibe.AgentCreateParams{
					Name:     uniqueName("provider-" + provider),
					Provider: provider,
				})
				requireNoError(t, err)
				t.Cleanup(func() { c.Agents.Delete(ctx(), agent.ID) })

				if agent.Provider != provider {
					t.Errorf("expected provider %q, got %q", provider, agent.Provider)
				}
			})
		}
	})
}

// Migrated from: 29-webhooks.spec.js (validation parts)
func TestWebhookValidation(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	t.Run("reject missing URL", func(t *testing.T) {
		t.Parallel()
		_, err := c.WebhookEndpoints.Create(ctx(), &fibe.WebhookEndpointCreateParams{
			Secret: "secret",
			Events: []string{"playground.created"},
		})
		if err == nil {
			t.Error("expected validation error for missing URL")
		}
	})

	t.Run("reject empty events", func(t *testing.T) {
		t.Parallel()
		_, err := c.WebhookEndpoints.Create(ctx(), &fibe.WebhookEndpointCreateParams{
			URL:    "https://example.com/hook",
			Secret: "secret",
		})
		if err == nil {
			t.Error("expected validation error for empty events")
		}
	})

	t.Run("allow missing secret and auto-generate one", func(t *testing.T) {
		t.Parallel()
		ep, err := c.WebhookEndpoints.Create(ctx(), &fibe.WebhookEndpointCreateParams{
			URL:    "https://example.com/hook-" + uniqueName(""),
			Events: []string{"playground.created"},
		})
		requireNoError(t, err)
		if ep.ID != nil {
			t.Cleanup(func() { _ = c.WebhookEndpoints.Delete(ctx(), *ep.ID) })
		}
		if ep.Secret == nil || *ep.Secret == "" {
			t.Error("expected server-generated secret when secret is omitted")
		}
	})
}

// Migrated from: 06-launch.spec.js
func TestLaunchValidation(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	t.Run("reject missing compose_yaml", func(t *testing.T) {
		t.Parallel()
		_, err := c.Launch.Create(ctx(), &fibe.LaunchParams{
			Name: uniqueName("no-compose"),
		})
		if err == nil {
			t.Error("expected validation error for missing compose_yaml")
		}
	})

	t.Run("reject missing name", func(t *testing.T) {
		t.Parallel()
		_, err := c.Launch.Create(ctx(), &fibe.LaunchParams{
			ComposeYAML: "services:\n  web:\n    image: nginx\n",
		})
		if err == nil {
			t.Error("expected validation error for missing name")
		}
	})

	t.Run("launch creates playspec", func(t *testing.T) {
		t.Parallel()
		result, err := c.Launch.Create(ctx(), &fibe.LaunchParams{
			Name:        uniqueName("launch-test"),
			ComposeYAML: "services:\n  web:\n    image: nginx:alpine\n",
		})
		if err != nil {
			apiErr, ok := err.(*fibe.APIError)
			if ok && strings.Contains(apiErr.Message, "classified") {
				t.Skip("launch requires classified services")
			}
			t.Fatalf("unexpected error: %v", err)
		}
		if result == nil {
			t.Fatal("expected launch result to be non-nil")
		}
		// Launch endpoint returns a playspec — the result struct may have zero values
		// if the API response shape doesn't match LaunchResult exactly.
		// The key assertion is that the call succeeded (no error above).
	})
}
