package integration

import (
	"testing"

	"github.com/fibegg/sdk/fibe"
)

// Marquees require real VMs for full testing — per user guidance, we don't
// thoroughly test connectivity. But we DO test the CRUD surface with params
// so that create/update payload shapes are validated end-to-end.

func TestMarquee_CreateWithDockerhubAuth(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	enabled := true
	username := "integration-test-user"
	token := "fake-dockerhub-pat"
	email := "fx-" + uniqueName("") + "@example.com"
	domains := uniqueName("dh") + ".example.com"
	mq, err := c.Marquees.Create(ctx(), &fibe.MarqueeCreateParams{
		Name:                 uniqueName("mq-dh"),
		Host:                 uniqueHost(),
		Port:                 22,
		User:                 "deploy",
		SSHPrivateKey:        "-----BEGIN OPENSSH PRIVATE KEY-----\nFAKE\n-----END OPENSSH PRIVATE KEY-----\n",
		AcmeEmail:            &email,
		DomainsInput:         &domains,
		DockerhubAuthEnabled: &enabled,
		DockerhubUsername:    &username,
		DockerhubToken:       &token,
	})
	requireNoError(t, err)
	t.Cleanup(func() { c.Marquees.Delete(ctx(), mq.ID) })

	// Re-read to ensure persistence
	got, err := c.Marquees.Get(ctx(), mq.ID)
	requireNoError(t, err)
	if got.DockerhubAuthEnabled == nil || !*got.DockerhubAuthEnabled {
		t.Errorf("expected DockerhubAuthEnabled=true, got %v", got.DockerhubAuthEnabled)
	}
}

func TestMarquee_CreateWithDNSProvider(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	dnsProvider := "cloudflare"
	creds := map[string]string{
		"api_token": "fake-cf-token",
	}
	email := "fx-" + uniqueName("") + "@example.com"
	domains := uniqueName("dns") + ".example.com"
	mq, err := c.Marquees.Create(ctx(), &fibe.MarqueeCreateParams{
		Name:           uniqueName("mq-dns"),
		Host:           uniqueHost(),
		Port:           22,
		User:           "deploy",
		SSHPrivateKey:  "-----BEGIN OPENSSH PRIVATE KEY-----\nFAKE\n-----END OPENSSH PRIVATE KEY-----\n",
		AcmeEmail:      &email,
		DomainsInput:   &domains,
		DnsProvider:    &dnsProvider,
		DnsCredentials: creds,
	})
	if err != nil {
		if apiErr, ok := err.(*fibe.APIError); ok && apiErr.StatusCode == 422 {
			t.Skipf("dns config rejected (optional validation): %s", apiErr.Message)
		}
		requireNoError(t, err)
	}
	t.Cleanup(func() { c.Marquees.Delete(ctx(), mq.ID) })
}

func TestMarquee_CreateWithBuildPlatform(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	platform := "linux/amd64"
	email := "fx-" + uniqueName("") + "@example.com"
	domains := uniqueName("plat") + ".example.com"
	mq, err := c.Marquees.Create(ctx(), &fibe.MarqueeCreateParams{
		Name:          uniqueName("mq-plat"),
		Host:          uniqueHost(),
		Port:          2222,
		User:          "ubuntu",
		SSHPrivateKey: "-----BEGIN OPENSSH PRIVATE KEY-----\nFAKE\n-----END OPENSSH PRIVATE KEY-----\n",
		AcmeEmail:     &email,
		DomainsInput:  &domains,
		BuildPlatform: &platform,
	})
	requireNoError(t, err)
	t.Cleanup(func() { c.Marquees.Delete(ctx(), mq.ID) })

	got, err := c.Marquees.Get(ctx(), mq.ID)
	requireNoError(t, err)
	if got.BuildPlatform == nil || *got.BuildPlatform != platform {
		t.Errorf("expected BuildPlatform=%s, got %v", platform, got.BuildPlatform)
	}
}

