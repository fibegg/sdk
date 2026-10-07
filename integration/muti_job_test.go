package integration

import (
	"testing"

	"github.com/fibegg/sdk/fibe"
)

func TestMutiJob_SpecConfig(t *testing.T) {
	t.Parallel()
	c := userClient(t)

	repository := seedWritableGiteaRepository(t, c, "muti-repository")

	t.Run("create spec with muti config", func(t *testing.T) {
		t.Parallel()
		spec, err := c.Specs.Create(ctx(), &fibe.SpecCreateParams{
			Name:            uniqueName("muti-spec"),
			BaseComposeYAML: "services:\n  app:\n    image: alpine:latest\n",
			Services:        []fibe.SpecServiceDef{{Name: "app", Type: fibe.ServiceTypeStatic}},
			MutiConfig: map[string]any{
				"enabled":       true,
				"repository_id": repository.ID,
				"agent_id":      nil,
			},
		})
		requireNoError(t, err)
		t.Cleanup(func() { c.Specs.Delete(ctx(), *spec.ID) })

		detail, err := c.Specs.Get(ctx(), *spec.ID)
		requireNoError(t, err)

		if detail.MutiConfig == nil {
			t.Error("expected muti_config in detail")
		}
	})
}
