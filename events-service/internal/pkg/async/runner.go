package async

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

// Runner handles asynchronous task execution with safety mechanisms
type Runner struct {
	logger *slog.Logger
}

// NewRunner creates a new async runner
func NewRunner(logger *slog.Logger) *Runner {
	return &Runner{
		logger: logger,
	}
}

// RunAsync executes a function in a goroutine with panic recovery
func (r *Runner) RunAsync(ctx context.Context, name string, fn func() error) {
	go func() {
		defer func() {
			if rec := recover(); rec != nil {
				r.logger.ErrorContext(ctx, "panic in async task",
					"task", name,
					"error", fmt.Sprintf("%v", rec),
				)
			}
		}()

		if err := fn(); err != nil {
			r.logger.ErrorContext(ctx, "async task failed",
				"task", name,
				"error", err.Error(),
			)
		}
	}()
}

// RunAsyncRetry executes a function in a goroutine with retry logic
func (r *Runner) RunAsyncRetry(ctx context.Context, name string, fn func() error, attempts int, delay time.Duration) {
	go func() {
		defer func() {
			if rec := recover(); rec != nil {
				r.logger.ErrorContext(ctx, "panic in async task (retry loop)",
					"task", name,
					"error", fmt.Sprintf("%v", rec),
				)
			}
		}()

		var err error
		for i := 0; i < attempts; i++ {
			if i > 0 {
				backoff := delay << (i - 1)
				select {
				case <-ctx.Done():
					r.logger.InfoContext(ctx, "async task canceled during retry", "task", name)
					return
				case <-time.After(backoff):
				}
			}

			if err = fn(); err == nil {
				return // Success
			}

			r.logger.WarnContext(ctx, "async task failed, retrying",
				"task", name,
				"attempt", i+1,
				"max_attempts", attempts,
				"error", err.Error(),
			)
		}

		// Final failure log
		r.logger.ErrorContext(ctx, "async task permanently failed after retries",
			"task", name,
			"attempts", attempts,
			"error", err.Error(),
		)
	}()
}
