package concurrency

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

// CircuitState represents the state of the circuit breaker
type CircuitState int32

const (
	StateClosed   CircuitState = 0 // Normal operation
	StateOpen     CircuitState = 1 // Failing, reject requests
	StateHalfOpen CircuitState = 2 // Testing recovery
)

// String returns the string representation of the circuit state
func (s CircuitState) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// CircuitBreaker implements the circuit breaker pattern
type CircuitBreaker struct {
	// Configuration
	maxFailures     int
	timeout         time.Duration
	halfOpenMax     int
	
	// State
	state           atomic.Int32 // CircuitState
	failures        atomic.Int32
	successes       atomic.Int32
	halfOpenAttempts atomic.Int32
	lastFailTime    atomic.Int64 // Unix nano
	lastStateChange atomic.Int64 // Unix nano
	
	// Metrics
	metrics *CircuitBreakerMetrics
	
	// Lock for state transitions
	mu sync.RWMutex
}

// CircuitBreakerMetrics tracks circuit breaker performance
type CircuitBreakerMetrics struct {
	// State counters
	TotalRequests     atomic.Int64
	SuccessfulRequests atomic.Int64
	FailedRequests    atomic.Int64
	RejectedRequests  atomic.Int64
	
	// State transitions
	StateTransitions  atomic.Int64
	TimeInClosed      atomic.Int64 // nanoseconds
	TimeInOpen        atomic.Int64 // nanoseconds
	TimeInHalfOpen    atomic.Int64 // nanoseconds
	
	// Current state duration
	CurrentStateStart atomic.Int64 // Unix nano
	
	mu sync.RWMutex
}

// CircuitBreakerConfig holds circuit breaker configuration
type CircuitBreakerConfig struct {
	MaxFailures int           // Number of failures before opening
	Timeout     time.Duration // Time before attempting recovery
	HalfOpenMax int           // Max test requests in half-open state
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(cfg CircuitBreakerConfig) *CircuitBreaker {
	// Set defaults
	if cfg.MaxFailures <= 0 {
		cfg.MaxFailures = 5
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 30 * time.Second
	}
	if cfg.HalfOpenMax <= 0 {
		cfg.HalfOpenMax = 3
	}
	
	now := time.Now().UnixNano()
	
	cb := &CircuitBreaker{
		maxFailures: cfg.MaxFailures,
		timeout:     cfg.Timeout,
		halfOpenMax: cfg.HalfOpenMax,
		metrics:     &CircuitBreakerMetrics{},
	}
	
	cb.state.Store(int32(StateClosed))
	cb.lastStateChange.Store(now)
	cb.metrics.CurrentStateStart.Store(now)
	
	log.Printf("Circuit breaker initialized (max failures: %d, timeout: %v, half-open max: %d)",
		cfg.MaxFailures, cfg.Timeout, cfg.HalfOpenMax)
	
	return cb
}

// Call executes a function with circuit breaker protection
func (cb *CircuitBreaker) Call(ctx context.Context, fn func(context.Context) error) error {
	cb.metrics.TotalRequests.Add(1)
	
	// Check if circuit breaker allows the request
	if err := cb.beforeRequest(); err != nil {
		cb.metrics.RejectedRequests.Add(1)
		return err
	}
	
	// Execute function
	err := fn(ctx)
	
	// Record result
	cb.afterRequest(err)
	
	return err
}

// beforeRequest checks if the request should be allowed
func (cb *CircuitBreaker) beforeRequest() error {
	state := CircuitState(cb.state.Load())
	
	switch state {
	case StateClosed:
		// Allow request
		return nil
		
	case StateOpen:
		// Check if timeout has elapsed
		lastFail := time.Unix(0, cb.lastFailTime.Load())
		if time.Since(lastFail) > cb.timeout {
			// Attempt recovery
			cb.transitionToHalfOpen()
			return nil
		}
		
		// Reject request
		return fmt.Errorf("circuit breaker is open (last failure: %v ago)", time.Since(lastFail))
		
	case StateHalfOpen:
		// Check if we've reached max test requests
		attempts := cb.halfOpenAttempts.Load()
		if attempts >= int32(cb.halfOpenMax) {
			return fmt.Errorf("circuit breaker is half-open (max test requests reached)")
		}
		
		// Allow test request
		cb.halfOpenAttempts.Add(1)
		return nil
		
	default:
		return fmt.Errorf("circuit breaker in unknown state: %d", state)
	}
}

// afterRequest records the result of a request
func (cb *CircuitBreaker) afterRequest(err error) {
	state := CircuitState(cb.state.Load())
	
	if err != nil {
		// Request failed
		cb.metrics.FailedRequests.Add(1)
		cb.failures.Add(1)
		cb.lastFailTime.Store(time.Now().UnixNano())
		
		switch state {
		case StateClosed:
			// Check if we should open the circuit
			if cb.failures.Load() >= int32(cb.maxFailures) {
				cb.transitionToOpen()
			}
			
		case StateHalfOpen:
			// Failure in half-open state, reopen circuit
			cb.transitionToOpen()
		}
	} else {
		// Request succeeded
		cb.metrics.SuccessfulRequests.Add(1)
		cb.successes.Add(1)
		
		switch state {
		case StateClosed:
			// Reset failure count on success
			cb.failures.Store(0)
			
		case StateHalfOpen:
			// Check if we should close the circuit
			if cb.successes.Load() >= int32(cb.halfOpenMax) {
				cb.transitionToClosed()
			}
		}
	}
}

// transitionToOpen transitions the circuit breaker to open state
func (cb *CircuitBreaker) transitionToOpen() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	
	oldState := CircuitState(cb.state.Load())
	if oldState == StateOpen {
		return // Already open
	}
	
	cb.updateStateMetrics(oldState)
	cb.state.Store(int32(StateOpen))
	cb.lastStateChange.Store(time.Now().UnixNano())
	cb.metrics.CurrentStateStart.Store(time.Now().UnixNano())
	cb.metrics.StateTransitions.Add(1)
	
	log.Printf("Circuit breaker opened (failures: %d, state: %s -> open)",
		cb.failures.Load(), oldState.String())
}

