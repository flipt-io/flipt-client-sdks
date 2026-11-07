package flipt_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	flipt "go.flipt.io/flipt-client"
)

func flagsDoc(keys ...string) string {
	flags := make([]string, 0, len(keys))
	for _, k := range keys {
		flags = append(flags, fmt.Sprintf(`{"key":%q,"name":%q,"enabled":true,"type":"BOOLEAN_FLAG_TYPE","rules":[],"rollouts":[]}`, k, k))
	}
	return fmt.Sprintf(`{"namespace":{"key":"default"},"flags":[%s]}`, strings.Join(flags, ","))
}

// streamServer serves the initial snapshot and then writes every document sent
// on the returned channel as a streaming message.
func streamServer(t *testing.T, initial string) (*httptest.Server, chan<- string) {
	t.Helper()
	msgs := make(chan string, 16)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/stream") {
			flusher := w.(http.Flusher)
			w.WriteHeader(http.StatusOK)
			flusher.Flush()
			for {
				select {
				case <-r.Context().Done():
					return
				case doc := <-msgs:
					b, _ := json.Marshal(map[string]json.RawMessage{"result": json.RawMessage(doc)})
					_, _ = w.Write(append(b, '\n'))
					flusher.Flush()
				}
			}
		}
		w.Header().Set("ETag", "initial")
		_, _ = w.Write([]byte(initial))
	}))
	t.Cleanup(srv.Close)
	return srv, msgs
}

func newStreamingClient(t *testing.T, url string, opts ...flipt.Option) *flipt.Client {
	t.Helper()
	opts = append([]flipt.Option{flipt.WithURL(url), flipt.WithFetchMode(flipt.FetchModeStreaming)}, opts...)
	client, err := flipt.NewClient(context.Background(), opts...)
	require.NoError(t, err)
	t.Cleanup(func() { _ = client.Close(context.Background()) })
	return client
}

func TestStateUpdateListener_CalledOnlyOnChange(t *testing.T) {
	srv, msgs := streamServer(t, flagsDoc("a"))

	calls := make(chan struct{}, 16)
	newStreamingClient(t, srv.URL, flipt.WithStateUpdateListener(func() { calls <- struct{}{} }))

	// the same state as the initial fetch is not a change
	msgs <- flagsDoc("a")
	require.Never(t, func() bool { return len(calls) > 0 }, 300*time.Millisecond, 10*time.Millisecond)

	msgs <- flagsDoc("a", "b")
	require.Eventually(t, func() bool { return len(calls) == 1 }, 2*time.Second, 10*time.Millisecond)

	// a repeated message is not a change
	msgs <- flagsDoc("a", "b")
	require.Never(t, func() bool { return len(calls) > 1 }, 300*time.Millisecond, 10*time.Millisecond)
}

func TestStateUpdateListener_ListenerSeesNewState(t *testing.T) {
	srv, msgs := streamServer(t, flagsDoc("a"))
	client := newStreamingClient(t, srv.URL)

	var flagCount atomic.Int32
	client.AddStateUpdateListener(func() {
		flags, err := client.ListFlags(context.Background())
		if err == nil {
			flagCount.Store(int32(len(flags)))
		}
	})

	msgs <- flagsDoc("a", "b", "c")
	require.Eventually(t, func() bool { return flagCount.Load() == 3 }, 2*time.Second, 10*time.Millisecond)
}

func TestStateUpdateListener_Remove(t *testing.T) {
	srv, msgs := streamServer(t, flagsDoc("a"))
	client := newStreamingClient(t, srv.URL)

	var removed, kept atomic.Int32
	remove := client.AddStateUpdateListener(func() { removed.Add(1) })
	client.AddStateUpdateListener(func() { kept.Add(1) })

	msgs <- flagsDoc("a", "b")
	require.Eventually(t, func() bool { return removed.Load() == 1 && kept.Load() == 1 }, 2*time.Second, 10*time.Millisecond)

	remove()
	remove() // idempotent

	msgs <- flagsDoc("a")
	require.Eventually(t, func() bool { return kept.Load() == 2 }, 2*time.Second, 10*time.Millisecond)
	require.Equal(t, int32(1), removed.Load())
}

func TestStateUpdateListener_PanicDoesNotStopOthers(t *testing.T) {
	srv, msgs := streamServer(t, flagsDoc("a"))
	client := newStreamingClient(t, srv.URL)

	var calls atomic.Int32
	client.AddStateUpdateListener(func() { panic("boom") })
	client.AddStateUpdateListener(func() { calls.Add(1) })

	msgs <- flagsDoc("a", "b")
	require.Eventually(t, func() bool { return calls.Load() == 1 }, 2*time.Second, 10*time.Millisecond)

	msgs <- flagsDoc("a")
	require.Eventually(t, func() bool { return calls.Load() == 2 }, 2*time.Second, 10*time.Millisecond)
}