func TestMarquee_AutoconnectToken(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	t.Run("token with minimal params returns hex token", func(t *testing.T) {
		t.Parallel()
		result, err := c.Marquees.AutoconnectToken(ctx(), &fibe.AutoconnectTokenParams{
			Email: "integration-test@example.com",
		})
		if err != nil {
			if apiErr, ok := err.(*fibe.APIError); ok && apiErr.StatusCode == 403 {
				t.Skip("autoconnect token requires admin-level scope")
			}
			requireNoError(t, err)
		}
		if result.Token == "" {
			t.Error("expected non-empty token")
		}
		if len(result.Token) < 32 {
			t.Errorf("expected token length >= 32, got %d", len(result.Token))
		}
	})

	t.Run("token with full params", func(t *testing.T) {
		t.Parallel()
		_, err := c.Marquees.AutoconnectToken(ctx(), &fibe.AutoconnectTokenParams{
			Email:       "test@example.com",
			Domain:      "app.example.com",
			IP:          "10.0.0.1",
			SSLMode:     "letsencrypt",
			DnsProvider: "route53",
			DnsCredentials: map[string]string{
				"aws_access_key_id":     "AKIA-FAKE",
				"aws_secret_access_key": "secret-fake",
			},
		})
		if err != nil {
			if apiErr, ok := err.(*fibe.APIError); ok && apiErr.StatusCode >= 400 && apiErr.StatusCode < 500 {
				t.Logf("autoconnect returned %d (may be expected if validation enforced): %s", apiErr.StatusCode, apiErr.Message)
				return
			}
			requireNoError(t, err)
		}
	})
}

func TestMarquee_UpdatePartialFields(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	email := "fx-" + uniqueName("") + "@example.com"
	domains := uniqueName("upd") + ".example.com"
	mq, err := c.Marquees.Create(ctx(), &fibe.MarqueeCreateParams{
		Name:          uniqueName("mq-upd"),
		Host:          uniqueHost(),
		Port:          22,
		User:          "deploy",
		SSHPrivateKey: "-----BEGIN OPENSSH PRIVATE KEY-----\nFAKE\n-----END OPENSSH PRIVATE KEY-----\n",
		AcmeEmail:     &email,
		DomainsInput:  &domains,
	})
	requireNoError(t, err)
	t.Cleanup(func() { c.Marquees.Delete(ctx(), mq.ID) })

	t.Run("update port", func(t *testing.T) {
		newPort := 2223
		upd, err := c.Marquees.Update(ctx(), mq.ID, &fibe.MarqueeUpdateParams{
			Port: &newPort,
		})
		requireNoError(t, err)
		if upd.Port != newPort {
			t.Errorf("expected Port=%d, got %d", newPort, upd.Port)
		}
	})

	t.Run("update user", func(t *testing.T) {
		newUser := "ubuntu-" + uniqueName("")
		upd, err := c.Marquees.Update(ctx(), mq.ID, &fibe.MarqueeUpdateParams{
			User: &newUser,
		})
		requireNoError(t, err)
		if upd.User != newUser {
			t.Errorf("expected User=%s, got %s", newUser, upd.User)
		}
	})

	t.Run("update acme email", func(t *testing.T) {
		email := "integration-" + uniqueName("") + "@example.com"
		upd, err := c.Marquees.Update(ctx(), mq.ID, &fibe.MarqueeUpdateParams{
			AcmeEmail: &email,
		})
		requireNoError(t, err)
		if upd.AcmeEmail == nil || *upd.AcmeEmail != email {
			t.Errorf("expected AcmeEmail=%s, got %v", email, upd.AcmeEmail)
		}
	})
}

func TestMarquee_CreateValidationErrors(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	t.Run("missing host returns validation error", func(t *testing.T) {
		t.Parallel()
		_, err := c.Marquees.Create(ctx(), &fibe.MarqueeCreateParams{
			Name:          uniqueName("mq-bad"),
			Host:          "",
			Port:          22,
			User:          "deploy",
			SSHPrivateKey: "fake-key",
		})
		if err == nil {
			t.Error("expected validation error for missing host")
		}
	})

	t.Run("invalid port returns error", func(t *testing.T) {
		t.Parallel()
		_, err := c.Marquees.Create(ctx(), &fibe.MarqueeCreateParams{
			Name:          uniqueName("mq-badport"),
			Host:          uniqueHost(),
			Port:          99999,
			User:          "deploy",
			SSHPrivateKey: "fake-key",
		})
		if err == nil {
			t.Error("expected error for out-of-range port")
		}
	})
}
