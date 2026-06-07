package utils

import (
	"context"
	"math/rand"
	"time"
)

// Backoff calls the provided function with exponential backoff and jitter.
func Backoff(ctx context.Context, maxRetries int, initialDelay time.Duration, maxDelay time.Duration, fn func() error) error {
	delay := initialDelay

	for attempt := 1; ; attempt++ {
		err := fn()
		if err == nil {
			return nil
		}

		if attempt > maxRetries {
			return err
		}

		// Add jitter to avoid thundering herd problem
		jitter := time.Duration(rand.Float64() * float64(delay))
		sleepDuration := delay + jitter

		if sleepDuration > maxDelay {
			sleepDuration = maxDelay
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(sleepDuration):
			// Exponential increase for next delay
			delay *= 2
		}
	}
}
