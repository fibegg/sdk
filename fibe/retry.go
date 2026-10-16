package fibe

import (
	"context"
	"errors"
	"math/rand/v2"
	"net"
	"time"
)

type retryPolicy struct {
	maxRetries int
	baseDelay  time.Duration
	maxDelay   time.Duration
	jitter     func() float64
}

type permanentRequestError struct{ err error }

func (e *permanentRequestError) Error() string { return e.err.Error() }

func (e *permanentRequestError) Unwrap() error { return e.err }

func (p *retryPolicy) shouldRetry(attempt int, statusCode int) bool {
	if attempt >= p.maxRetries {
		return false
	}
	switch statusCode {
	case 429, 500, 502, 503, 504:
		return true
	default:
		return false
	}
}

func (p *retryPolicy) shouldRetryError(attempt int, err error) bool {
	return attempt < p.maxRetries && p.isTransientError(err)
}

func (p *retryPolicy) isTransientError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	var permanent *permanentRequestError
	if errors.As(err, &permanent) {
		return false
	}

	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return false
	}

	return true
}

func (p *retryPolicy) delay(attempt int, retryAfter time.Duration) time.Duration {
	if retryAfter > 0 {
		if p.maxDelay > 0 && retryAfter > p.maxDelay {
			return p.maxDelay
		}
		return retryAfter
	}
	calculated := p.baseDelay
	if calculated <= 0 {
		return 0
	}
	for i := 0; i < attempt; i++ {
		if p.maxDelay > 0 && calculated >= p.maxDelay/2 {
			calculated = p.maxDelay
			break
		}
		if calculated > time.Duration(1<<62) {
			if p.maxDelay > 0 {
				calculated = p.maxDelay
			}
			break
		}
		calculated *= 2
	}
	if p.maxDelay > 0 && calculated > p.maxDelay {
		calculated = p.maxDelay
	}
	// #nosec G404 -- retry jitter is deliberately non-cryptographic and never protects secrets.
	jitter := rand.Float64()
	if p.jitter != nil {
		jitter = p.jitter()
		if jitter < 0 {
			jitter = 0
		} else if jitter > 1 {
			jitter = 1
		}
	}
	return time.Duration(float64(calculated) * jitter)
}
