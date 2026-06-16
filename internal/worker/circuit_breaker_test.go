package worker

import (
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCircuitBreaker_InitiallyClosed(t *testing.T) {
	reg := NewCircuitBreakerRegistry(DefaultCircuitBreakerConfig())
	id := uuid.New()
	assert.True(t, reg.Allow(id))
	assert.False(t, reg.IsOpen(id))
}

func TestCircuitBreaker_OpensAfterThreshold(t *testing.T) {
	reg := NewCircuitBreakerRegistry(CircuitBreakerConfig{
		FailureThreshold: 3,
		CooldownPeriod:   time.Second,
	})
	id := uuid.New()

	for range 3 {
		reg.RecordFailure(id)
	}

	assert.True(t, reg.IsOpen(id))
	assert.False(t, reg.Allow(id))
}

func TestCircuitBreaker_BlocksWhenOpen(t *testing.T) {
	reg := NewCircuitBreakerRegistry(CircuitBreakerConfig{
		FailureThreshold: 2,
		CooldownPeriod:   time.Hour,
	})
	id := uuid.New()

	reg.RecordFailure(id)
	reg.RecordFailure(id)
	assert.False(t, reg.Allow(id))
}

func TestCircuitBreaker_HalfOpenAfterCooldown(t *testing.T) {
	reg := NewCircuitBreakerRegistry(CircuitBreakerConfig{
		FailureThreshold: 2,
		CooldownPeriod:   50 * time.Millisecond,
	})
	id := uuid.New()

	reg.RecordFailure(id)
	reg.RecordFailure(id)
	assert.False(t, reg.Allow(id))

	time.Sleep(60 * time.Millisecond)
	assert.True(t, reg.Allow(id)) // transitions to half-open
}

func TestCircuitBreaker_ClosesOnSuccess(t *testing.T) {
	reg := NewCircuitBreakerRegistry(CircuitBreakerConfig{
		FailureThreshold: 2,
		CooldownPeriod:   50 * time.Millisecond,
	})
	id := uuid.New()

	reg.RecordFailure(id)
	reg.RecordFailure(id)
	time.Sleep(60 * time.Millisecond)

	reg.Allow(id)         // half-open
	reg.RecordSuccess(id) // should close
	assert.True(t, reg.Allow(id))
	assert.False(t, reg.IsOpen(id))
}

func TestCircuitBreaker_ResetCounter(t *testing.T) {
	reg := NewCircuitBreakerRegistry(CircuitBreakerConfig{
		FailureThreshold: 3,
		CooldownPeriod:   time.Second,
	})
	id := uuid.New()

	reg.RecordFailure(id)
	reg.RecordFailure(id)
	reg.RecordSuccess(id) // resets counter
	reg.RecordFailure(id)

	assert.False(t, reg.IsOpen(id)) // still closed, only 1 failure since reset
}

func TestCircuitBreaker_DifferentSMTPIDs(t *testing.T) {
	reg := NewCircuitBreakerRegistry(CircuitBreakerConfig{
		FailureThreshold: 2,
		CooldownPeriod:   time.Second,
	})
	id1 := uuid.New()
	id2 := uuid.New()

	reg.RecordFailure(id1)
	reg.RecordFailure(id1)
	assert.True(t, reg.IsOpen(id1))
	assert.False(t, reg.IsOpen(id2))
	assert.True(t, reg.Allow(id2))
}

func TestCircuitBreaker_DefaultConfig(t *testing.T) {
	cfg := DefaultCircuitBreakerConfig()
	assert.Equal(t, 5, cfg.FailureThreshold)
	assert.Equal(t, 30*time.Second, cfg.CooldownPeriod)
}

func TestCircuitBreaker_CustomConfig(t *testing.T) {
	reg := NewCircuitBreakerRegistry(CircuitBreakerConfig{
		FailureThreshold: 10,
		CooldownPeriod:   time.Minute,
	})
	id := uuid.New()

	for range 9 {
		reg.RecordFailure(id)
	}
	assert.False(t, reg.IsOpen(id)) // 9 < 10

	reg.RecordFailure(id) // 10th
	assert.True(t, reg.IsOpen(id))
}

func TestCircuitBreaker_ConcurrentSafety(t *testing.T) {
	reg := NewCircuitBreakerRegistry(CircuitBreakerConfig{
		FailureThreshold: 100,
		CooldownPeriod:   time.Second,
	})
	id := uuid.New()

	var wg sync.WaitGroup
	for range 50 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			reg.Allow(id)
			reg.RecordFailure(id)
			reg.RecordSuccess(id)
		}()
	}
	wg.Wait()

	require.NotPanics(t, func() {
		reg.Allow(id)
		reg.IsOpen(id)
	})
}
