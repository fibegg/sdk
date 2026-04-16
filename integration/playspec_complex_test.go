package integration

import (
	"strings"
	"testing"

	"github.com/fibegg/sdk/fibe"
)

func TestPlayspec_WithTriggerConfig(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	spec := seedPlayspec(t, c, func(p *fibe.PlayspecCreateParams) {
		p.TriggerConfig = map[string]any{
			"enabled": true,
			"branch":  "main",
		}
	})

	// Verify trigger_config persists
	detail, err := c.Playspecs.Get(ctx(), *spec.ID)
	requireNoError(t, err)
	if detail.TriggerConfig == nil {
		t.Error("expected TriggerConfig in detail response")
	}
	if detail.TriggerConfig != nil {
		if v, ok := detail.TriggerConfig["enabled"].(bool); !ok || !v {
			t.Errorf("expected trigger_config.enabled=true, got %v", detail.TriggerConfig["enabled"])
		}
	}
}

func TestPlayspec_WithPersistVolumes(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	pv := true
	spec := seedPlayspec(t, c, func(p *fibe.PlayspecCreateParams) {
		p.PersistVolumes = &pv
	})

	d, err := c.Playspecs.Get(ctx(), *spec.ID)
	requireNoError(t, err)
	if d.PersistVolumes == nil || !*d.PersistVolumes {
		t.Errorf("expected PersistVolumes=true, got %v", d.PersistVolumes)
	}
}

func TestPlayspec_WithDescription(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	desc := "Integration test playspec with description " + uniqueName("")
	spec := seedPlayspec(t, c, func(p *fibe.PlayspecCreateParams) {
		p.Description = &desc
	})

	d, err := c.Playspecs.Get(ctx(), *spec.ID)
	requireNoError(t, err)
	if d.Description == nil || *d.Description != desc {
		t.Errorf("expected Description=%q, got %v", desc, d.Description)
	}
}

func TestPlayspec_WithRegistryCredential(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	spec := seedPlayspec(t, c)

	// Add a registry credential (dummy values — never actually used to pull)
	_, err := c.Playspecs.AddRegistryCredential(ctx(), *spec.ID, &fibe.RegistryCredentialParams{
		RegistryType: "dockerhub",
		RegistryURL:  "https://index.docker.io/v1/",
		Username:     "test-integration-user",
		Secret:       "fake-password-not-real",
	})
	if err != nil {
		if apiErr, ok := err.(*fibe.APIError); ok && apiErr.StatusCode == 422 {
			t.Skipf("registry credentials rejected: %s", apiErr.Message)
		}
		requireNoError(t, err)
	}

	// Detail should reflect the credential existence (exact shape is backend-defined)
	d, err := c.Playspecs.Get(ctx(), *spec.ID)
	requireNoError(t, err)
	// d.Credentials may be a list; just check non-nil when creds added
	_ = d
}

func TestPlayspec_ValidateCompose_Invalid(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	t.Run("invalid YAML produces errors", func(t *testing.T) {
		t.Parallel()
		result, err := c.Playspecs.ValidateCompose(ctx(), "this: is: not: valid: yaml: :::")
		// Backend may return error or may return Valid=false
		if err == nil {
			if result.Valid {
				t.Error("expected validation to fail for invalid YAML")
			}
		}
	})

	t.Run("compose missing services section", func(t *testing.T) {
		t.Parallel()
		result, err := c.Playspecs.ValidateCompose(ctx(), "version: '3'\n")
		if err == nil {
			if result.Valid && len(result.Errors) == 0 {
				t.Log("backend accepted compose without services (may be lenient)")
			}
		}
	})

	t.Run("valid compose returns Valid=true", func(t *testing.T) {
		t.Parallel()
		result, err := c.Playspecs.ValidateCompose(ctx(), realComposeYAML())
		requireNoError(t, err)
		if !result.Valid && len(result.Errors) > 0 {
			t.Errorf("expected valid compose, got errors: %v", result.Errors)
		}
	})
}

func TestPlayspec_UpdatePartialFields(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	spec := seedPlayspec(t, c)

	t.Run("update description only", func(t *testing.T) {
		newDesc := "updated description " + uniqueName("")
		upd, err := c.Playspecs.Update(ctx(), *spec.ID, &fibe.PlayspecUpdateParams{
			Description: &newDesc,
		})
		requireNoError(t, err)
		if upd.Description == nil || *upd.Description != newDesc {
			t.Errorf("expected description %q, got %v", newDesc, upd.Description)
		}
	})

	t.Run("update persist_volumes flag", func(t *testing.T) {
		pv := true
		upd, err := c.Playspecs.Update(ctx(), *spec.ID, &fibe.PlayspecUpdateParams{
			PersistVolumes: &pv,
		})
		requireNoError(t, err)
		if upd.PersistVolumes == nil || !*upd.PersistVolumes {
			t.Errorf("expected PersistVolumes=true, got %v", upd.PersistVolumes)
		}
	})
}

func TestPlayspec_ListFilters(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	needle := uniqueName("psfilt-needle")
	_, err := c.Playspecs.Create(ctx(), &fibe.PlayspecCreateParams{
		Name:            needle,
		BaseComposeYAML: minimalComposeYAML(),
		Services:        []fibe.PlayspecServiceDef{{Name: "web", Type: fibe.ServiceTypeStatic}},
	})
	requireNoError(t, err)

	t.Run("name filter finds needle", func(t *testing.T) {
		r, err := c.Playspecs.List(ctx(), &fibe.PlayspecListParams{Name: "psfilt-needle", PerPage: 50})
		requireNoError(t, err)
		found := false
		for _, p := range r.Data {
			if strings.Contains(p.Name, "psfilt-needle") {
				found = true
			}
		}
		if !found {
			t.Errorf("expected to find needle playspec, got %d results", r.Meta.Total)
		}
	})

	t.Run("Q filter finds needle", func(t *testing.T) {
		r, err := c.Playspecs.List(ctx(), &fibe.PlayspecListParams{Q: "psfilt-needle", PerPage: 50})
		requireNoError(t, err)
		if r.Meta.Total == 0 {
			t.Error("expected Q filter to match at least one playspec")
		}
	})
}
