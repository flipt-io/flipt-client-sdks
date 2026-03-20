package flipt

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/rand/v2"
	"time"
)

type backoff struct {
	base       time.Duration
	jitter     float64
	maxDelay   time.Duration
	maxRetries uint
}

func (b backoff) next(attempt int) time.Duration {
	if attempt < 0 {
		attempt = 0
	}

	delay := b.base
	if attempt > 0 {
		delay = b.base * (1 << attempt)
	}

	if b.maxDelay > 0 && delay > b.maxDelay {
		delay = b.maxDelay
	}

	if b.jitter <= 0 {
		return delay
	}

	factor := b.jitter + rand.Float64()*b.jitter
	jittered := time.Duration(float64(delay) * factor)
	if b.maxDelay > 0 {
		jittered = min(jittered, b.maxDelay)
	}
	return jittered
}

type retryableError struct{ err error }

func (e *retryableError) Error() string { return e.err.Error() }
func (e *retryableError) Unwrap() error { return e.err }

func newRetryableError(err error) error {
	if err == nil {
		return nil
	}
	return &retryableError{err: err}
}

func isRetryable(err error) bool {
	var re *retryableError
	return errors.As(err, &re)
}

func doWithRetry[T any](ctx context.Context, maxRetries uint, fn func(ctx context.Context) (T, error)) (T, error) {
	var zero T
	b := backoff{
		base:       time.Second,
		jitter:     0.1,
		maxDelay:   30 * time.Second,
		maxRetries: maxRetries,
	}

	for attempt := 0; ; attempt++ {
		select {
		case <-ctx.Done():
			return zero, ctx.Err()
		default:
		}
		v, err := fn(ctx)
		if err == nil {
			return v, nil
		}
		if !isRetryable(err) {
			return zero, err
		}

		if (b.maxRetries != 0 && attempt >= int(b.maxRetries)) || attempt >= math.MaxInt32 {
			return zero, fmt.Errorf("failed after %d retries, last error: %w", b.maxRetries, err)
		}

		delay := b.next(attempt)
		t := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			t.Stop()
			return zero, ctx.Err()
		case <-t.C:
		}
	}
}
