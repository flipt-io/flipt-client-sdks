package flipt

import (
	"context"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestDecodeResultVariant_NilResult(t *testing.T) {
	_, err := decodeResult[VariantEvaluationResponse]([]byte(`{"status":"success","result":null}`))
	require.EqualError(t, err, "failed to unmarshal result: nil result")
}

func TestDecodeResultBoolean_NilResult(t *testing.T) {
	_, err := decodeResult[BooleanEvaluationResponse]([]byte(`{"status":"success","result":null}`))
	require.EqualError(t, err, "failed to unmarshal result: nil result")
}

func TestDecodeResultBatch_NilResult(t *testing.T) {
	_, err := decodeResult[BatchEvaluationResponse]([]byte(`{"status":"success","result":null}`))
	require.EqualError(t, err, "failed to unmarshal result: nil result")
}

func TestHandleUpdates_EmptySnapshotPreservesError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	expectedErr := errors.New("previous error")
	client := &Client{
		err:          expectedErr,
		snapshotChan: make(chan snapshot, 1),
	}

	done := make(chan error, 1)
	go func() {
		done <- client.handleUpdates(ctx)
	}()

	client.snapshotChan <- snapshot{etag: "etag-1"}
	require.Eventually(t, func() bool {
		client.mu.RLock()
		defer client.mu.RUnlock()
		return client.etag == "etag-1"
	}, time.Second, 10*time.Millisecond)

	client.mu.RLock()
	require.ErrorIs(t, client.err, expectedErr)
	client.mu.RUnlock()

	cancel()
	err := <-done
	require.ErrorIs(t, err, context.Canceled)
}

func TestHandleStream_CancelStopsReader(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client := &Client{
		snapshotChan: make(chan snapshot, 1),
	}

	reader := newBlockingReadCloser()
	done := make(chan error, 1)
	go func() {
		done <- client.handleStream(ctx, reader)
	}()

	cancel()

	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("handleStream did not return after context cancellation")
	}
}

func TestHandleStream_BufferLimit(t *testing.T) {
	client := &Client{
		snapshotChan: make(chan snapshot, 1),
	}

	payload := strings.Repeat("a", maxStreamBufferSize+1) + "\n"
	err := client.handleStream(context.Background(), io.NopCloser(strings.NewReader(payload)))
	require.EqualError(t, err, "stream buffer exceeded 1048576 bytes without a complete JSON message")
}

type blockingReadCloser struct {
	closeOnce sync.Once
	closed    chan struct{}
}

func newBlockingReadCloser() *blockingReadCloser {
	return &blockingReadCloser{closed: make(chan struct{})}
}

func (b *blockingReadCloser) Read(p []byte) (int, error) {
	<-b.closed
	return 0, io.EOF
}

func (b *blockingReadCloser) Close() error {
	b.closeOnce.Do(func() {
		close(b.closed)
	})
	return nil
}
