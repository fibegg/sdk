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
	mu               sync.RWMutex
	config           CircuitBreakerConfig
	state            circuitState
	failures         int
	halfOpenSuccesses int
	lastFailure      time.Time
}

func newCircuitBreaker(cfg CircuitBreakerConfig) *circuitBreaker {
	if cfg.FailureThreshold == 0 {
		cfg.FailureThreshold = DefaultBreakerConfig.FailureThreshold
	}
	if cfg.ResetTimeout == 0 {
		cfg.ResetTimeout = DefaultBreakerConfig.ResetTimeout
	}
	if cfg.HalfOpenRequests == 0 {
		cfg.HalfOpenRequests = DefaultBreakerConfig.HalfOpenRequests
	}
	return &circuitBreaker{config: cfg}
}

func (cb *circuitBreaker) allow() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case circuitClosed:
		return true
	case circuitOpen:
		if time.Since(cb.lastFailure) > cb.config.ResetTimeout {
			cb.state = circuitHalfOpen
			cb.halfOpenSuccesses = 0
			return true
		}
		return false
	case circuitHalfOpen:
		return true
	}
	return false
}

func (cb *circuitBreaker) recordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case circuitHalfOpen:
		cb.halfOpenSuccesses++
		if cb.halfOpenSuccesses >= cb.config.HalfOpenRequests {
			cb.state = circuitClosed
			cb.failures = 0
		}
	case circuitClosed:
		cb.failures = 0
	}
}

func (cb *circuitBreaker) recordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failures++
	cb.lastFailure = time.Now()

	switch cb.state {
	case circuitClosed:
		if cb.failures >= cb.config.FailureThreshold {
			cb.state = circuitOpen
		}
	case circuitHalfOpen:
		cb.state = circuitOpen
	}
}
