package fibe

import (
	"sync"
	"time"
)

type circuitState int

const (
	circuitClosed circuitState = iota
	circuitOpen
	circuitHalfOpen
)

type CircuitBreakerConfig struct {
	FailureThreshold int
	ResetTimeout     time.Duration
	HalfOpenRequests int
}

var DefaultBreakerConfig = CircuitBreakerConfig{
	FailureThreshold: 5,
	ResetTimeout:     30 * time.Second,
	HalfOpenRequests: 2,
}

type circuitBreaker struct {
	mu                sync.RWMutex
	config            CircuitBreakerConfig
	state             circuitState
	failures          int
	halfOpenSuccesses int
	halfOpenInFlight  int
	lastFailure       time.Time
	now               func() time.Time
}

func newCircuitBreaker(cfg CircuitBreakerConfig) *circuitBreaker {
	if cfg.FailureThreshold <= 0 {
		cfg.FailureThreshold = DefaultBreakerConfig.FailureThreshold
	}
	if cfg.ResetTimeout <= 0 {
		cfg.ResetTimeout = DefaultBreakerConfig.ResetTimeout
	}
	if cfg.HalfOpenRequests <= 0 {
		cfg.HalfOpenRequests = DefaultBreakerConfig.HalfOpenRequests
	}
	return &circuitBreaker{config: cfg, now: time.Now}
}

func (cb *circuitBreaker) allow() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case circuitClosed:
		return true
	case circuitOpen:
		if cb.now().Sub(cb.lastFailure) > cb.config.ResetTimeout {
			cb.state = circuitHalfOpen
			cb.halfOpenSuccesses = 0
			cb.halfOpenInFlight = 1
			return true
		}
		return false
	case circuitHalfOpen:
		if cb.halfOpenInFlight >= cb.config.HalfOpenRequests {
			return false
		}
		cb.halfOpenInFlight++
		return true
	}
	return false
}

func (cb *circuitBreaker) recordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case circuitHalfOpen:
		if cb.halfOpenInFlight > 0 {
			cb.halfOpenInFlight--
		}
		cb.halfOpenSuccesses++
		if cb.halfOpenSuccesses >= cb.config.HalfOpenRequests {
			cb.state = circuitClosed
			cb.failures = 0
			cb.halfOpenInFlight = 0
		}
	case circuitClosed:
		cb.failures = 0
	}
}

func (cb *circuitBreaker) recordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failures++
	cb.lastFailure = cb.now()
	if cb.state == circuitHalfOpen && cb.halfOpenInFlight > 0 {
		cb.halfOpenInFlight--
	}

	switch cb.state {
	case circuitClosed:
		if cb.failures >= cb.config.FailureThreshold {
			cb.state = circuitOpen
		}
	case circuitHalfOpen:
		cb.state = circuitOpen
		cb.halfOpenInFlight = 0
	}
}

func (cb *circuitBreaker) releaseProbe() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	if cb.state == circuitHalfOpen && cb.halfOpenInFlight > 0 {
		cb.halfOpenInFlight--
	}
}
