package fibe

import (
	"encoding/json"
	"fmt"
	"strings"
)

const ErrCodePlaygroundTerminalState = "PLAYGROUND_TERMINAL_STATE"

type PlaygroundTerminalError struct {
	Status *PlaygroundStatus
}

func NewPlaygroundTerminalStateError(status *PlaygroundStatus) *PlaygroundTerminalError {
	return &PlaygroundTerminalError{Status: status}
}

func (e *PlaygroundTerminalError) Error() string {
	return PlaygroundTerminalStateError(e.Status)
}

func (e *PlaygroundTerminalError) Details() map[string]any {
	if e == nil || e.Status == nil {
		return nil
	}
	status := e.Status
	details := map[string]any{
		"status": status.Status,
	}
	if status.StateReason != nil && strings.TrimSpace(*status.StateReason) != "" {
		details["state_reason"] = strings.TrimSpace(*status.StateReason)
	}
	if len(status.StateReasons) > 0 {
		details["state_reasons"] = status.StateReasons
	}
	if status.ErrorMessage != nil && strings.TrimSpace(*status.ErrorMessage) != "" {
		details["error_message"] = strings.TrimSpace(*status.ErrorMessage)
	}
	if status.ErrorStep != nil && strings.TrimSpace(*status.ErrorStep) != "" {
		details["error_step"] = strings.TrimSpace(*status.ErrorStep)
	}
	if status.ErrorStepLabel != nil && strings.TrimSpace(*status.ErrorStepLabel) != "" {
		details["error_step_label"] = strings.TrimSpace(*status.ErrorStepLabel)
	}
	if len(status.FailureDiagnostics) > 0 {
		details["failure_diagnostics"] = status.FailureDiagnostics
	}
	if len(status.ErrorDetails) > 0 {
		details["error_details"] = status.ErrorDetails
	}
	if len(status.BuildStatuses) > 0 {
		details["build_statuses"] = status.BuildStatuses
	}
	return details
}

func PlaygroundTerminalStateError(status *PlaygroundStatus) string {
	if status == nil {
		return "playground reached terminal state"
	}

	lines := []string{fmt.Sprintf("playground reached terminal state: %s", status.Status)}
	if status.StateReason != nil && strings.TrimSpace(*status.StateReason) != "" {
		lines = append(lines, fmt.Sprintf("state_reason: %s", strings.TrimSpace(*status.StateReason)))
	}
	if len(status.StateReasons) > 1 {
		lines = append(lines, fmt.Sprintf("state_reasons: %s", strings.Join(status.StateReasons, "; ")))
	}
	if status.ErrorMessage != nil && strings.TrimSpace(*status.ErrorMessage) != "" {
		lines = append(lines, fmt.Sprintf("error_message: %s", strings.TrimSpace(*status.ErrorMessage)))
	}
	if status.ErrorStep != nil && strings.TrimSpace(*status.ErrorStep) != "" {
		step := strings.TrimSpace(*status.ErrorStep)
		if status.ErrorStepLabel != nil && strings.TrimSpace(*status.ErrorStepLabel) != "" {
			step = fmt.Sprintf("%s (%s)", step, strings.TrimSpace(*status.ErrorStepLabel))
		}
		lines = append(lines, fmt.Sprintf("error_step: %s", step))
	}
	appendJSONBlock := func(label string, value map[string]any) {
		if len(value) == 0 {
			return
		}
		encoded, err := json.MarshalIndent(value, "", "  ")
		if err != nil {
			lines = append(lines, fmt.Sprintf("%s: %#v", label, value))
			return
		}
		lines = append(lines, fmt.Sprintf("%s:\n%s", label, encoded))
	}
	appendJSONBlock("failure_diagnostics", status.FailureDiagnostics)
	appendJSONBlock("error_details", status.ErrorDetails)
	if len(status.BuildStatuses) > 0 {
		encoded, err := json.MarshalIndent(status.BuildStatuses, "", "  ")
		if err == nil {
			lines = append(lines, fmt.Sprintf("build_statuses:\n%s", encoded))
		}
	}
	return strings.Join(lines, "\n")
}
