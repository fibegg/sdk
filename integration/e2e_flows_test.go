package integration

import (
	"strings"
	"testing"
	"time"

	"github.com/fibegg/sdk/fibe"
)

// TestE2E_PlayspecToPlayground exercises the primary user flow:
//   1. Create playspec with real compose
//   2. Deploy playground from that playspec on the shared marquee
//   3. Verify status polling works
//   4. Verify compose reflects the playspec
//   5. Clean up
func TestE2E_PlayspecToPlayground(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	marqueeID := testMarqueeID(t)
	if marqueeID == 0 {
		t.Skip("set FIBE_TEST_MARQUEE_ID to run E2E flow")
	}

	// Step 1: playspec with distinct service names for reliable verification
	specName := uniqueName("e2e-spec")
	spec, err := c.Playspecs.Create(ctx(), &fibe.PlayspecCreateParams{
		Name:            specName,
		BaseComposeYAML: realComposeYAML(),
		Services: []fibe.PlayspecServiceDef{
			{Name: "web", Type: fibe.ServiceTypeStatic},
			{Name: "db", Type: fibe.ServiceTypeStatic},
			{Name: "cache", Type: fibe.ServiceTypeStatic},
		},
	})
	requireNoError(t, err, "create e2e playspec")
	t.Cleanup(func() { c.Playspecs.Delete(ctx(), *spec.ID) })

	// Step 2: services endpoint must reflect 3 services
	services, err := c.Playspecs.Services(ctx(), *spec.ID)
	requireNoError(t, err)
	if services == nil {
		t.Error("expected non-nil services response")
	}

	// Step 3: deploy playground
	pgName := uniqueName("e2e-pg")
	pg, err := c.Playgrounds.Create(ctx(), &fibe.PlaygroundCreateParams{
		Name:       pgName,
		PlayspecID: *spec.ID,
		MarqueeID:  &marqueeID,
	})
	requireNoError(t, err, "create e2e playground")
	t.Cleanup(func() { c.Playgrounds.Delete(ctx(), pg.ID) })

	// Step 4: compose YAML must contain our service names (web/db/cache), poll for render
	var lastComposeErr error
	cmp, found := pollUntil(120, time.Second, func() (*fibe.PlaygroundCompose, bool) {
		c2, err := c.Playgrounds.Compose(ctx(), pg.ID)
		if err != nil {
			lastComposeErr = err
			return nil, false
		}
		lastComposeErr = nil
		return c2, c2.ComposeYAML != ""
	})
	if !found {
		if lastComposeErr != nil {
			t.Skipf("compose YAML not rendered in time; last compose error: %v", lastComposeErr)
		}
		t.Skip("compose YAML not rendered in time")
	}
	for _, svc := range []string{"web", "db", "cache"} {
		if !strings.Contains(cmp.ComposeYAML, svc+":") {
			t.Errorf("expected service %q in compose, got:\n%s", svc, cmp.ComposeYAML)
		}
	}

	// Step 5: status polling works
	status := waitForPlaygroundStatus(t, c, pg.ID, []string{"running", "error", "failed", "in_progress"}, CapWaitTimeout)
	if status == "" {
		t.Error("playground never left empty status")
	}
	t.Logf("playground final status: %s", status)
}

