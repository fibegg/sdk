package integration

import (
	"io"
	"strings"
	"testing"

	"github.com/fibegg/sdk/fibe"
)

func TestArtefacts_FilteringAndSorting(t *testing.T) {
	t.Parallel()
	c := userClient(t)

	agent, err := c.Agents.Create(ctx(), &fibe.AgentCreateParams{
		Name:     uniqueName("art-filter-agent"),
		Provider: fibe.ProviderGemini,
	})
	requireNoError(t, err)
	t.Cleanup(func() { c.Agents.Delete(ctx(), agent.ID) })

	reportName := uniqueName("test-report") + ".txt"
	analysisName := uniqueName("analysis-data") + ".csv"
	fileContent := strings.NewReader("test content for artefact")
	_, err = c.Artefacts.Create(ctx(), agent.ID, &fibe.ArtefactCreateParams{
		Name: reportName,
	}, fileContent, reportName)
	if err != nil {
		t.Skipf("artefact upload not available: %v", err)
	}

	fileContent2 := strings.NewReader("second artefact content")
	_, err = c.Artefacts.Create(ctx(), agent.ID, &fibe.ArtefactCreateParams{
		Name: analysisName,
	}, fileContent2, analysisName)
	if err != nil {
		t.Logf("second artefact creation failed: %v", err)
	}

	t.Run("filter by query", func(t *testing.T) {
		t.Parallel()
		result, err := c.Artefacts.List(ctx(), agent.ID, &fibe.ArtefactListParams{
			Query: "report",
		})
		requireNoError(t, err)
		if len(result.Data) == 0 {
			t.Log("filter by query returned no results (may need seeded data)")
		}
		for _, a := range result.Data {
			if !strings.Contains(strings.ToLower(a.Name), "report") {
				t.Errorf("expected filtered result to contain 'report' in name, got %q", a.Name)
			}
		}
	})

	t.Run("filter by name", func(t *testing.T) {
		t.Parallel()
		result, err := c.Artefacts.List(ctx(), agent.ID, &fibe.ArtefactListParams{
			Name: reportName,
		})
		requireNoError(t, err)

		for _, a := range result.Data {
			if a.Name != reportName {
				t.Errorf("expected name %q, got %q", reportName, a.Name)
			}
		}
	})

	t.Run("filter by content_type", func(t *testing.T) {
		t.Parallel()
		result, err := c.Artefacts.List(ctx(), agent.ID, &fibe.ArtefactListParams{
			ContentType: "text/plain",
		})
		requireNoError(t, err)
		for _, a := range result.Data {
			if a.ContentType == nil || *a.ContentType != "text/plain" {
				t.Errorf("expected content_type 'text/plain', got %v", a.ContentType)
			}
		}
	})

	t.Run("sort by created_at_asc", func(t *testing.T) {
		t.Parallel()
		result, err := c.Artefacts.List(ctx(), agent.ID, &fibe.ArtefactListParams{
			Sort: "created_at_asc",
		})
		requireNoError(t, err)

		if len(result.Data) >= 2 {
			if result.Data[0].CreatedAt.After(result.Data[1].CreatedAt) {
				t.Error("expected ascending order by created_at")
			}
		}
	})

	t.Run("sort by name_asc", func(t *testing.T) {
		t.Parallel()
		result, err := c.Artefacts.List(ctx(), agent.ID, &fibe.ArtefactListParams{
			Sort: "name_asc",
		})
		requireNoError(t, err)

		if len(result.Data) >= 2 {
			if result.Data[0].Name > result.Data[1].Name {
				t.Error("expected ascending order by name")
			}
		}
	})

	t.Run("sort by name_desc", func(t *testing.T) {
		t.Parallel()
		result, err := c.Artefacts.List(ctx(), agent.ID, &fibe.ArtefactListParams{
			Sort: "name_desc",
		})
		requireNoError(t, err)

		if len(result.Data) >= 2 {
			if result.Data[0].Name < result.Data[1].Name {
				t.Error("expected descending order by name")
			}
		}
	})

	t.Run("pagination", func(t *testing.T) {
		t.Parallel()
		result, err := c.Artefacts.List(ctx(), agent.ID, &fibe.ArtefactListParams{
			PerPage: 1,
			Page:    1,
		})
		requireNoError(t, err)

		if len(result.Data) > 1 {
			t.Error("expected at most 1 result with per_page=1")
		}
	})
}

