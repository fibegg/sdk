package fibe

import (
	"context"
	"io"
	"testing"
	"time"
)

type timeoutTransportError struct{}

func (timeoutTransportError) Error() string   { return "timeout" }
func (timeoutTransportError) Timeout() bool   { return true }
func (timeoutTransportError) Temporary() bool { return true }

func TestRetryPolicy_ShouldRetry(t *testing.T) {
	p := &retryPolicy{maxRetries: 3}

	tests := []struct {
		attempt    int
		statusCode int
		want       bool
	}{
		{0, 429, true},
		{0, 500, true},
		{0, 502, true},
		{0, 503, true},
		{0, 504, true},
		{0, 400, false},
		{0, 401, false},
		{0, 403, false},
		{0, 404, false},
		{0, 422, false},
		{3, 500, false},
		{4, 500, false},
	}

	for _, tt := range tests {
		got := p.shouldRetry(tt.attempt, tt.statusCode)
		if got != tt.want {
			t.Errorf("shouldRetry(attempt=%d, status=%d) = %v, want %v",
				tt.attempt, tt.statusCode, got, tt.want)
		}
	}
}

func TestRetryPolicy_ShouldRetryError(t *testing.T) {
	p := &retryPolicy{maxRetries: 3}

	tests := []struct {
		name    string
		attempt int
		err     error
		want    bool
	}{
		{"nil error", 0, nil, false},
		{"max retries reached", 3, io.ErrUnexpectedEOF, false},
		{"context canceled", 0, context.Canceled, false},
		{"context deadline exceeded", 0, context.DeadlineExceeded, false},
		{"transport timeout", 0, timeoutTransportError{}, false},
		{"transient transport error", 0, io.ErrUnexpectedEOF, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := p.shouldRetryError(tt.attempt, tt.err)
			if got != tt.want {
				t.Errorf("shouldRetryError(attempt=%d, err=%v) = %v, want %v",
					tt.attempt, tt.err, got, tt.want)
			}
		})
	}
}

func TestRetryPolicy_DelayRespectsRetryAfter(t *testing.T) {
	p := &retryPolicy{
		maxRetries: 3,
		baseDelay:  100 * time.Millisecond,
		maxDelay:   10 * time.Second,
	}

	retryAfter := 5 * time.Second
	delay := p.delay(0, retryAfter)
	if delay != retryAfter {
		t.Errorf("expected delay=%v when Retry-After set, got %v", retryAfter, delay)
	}
}

func TestRetryPolicy_DelayCapped(t *testing.T) {
	p := &retryPolicy{
		maxRetries: 10,
		baseDelay:  1 * time.Second,
		maxDelay:   5 * time.Second,
	}

	for attempt := 0; attempt < 10; attempt++ {
		delay := p.delay(attempt, 0)
		if delay > p.maxDelay {
			t.Errorf("delay %v exceeds maxDelay %v at attempt %d", delay, p.maxDelay, attempt)
		}
	}
}
