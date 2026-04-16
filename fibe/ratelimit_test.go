package fibe

import (
	"net/http"
	"testing"
	"time"
)

func TestRateLimitTracker_Update(t *testing.T) {
	tracker := &rateLimitTracker{}

	header := http.Header{}
	header.Set("X-RateLimit-Limit", "5000")
	header.Set("X-RateLimit-Remaining", "4500")
	header.Set("X-RateLimit-Reset", "1700000000")

	resp := &http.Response{Header: header}
	tracker.update(resp)

	rl := tracker.current()
	if rl.Limit != 5000 {
		t.Errorf("limit: expected 5000, got %d", rl.Limit)
	}
	if rl.Remaining != 4500 {
		t.Errorf("remaining: expected 4500, got %d", rl.Remaining)
	}
	if rl.Reset.Unix() != 1700000000 {
		t.Errorf("reset: expected 1700000000, got %d", rl.Reset.Unix())
	}
}

func TestRateLimitTracker_WaitTime(t *testing.T) {
	tracker := &rateLimitTracker{}

	header := http.Header{}
	header.Set("X-RateLimit-Remaining", "0")
	header.Set("X-RateLimit-Reset", "99999999999")

	resp := &http.Response{Header: header}
	tracker.update(resp)

	wait := tracker.waitTime()
	if wait <= 0 {
		t.Error("expected positive wait time when remaining is 0")
	}
}

func TestRateLimitTracker_NoWaitWhenRemaining(t *testing.T) {
	tracker := &rateLimitTracker{}

	header := http.Header{}
	header.Set("X-RateLimit-Remaining", "100")
	header.Set("X-RateLimit-Reset", "99999999999")

	resp := &http.Response{Header: header}
	tracker.update(resp)

	wait := tracker.waitTime()
	if wait != 0 {
		t.Errorf("expected 0 wait time when remaining > 0, got %v", wait)
	}
}

func TestParseRetryAfter_Seconds(t *testing.T) {
	resp := &http.Response{Header: http.Header{}}
	resp.Header.Set("Retry-After", "30")

	d := parseRetryAfter(resp)
	if d != 30*time.Second {
		t.Errorf("expected 30s, got %v", d)
	}
}

func TestParseRetryAfter_Empty(t *testing.T) {
	resp := &http.Response{Header: http.Header{}}

	d := parseRetryAfter(resp)
	if d != 0 {
		t.Errorf("expected 0, got %v", d)
	}
}
