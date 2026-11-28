package integration

import (
	"context"
	"testing"
	"time"

	"github.com/fibegg/sdk/fibe"
)

func waitForGitHubFixtureLaunch(t *testing.T, c *fibe.Client, id int64) string {
	t.Helper()

	const timeout = 105 * time.Second
	started := time.Now()
	waitCtx, cancelWait := context.WithDeadline(ctx(), started.Add(timeout))
	defer cancelWait()
	deadline, _ := waitCtx.Deadline()

	lastStatus, lastStep := "", ""
	var lastPollErr error
	for time.Now().Before(deadline) {
		pollCtx, cancelPoll := context.WithTimeout(waitCtx, 5*time.Second)
		status, err := c.Playgrounds.Status(pollCtx, id)
		cancelPoll()
		if err != nil {
			lastPollErr = err
		} else {
			lastStatus = status.Status
			lastStep = ""
			if status.CreationStep != nil {
				lastStep = *status.CreationStep
			}
			switch lastStatus {
			case "running", "completed":
				t.Logf("fixture Playground %d reached %q after %s", id, lastStatus, time.Since(started).Round(time.Millisecond))
				return lastStatus
			case "error", "failed":
				return lastStatus
			}
		}
		time.Sleep(minWaitDuration(2*time.Second, time.Until(deadline)))
	}

	if lastPollErr != nil {
		t.Logf("last fixture Playground status poll error: %s", describeAPIError(lastPollErr))
	}

	// Playground detail exposes the saved result code as state_reason.
	diagnosticCtx, cancelDiagnostic := context.WithTimeout(ctx(), 2*time.Second)
	defer cancelDiagnostic()
	resultCode := "unavailable"
	if pg, err := c.Playgrounds.Get(diagnosticCtx, id); err != nil {
		t.Logf("fixture Playground detail read error: %s", describeAPIError(err))
	} else {
		resultCode = "unset"
		if pg.StateReason != nil {
			resultCode = *pg.StateReason
		}
		errorMessage := ""
		if pg.ErrorMessage != nil {
			errorMessage = *pg.ErrorMessage
		}
		t.Logf("fixture Playground detail: status=%q error message=%q", pg.Status, errorMessage)
	}
	t.Fatalf("fixture Playground %d timed out after %s: last status=%q creation step=%q saved result code=%q",
		id, timeout, lastStatus, lastStep, resultCode)
	return ""
}
