package utils

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestBackoff_SucceedsWithoutRetryingOnFirstSuccess(t *testing.T) {
	calls := 0
	err := Backoff(context.Background(), 3, time.Millisecond, 10*time.Millisecond, func() error {
		calls++
		return nil
	})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if calls != 1 {
		t.Fatalf("expected exactly 1 call, got: %d", calls)
	}
}

func TestBackoff_RetriesThenSucceeds(t *testing.T) {
	calls := 0
	err := Backoff(context.Background(), 3, time.Millisecond, 10*time.Millisecond, func() error {
		calls++
		if calls < 3 {
			return errors.New("transient failure")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("expected eventual success, got: %v", err)
	}
	if calls != 3 {
		t.Fatalf("expected 3 calls (2 failures + 1 success), got: %d", calls)
	}
}

func TestBackoff_GivesUpAfterMaxRetries(t *testing.T) {
	calls := 0
	wantErr := errors.New("persistent failure")
	err := Backoff(context.Background(), 2, time.Millisecond, 10*time.Millisecond, func() error {
		calls++
		return wantErr
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected persistent failure to propagate, got: %v", err)
	}
	// 1 initial attempt + 2 retries = 3 calls
	if calls != 3 {
		t.Fatalf("expected 3 calls (1 + maxRetries), got: %d", calls)
	}
}

func TestBackoff_StopsOnContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := Backoff(ctx, 5, 50*time.Millisecond, 500*time.Millisecond, func() error {
		return errors.New("always fails")
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got: %v", err)
	}
}
