package integration

import (
	"testing"

	"github.com/fibegg/sdk/fibe"
)

// TestTasks_FullLifecycle exercises:
func TestTasks_FullLifecycle(t *testing.T) {
	c := userClient(t)

	hostID := testHostID(t)
	if hostID == 0 {
		t.Skip("set FIBE_TEST_HOST_ID to run tasks lifecycle")
	}

	jm := true
	spec := seedSpec(t, c, func(p *fibe.SpecCreateParams) {
		p.JobMode = &jm
		p.BaseComposeYAML = jobComposeYAML()
		p.Services = []fibe.SpecServiceDef{jobWatchedService("worker")}
	})

	task, err := c.Tasks.Trigger(ctx(), &fibe.TaskTriggerParams{
		SpecID: *spec.ID,
		HostID: &hostID,
	})
	requireNoError(t, err)
	t.Cleanup(func() { c.Tasks.Delete(ctx(), task.ID) })

	if task.ID == 0 || task.Name == "" || task.Status == "" {
		t.Errorf("trigger missing core fields: id=%d name=%q status=%q", task.ID, task.Name, task.Status)
	}
	if !task.JobMode {
		t.Error("expected JobMode=true for task")
	}

	t.Run("list q finds triggered task", func(t *testing.T) {
		r, err := c.Tasks.List(ctx(), &fibe.PlaygroundListParams{Q: task.Name, PerPage: 50})
		requireNoError(t, err)
		found := false
		for _, listed := range r.Data {
			if listed.ID == task.ID {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected Tasks.List(Q=%q) to include task %d", task.Name, task.ID)
		}
	})

	t.Run("status reaches terminal state", func(t *testing.T) {
		final := waitForTaskTerminal(t, c, task.ID, CapWaitTimeout)
		if final == "" {
			t.Error("task status never left empty")
		}
		t.Logf("task final status: %s", final)
	})

	t.Run("get status returns ID, Status, potentially JobResult", func(t *testing.T) {
		s, err := c.Tasks.Status(ctx(), task.ID)
		requireNoError(t, err)
		if s.ID != task.ID {
			t.Errorf("expected ID=%d, got %d", task.ID, s.ID)
		}
		if s.Status == "" {
			t.Error("expected non-empty Status")
		}
		if s.Status == "completed" && s.JobResult != nil {
			if s.JobResult.CompletedAt == nil {
				t.Error("expected CompletedAt on completed JobResult")
			}
		}
	})

	t.Run("rerun creates new task", func(t *testing.T) {
		re, err := c.Tasks.Rerun(ctx(), task.ID)
		if err != nil {
			if apiErr, ok := err.(*fibe.APIError); ok && apiErr.StatusCode == 409 {
				t.Skipf("rerun rejected: %s", apiErr.Message)
			}
			requireNoError(t, err)
		}
		t.Cleanup(func() { c.Tasks.Delete(ctx(), re.ID) })
		if re.ID == task.ID {
			t.Error("expected rerun to produce a NEW task ID")
		}
		if re.SpecID == nil || *re.SpecID != *spec.ID {
			t.Errorf("expected rerun SpecID=%d, got %v", *spec.ID, re.SpecID)
		}
	})
}

func TestTasks_ListOnlyShowsJobMode(t *testing.T) {
	t.Parallel()
	c := userClient(t)

	r, err := c.Tasks.List(ctx(), &fibe.PlaygroundListParams{PerPage: 50})
	requireNoError(t, err)
	for _, tr := range r.Data {
		if !tr.JobMode {
			t.Errorf("Tasks.List returned non-job-mode playground %d: %s", tr.ID, tr.Name)
		}
	}
}

func TestTasks_TriggerAutoName(t *testing.T) {
	t.Parallel()
	c := userClient(t)

	hostID := testHostID(t)
	if hostID == 0 {
		t.Skip("set FIBE_TEST_HOST_ID")
	}

	jm := true
	spec := seedSpec(t, c, func(p *fibe.SpecCreateParams) {
		p.JobMode = &jm
		p.BaseComposeYAML = jobComposeYAML()
		p.Services = []fibe.SpecServiceDef{jobWatchedService("worker")}
	})

	task, err := c.Tasks.Trigger(ctx(), &fibe.TaskTriggerParams{
		SpecID: *spec.ID,
		HostID: &hostID,
	})
	requireNoError(t, err)
	t.Cleanup(func() { c.Tasks.Delete(ctx(), task.ID) })

	// Auto-generated name should start with spec name
	if task.Name == "" {
		t.Error("expected auto-generated name")
	}
}