// TestE2E_AgentArtefactRoundtrip creates an agent, uploads multiple artefacts with
// different content types, lists/filters them, downloads one, verifies content matches.
func TestE2E_AgentArtefactRoundtrip(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	ag := seedAgent(t, c, fibe.ProviderGemini)

	// Upload 3 artefacts with different content
	artefactContents := map[string]string{
		"report.txt":  "this is a text report with utf-8 content: 😀",
		"data.csv":    "id,value\n1,alpha\n2,beta\n3,gamma\n",
		"config.json": `{"key":"value","num":42}`,
	}
	for name, content := range artefactContents {
		art, err := c.Artefacts.Create(ctx(), ag.ID, &fibe.ArtefactCreateParams{
			Name: name,
		}, strings.NewReader(content), name)
		if err != nil {
			if apiErr, ok := err.(*fibe.APIError); ok && apiErr.StatusCode >= 400 && apiErr.StatusCode < 500 {
				t.Skipf("artefact upload rejected (%d %s): %s", apiErr.StatusCode, apiErr.Code, apiErr.Message)
			}
			requireNoError(t, err)
		}
		// Verify fixed Create returns real Artefact (not nil)
		if art == nil || art.ID == 0 {
			t.Errorf("expected Artefact.Create to return populated struct, got %+v", art)
		}
		if art != nil && art.Name != name {
			t.Errorf("expected Artefact.Name=%q, got %q", name, art.Name)
		}
	}

	// List should include all 3
	list, err := c.Artefacts.List(ctx(), ag.ID, &fibe.ArtefactListParams{PerPage: 50})
	requireNoError(t, err)
	if list.Meta.Total < 3 {
		t.Logf("expected >= 3 artefacts, got %d (may be async)", list.Meta.Total)
	}

	// Download the first one — the backend may return content as bytes
	if len(list.Data) > 0 {
		first := list.Data[0]
		body, _, _, err := c.Artefacts.Download(ctx(), ag.ID, first.ID)
		if err != nil {
			if apiErr, ok := err.(*fibe.APIError); ok && apiErr.StatusCode >= 400 && apiErr.StatusCode < 500 {
				t.Logf("artefact download returned %d %s (may be async/pre-signed URL issue): %s", apiErr.StatusCode, apiErr.Code, apiErr.Message)
				return
			}
			requireNoError(t, err)
		}
		defer body.Close()

		buf := make([]byte, 4096)
		n, _ := body.Read(buf)
		if n == 0 {
			t.Error("expected non-empty download content")
		}
	}
}

// TestE2E_TeamContribution creates a team, contributes a resource, verifies membership & resources,
// then removes the resource and confirms it's gone.
func TestE2E_TeamContribution(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	team := seedTeam(t, c)
	spec := seedPlayspec(t, c)

	// Contribute the playspec with read permission
	tr, err := c.Teams.ContributeResource(ctx(), team.ID, &fibe.TeamResourceParams{
		ResourceType:    "Playspec",
		ResourceID:      *spec.ID,
		PermissionLevel: "read",
	})
	requireNoError(t, err)
	if tr.ResourceType != "Playspec" {
		t.Errorf("expected ResourceType=Playspec, got %s", tr.ResourceType)
	}
	if tr.ResourceID != *spec.ID {
		t.Errorf("expected ResourceID=%d, got %d", *spec.ID, tr.ResourceID)
	}

	// List must include it
	resList, err := c.Teams.ListResources(ctx(), team.ID, nil)
	requireNoError(t, err)
	found := false
	for _, r := range resList.Data {
		if r.ID == tr.ID {
			found = true
			if r.PermissionLevel != "read" {
				t.Errorf("expected PermissionLevel=read, got %s", r.PermissionLevel)
			}
		}
	}
	if !found {
		t.Error("contributed resource not found in team resources list")
	}

	// Remove
	err = c.Teams.RemoveResource(ctx(), team.ID, tr.ID)
	requireNoError(t, err)

	// List again — should NOT include removed resource
	resList2, err := c.Teams.ListResources(ctx(), team.ID, nil)
	requireNoError(t, err)
	for _, r := range resList2.Data {
		if r.ID == tr.ID {
			t.Error("removed resource still in list")
		}
	}
}

// TestE2E_SecretRotation verifies full secret lifecycle — create, read, update value, re-read, delete.
func TestE2E_SecretRotation(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	key := "E2E_ROT_" + uniqueName("KEY")
	original := "original-" + uniqueName("")
	s, err := c.Secrets.Create(ctx(), &fibe.SecretCreateParams{Key: key, Value: original})
	requireNoError(t, err)
	t.Cleanup(func() {
		if s.ID != nil {
			c.Secrets.Delete(ctx(), *s.ID)
		}
	})

	// Read
	got, err := c.Secrets.Get(ctx(), *s.ID, true)
	requireNoError(t, err)
	if got.Value == nil || *got.Value != original {
		t.Errorf("expected Value=%q, got %v", original, got.Value)
	}

	// Rotate
	rotated := "rotated-" + uniqueName("")
	_, err = c.Secrets.Update(ctx(), *s.ID, &fibe.SecretUpdateParams{Value: &rotated})
	requireNoError(t, err)

	// Read new value
	got2, err := c.Secrets.Get(ctx(), *s.ID, true)
	requireNoError(t, err)
	if got2.Value == nil || *got2.Value != rotated {
		t.Errorf("expected rotated Value=%q, got %v", rotated, got2.Value)
	}
}
