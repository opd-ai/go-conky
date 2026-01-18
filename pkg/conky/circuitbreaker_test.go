package conky

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestCircuitState_String(t *testing.T) {
	tests := []struct {
		state CircuitState
		want  string
	}{
		{CircuitClosed, "closed"},
		{CircuitOpen, "open"},
		{CircuitHalfOpen, "half-open"},
		{CircuitState(99), "unknown"},
	}

	for _, tt := range tests {
		got := tt.state.String()
		if got != tt.want {
			t.Errorf("CircuitState(%d).String() = %q, want %q", tt.state, got, tt.want)
		}
	}
}

func TestDefaultCircuitBreakerConfig(t *testing.T) {
	config := DefaultCircuitBreakerConfig()

	if config.FailureThreshold != 5 {
		t.Errorf("FailureThreshold = %d, want 5", config.FailureThreshold)
	}
	if config.SuccessThreshold != 2 {
		t.Errorf("SuccessThreshold = %d, want 2", config.SuccessThreshold)
	}
	if config.Timeout != 30*time.Second {
		t.Errorf("Timeout = %v, want 30s", config.Timeout)
	}
	if config.MaxHalfOpenRequests != 1 {
		t.Errorf("MaxHalfOpenRequests = %d, want 1", config.MaxHalfOpenRequests)
	}
}

func TestNewCircuitBreaker_AppliesDefaults(t *testing.T) {
	// Zero values should get defaults
	cb := NewCircuitBreaker(CircuitBreakerConfig{})

	if cb.config.FailureThreshold != 5 {
		t.Errorf("FailureThreshold = %d, want 5", cb.config.FailureThreshold)
	}
	if cb.config.SuccessThreshold != 2 {
		t.Errorf("SuccessThreshold = %d, want 2", cb.config.SuccessThreshold)
	}
	if cb.config.Timeout != 30*time.Second {
		t.Errorf("Timeout = %v, want 30s", cb.config.Timeout)
	}
	if cb.config.MaxHalfOpenRequests != 1 {
		t.Errorf("MaxHalfOpenRequests = %d, want 1", cb.config.MaxHalfOpenRequests)
	}
	if cb.state != CircuitClosed {
		t.Errorf("Initial state = %v, want closed", cb.state)
	}
}

func TestCircuitBreaker_ClosedState(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		FailureThreshold: 3,
	})

	// Successful calls should work
	err := cb.Execute(func() error {
		return nil
	})
	if err != nil {
		t.Errorf("Execute returned error: %v", err)
	}

	if cb.State() != CircuitClosed {
		t.Errorf("State = %v, want closed", cb.State())
	}

	stats := cb.Stats()
	if stats.TotalSuccesses != 1 {
		t.Errorf("TotalSuccesses = %d, want 1", stats.TotalSuccesses)
	}
}

func TestCircuitBreaker_OpensAfterThreshold(t *testing.T) {
	stateChanges := make([]CircuitState, 0)
	var mu sync.Mutex

	cb := NewCircuitBreaker(CircuitBreakerConfig{
		FailureThreshold: 3,
		Timeout:          100 * time.Millisecond,
		OnStateChange: func(from, to CircuitState) {
			mu.Lock()
			stateChanges = append(stateChanges, to)
			mu.Unlock()
		},
	})

	testErr := errors.New("test error")

	// First two failures - circuit should stay closed
	for i := 0; i < 2; i++ {
		err := cb.Execute(func() error { return testErr })
		if err != testErr {
			t.Errorf("Execute %d returned %v, want %v", i, err, testErr)
		}
		if cb.State() != CircuitClosed {
			t.Errorf("State after failure %d = %v, want closed", i, cb.State())
		}
	}

	// Third failure - circuit should open
	err := cb.Execute(func() error { return testErr })
	if err != testErr {
		t.Errorf("Execute returned %v, want %v", err, testErr)
	}

	// Give state change callback time to execute
	time.Sleep(10 * time.Millisecond)

	if cb.State() != CircuitOpen {
		t.Errorf("State after threshold = %v, want open", cb.State())
	}

	stats := cb.Stats()
	if stats.TotalFailures != 3 {
		t.Errorf("TotalFailures = %d, want 3", stats.TotalFailures)
	}

	mu.Lock()
	if len(stateChanges) < 1 || stateChanges[0] != CircuitOpen {
		t.Errorf("OnStateChange not called correctly, got %v", stateChanges)
	}
	mu.Unlock()
}