// transitionToHalfOpen transitions the circuit breaker to half-open state
func (cb *CircuitBreaker) transitionToHalfOpen() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	
	oldState := CircuitState(cb.state.Load())
	if oldState == StateHalfOpen {
		return // Already half-open
	}
	
	cb.updateStateMetrics(oldState)
	cb.state.Store(int32(StateHalfOpen))
	cb.lastStateChange.Store(time.Now().UnixNano())
	cb.metrics.CurrentStateStart.Store(time.Now().UnixNano())
	cb.metrics.StateTransitions.Add(1)
	cb.halfOpenAttempts.Store(0)
	cb.successes.Store(0)
	
	log.Printf("Circuit breaker half-opened (state: %s -> half-open)", oldState.String())
}

// transitionToClosed transitions the circuit breaker to closed state
func (cb *CircuitBreaker) transitionToClosed() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	
	oldState := CircuitState(cb.state.Load())
	if oldState == StateClosed {
		return // Already closed
	}
	
	cb.updateStateMetrics(oldState)
	cb.state.Store(int32(StateClosed))
	cb.lastStateChange.Store(time.Now().UnixNano())
	cb.metrics.CurrentStateStart.Store(time.Now().UnixNano())
	cb.metrics.StateTransitions.Add(1)
	cb.failures.Store(0)
	cb.successes.Store(0)
	cb.halfOpenAttempts.Store(0)
	
	log.Printf("Circuit breaker closed (state: %s -> closed)", oldState.String())
}

// updateStateMetrics updates time-in-state metrics
func (cb *CircuitBreaker) updateStateMetrics(oldState CircuitState) {
	stateStart := time.Unix(0, cb.metrics.CurrentStateStart.Load())
	duration := time.Since(stateStart).Nanoseconds()
	
	switch oldState {
	case StateClosed:
		cb.metrics.TimeInClosed.Add(duration)
	case StateOpen:
		cb.metrics.TimeInOpen.Add(duration)
	case StateHalfOpen:
		cb.metrics.TimeInHalfOpen.Add(duration)
	}
}

// GetState returns the current state of the circuit breaker
func (cb *CircuitBreaker) GetState() CircuitState {
	return CircuitState(cb.state.Load())
}

// IsOpen returns true if the circuit breaker is open
func (cb *CircuitBreaker) IsOpen() bool {
	return cb.GetState() == StateOpen
}

// IsClosed returns true if the circuit breaker is closed
func (cb *CircuitBreaker) IsClosed() bool {
	return cb.GetState() == StateClosed
}

// IsHalfOpen returns true if the circuit breaker is half-open
func (cb *CircuitBreaker) IsHalfOpen() bool {
	return cb.GetState() == StateHalfOpen
}

// Reset resets the circuit breaker to closed state
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	
	oldState := CircuitState(cb.state.Load())
	cb.updateStateMetrics(oldState)
	
	cb.state.Store(int32(StateClosed))
	cb.failures.Store(0)
	cb.successes.Store(0)
	cb.halfOpenAttempts.Store(0)
	cb.lastStateChange.Store(time.Now().UnixNano())
	cb.metrics.CurrentStateStart.Store(time.Now().UnixNano())
	
	log.Printf("Circuit breaker reset (state: %s -> closed)", oldState.String())
}

// GetStats returns human-readable statistics
func (cb *CircuitBreaker) GetStats() map[string]interface{} {
	state := cb.GetState()
	stateStart := time.Unix(0, cb.metrics.CurrentStateStart.Load())
	stateDuration := time.Since(stateStart)
	
	totalRequests := cb.metrics.TotalRequests.Load()
	successfulRequests := cb.metrics.SuccessfulRequests.Load()
	failedRequests := cb.metrics.FailedRequests.Load()
	rejectedRequests := cb.metrics.RejectedRequests.Load()
	
	var successRate float64
	if totalRequests > 0 {
		successRate = float64(successfulRequests) / float64(totalRequests) * 100
	}
	
	return map[string]interface{}{
		"state":                state.String(),
		"state_duration":       stateDuration.String(),
		"failures":             cb.failures.Load(),
		"successes":            cb.successes.Load(),
		"total_requests":       totalRequests,
		"successful_requests":  successfulRequests,
		"failed_requests":      failedRequests,
		"rejected_requests":    rejectedRequests,
		"success_rate":         successRate,
		"state_transitions":    cb.metrics.StateTransitions.Load(),
		"time_in_closed":       time.Duration(cb.metrics.TimeInClosed.Load()).String(),
		"time_in_open":         time.Duration(cb.metrics.TimeInOpen.Load()).String(),
		"time_in_half_open":    time.Duration(cb.metrics.TimeInHalfOpen.Load()).String(),
	}
}

// IsHealthy returns true if the circuit breaker is healthy
func (cb *CircuitBreaker) IsHealthy() bool {
	state := cb.GetState()
	
	// Circuit breaker is healthy if closed or half-open (attempting recovery)
	return state == StateClosed || state == StateHalfOpen
}
