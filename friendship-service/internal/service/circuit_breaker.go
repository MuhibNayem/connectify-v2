package service

import (
	"context"
	"log/slog"

	"github.com/MuhibNayem/connectify-v2/shared-entity/resilience"
	"github.com/sony/gobreaker"
)

// CircuitBreakerWrapper wraps operations with circuit breaker protection
type CircuitBreakerWrapper struct {
	graphBreaker *resilience.CircuitBreaker
	userBreaker  *resilience.CircuitBreaker
	kafkaBreaker *resilience.CircuitBreaker
	logger       *slog.Logger
}

// NewCircuitBreakerWrapper creates a new circuit breaker wrapper
func NewCircuitBreakerWrapper(logger *slog.Logger) *CircuitBreakerWrapper {
	if logger == nil {
		logger = slog.Default()
	}

	graphCfg := resilience.DefaultConfig("graph-operations")
	graphCfg.OnStateChange = func(name string, from, to gobreaker.State) {
		logger.Warn("Circuit breaker state change",
			"name", name,
			"from", from.String(),
			"to", to.String(),
		)
	}

	userCfg := resilience.DefaultConfig("user-service")
	userCfg.OnStateChange = graphCfg.OnStateChange

	kafkaCfg := resilience.DefaultConfig("kafka-producer")
	kafkaCfg.OnStateChange = graphCfg.OnStateChange

	return &CircuitBreakerWrapper{
		graphBreaker: resilience.NewCircuitBreaker(graphCfg),
		userBreaker:  resilience.NewCircuitBreaker(userCfg),
		kafkaBreaker: resilience.NewCircuitBreaker(kafkaCfg),
		logger:       logger,
	}
}

// ExecuteGraphOp executes a graph operation with circuit breaker protection
func (c *CircuitBreakerWrapper) ExecuteGraphOp(ctx context.Context, name string, fn func() error) error {
	_, err := c.graphBreaker.Execute(ctx, func() (interface{}, error) {
		return nil, fn()
	})
	if err != nil {
		c.logger.Warn("Graph operation failed",
			"operation", name,
			"error", err,
		)
	}
	return err
}

// ExecuteUserServiceOp executes a user service operation with circuit breaker protection
func (c *CircuitBreakerWrapper) ExecuteUserServiceOp(ctx context.Context, name string, fn func() error) error {
	_, err := c.userBreaker.Execute(ctx, func() (interface{}, error) {
		return nil, fn()
	})
	if err != nil {
		c.logger.Warn("User service operation failed",
			"operation", name,
			"error", err,
		)
	}
	return err
}

// ExecuteKafkaOp executes a Kafka operation with circuit breaker protection
func (c *CircuitBreakerWrapper) ExecuteKafkaOp(ctx context.Context, name string, fn func() error) error {
	_, err := c.kafkaBreaker.Execute(ctx, func() (interface{}, error) {
		return nil, fn()
	})
	if err != nil {
		c.logger.Warn("Kafka operation failed",
			"operation", name,
			"error", err,
		)
	}
	return err
}
