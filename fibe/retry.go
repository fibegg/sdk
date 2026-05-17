package fibe

import (
	"context"
	"errors"
	"math"
	"math/rand/v2"
	"net"
	"time"
)

type retryPolicy struct {
	maxRetries int
	baseDelay  time.Duration
	maxDelay   time.Duration
}

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
	if attempt >= p.maxRetries || err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
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
	exp := math.Pow(2, float64(attempt))
	calculated := time.Duration(float64(p.baseDelay) * exp)
	if calculated > p.maxDelay {
		calculated = p.maxDelay
	}
	jitter := rand.Float64()
	return time.Duration(float64(calculated) * jitter)
}
