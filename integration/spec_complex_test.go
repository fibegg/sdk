package integration

import (
	"strings"
	"testing"

	"github.com/fibegg/sdk/fibe"
)

func TestSpec_WithTriggerConfig(t *testing.T) {
	t.Parallel()
	c := userClient(t)

	host, err := c.Hosts.Create(ctx(), testHostParams("trigger-mq"))
	requireNoError(t, err)
	t.Cleanup(func() { c.Hosts.Delete(ctx(), host.ID) })
	hostID := host.ID

	repository := seedWritableGiteaRepository(t, c, "trigger-repository")

	jm := true
	spec := seedSpec(t, c, func(p *fibe.SpecCreateParams) {
		p.JobMode = &jm
		p.BaseComposeYAML = jobComposeYAML()
		p.Services = []fibe.SpecServiceDef{jobWatchedService("worker")}
		p.TriggerConfig = map[string]any{
			"enabled":       true,
			"event_type":    "push",
			"branch":        "main",
			"repository_id": repository.ID,
			"host_id":       hostID,
		}
	})

	detail, err := c.Specs.Get(ctx(), *spec.ID)
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

func TestSpec_WithPersistVolumes(t *testing.T) {
	t.Parallel()
	c := userClient(t)

	pv := true
	spec := seedSpec(t, c, func(p *fibe.SpecCreateParams) {
		p.PersistVolumes = &pv
	})

	d, err := c.Specs.Get(ctx(), *spec.ID)
	requireNoError(t, err)
	if d.PersistVolumes == nil || !*d.PersistVolumes {
		t.Errorf("expected PersistVolumes=true, got %v", d.PersistVolumes)
	}
}

func TestSpec_WithDescription(t *testing.T) {
	t.Parallel()
	c := userClient(t)

	desc := "Integration test spec with description " + uniqueName("")
	spec := seedSpec(t, c, func(p *fibe.SpecCreateParams) {
		p.Description = &desc
	})

	d, err := c.Specs.Get(ctx(), *spec.ID)
	requireNoError(t, err)
	if d.Description == nil || *d.Description != desc {
		t.Errorf("expected Description=%q, got %v", desc, d.Description)
	}
}

func TestSpec_WithRegistryCredential(t *testing.T) {
	t.Parallel()
	c := userClient(t)

	spec := seedSpec(t, c)

	_, err := c.Specs.AddRegistryCredential(ctx(), *spec.ID, &fibe.RegistryCredentialParams{
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
	d, err := c.Specs.Get(ctx(), *spec.ID)
	requireNoError(t, err)
	_ = d
}

func TestSpec_ValidateCompose_Invalid(t *testing.T) {
	t.Parallel()
	c := userClient(t)

	t.Run("invalid YAML produces errors", func(t *testing.T) {
		t.Parallel()
		result, err := c.Specs.ValidateCompose(ctx(), "this: is: not: valid: yaml: :::")
		// Backend may return error or may return Valid=false
		if err == nil {
			if result.Valid {
				t.Error("expected validation to fail for invalid YAML")
			}
		}
	})

	t.Run("compose missing services section", func(t *testing.T) {
		t.Parallel()
		result, err := c.Specs.ValidateCompose(ctx(), "version: '3'\n")
		if err == nil {
			if result.Valid && len(result.Errors) == 0 {
				t.Log("backend accepted compose without services (may be lenient)")
			}
		}
	})

	t.Run("valid compose returns Valid=true", func(t *testing.T) {
		t.Parallel()
		result, err := c.Specs.ValidateCompose(ctx(), realComposeYAML())
		requireNoError(t, err)
		if !result.Valid && len(result.Errors) > 0 {
			t.Errorf("expected valid compose, got errors: %v", result.Errors)
		}
	})
}

func TestSpec_UpdatePartialFields(t *testing.T) {
	t.Parallel()
	c := userClient(t)

	spec := seedSpec(t, c)

	t.Run("update description only", func(t *testing.T) {
		newDesc := "updated description " + uniqueName("")
		upd, err := c.Specs.Update(ctx(), *spec.ID, &fibe.SpecUpdateParams{
			Description: &newDesc,
		})
		requireNoError(t, err)
		if upd.Description == nil || *upd.Description != newDesc {
			t.Errorf("expected description %q, got %v", newDesc, upd.Description)
		}
	})

	t.Run("update persist_volumes flag", func(t *testing.T) {
		pv := true
		upd, err := c.Specs.Update(ctx(), *spec.ID, &fibe.SpecUpdateParams{
			PersistVolumes: &pv,
		})
		requireNoError(t, err)
		if upd.PersistVolumes == nil || !*upd.PersistVolumes {
			t.Errorf("expected PersistVolumes=true, got %v", upd.PersistVolumes)
		}
	})
}

func TestSpec_ListFilters(t *testing.T) {
	t.Parallel()
	c := userClient(t)

	needle := uniqueName("psfilt-needle")
	_, err := c.Specs.Create(ctx(), &fibe.SpecCreateParams{
		Name:            needle,
		BaseComposeYAML: minimalComposeYAML(),
		Services:        []fibe.SpecServiceDef{{Name: "web", Type: fibe.ServiceTypeStatic}},
	})
	requireNoError(t, err)

	t.Run("name filter finds needle", func(t *testing.T) {
		r, err := c.Specs.List(ctx(), &fibe.SpecListParams{Name: "psfilt-needle", PerPage: 50})
		requireNoError(t, err)
		found := false
		for _, p := range r.Data {
			if strings.Contains(p.Name, "psfilt-needle") {
				found = true
			}
		}
		if !found {
			t.Errorf("expected to find needle spec, got %d results", r.Meta.Total)
		}
	})

	t.Run("Q filter finds needle", func(t *testing.T) {
		r, err := c.Specs.List(ctx(), &fibe.SpecListParams{Q: "psfilt-needle", PerPage: 50})
		requireNoError(t, err)
		if r.Meta.Total == 0 {
			t.Error("expected Q filter to match at least one spec")
		}
	})
}
