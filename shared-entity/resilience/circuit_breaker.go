package resilience

import (
	"context"
	"fmt"
	"time"

	"github.com/sony/gobreaker"
)

// CircuitBreakerConfig holds configuration for a circuit breaker
type CircuitBreakerConfig struct {
	Name          string
	MaxRequests   uint32        // Max requests to allow in half-open state
	Interval      time.Duration // Cyclic period of the closed state to clear internal stats
	Timeout       time.Duration // Period of the open state before going to half-open
	ReadyToTrip   func(counts gobreaker.Counts) bool
	OnStateChange func(name string, from gobreaker.State, to gobreaker.State)
}

// DefaultConfig returns a sensible default configuration
func DefaultConfig(name string) CircuitBreakerConfig {
	return CircuitBreakerConfig{
		Name:        name,
		MaxRequests: 3,
		Interval:    30 * time.Second,
		Timeout:     10 * time.Second,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			// Trip if failure ratio > 50% with at least 5 requests
			return counts.Requests >= 5 && counts.ConsecutiveFailures >= 3
		},
	}
}

// CircuitBreaker wraps gobreaker.CircuitBreaker with additional functionality
type CircuitBreaker struct {
	cb *gobreaker.CircuitBreaker
}

// NewCircuitBreaker creates a new circuit breaker with the given config
func NewCircuitBreaker(cfg CircuitBreakerConfig) *CircuitBreaker {
	settings := gobreaker.Settings{
		Name:        cfg.Name,
		MaxRequests: cfg.MaxRequests,
		Interval:    cfg.Interval,
		Timeout:     cfg.Timeout,
		ReadyToTrip: cfg.ReadyToTrip,
		OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
			fmt.Printf("[CircuitBreaker] %s: state changed from %s to %s\n", name, from, to)
			if cfg.OnStateChange != nil {
				cfg.OnStateChange(name, from, to)
			}
		},
	}

	return &CircuitBreaker{
		cb: gobreaker.NewCircuitBreaker(settings),
	}
}

// Execute wraps a function call with circuit breaker protection
func (c *CircuitBreaker) Execute(ctx context.Context, fn func() (interface{}, error)) (interface{}, error) {
	return c.cb.Execute(func() (interface{}, error) {
		// Check context cancellation before executing
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			return fn()
		}
	})
}

// State returns the current state of the circuit breaker
func (c *CircuitBreaker) State() gobreaker.State {
	return c.cb.State()
}

// Name returns the name of the circuit breaker
func (c *CircuitBreaker) Name() string {
	return c.cb.Name()
}

// Counts returns the internal counter values
func (c *CircuitBreaker) Counts() gobreaker.Counts {
	return c.cb.Counts()
}
