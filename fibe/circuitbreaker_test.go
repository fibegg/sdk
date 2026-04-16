package fibe

import (
	"testing"
	"time"
)

func TestCircuitBreaker_StartsAllowing(t *testing.T) {
	cb := newCircuitBreaker(CircuitBreakerConfig{
		FailureThreshold: 3,
		ResetTimeout:     100 * time.Millisecond,
		HalfOpenRequests: 1,
	})

	if !cb.allow() {
		t.Error("new circuit breaker should allow requests")
	}
}

func TestCircuitBreaker_OpensAfterThreshold(t *testing.T) {
	cb := newCircuitBreaker(CircuitBreakerConfig{
		FailureThreshold: 3,
		ResetTimeout:     100 * time.Millisecond,
		HalfOpenRequests: 1,
	})

	cb.recordFailure()
	cb.recordFailure()
	if !cb.allow() {
		t.Error("should still allow before threshold")
	}

	cb.recordFailure()
	if cb.allow() {
		t.Error("should block after threshold reached")
	}
}

func TestCircuitBreaker_TransitionsToHalfOpen(t *testing.T) {
	cb := newCircuitBreaker(CircuitBreakerConfig{
		FailureThreshold: 2,
		ResetTimeout:     10 * time.Millisecond,
		HalfOpenRequests: 1,
	})

	cb.recordFailure()
	cb.recordFailure()

	if cb.allow() {
		t.Error("should be open")
	}

	time.Sleep(15 * time.Millisecond)

	if !cb.allow() {
		t.Error("should allow in half-open state after reset timeout")
	}
}

func TestCircuitBreaker_ClosesAfterHalfOpenSuccess(t *testing.T) {
	cb := newCircuitBreaker(CircuitBreakerConfig{
		FailureThreshold: 2,
		ResetTimeout:     10 * time.Millisecond,
		HalfOpenRequests: 2,
	})

	cb.recordFailure()
	cb.recordFailure()

	time.Sleep(15 * time.Millisecond)
	cb.allow()

	cb.recordSuccess()
	cb.recordSuccess()

	if !cb.allow() {
		t.Error("should be closed after successful half-open probes")
	}
}

func TestCircuitBreaker_ReopensOnHalfOpenFailure(t *testing.T) {
	cb := newCircuitBreaker(CircuitBreakerConfig{
		FailureThreshold: 2,
		ResetTimeout:     10 * time.Millisecond,
		HalfOpenRequests: 2,
	})

	cb.recordFailure()
	cb.recordFailure()

	time.Sleep(15 * time.Millisecond)
	cb.allow()

	cb.recordFailure()

	if cb.allow() {
		t.Error("should reopen after failure in half-open state")
	}
}

func TestCircuitBreaker_SuccessResetsFailures(t *testing.T) {
	cb := newCircuitBreaker(CircuitBreakerConfig{
		FailureThreshold: 3,
		ResetTimeout:     100 * time.Millisecond,
		HalfOpenRequests: 1,
	})

	cb.recordFailure()
	cb.recordFailure()
	cb.recordSuccess()
	cb.recordFailure()
	cb.recordFailure()

	if !cb.allow() {
		t.Error("should still allow — success reset the counter")
	}
}
