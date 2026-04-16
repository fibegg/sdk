package fibe

import (
	"testing"
	"time"
)

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

func TestRetryPolicy_DelayExponentialBackoff(t *testing.T) {
	p := &retryPolicy{
		maxRetries: 5,
		baseDelay:  100 * time.Millisecond,
		maxDelay:   10 * time.Second,
	}

	for attempt := 0; attempt < 5; attempt++ {
		delay := p.delay(attempt, 0)
		if delay < 0 {
			t.Errorf("delay should be non-negative, got %v for attempt %d", delay, attempt)
		}
		if delay > p.maxDelay {
			t.Errorf("delay %v exceeds max %v for attempt %d", delay, p.maxDelay, attempt)
		}
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