func TestCircuitBreaker_OpenRejectsRequests(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		FailureThreshold: 1,
		Timeout:          1 * time.Hour, // Long timeout to keep open
	})

	// Open the circuit
	testErr := errors.New("test error")
	_ = cb.Execute(func() error { return testErr })

	if cb.State() != CircuitOpen {
		t.Fatalf("State = %v, want open", cb.State())
	}

	// Subsequent requests should be rejected
	executed := false
	err := cb.Execute(func() error {
		executed = true
		return nil
	})

	if !errors.Is(err, ErrCircuitOpen) {
		t.Errorf("Execute returned %v, want ErrCircuitOpen", err)
	}
	if executed {
		t.Error("Function was executed when circuit was open")
	}

	stats := cb.Stats()
	if stats.TotalRejections != 1 {
		t.Errorf("TotalRejections = %d, want 1", stats.TotalRejections)
	}
}

func TestCircuitBreaker_TransitionsToHalfOpen(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		FailureThreshold: 1,
		Timeout:          50 * time.Millisecond,
	})

	// Open the circuit
	testErr := errors.New("test error")
	_ = cb.Execute(func() error { return testErr })

	if cb.State() != CircuitOpen {
		t.Fatalf("State = %v, want open", cb.State())
	}

	// Wait for timeout
	time.Sleep(60 * time.Millisecond)

	// State should now show half-open
	if cb.State() != CircuitHalfOpen {
		t.Errorf("State after timeout = %v, want half-open", cb.State())
	}
}

func TestCircuitBreaker_HalfOpenToClosedOnSuccess(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		FailureThreshold:    1,
		SuccessThreshold:    2,
		MaxHalfOpenRequests: 2, // Allow 2 requests in half-open to test success threshold
		Timeout:             50 * time.Millisecond,
	})

	// Open the circuit
	testErr := errors.New("test error")
	_ = cb.Execute(func() error { return testErr })

	// Wait for timeout to transition to half-open
	time.Sleep(60 * time.Millisecond)

	// First success
	err := cb.Execute(func() error { return nil })
	if err != nil {
		t.Errorf("Execute returned error: %v", err)
	}

	// Still half-open (need 2 successes)
	if cb.State() != CircuitHalfOpen {
		t.Errorf("State after 1 success = %v, want half-open", cb.State())
	}

	// Second success - should close
	err = cb.Execute(func() error { return nil })
	if err != nil {
		t.Errorf("Execute returned error: %v", err)
	}

	// Give state change time to execute
	time.Sleep(10 * time.Millisecond)

	if cb.State() != CircuitClosed {
		t.Errorf("State after 2 successes = %v, want closed", cb.State())
	}
}

func TestCircuitBreaker_HalfOpenToOpenOnFailure(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		FailureThreshold: 1,
		Timeout:          50 * time.Millisecond,
	})

	// Open the circuit
	testErr := errors.New("test error")
	_ = cb.Execute(func() error { return testErr })

	// Wait for timeout to transition to half-open
	time.Sleep(60 * time.Millisecond)

	// Failure in half-open should re-open
	err := cb.Execute(func() error { return testErr })
	if err != testErr {
		t.Errorf("Execute returned %v, want %v", err, testErr)
	}

	// Give state change time to execute
	time.Sleep(10 * time.Millisecond)

	if cb.State() != CircuitOpen {
		t.Errorf("State after failure = %v, want open", cb.State())
	}
}

func TestCircuitBreaker_Reset(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		FailureThreshold: 1,
		Timeout:          1 * time.Hour,
	})

	// Open the circuit
	testErr := errors.New("test error")
	_ = cb.Execute(func() error { return testErr })

	if cb.State() != CircuitOpen {
		t.Fatalf("State = %v, want open", cb.State())
	}

	// Reset
	cb.Reset()

	if cb.State() != CircuitClosed {
		t.Errorf("State after reset = %v, want closed", cb.State())
	}

	// Should accept requests again
	err := cb.Execute(func() error { return nil })
	if err != nil {
		t.Errorf("Execute after reset returned error: %v", err)
	}
}

