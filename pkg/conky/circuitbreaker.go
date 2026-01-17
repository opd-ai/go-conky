// Package conky provides the public API for go-conky.
package conky

import (
	"errors"
	"sync"
	"time"
)

// CircuitState represents the current state of a circuit breaker.
type CircuitState int

const (
	// CircuitClosed indicates the circuit is functioning normally.
	CircuitClosed CircuitState = iota
	// CircuitOpen indicates the circuit is open and rejecting requests.
	CircuitOpen
	// CircuitHalfOpen indicates the circuit is testing if the service recovered.
	CircuitHalfOpen
)

// String returns the string representation of the circuit state.
func (s CircuitState) String() string {
	switch s {
	case CircuitClosed:
		return "closed"
	case CircuitOpen:
		return "open"
	case CircuitHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// ErrCircuitOpen is returned when a circuit breaker is open and rejecting requests.
var ErrCircuitOpen = errors.New("circuit breaker is open")

// CircuitBreakerConfig contains configuration for a circuit breaker.
type CircuitBreakerConfig struct {
	// FailureThreshold is the number of consecutive failures before opening the circuit.
	// Default: 5
	FailureThreshold int

	// SuccessThreshold is the number of consecutive successes in half-open state
	// required to close the circuit. Default: 2
	SuccessThreshold int

	// Timeout is how long the circuit stays open before transitioning to half-open.
	// Default: 30 seconds
	Timeout time.Duration

	// MaxHalfOpenRequests is the maximum number of requests allowed in half-open state.
	// Default: 1
	MaxHalfOpenRequests int

	// OnStateChange is called when the circuit state changes.
	// The callback receives the old state and new state.
	OnStateChange func(from, to CircuitState)
}

// DefaultCircuitBreakerConfig returns a CircuitBreakerConfig with sensible defaults.
func DefaultCircuitBreakerConfig() CircuitBreakerConfig {
	return CircuitBreakerConfig{
		FailureThreshold:    5,
		SuccessThreshold:    2,
		Timeout:             30 * time.Second,
		MaxHalfOpenRequests: 1,
	}
}

// CircuitBreaker implements the circuit breaker pattern for protecting
// external service calls from cascading failures.
type CircuitBreaker struct {
	config CircuitBreakerConfig

	mu                sync.RWMutex
	state             CircuitState
	failures          int
	successes         int
	lastFailure       time.Time
	halfOpenRequests  int
	totalSuccesses    int64
	totalFailures     int64
	totalRejections   int64
	consecutiveErrors int
}

// NewCircuitBreaker creates a new circuit breaker with the given configuration.
func NewCircuitBreaker(config CircuitBreakerConfig) *CircuitBreaker {
	// Apply defaults for zero values
	if config.FailureThreshold <= 0 {
		config.FailureThreshold = 5
	}
	if config.SuccessThreshold <= 0 {
		config.SuccessThreshold = 2
	}
	if config.Timeout <= 0 {
		config.Timeout = 30 * time.Second
	}
	if config.MaxHalfOpenRequests <= 0 {
		config.MaxHalfOpenRequests = 1
	}

	return &CircuitBreaker{
		config: config,
		state:  CircuitClosed,
	}
}

// Execute runs the given function through the circuit breaker.
// If the circuit is open, it returns ErrCircuitOpen without executing the function.
// If the circuit is half-open, it allows limited requests to test recovery.
func (cb *CircuitBreaker) Execute(fn func() error) error {
	if !cb.allowRequest() {
		cb.mu.Lock()
		cb.totalRejections++
		cb.mu.Unlock()
		return ErrCircuitOpen
	}

	err := fn()
	cb.recordResult(err)
	return err
}

// State returns the current state of the circuit breaker.
func (cb *CircuitBreaker) State() CircuitState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	// Check if open circuit should transition to half-open
	if cb.state == CircuitOpen {
		if time.Since(cb.lastFailure) >= cb.config.Timeout {
			return CircuitHalfOpen
		}
	}
	return cb.state
}

// Stats returns circuit breaker statistics.
func (cb *CircuitBreaker) Stats() CircuitBreakerStats {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	return CircuitBreakerStats{
		State:             cb.state,
		Failures:          cb.failures,
		Successes:         cb.successes,
		TotalSuccesses:    cb.totalSuccesses,
		TotalFailures:     cb.totalFailures,
		TotalRejections:   cb.totalRejections,
		LastFailure:       cb.lastFailure,
		ConsecutiveErrors: cb.consecutiveErrors,
	}
}

// CircuitBreakerStats contains statistics about circuit breaker operation.
type CircuitBreakerStats struct {
	// State is the current circuit state.
	State CircuitState
	// Failures is the current failure count in the closed state window.
	Failures int
	// Successes is the current success count in the half-open state.
	Successes int
	// TotalSuccesses is the total number of successful operations.
	TotalSuccesses int64
	// TotalFailures is the total number of failed operations.
	TotalFailures int64
	// TotalRejections is the total number of requests rejected due to open circuit.
	TotalRejections int64
	// LastFailure is the time of the last failure.
	LastFailure time.Time
	// ConsecutiveErrors is the current number of consecutive errors.
	ConsecutiveErrors int
}

// Reset resets the circuit breaker to closed state.
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	oldState := cb.state
	cb.state = CircuitClosed
	cb.failures = 0
	cb.successes = 0
	cb.halfOpenRequests = 0
	cb.consecutiveErrors = 0

	if oldState != CircuitClosed && cb.config.OnStateChange != nil {
		cb.config.OnStateChange(oldState, CircuitClosed)
	}
}

