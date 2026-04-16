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
		if n, err := strconv.Atoi(v); err == nil {
			r.limit.Limit = n
		}
	}
	if v := resp.Header.Get("X-RateLimit-Remaining"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			r.limit.Remaining = n
		}
	}
	if v := resp.Header.Get("X-RateLimit-Reset"); v != "" {
		if epoch, err := strconv.ParseInt(v, 10, 64); err == nil {
			r.limit.Reset = time.Unix(epoch, 0)
		}
	}
}

func (r *rateLimitTracker) current() RateLimit {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.limit
}

func (r *rateLimitTracker) waitTime() time.Duration {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.limit.Remaining > 0 {
		return 0
	}
	wait := time.Until(r.limit.Reset)
	if wait < 0 {
		return 0
	}
	return wait
}

func parseRetryAfter(resp *http.Response) time.Duration {
	v := resp.Header.Get("Retry-After")
	if v == "" {
		return 0
	}
	if seconds, err := strconv.Atoi(v); err == nil {
		return time.Duration(seconds) * time.Second
	}
	if t, err := time.Parse(time.RFC1123, v); err == nil {
		d := time.Until(t)
		if d > 0 {
			return d
		}
	}
	return 0
}