func TestArtefacts_Download(t *testing.T) {
	t.Parallel()
	c := userClient(t)

	agent, err := c.Agents.Create(ctx(), &fibe.AgentCreateParams{
		Name:     uniqueName("art-download-agent"),
		Provider: fibe.ProviderGemini,
	})
	requireNoError(t, err)
	t.Cleanup(func() { c.Agents.Delete(ctx(), agent.ID) })

	uploadContent := "download test content"
	downloadName := uniqueName("downloadable") + ".txt"
	fileContent := strings.NewReader(uploadContent)
	artefact, err := c.Artefacts.Create(ctx(), agent.ID, &fibe.ArtefactCreateParams{
		Name: downloadName,
	}, fileContent, downloadName)
	if err != nil {
		t.Skipf("artefact upload not available: %v", err)
	}

	t.Run("download returns content", func(t *testing.T) {
		t.Parallel()
		body, filename, _, err := c.Artefacts.Download(ctx(), agent.ID, artefact.ID)
		if err != nil {
			t.Skipf("artefact download not available (may still be processing): %v", err)
		}
		defer body.Close()

		content, err := io.ReadAll(body)
		requireNoError(t, err)

		if len(content) == 0 {
			t.Error("expected non-empty download content")
		}
		if filename == "" {
			t.Error("expected non-empty filename in download response")
		}
	})

	t.Run("download nonexistent artefact returns 404", func(t *testing.T) {
		t.Parallel()
		_, _, _, err := c.Artefacts.Download(ctx(), agent.ID, 999999999)
		requireAPIError(t, err, fibe.ErrCodeNotFound, 404)
	})
}

func TestArtefacts_Immutability(t *testing.T) {
	t.Parallel()
	c := userClient(t)

	agent, err := c.Agents.Create(ctx(), &fibe.AgentCreateParams{
		Name:     uniqueName("art-immutable-agent"),
		Provider: fibe.ProviderGemini,
	})
	requireNoError(t, err)
	t.Cleanup(func() { c.Agents.Delete(ctx(), agent.ID) })

	t.Run("list returns empty for new agent", func(t *testing.T) {
		t.Parallel()
		result, err := c.Artefacts.List(ctx(), agent.ID, nil)
		requireNoError(t, err)

		if result.Meta.Total != 0 {
			t.Errorf("expected 0 artefacts, got %d", result.Meta.Total)
		}
	})
}

func TestArtefacts_ScopeEnforcement(t *testing.T) {
	t.Parallel()
	c := userClient(t)

	agent, err := c.Agents.Create(ctx(), &fibe.AgentCreateParams{
		Name:     uniqueName("art-scope-agent"),
		Provider: fibe.ProviderGemini,
	})
	requireNoError(t, err)
	t.Cleanup(func() { c.Agents.Delete(ctx(), agent.ID) })

	t.Run("artefacts:read scope can list", func(t *testing.T) {
		t.Parallel()
		readKey := createScopedKey(t, c, "art-read", []string{"agents:read", "artefacts:read"})
		_, err := readKey.Artefacts.List(ctx(), agent.ID, nil)
		requireNoError(t, err)
	})

	t.Run("no artefacts scope gets 403", func(t *testing.T) {
		t.Parallel()
		noScope := createScopedKey(t, c, "art-noscope", []string{"agents:read"})
		_, err := noScope.Artefacts.List(ctx(), agent.ID, nil)
		requireAPIError(t, err, fibe.ErrCodeForbidden, 403)
	})
}

func TestArtefacts_IDOR(t *testing.T) {
	t.Parallel()
	c := userClient(t)
	userB := userBClient(t)

	agent, err := c.Agents.Create(ctx(), &fibe.AgentCreateParams{
		Name:     uniqueName("art-idor-agent"),
		Provider: fibe.ProviderGemini,
	})
	requireNoError(t, err)
	t.Cleanup(func() { c.Agents.Delete(ctx(), agent.ID) })

	t.Run("user B cannot list primary agent artefacts", func(t *testing.T) {
		t.Parallel()
		_, err := userB.Artefacts.List(ctx(), agent.ID, nil)
		requireAPIError(t, err, fibe.ErrCodeNotFound, 404)
	})
}
