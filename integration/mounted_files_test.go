package integration

import (
	"strings"
	"testing"

	"github.com/fibegg/sdk/fibe"
)

// Migrated from: 38-agent-mounted-files.spec.js
func TestAgentMountedFiles(t *testing.T) {
	t.Parallel()
	c := adminClient(t)
	mountedFilename := "test.txt"

	agent, err := c.Agents.Create(ctx(), &fibe.AgentCreateParams{
		Name:     uniqueName("mf-agent"),
		Provider: fibe.ProviderGemini,
	})
	requireNoError(t, err)
	t.Cleanup(func() { c.Agents.Delete(ctx(), agent.ID) })

	t.Run("add mounted file", func(t *testing.T) {
		file := strings.NewReader("hello world test content")
		updated, err := c.Agents.AddMountedFile(ctx(), agent.ID, file, "test.txt", &fibe.MountedFileParams{
			MountPath: "/app/config/test.txt",
			ReadOnly:  ptr(true),
		})
		requireNoError(t, err)
		if updated != nil && len(updated.MountedFiles) > 0 {
			name := updated.MountedFiles[len(updated.MountedFiles)-1].Name
			if name != "" {
				mountedFilename = name
			}
		}
	})

	t.Run("update mounted file path", func(t *testing.T) {
		_, err := c.Agents.UpdateMountedFile(ctx(), agent.ID, &fibe.MountedFileUpdateParams{
			Filename:  mountedFilename,
			MountPath: "/app/data/test.txt",
		})
		requireNoError(t, err)
	})

	t.Run("remove mounted file", func(t *testing.T) {
		_, err := c.Agents.RemoveMountedFile(ctx(), agent.ID, mountedFilename)
		requireNoError(t, err)
	})
}

// Migrated from: 39-playspec-mounted-files.spec.js
func TestPlayspecMountedFiles(t *testing.T) {
	t.Parallel()
	c := adminClient(t)
	mountedFilename := "nginx.conf"

	spec, err := c.Playspecs.Create(ctx(), &fibe.PlayspecCreateParams{
		Name:            uniqueName("mf-spec"),
		BaseComposeYAML: "services:\n  web:\n    image: nginx:alpine\n",
		Services:        []fibe.PlayspecServiceDef{{Name: "web", Type: fibe.ServiceTypeStatic}},
	})
	requireNoError(t, err)
	t.Cleanup(func() { c.Playspecs.Delete(ctx(), *spec.ID) })

	t.Run("add mounted file", func(t *testing.T) {
		file := strings.NewReader("nginx config content")
		err := c.Playspecs.AddMountedFile(ctx(), *spec.ID, file, "nginx.conf", &fibe.MountedFileParams{
			MountPath:      "/etc/nginx/nginx.conf",
			TargetServices: []string{"web"},
			ReadOnly:       ptr(true),
		})
		requireNoError(t, err)
	})

	t.Run("file visible in detail", func(t *testing.T) {
		detail, err := c.Playspecs.Get(ctx(), *spec.ID)
		requireNoError(t, err)

		if len(detail.MountedFiles) == 0 {
			t.Error("expected mounted_files in playspec detail")
			return
		}
		if detail.MountedFiles[0].Filename != "" {
			mountedFilename = detail.MountedFiles[0].Filename
		}
	})

	t.Run("update mounted file", func(t *testing.T) {
		err := c.Playspecs.UpdateMountedFile(ctx(), *spec.ID, &fibe.MountedFileUpdateParams{
			Filename:  mountedFilename,
			MountPath: "/etc/nginx/conf.d/default.conf",
		})
		requireNoError(t, err)
	})

	t.Run("remove mounted file", func(t *testing.T) {
		err := c.Playspecs.RemoveMountedFile(ctx(), *spec.ID, mountedFilename)
		requireNoError(t, err)
	})
}
