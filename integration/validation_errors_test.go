package integration

import (
	"strings"
	"testing"

	"github.com/fibegg/sdk/fibe"
)

// TestValidation_EmptyRequiredFields verifies each create endpoint rejects
// missing required fields with a structured 422/400.
func TestValidation_EmptyRequiredFields(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	cases := []struct {
		name string
		run  func() error
	}{
		{"agent empty name", func() error {
			_, err := c.Agents.Create(ctx(), &fibe.AgentCreateParams{Name: "", Provider: fibe.ProviderGemini})
			return err
		}},
		{"agent invalid provider", func() error {
			_, err := c.Agents.Create(ctx(), &fibe.AgentCreateParams{Name: uniqueName("t"), Provider: "bogus-provider-xyz"})
			return err
		}},
		{"playspec empty name", func() error {
			_, err := c.Playspecs.Create(ctx(), &fibe.PlayspecCreateParams{Name: "", BaseComposeYAML: minimalComposeYAML()})
			return err
		}},
		{"playspec empty compose", func() error {
			_, err := c.Playspecs.Create(ctx(), &fibe.PlayspecCreateParams{Name: uniqueName("t"), BaseComposeYAML: ""})
			return err
		}},
		{"secret empty key", func() error {
			_, err := c.Secrets.Create(ctx(), &fibe.SecretCreateParams{Key: "", Value: "v"})
			return err
		}},
		{"secret empty value", func() error {
			_, err := c.Secrets.Create(ctx(), &fibe.SecretCreateParams{Key: "KEY_" + uniqueName(""), Value: ""})
			return err
		}},
		{"prop empty url", func() error {
			_, err := c.Props.Create(ctx(), &fibe.PropCreateParams{RepositoryURL: ""})
			return err
		}},
		{"webhook empty url", func() error {
			_, err := c.WebhookEndpoints.Create(ctx(), &fibe.WebhookEndpointCreateParams{URL: "", Secret: "s", Events: []string{"playground.created"}})
			return err
		}},
		{"webhook empty events", func() error {
			_, err := c.WebhookEndpoints.Create(ctx(), &fibe.WebhookEndpointCreateParams{URL: "https://x", Secret: "s", Events: []string{}})
			return err
		}},
		{"marquee empty name", func() error {
			_, err := c.Marquees.Create(ctx(), &fibe.MarqueeCreateParams{Name: "", Host: "h", Port: 22, User: "u", SSHPrivateKey: "k"})
			return err
		}},
		{"api_key empty label", func() error {
			_, err := c.APIKeys.Create(ctx(), &fibe.APIKeyCreateParams{Label: ""})
			return err
		}},
		{"team empty name", func() error {
			_, err := c.Teams.Create(ctx(), &fibe.TeamCreateParams{Name: ""})
			return err
		}},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := tc.run()
			if err == nil {
				t.Error("expected validation error, got nil")
				return
			}
			// Either SDK-side validation or API-side — both are acceptable, but it must be an error.
			if apiErr, ok := err.(*fibe.APIError); ok {
				if apiErr.StatusCode >= 500 {
					t.Errorf("expected 4xx, got 5xx: %v", err)
				}
			}
		})
	}
}

