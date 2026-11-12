package integration

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/fibegg/sdk/fibe"
)

// waitForPlaygroundStatus polls playground status until it matches one of the
// target statuses, or times out. Returns the final status seen.
//
// Timeout is aggressively capped to keep total suite time bounded under CI limits.
// If the provided timeout exceeds CapWaitTimeout it is clamped.
func waitForPlaygroundStatus(t *testing.T, c *fibe.Client, id int64, targets []string, timeout time.Duration) string {
	t.Helper()
	if timeout > CapWaitTimeout {
		timeout = CapWaitTimeout
	}
	return waitForPlaygroundStatusWithin(t, c, id, targets, timeout)
}

func waitForPlaygroundStatusWithin(t *testing.T, c *fibe.Client, id int64, targets []string, timeout time.Duration) string {
	t.Helper()
	waitCtx, waitCancel := ctxTimeout(minWaitDuration(timeout, 90*time.Second))
	defer waitCancel()
	deadline, _ := waitCtx.Deadline()
	interval := 2 * time.Second
	lastStatus := ""
	for time.Now().Before(deadline) {
		cctx, cancel := context.WithTimeout(waitCtx, 5*time.Second)
		s, err := c.Playgrounds.Status(cctx, id)
		cancel()
		if err != nil {
			t.Logf("status poll error: %v", err)
			time.Sleep(minWaitDuration(interval, time.Until(deadline)))
			continue
		}
		lastStatus = s.Status
		for _, target := range targets {
			if s.Status == target {
				return s.Status
			}
		}
		time.Sleep(minWaitDuration(interval, time.Until(deadline)))
	}
	t.Logf("wait timeout after %s: last status=%q, wanted one of %v", timeout, lastStatus, targets)
	return lastStatus
}

// CapWaitTimeout caps any wait to this value so a single test can't consume
// the full suite timeout.
const CapWaitTimeout = 20 * time.Second
const PlaygroundLaunchWaitTimeout = 90 * time.Second
const playgroundActionRetryTimeout = 90 * time.Second

// waitForTaskTerminal polls task status until completed/error/failed.
func waitForTaskTerminal(t *testing.T, c *fibe.Client, id int64, timeout time.Duration) string {
	t.Helper()
	return waitForPlaygroundStatus(t, c, id, []string{"completed", "error", "failed", "destroyed"}, timeout)
}

// webhookTimeout returns the canonical timeout for webhook delivery polling.
func webhookTimeout() time.Duration {
	return 90 * time.Second
}

func skipIfPlaygroundActionStateRejected(t *testing.T, err error, action string) bool {
	t.Helper()

	if message, ok := playgroundActionStateRejected(err); ok {
		t.Logf("%s rejected by current playground state: %s", action, message)
		return false
	}
	return false
}

func playgroundActionEventuallyAccepted(t *testing.T, c *fibe.Client, id int64, actionType string, actionLabel string) *fibe.PlaygroundStatus {
	t.Helper()

	waitCtx, cancel := ctxTimeout(playgroundActionRetryTimeout)
	defer cancel()
	deadline, _ := waitCtx.Deadline()
	var lastErr error
	for attempt := 1; ; attempt++ {
		accepted, err := c.Playgrounds.Action(waitCtx, id, &fibe.PlaygroundActionParams{ActionType: actionType})
		if err == nil {
			return accepted
		}
		lastErr = err
		message, rejected := playgroundActionStateRejected(err)
		if !rejected {
			requireNoError(t, err)
		}
		if time.Now().After(deadline) {
			t.Fatalf("%s was not accepted within %s after %d attempts: %s", actionLabel, playgroundActionRetryTimeout, attempt, describeAPIError(lastErr))
		}
		t.Logf("%s waiting for playground action readiness: %s", actionLabel, message)
		time.Sleep(minWaitDuration(3*time.Second, time.Until(deadline)))
	}
}

func playgroundActionStateRejected(err error) (string, bool) {
	apiErr, ok := err.(*fibe.APIError)
	if !ok {
		return "", false
	}
	if apiErr.StatusCode == 409 || (apiErr.StatusCode == 422 && apiErr.Code == "INVALID_STATE") {
		return apiErr.Message, true
	}
	if apiErr.Code == "REMOTE_REQUEST_FAILED" && strings.Contains(apiErr.Message, "deployment is already active") {
		return apiErr.Message, true
	}
	return "", false
}

// pollUntil retries fn up to `attempts` times with delay; returns first non-nil result.
func pollUntil[T any](attempts int, delay time.Duration, fn func(context.Context) (T, bool)) (T, bool) {
	var zero T
	pollCtx, cancel := ctxTimeout(90 * time.Second)
	defer cancel()
	deadline, _ := pollCtx.Deadline()
	for i := 0; i < attempts && time.Now().Before(deadline); i++ {
		v, ok := fn(pollCtx)
		if ok {
			return v, true
		}
		time.Sleep(minWaitDuration(delay, time.Until(deadline)))
	}
	return zero, false
}

// describeAPIError formats an APIError for test logs with request id.
func describeAPIError(err error) string {
	if err == nil {
		return "<nil>"
	}
	if apiErr, ok := err.(*fibe.APIError); ok {
		return fmt.Sprintf("%s (%d) %s [req:%s]", apiErr.Code, apiErr.StatusCode, apiErr.Message, apiErr.RequestID)
	}
	return err.Error()
}

func minWaitDuration(a, b time.Duration) time.Duration {
	if a < b {
		return a
	}
	return b
}
