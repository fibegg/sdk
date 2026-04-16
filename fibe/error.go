package fibe

import (
	"fmt"
	"time"
)

const (
	ErrCodeUnauthorized     = "UNAUTHORIZED"
	ErrCodeForbidden        = "FORBIDDEN"
	ErrCodeNotFound         = "RESOURCE_NOT_FOUND"
	ErrCodeValidationFailed = "VALIDATION_FAILED"
	ErrCodeQuotaExceeded    = "QUOTA_EXCEEDED"
	ErrCodeConflict         = "CONFLICT"
	ErrCodeLocked           = "LOCKED"
	ErrCodeBadRequest       = "BAD_REQUEST"
	ErrCodeNotImplemented   = "NOT_IMPLEMENTED"
	ErrCodeInternalError    = "INTERNAL_ERROR"
	ErrCodeFeatureDisabled  = "FEATURE_DISABLED"
	ErrCodeRateLimited      = "RATE_LIMITED"
)

type APIError struct {
	StatusCode         int                    `json:"-"`
	Code               string                 `json:"code"`
	Message            string                 `json:"message"`
	Details            map[string]any         `json:"details,omitempty"`
	RetryAfter         time.Duration          `json:"-"`
	RequestID          string                 `json:"-"`
	IdempotentReplayed bool                   `json:"-"`
}

func (e *APIError) Error() string {
	prefix := "fibe"
	if e.RequestID != "" {
		prefix = fmt.Sprintf("fibe [%s]", e.RequestID)
	}
	if e.Details != nil {
		return fmt.Sprintf("%s: %s (%d): %s — %v", prefix, e.Code, e.StatusCode, e.Message, e.Details)
	}
	return fmt.Sprintf("%s: %s (%d): %s", prefix, e.Code, e.StatusCode, e.Message)
}

// Unwrap returns nil; APIError is a terminal error.
// Implementing Unwrap allows callers to use errors.As(err, &apiErr).
func (e *APIError) Unwrap() error { return nil }

func (e *APIError) IsRetryable() bool {
	switch e.StatusCode {
	case 429, 500, 502, 503, 504:
		return true
	default:
		return false
	}
}

func (e *APIError) IsNotFound() bool     { return e.Code == ErrCodeNotFound }
func (e *APIError) IsForbidden() bool     { return e.Code == ErrCodeForbidden }
func (e *APIError) IsUnauthorized() bool  { return e.Code == ErrCodeUnauthorized }
func (e *APIError) IsRateLimited() bool   { return e.StatusCode == 429 }
func (e *APIError) IsValidation() bool    { return e.Code == ErrCodeValidationFailed }

type CircuitOpenError struct {
	Resource string
}

func (e *CircuitOpenError) Error() string {
	return fmt.Sprintf("fibe: circuit breaker open for %s — too many recent failures", e.Resource)
}

type apiErrorResponse struct {
	Error struct {
		Code    string         `json:"code"`
		Message string         `json:"message"`
		Details map[string]any `json:"details,omitempty"`
	} `json:"error"`
}
