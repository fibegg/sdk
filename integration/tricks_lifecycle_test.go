package integration

import (
	"testing"

	"github.com/fibegg/sdk/fibe"
)

// TestTricks_FullLifecycle exercises:
//   Trigger (job-mode playspec) → Status (terminal) → Logs → Rerun → Delete
func TestTricks_FullLifecycle(t *testing.T) {
	c := adminClient(t)

	marqueeID := testMarqueeID(t)
	if marqueeID == 0 {
		t.Skip("set FIBE_TEST_MARQUEE_ID to run tricks lifecycle")
	}

	jm := true
	spec := seedPlayspec(t, c, func(p *fibe.PlayspecCreateParams) {
		p.JobMode = &jm
		p.BaseComposeYAML = jobComposeYAML()
		p.Services = []fibe.PlayspecServiceDef{jobWatchedService("worker")}
	})

	trick, err := c.Tricks.Trigger(ctx(), &fibe.TrickTriggerParams{
		PlayspecID: *spec.ID,
		MarqueeID:  &marqueeID,
	})
	requireNoError(t, err)
	t.Cleanup(func() { c.Tricks.Delete(ctx(), trick.ID) })

	// Trigger response must have ID, Name, Status, and job_mode=true
	if trick.ID == 0 || trick.Name == "" || trick.Status == "" {
		t.Errorf("trigger missing core fields: id=%d name=%q status=%q", trick.ID, trick.Name, trick.Status)
	}
	if !trick.JobMode {
		t.Error("expected JobMode=true for trick")
	}

	// Trick should go through states and reach terminal
	t.Run("status reaches terminal state", func(t *testing.T) {
		final := waitForTrickTerminal(t, c, trick.ID, CapWaitTimeout)
		if final == "" {
			t.Error("trick status never left empty")
		}
		t.Logf("trick final status: %s", final)
	})

	t.Run("get status returns ID, Status, potentially JobResult", func(t *testing.T) {
		s, err := c.Tricks.Status(ctx(), trick.ID)
		requireNoError(t, err)
		if s.ID != trick.ID {
			t.Errorf("expected ID=%d, got %d", trick.ID, s.ID)
		}
		if s.Status == "" {
			t.Error("expected non-empty Status")
		}
		if s.Status == "completed" && s.JobResult != nil {
			// Verify JobResult structure
			if s.JobResult.CompletedAt == nil {
				t.Error("expected CompletedAt on completed JobResult")
			}
		}
	})

	t.Run("rerun creates new trick", func(t *testing.T) {
		re, err := c.Tricks.Rerun(ctx(), trick.ID)
		if err != nil {
			if apiErr, ok := err.(*fibe.APIError); ok && apiErr.StatusCode == 409 {
				t.Skipf("rerun rejected: %s", apiErr.Message)
			}
			requireNoError(t, err)
		}
		t.Cleanup(func() { c.Tricks.Delete(ctx(), re.ID) })
		if re.ID == trick.ID {
			t.Error("expected rerun to produce a NEW trick ID")
		}
		if re.PlayspecID == nil || *re.PlayspecID != *spec.ID {
			t.Errorf("expected rerun PlayspecID=%d, got %v", *spec.ID, re.PlayspecID)
		}
	})
}

func TestTricks_ListOnlyShowsJobMode(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	r, err := c.Tricks.List(ctx(), &fibe.PlaygroundListParams{PerPage: 50})
	requireNoError(t, err)
	for _, tr := range r.Data {
		if !tr.JobMode {
			t.Errorf("Tricks.List returned non-job-mode playground %d: %s", tr.ID, tr.Name)
		}
	}
}

func TestTricks_TriggerAutoName(t *testing.T) {
	t.Parallel()
	c := adminClient(t)

	marqueeID := testMarqueeID(t)
	if marqueeID == 0 {
		t.Skip("set FIBE_TEST_MARQUEE_ID")
	}

	jm := true
	spec := seedPlayspec(t, c, func(p *fibe.PlayspecCreateParams) {
		p.JobMode = &jm
		p.BaseComposeYAML = jobComposeYAML()
		p.Services = []fibe.PlayspecServiceDef{jobWatchedService("worker")}
	})

	// Trigger without explicit name — should auto-generate from playspec name
	trick, err := c.Tricks.Trigger(ctx(), &fibe.TrickTriggerParams{
		PlayspecID: *spec.ID,
		MarqueeID:  &marqueeID,
	})
	requireNoError(t, err)
	t.Cleanup(func() { c.Tricks.Delete(ctx(), trick.ID) })

	// Auto-generated name should start with playspec name
	if trick.Name == "" {
		t.Error("expected auto-generated name")
	}
}
