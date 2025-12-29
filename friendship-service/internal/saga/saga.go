package saga

import (
	"context"
	"fmt"
	"log/slog"
)

// Step represents a single step in a saga with forward and compensating actions
type Step struct {
	Name       string
	Forward    func(ctx context.Context) error
	Compensate func(ctx context.Context) error
}

// Saga orchestrates distributed transactions with automatic rollback
type Saga struct {
	name   string
	steps  []Step
	logger *slog.Logger
}

// New creates a new Saga orchestrator
func New(name string, logger *slog.Logger) *Saga {
	if logger == nil {
		logger = slog.Default()
	}
	return &Saga{
		name:   name,
		steps:  []Step{},
		logger: logger,
	}
}

// AddStep adds a step with forward and compensating actions
func (s *Saga) AddStep(name string, forward, compensate func(ctx context.Context) error) *Saga {
	s.steps = append(s.steps, Step{
		Name:       name,
		Forward:    forward,
		Compensate: compensate,
	})
	return s
}

// Execute runs the saga with automatic rollback on failure
func (s *Saga) Execute(ctx context.Context) error {
	executed := []Step{}

	s.logger.Info("Starting saga", "saga", s.name, "steps", len(s.steps))

	for i, step := range s.steps {
		s.logger.Debug("Executing saga step", "saga", s.name, "step", step.Name, "index", i)

		if err := step.Forward(ctx); err != nil {
			s.logger.Warn("Saga step failed, initiating rollback",
				"saga", s.name,
				"failed_step", step.Name,
				"error", err,
			)

			// Rollback all executed steps in reverse order
			rollbackErr := s.rollback(ctx, executed)
			if rollbackErr != nil {
				return fmt.Errorf("saga '%s' failed at step '%s' (%w), rollback also failed: %v",
					s.name, step.Name, err, rollbackErr)
			}

			return fmt.Errorf("saga '%s' failed at step '%s': %w (rollback successful)", s.name, step.Name, err)
		}

		executed = append(executed, step)
	}

	s.logger.Info("Saga completed successfully", "saga", s.name)
	return nil
}

func (s *Saga) rollback(ctx context.Context, executed []Step) error {
	var lastErr error

	// Rollback in reverse order
	for i := len(executed) - 1; i >= 0; i-- {
		step := executed[i]

		s.logger.Debug("Rolling back saga step", "saga", s.name, "step", step.Name)

		if step.Compensate != nil {
			if err := step.Compensate(ctx); err != nil {
				s.logger.Error("Compensation failed",
					"saga", s.name,
					"step", step.Name,
					"error", err,
				)
				lastErr = err
				// Continue rolling back other steps even if one fails
			}
		}
	}

	return lastErr
}

// Result contains the result of saga execution
type Result struct {
	Success       bool
	FailedStep    string
	Error         error
	RollbackError error
}

// ExecuteWithResult runs the saga and returns detailed result
func (s *Saga) ExecuteWithResult(ctx context.Context) *Result {
	executed := []Step{}

	for _, step := range s.steps {
		if err := step.Forward(ctx); err != nil {
			result := &Result{
				Success:    false,
				FailedStep: step.Name,
				Error:      err,
			}

			if rollbackErr := s.rollback(ctx, executed); rollbackErr != nil {
				result.RollbackError = rollbackErr
			}

			return result
		}
		executed = append(executed, step)
	}

	return &Result{Success: true}
}
