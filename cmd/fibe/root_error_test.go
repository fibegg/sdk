package main

import (
	"fmt"
	"testing"

	"github.com/fibegg/sdk/fibe"
)

func TestStructuredErrorFieldsUsesWrappedAPIError(t *testing.T) {
	apiErr := &fibe.APIError{
		StatusCode: 409,
		Code:       fibe.ErrCodeConflict,
		Message:    "conflict",
		Details:    map[string]any{"status": "in_progress"},
		RequestID:  "req-123",
	}

	code, status, details, requestID := structuredErrorFields(fmt.Errorf("wrapped: %w", apiErr))

	if code != fibe.ErrCodeConflict || status != 409 || requestID != "req-123" {
		t.Fatalf("got code=%q status=%d requestID=%q", code, status, requestID)
	}
	if details["status"] != "in_progress" {
		t.Fatalf("details=%#v", details)
	}
}

func TestStructuredErrorFieldsUsesPlaygroundTerminalError(t *testing.T) {
	message := "compose failed"
	step := "compose_up"
	status := &fibe.PlaygroundStatus{
		Status:             "error",
		ErrorMessage:       &message,
		ErrorStep:          &step,
		FailureDiagnostics: map[string]any{"failed_service": "web"},
		ErrorDetails:       map[string]any{"compose_failure": map[string]any{"category": "service_exit"}},
	}

	code, httpStatus, details, requestID := structuredErrorFields(fibe.NewPlaygroundTerminalStateError(status))

	if code != fibe.ErrCodePlaygroundTerminalState || httpStatus != 422 || requestID != "" {
		t.Fatalf("got code=%q status=%d requestID=%q", code, httpStatus, requestID)
	}
	if details["status"] != "error" || details["error_message"] != message || details["error_step"] != step {
		t.Fatalf("details=%#v", details)
	}
	if details["failure_diagnostics"] == nil || details["error_details"] == nil {
		t.Fatalf("details missing diagnostics: %#v", details)
	}
}