func TestCircuitBreaker_SuccessResetsFailureCount(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		FailureThreshold: 3,
	})

	testErr := errors.New("test error")

	// Two failures
	_ = cb.Execute(func() error { return testErr })
	_ = cb.Execute(func() error { return testErr })

	// One success - should reset failure count
	_ = cb.Execute(func() error { return nil })

	// Two more failures - circuit should still be closed
	_ = cb.Execute(func() error { return testErr })
	_ = cb.Execute(func() error { return testErr })

	if cb.State() != CircuitClosed {
		t.Errorf("State = %v, want closed (failure count should have reset)", cb.State())
	}

	// Third failure - now it should open
	_ = cb.Execute(func() error { return testErr })

	// Give state change time
	time.Sleep(10 * time.Millisecond)

	if cb.State() != CircuitOpen {
		t.Errorf("State after 3 consecutive failures = %v, want open", cb.State())
	}
}

func TestCircuitBreaker_ConcurrentAccess(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		FailureThreshold: 100,
		Timeout:          100 * time.Millisecond,
	})

	var wg sync.WaitGroup
	var successCount int64
	var failureCount int64

	// Run 100 concurrent operations
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			var err error
			if idx%2 == 0 {
				err = cb.Execute(func() error { return nil })
				if err == nil {
					atomic.AddInt64(&successCount, 1)
				}
			} else {
				testErr := errors.New("test error")
				err = cb.Execute(func() error { return testErr })
				if err == testErr {
					atomic.AddInt64(&failureCount, 1)
				}
			}
		}(i)
	}

	wg.Wait()

	stats := cb.Stats()

	// All operations should have executed (circuit shouldn't have opened with threshold 100)
	total := atomic.LoadInt64(&successCount) + atomic.LoadInt64(&failureCount)
	if total != 100 {
		t.Errorf("Total operations = %d, want 100", total)
	}

	if stats.TotalSuccesses != atomic.LoadInt64(&successCount) {
		t.Errorf("TotalSuccesses = %d, want %d", stats.TotalSuccesses, successCount)
	}
}

func TestCircuitBreaker_HalfOpenLimitsRequests(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		FailureThreshold:    1,
		MaxHalfOpenRequests: 1,
		Timeout:             50 * time.Millisecond,
	})

	// Open the circuit
	testErr := errors.New("test error")
	_ = cb.Execute(func() error { return testErr })

	// Wait for timeout
	time.Sleep(60 * time.Millisecond)

	// First request in half-open should be allowed
	executed := 0
	var mu sync.Mutex

	var wg sync.WaitGroup
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := cb.Execute(func() error {
				mu.Lock()
				executed++
				mu.Unlock()
				time.Sleep(10 * time.Millisecond) // Simulate work
				return nil
			})
			_ = err // May be ErrCircuitOpen
		}()
	}

	wg.Wait()

	// Only MaxHalfOpenRequests should have executed
	mu.Lock()
	if executed > 1 {
		t.Errorf("Executed %d requests in half-open, want at most 1", executed)
	}
	mu.Unlock()
}

func TestCircuitBreaker_ConsecutiveErrors(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		FailureThreshold: 5,
	})

	testErr := errors.New("test error")

	// Generate errors
	for i := 0; i < 3; i++ {
		_ = cb.Execute(func() error { return testErr })
	}

	stats := cb.Stats()
	if stats.ConsecutiveErrors != 3 {
		t.Errorf("ConsecutiveErrors = %d, want 3", stats.ConsecutiveErrors)
	}

	// Success resets consecutive errors
	_ = cb.Execute(func() error { return nil })

	stats = cb.Stats()
	if stats.ConsecutiveErrors != 0 {
		t.Errorf("ConsecutiveErrors after success = %d, want 0", stats.ConsecutiveErrors)
	}
}
