package fibe

import (
	"net/http"
	"strconv"
	"sync"
	"time"
)

type rateLimitTracker struct {
	mu    sync.RWMutex
	limit RateLimit
}

func (r *rateLimitTracker) update(resp *http.Response) {
	if resp == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	if v := resp.Header.Get("X-RateLimit-Limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			r.limit.Limit = n
		}
	}
	if v := resp.Header.Get("X-RateLimit-Remaining"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			r.limit.Remaining = n
		}
	}
	if v := resp.Header.Get("X-RateLimit-Reset"); v != "" {
		if epoch, err := strconv.ParseInt(v, 10, 64); err == nil && epoch >= 0 {
			reset := time.Unix(epoch, 0)
			if r.limit.Reset.IsZero() || !reset.Before(r.limit.Reset) {
				r.limit.Reset = reset
			}
		}
	}
}

func (r *rateLimitTracker) current() RateLimit {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.limit
}

func (r *rateLimitTracker) waitTime() time.Duration {
	return r.waitTimeAt(time.Now())
}

func (r *rateLimitTracker) waitTimeAt(now time.Time) time.Duration {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.limit.Remaining > 0 {
		return 0
	}
	wait := r.limit.Reset.Sub(now)
	if wait < 0 {
		return 0
	}
	return wait
}

func parseRetryAfter(resp *http.Response) time.Duration {
	return parseRetryAfterAt(resp, time.Now())
}

func parseRetryAfterAt(resp *http.Response, now time.Time) time.Duration {
	if resp == nil {
		return 0
	}
	v := resp.Header.Get("Retry-After")
	if v == "" {
		return 0
	}
	if seconds, err := strconv.ParseInt(v, 10, 64); err == nil && seconds >= 0 {
		if seconds > int64((time.Duration(1<<63-1))/time.Second) {
			return time.Duration(1<<63 - 1)
		}
		return time.Duration(seconds) * time.Second
	}
	if t, err := http.ParseTime(v); err == nil {
		d := t.Sub(now)
		if d > 0 {
			return d
		}
	}
	return 0
}
