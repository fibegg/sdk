package fibe

import (
	"errors"
	"testing"
)

func TestAPIError_Error(t *testing.T) {
	err := &APIError{
		StatusCode: 404,
		Code:       ErrCodeNotFound,
		Message:    "Not found",
	}
	s := err.Error()
	if s == "" {
		t.Error("expected non-empty error string")
	}
}

func TestAPIError_ErrorWithDetails(t *testing.T) {
	err := &APIError{
		StatusCode: 422,
		Code:       ErrCodeValidationFailed,
		Message:    "Validation failed",
		Details:    map[string]any{"name": []any{"can't be blank"}},
	}
	s := err.Error()
	if s == "" {
		t.Error("expected non-empty error string")
	}
}

func TestAPIError_IsRetryable(t *testing.T) {
	tests := []struct {
		status int
		want   bool
	}{
		{429, true},
		{500, true},
		{502, true},
		{503, true},
		{504, true},
		{400, false},
		{401, false},
		{403, false},
		{404, false},
		{422, false},
	}

	for _, tt := range tests {
		err := &APIError{StatusCode: tt.status}
		if got := err.IsRetryable(); got != tt.want {
			t.Errorf("IsRetryable() for status %d: got %v, want %v", tt.status, got, tt.want)
		}
	}
}

func TestAPIError_Helpers(t *testing.T) {
	notFound := &APIError{Code: ErrCodeNotFound}
	if !notFound.IsNotFound() {
		t.Error("expected IsNotFound")
	}

	forbidden := &APIError{Code: ErrCodeForbidden}
	if !forbidden.IsForbidden() {
		t.Error("expected IsForbidden")
	}

	unauth := &APIError{Code: ErrCodeUnauthorized}
	if !unauth.IsUnauthorized() {
		t.Error("expected IsUnauthorized")
	}

	rateLimited := &APIError{StatusCode: 429}
	if !rateLimited.IsRateLimited() {
		t.Error("expected IsRateLimited")
	}

	validation := &APIError{Code: ErrCodeValidationFailed}
	if !validation.IsValidation() {
		t.Error("expected IsValidation")
	}
}

func TestAPIError_ErrorsAs(t *testing.T) {
	var apiErr *APIError
	err := error(&APIError{StatusCode: 404, Code: ErrCodeNotFound, Message: "not found"})

	if !errors.As(err, &apiErr) {
		t.Error("expected errors.As to work with *APIError")
	}
	if apiErr.Code != ErrCodeNotFound {
		t.Errorf("expected code %q, got %q", ErrCodeNotFound, apiErr.Code)
	}
}

func TestCircuitOpenError(t *testing.T) {
	err := &CircuitOpenError{Resource: "/api/playgrounds"}
	s := err.Error()
	if s == "" {
		t.Error("expected non-empty error string")
	}
}