// allowRequest checks if a request is allowed through the circuit.
func (cb *CircuitBreaker) allowRequest() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case CircuitClosed:
		return true

	case CircuitOpen:
		// Check if timeout has elapsed
		if time.Since(cb.lastFailure) >= cb.config.Timeout {
			cb.transitionTo(CircuitHalfOpen)
			cb.halfOpenRequests = 1
			return true
		}
		return false

	case CircuitHalfOpen:
		// Limit requests in half-open state
		if cb.halfOpenRequests < cb.config.MaxHalfOpenRequests {
			cb.halfOpenRequests++
			return true
		}
		return false

	default:
		return false
	}
}

// recordResult records the result of an operation.
func (cb *CircuitBreaker) recordResult(err error) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if err != nil {
		cb.recordFailure()
	} else {
		cb.recordSuccess()
	}
}

// recordSuccess handles a successful operation.
func (cb *CircuitBreaker) recordSuccess() {
	cb.totalSuccesses++
	cb.consecutiveErrors = 0

	switch cb.state {
	case CircuitClosed:
		// Reset failure count on success
		cb.failures = 0

	case CircuitHalfOpen:
		cb.successes++
		// Check if we have enough successes to close the circuit
		if cb.successes >= cb.config.SuccessThreshold {
			cb.transitionTo(CircuitClosed)
			cb.failures = 0
			cb.successes = 0
		}
	}
}

// recordFailure handles a failed operation.
func (cb *CircuitBreaker) recordFailure() {
	cb.totalFailures++
	cb.consecutiveErrors++
	cb.lastFailure = time.Now()

	switch cb.state {
	case CircuitClosed:
		cb.failures++
		// Check if we should open the circuit
		if cb.failures >= cb.config.FailureThreshold {
			cb.transitionTo(CircuitOpen)
		}

	case CircuitHalfOpen:
		// Any failure in half-open state opens the circuit again
		cb.transitionTo(CircuitOpen)
		cb.successes = 0
	}
}

// transitionTo changes the circuit state.
func (cb *CircuitBreaker) transitionTo(newState CircuitState) {
	if cb.state == newState {
		return
	}

	oldState := cb.state
	cb.state = newState
	cb.halfOpenRequests = 0

	if cb.config.OnStateChange != nil {
		// Call callback without holding lock to prevent deadlocks
		go cb.config.OnStateChange(oldState, newState)
	}
}
