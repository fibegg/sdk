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

func TestParseRetryAfter_HTTPDateFormats(t *testing.T) {
	now := time.Date(2026, time.July, 13, 12, 0, 0, 0, time.UTC)
	for _, value := range []string{
		now.Add(30 * time.Second).Format(http.TimeFormat),
		now.Add(30 * time.Second).Format(time.RFC850),
		now.Add(30 * time.Second).Format(time.ANSIC),
	} {
		resp := &http.Response{Header: http.Header{"Retry-After": []string{value}}}
		if got := parseRetryAfterAt(resp, now); got != 30*time.Second {
			t.Errorf("Retry-After %q = %v, want 30s", value, got)
		}
	}
}

func TestRateLimitTracker_IgnoresNegativeAndStaleHeaders(t *testing.T) {
	tracker := &rateLimitTracker{}
	valid := http.Header{}
	valid.Set("X-RateLimit-Limit", "100")
	valid.Set("X-RateLimit-Remaining", "50")
	valid.Set("X-RateLimit-Reset", "200")
	tracker.update(&http.Response{Header: valid})
	invalid := http.Header{}
	invalid.Set("X-RateLimit-Limit", "-1")
	invalid.Set("X-RateLimit-Remaining", "-2")
	invalid.Set("X-RateLimit-Reset", "100")
	tracker.update(&http.Response{Header: invalid})
	got := tracker.current()
	if got.Limit != 100 || got.Remaining != 50 || got.Reset.Unix() != 200 {
		t.Fatalf("invalid headers overwrote state: %#v", got)
	}
}