// TestValidation_NotFoundIDs verifies 404 behavior across Get/Update/Delete operations.
func TestValidation_NotFoundIDs(t *testing.T) {
	t.Parallel()
	c := adminClient(t)
	bogus := int64(999999999)

	cases := []struct {
		name string
		run  func() error
	}{
		{"agent get", func() error { _, err := c.Agents.Get(ctx(), bogus); return err }},
		{"playspec get", func() error { _, err := c.Playspecs.Get(ctx(), bogus); return err }},
		{"prop get", func() error { _, err := c.Props.Get(ctx(), bogus); return err }},
		{"playground get", func() error { _, err := c.Playgrounds.Get(ctx(), bogus); return err }},
		{"marquee get", func() error { _, err := c.Marquees.Get(ctx(), bogus); return err }},
		{"secret get", func() error { _, err := c.Secrets.Get(ctx(), bogus, false); return err }},
		{"webhook get", func() error { _, err := c.WebhookEndpoints.Get(ctx(), bogus); return err }},
		{"template get", func() error { _, err := c.ImportTemplates.Get(ctx(), bogus); return err }},
		{"team get", func() error { _, err := c.Teams.Get(ctx(), bogus); return err }},
		{"playground status", func() error { _, err := c.Playgrounds.Status(ctx(), bogus); return err }},
		{"playground compose", func() error { _, err := c.Playgrounds.Compose(ctx(), bogus); return err }},
		{"playground debug", func() error { _, err := c.Playgrounds.Debug(ctx(), bogus); return err }},
		{"playground env_metadata", func() error { _, err := c.Playgrounds.EnvMetadata(ctx(), bogus); return err }},
		{"prop sync", func() error { return c.Props.Sync(ctx(), bogus) }},
		{"prop branches", func() error { _, err := c.Props.Branches(ctx(), bogus, "", 0); return err }},
		{"marquee test connection", func() error { _, err := c.Marquees.TestConnection(ctx(), bogus); return err }},
		{"webhook test", func() error { return c.WebhookEndpoints.Test(ctx(), bogus) }},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := tc.run()
			if err == nil {
				t.Errorf("expected 404 error for ID %d, got nil", bogus)
				return
			}
			apiErr, ok := err.(*fibe.APIError)
			if !ok {
				t.Errorf("expected APIError, got %T: %v", err, err)
				return
			}
			if apiErr.StatusCode != 404 && apiErr.StatusCode != 403 {
				// Some resources return 403 for not-owned to prevent ID enumeration — also acceptable
				t.Errorf("expected 404/403, got %d (code=%s)", apiErr.StatusCode, apiErr.Code)
			}
		})
	}
}

// TestValidation_SecretKeyFormat verifies secret key conventions.
func TestValidation_SecretKeyFormat(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	cases := []struct {
		name    string
		key     string
		wantErr bool
	}{
		{"lowercase allowed", "lowercase_" + uniqueName("K"), false},
		{"hyphens allowed", "WITH-HYPHEN-" + uniqueName("K"), false},
		{"spaces rejected", "WITH SPACE " + uniqueName("K"), true},
		{"dots rejected", "with.dot." + uniqueName("K"), true},
		{"slashes rejected", "with/slash/" + uniqueName("K"), true},
		{"valid uppercase snake", "VALID_UPPER_" + uniqueName("KEY"), false},
		{"valid digits", "KEY_123_" + uniqueName(""), false},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			s, err := c.Secrets.Create(ctx(), &fibe.SecretCreateParams{Key: tc.key, Value: "v"})
			if tc.wantErr {
				if err == nil {
					t.Errorf("expected error for key %q", tc.key)
					if s != nil && s.ID != nil {
						c.Secrets.Delete(ctx(), *s.ID)
					}
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error for key %q: %v", tc.key, err)
					return
				}
				t.Cleanup(func() { c.Secrets.Delete(ctx(), *s.ID) })
			}
		})
	}
}

// TestValidation_ErrorPayloadStructure verifies error responses include RequestID, Code, Message.
func TestValidation_ErrorPayloadStructure(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	_, err := c.Agents.Get(ctx(), 999999999)
	if err == nil {
		t.Fatal("expected error")
	}
	apiErr, ok := err.(*fibe.APIError)
	if !ok {
		t.Fatalf("expected APIError, got %T", err)
	}
	if apiErr.Code == "" {
		t.Error("expected non-empty error Code")
	}
	if apiErr.Message == "" {
		t.Error("expected non-empty error Message")
	}
	if apiErr.RequestID == "" {
		t.Error("expected non-empty RequestID in error")
	}
	// Error() should format with request ID
	if !strings.Contains(apiErr.Error(), apiErr.Code) {
		t.Error("APIError.Error() should contain Code")
	}
}
