package resilience

import (
	"context"
	"log/slog"
	"time"

	"github.com/sony/gobreaker"
)

// CircuitBreakerConfig configures the circuit breaker
type CircuitBreakerConfig struct {
	Name         string
	MaxRequests  uint32        // max requests in half-open state
	Interval     time.Duration // cyclic period of closed state
	Timeout      time.Duration // period of open state
	FailureRatio float64       // ratio to trip (0.5 = 50% failures)
}

// DefaultConfig returns sensible defaults
func DefaultConfig(name string) CircuitBreakerConfig {
	return CircuitBreakerConfig{
		Name:         name,
		MaxRequests:  3,
		Interval:     10 * time.Second,
		Timeout:      30 * time.Second,
		FailureRatio: 0.5,
	}
}

// CircuitBreaker wraps gobreaker with logging
type CircuitBreaker struct {
	cb     *gobreaker.CircuitBreaker
	logger *slog.Logger
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(cfg CircuitBreakerConfig, logger *slog.Logger) *CircuitBreaker {
	if logger == nil {
		logger = slog.Default()
	}

	settings := gobreaker.Settings{
		Name:        cfg.Name,
		MaxRequests: cfg.MaxRequests,
		Interval:    cfg.Interval,
		Timeout:     cfg.Timeout,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
			return counts.Requests >= 5 && failureRatio >= cfg.FailureRatio
		},
		OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
			logger.Warn("Circuit breaker state change",
				"name", name,
				"from", from.String(),
				"to", to.String(),
			)
		},
	}

	return &CircuitBreaker{
		cb:     gobreaker.NewCircuitBreaker(settings),
		logger: logger,
	}
}

// Execute wraps a function with circuit breaker protection
func (c *CircuitBreaker) Execute(ctx context.Context, fn func() (interface{}, error)) (interface{}, error) {
	return c.cb.Execute(func() (interface{}, error) {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			return fn()
		}
	})
}

// ExecuteSimple wraps a simple error-returning function
func (c *CircuitBreaker) ExecuteSimple(ctx context.Context, fn func() error) error {
	_, err := c.Execute(ctx, func() (interface{}, error) {
		return nil, fn()
	})
	return err
}
