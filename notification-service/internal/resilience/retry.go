package resilience

import (
	"context"
	"math/rand"
	"time"
)

// Retry executes the given operation with exponential backoff and jitter.
func Retry(ctx context.Context, op func() error, maxAttempts int, initialDelay time.Duration) error {
	var err error
	delay := initialDelay

	for attempt := 0; attempt < maxAttempts; attempt++ {
		// Check context before trying
		if ctx.Err() != nil {
			return ctx.Err()
		}

		if err = op(); err == nil {
			return nil
		}

		// If this was the last attempt, return the error
		if attempt == maxAttempts-1 {
			return err
		}

		// Wait with backoff and jitter
		// Jitter: +/- 20% of current delay
		jitter := time.Duration(rand.Int63n(int64(delay)/5) - int64(delay)/10)
		sleepDuration := delay + jitter

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(sleepDuration):
			// Increase delay for next attempt (Factor: 2)
			delay *= 2
		}
	}
	return err
}
