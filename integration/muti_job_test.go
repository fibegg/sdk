package integration

import (
	"testing"

	"github.com/fibegg/sdk/fibe"
)

// Migrated from: 34-playspec-muti-job.spec.js
func TestMutiJob_PlayspecConfig(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	prop, err := c.Props.Create(ctx(), &fibe.PropCreateParams{
		RepositoryURL: "https://github.com/octocat/" + uniqueName("Hello-World"),
		Name:          ptr(uniqueName("muti-prop")),
	})
	requireNoError(t, err)
	t.Cleanup(func() { c.Props.Delete(ctx(), prop.ID) })

	t.Run("create playspec with muti config", func(t *testing.T) {
		t.Parallel()
		spec, err := c.Playspecs.Create(ctx(), &fibe.PlayspecCreateParams{
			Name:            uniqueName("muti-spec"),
			BaseComposeYAML: "services:\n  app:\n    image: alpine:latest\n",
			Services:        []fibe.PlayspecServiceDef{{Name: "app", Type: fibe.ServiceTypeStatic}},
			MutiConfig: map[string]any{
				"enabled":  true,
				"prop_id":  prop.ID,
				"agent_id": nil,
			},
		})
		requireNoError(t, err)
		t.Cleanup(func() { c.Playspecs.Delete(ctx(), *spec.ID) })

		detail, err := c.Playspecs.Get(ctx(), *spec.ID)
		requireNoError(t, err)

		if detail.MutiConfig == nil {
			t.Error("expected muti_config in detail")
		}
	})
}
