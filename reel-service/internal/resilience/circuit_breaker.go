package resilience

import (
	"context"
	"log/slog"
	"time"

	"github.com/sony/gobreaker"
)

type CircuitBreaker struct {
	cb     *gobreaker.CircuitBreaker
	logger *slog.Logger
}

type Config struct {
	Name         string
	MaxRequests  uint32
	Interval     time.Duration
	Timeout      time.Duration
	FailureRatio float64
	MinRequests  uint32
}

func DefaultConfig(name string) Config {
	return Config{
		Name:         name,
		MaxRequests:  5,
		Interval:     10 * time.Second,
		Timeout:      30 * time.Second,
		FailureRatio: 0.5,
		MinRequests:  3,
	}
}

func NewCircuitBreaker(cfg Config, logger *slog.Logger) *CircuitBreaker {
	settings := gobreaker.Settings{
		Name:        cfg.Name,
		MaxRequests: cfg.MaxRequests,
		Interval:    cfg.Interval,
		Timeout:     cfg.Timeout,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
			return counts.Requests >= cfg.MinRequests && failureRatio >= cfg.FailureRatio
		},
		OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
			logger.Info("Circuit breaker state changed",
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

func (c *CircuitBreaker) Execute(ctx context.Context, fn func() (interface{}, error)) (interface{}, error) {
	return c.cb.Execute(fn)
}

func (c *CircuitBreaker) State() gobreaker.State {
	return c.cb.State()
}
