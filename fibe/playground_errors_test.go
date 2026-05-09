package fibe

import (
	"errors"
	"strings"
	"testing"
)

func TestPlaygroundTerminalErrorPreservesStatusDetails(t *testing.T) {
	message := "compose failed"
	step := "compose_up"
	err := NewPlaygroundTerminalStateError(&PlaygroundStatus{
		Status:             "error",
		ErrorMessage:       &message,
		ErrorStep:          &step,
		FailureDiagnostics: map[string]any{"failed_service": "web"},
		ErrorDetails:       map[string]any{"compose_failure": map[string]any{"category": "service_exit"}},
	})

	var terminalErr *PlaygroundTerminalError
	if !errors.As(err, &terminalErr) {
		t.Fatal("expected errors.As to find PlaygroundTerminalError")
	}
	if !strings.Contains(err.Error(), "error_step: compose_up") {
		t.Fatalf("error=%q", err.Error())
	}
	details := terminalErr.Details()
	if details["status"] != "error" || details["error_message"] != message || details["error_step"] != step {
		t.Fatalf("details=%#v", details)
	}
	if details["failure_diagnostics"] == nil || details["error_details"] == nil {
		t.Fatalf("details missing diagnostics: %#v", details)
	}
}