func TestStateUpdateListener_CloseFromListener(t *testing.T) {
	srv, msgs := streamServer(t, flagsDoc("a"))
	client := newStreamingClient(t, srv.URL)

	type result struct{ closeErr, evalErr error }
	done := make(chan result, 1)
	var skipped atomic.Int32
	client.AddStateUpdateListener(func() {
		closeErr := client.Close(context.Background())
		_, evalErr := client.EvaluateBoolean(context.Background(), &flipt.EvaluationRequest{FlagKey: "a", EntityID: "e"})
		done <- result{closeErr, evalErr}
	})
	// listeners after the one that closed are skipped
	client.AddStateUpdateListener(func() { skipped.Add(1) })

	msgs <- flagsDoc("a", "b")
	select {
	case r := <-done:
		require.NoError(t, r.closeErr)
		require.ErrorIs(t, r.evalErr, flipt.ErrClientClosed)
	case <-time.After(5 * time.Second):
		t.Fatal("close from a listener did not return")
	}
	require.Never(t, func() bool { return skipped.Load() > 0 }, 200*time.Millisecond, 10*time.Millisecond)
}

func TestStateUpdateListener_CloseWaitsForRunningListener(t *testing.T) {
	srv, msgs := streamServer(t, flagsDoc("a"))
	client := newStreamingClient(t, srv.URL)

	var (
		started  = make(chan struct{})
		release  = make(chan struct{})
		finished atomic.Bool
	)
	client.AddStateUpdateListener(func() {
		close(started)
		<-release
		finished.Store(true)
	})

	msgs <- flagsDoc("a", "b")
	<-started

	closed := make(chan error, 1)
	go func() { closed <- client.Close(context.Background()) }()

	require.Never(t, func() bool { return len(closed) > 0 }, 200*time.Millisecond, 10*time.Millisecond)
	close(release)
	select {
	case err := <-closed:
		require.NoError(t, err)
		require.True(t, finished.Load(), "Close returned before the listener finished")
	case <-time.After(5 * time.Second):
		t.Fatal("Close did not return after the listener finished")
	}
}

func TestClosedClientReturnsErrClientClosed(t *testing.T) {
	srv, _ := streamServer(t, flagsDoc("a"))
	client := newStreamingClient(t, srv.URL)
	require.NoError(t, client.Close(context.Background()))

	ctx := context.Background()
	req := &flipt.EvaluationRequest{FlagKey: "a", EntityID: "e"}

	_, err := client.EvaluateBoolean(ctx, req)
	require.ErrorIs(t, err, flipt.ErrClientClosed)
	_, err = client.EvaluateVariant(ctx, req)
	require.ErrorIs(t, err, flipt.ErrClientClosed)
	_, err = client.EvaluateBatch(ctx, []*flipt.EvaluationRequest{req})
	require.ErrorIs(t, err, flipt.ErrClientClosed)
	_, err = client.ListFlags(ctx)
	require.ErrorIs(t, err, flipt.ErrClientClosed)
	_, err = client.GetSnapshot(ctx)
	require.ErrorIs(t, err, flipt.ErrClientClosed)

	// adding a listener after close returns a no-op remove func
	client.AddStateUpdateListener(func() {})()
}

func TestStateUpdateListener_RecoveryFromError(t *testing.T) {
	for _, tc := range []struct {
		name    string
		newEtag bool
	}{
		// the server comes back with the same etag the client held before
		// the error; the client must not get a 304 and stay in the error state
		{name: "same etag", newEtag: false},
		{name: "new etag", newEtag: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			testRecoveryFromError(t, tc.newEtag)
		})
	}
}

func testRecoveryFromError(t *testing.T, newEtag bool) {
	var (
		fail atomic.Bool
		etag atomic.Value
	)
	etag.Store("v1")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if fail.Load() {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		current := etag.Load().(string)
		if r.Header.Get("If-None-Match") == current {
			w.WriteHeader(http.StatusNotModified)
			return
		}
		// the flags never change; only the recovery should report
		w.Header().Set("ETag", current)
		_, _ = w.Write([]byte(flagsDoc("a")))
	}))
	t.Cleanup(srv.Close)

	var calls atomic.Int32
	client, err := flipt.NewClient(context.Background(),
		flipt.WithURL(srv.URL),
		flipt.WithUpdateInterval(time.Second),
		flipt.WithStateUpdateListener(func() { calls.Add(1) }),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = client.Close(context.Background()) })

	// unchanged polls (304) don't report
	require.Never(t, func() bool { return calls.Load() > 0 }, 1500*time.Millisecond, 50*time.Millisecond)

	fail.Store(true)
	require.Eventually(t, func() bool { return client.Err() != nil }, 5*time.Second, 50*time.Millisecond)
	require.Equal(t, int32(0), calls.Load())

	if newEtag {
		etag.Store("v2")
	}
	fail.Store(false)
	require.Eventually(t, func() bool { return calls.Load() == 1 && client.Err() == nil }, 5*time.Second, 50*time.Millisecond)
}
