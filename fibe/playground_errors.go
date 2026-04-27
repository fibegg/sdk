package fibe

import (
	"encoding/json"
	"fmt"
	"strings"
)

func PlaygroundTerminalStateError(status *PlaygroundStatus) string {
	if status == nil {
		return "playground reached terminal state"
	}

	lines := []string{fmt.Sprintf("playground reached terminal state: %s", status.Status)}
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
	return strings.Join(lines, "\n")
}
