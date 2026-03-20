package flipt

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDoWithRetry_Retryable(t *testing.T) {
	var calls int

	_, err := doWithRetry(t.Context(), 2, func(ctx context.Context) (struct{}, error) {
		calls++
		if calls < 3 {
			return struct{}{}, newRetryableError(errors.New("transient"))
		}
		return struct{}{}, nil
	})
	require.NoError(t, err)
	require.Equal(t, 3, calls)
}

func TestDoWithRetry_NonRetryable(t *testing.T) {
	var calls int

	_, err := doWithRetry(t.Context(), 5, func(ctx context.Context) (struct{}, error) {
		calls++
		return struct{}{}, errors.New("permanent")
	})
	require.Error(t, err)
	require.Equal(t, 1, calls)
}
