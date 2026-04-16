package fibe

import (
	"context"
	"errors"
	"testing"
)

func TestPlaygroundCreateParams_Validate(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		p := &PlaygroundCreateParams{Name: "test", PlayspecID: 1}
		if err := p.Validate(); err != nil {
			t.Errorf("expected no error, got: %v", err)
		}
	})

	t.Run("missing name", func(t *testing.T) {
		p := &PlaygroundCreateParams{PlayspecID: 1}
		err := p.Validate()
		if err == nil {
			t.Fatal("expected error")
		}
		var ve ValidationErrors
		if !errors.As(err, &ve) {
			t.Fatal("expected ValidationErrors")
		}
		if ve[0].Field != "name" {
			t.Errorf("expected field 'name', got %q", ve[0].Field)
		}
	})

	t.Run("missing playspec_id", func(t *testing.T) {
		p := &PlaygroundCreateParams{Name: "test"}
		err := p.Validate()
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("invalid subdomain in service", func(t *testing.T) {
		p := &PlaygroundCreateParams{
			Name:       "test",
			PlayspecID: 1,
			Services: map[string]*ServiceConfig{
				"web": {Exposure: &ServiceExposure{Subdomain: "INVALID SUBDOMAIN!"}},
			},
		}
		err := p.Validate()
		if err == nil {
			t.Fatal("expected validation error for subdomain")
		}
	})
}

func TestAgentCreateParams_Validate(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		p := &AgentCreateParams{Name: "test", Provider: ProviderGemini}
		if err := p.Validate(); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("invalid provider", func(t *testing.T) {
		p := &AgentCreateParams{Name: "test", Provider: "invalid-provider"}
		err := p.Validate()
		if err == nil {
			t.Fatal("expected error for invalid provider")
		}
	})

	t.Run("missing name", func(t *testing.T) {
		p := &AgentCreateParams{Provider: ProviderClaudeCode}
		err := p.Validate()
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestPlayspecCreateParams_Validate(t *testing.T) {
	t.Run("valid minimal", func(t *testing.T) {
		p := &PlayspecCreateParams{Name: "test", BaseComposeYAML: "services:\n  web:\n    image: nginx\n"}
		if err := p.Validate(); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("missing compose", func(t *testing.T) {
		p := &PlayspecCreateParams{Name: "test"}
		err := p.Validate()
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("duplicate service names", func(t *testing.T) {
		p := &PlayspecCreateParams{
			Name:            "test",
			BaseComposeYAML: "services:\n  web:\n    image: nginx\n",
			Services: []PlayspecServiceDef{
				{Name: "web", Type: ServiceTypeStatic},
				{Name: "web", Type: ServiceTypeStatic},
			},
		}
		err := p.Validate()
		if err == nil {
			t.Fatal("expected error for duplicate service names")
		}
	})

	t.Run("invalid service type", func(t *testing.T) {
		p := &PlayspecCreateParams{
			Name:            "test",
			BaseComposeYAML: "yaml",
			Services:        []PlayspecServiceDef{{Name: "web", Type: "invalid"}},
		}
		err := p.Validate()
		if err == nil {
			t.Fatal("expected error for invalid service type")
		}
	})
}

func TestMarqueeCreateParams_Validate(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		p := &MarqueeCreateParams{Name: "test", Host: "10.0.1.5", Port: 22, User: "deploy", SSHPrivateKey: "key"}
		if err := p.Validate(); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("invalid port", func(t *testing.T) {
		p := &MarqueeCreateParams{Name: "test", Host: "host", Port: 99999, User: "u", SSHPrivateKey: "k"}
		err := p.Validate()
		if err == nil {
			t.Fatal("expected error for invalid port")
		}
	})

	t.Run("dockerhub enabled without credentials", func(t *testing.T) {
		enabled := true
		p := &MarqueeCreateParams{
			Name: "test", Host: "host", Port: 22, User: "u", SSHPrivateKey: "k",
			DockerhubAuthEnabled: &enabled,
		}
		err := p.Validate()
		if err == nil {
			t.Fatal("expected error when dockerhub enabled without username/token")
		}
		var ve ValidationErrors
		errors.As(err, &ve)
		if len(ve) < 2 {
			t.Errorf("expected 2 errors (username + token), got %d", len(ve))
		}
	})
}

func TestSecretCreateParams_Validate(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		p := &SecretCreateParams{Key: "DB_URL", Value: "postgres://..."}
		if err := p.Validate(); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("invalid key format", func(t *testing.T) {
		p := &SecretCreateParams{Key: "invalid key!", Value: "val"}
		err := p.Validate()
		if err == nil {
			t.Fatal("expected error for invalid key format")
		}
	})

	t.Run("missing value", func(t *testing.T) {
		p := &SecretCreateParams{Key: "KEY"}
		err := p.Validate()
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestWebhookCreateParams_Validate(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		p := &WebhookEndpointCreateParams{URL: "https://example.com", Events: []string{"playground.created"}}
		if err := p.Validate(); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("no events", func(t *testing.T) {
		p := &WebhookEndpointCreateParams{URL: "https://example.com"}
		err := p.Validate()
		if err == nil {
			t.Fatal("expected error for empty events")
		}
	})
}

func TestLaunchParams_Validate(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		p := &LaunchParams{ComposeYAML: "yaml", Name: "test"}
		if err := p.Validate(); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("missing both", func(t *testing.T) {
		p := &LaunchParams{}
		err := p.Validate()
		if err == nil {
			t.Fatal("expected error")
		}
		var ve ValidationErrors
		errors.As(err, &ve)
		if len(ve) != 2 {
			t.Errorf("expected 2 errors, got %d", len(ve))
		}
	})
}

func TestValidation_SkipsNetworkCall(t *testing.T) {
	c := NewClient(WithAPIKey("test"), WithDomain("nonexistent.invalid"), WithMaxRetries(0))

	_, err := c.Agents.Create(context.Background(), &AgentCreateParams{Provider: "bad"})
	if err == nil {
		t.Fatal("expected validation error")
	}
	var ve ValidationErrors
	if !errors.As(err, &ve) {
		t.Fatalf("expected ValidationErrors (no network call), got %T: %v", err, err)
	}
}
