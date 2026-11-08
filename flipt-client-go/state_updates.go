package flipt

import (
	"bytes"
	"log/slog"
	"runtime"
	"slices"
	"strconv"
	"sync"
	"sync/atomic"
)

type stateUpdateListener struct {
	fn      func()
	removed atomic.Bool
}

// stateUpdates fans state changes out to listeners on one goroutine, started
// when the first listener is added. Changes published while listeners run are
// coalesced into one follow-up call.
type stateUpdates struct {
	mu        sync.Mutex
	listeners []*stateUpdateListener
	started   bool
	closed    bool

	pending chan struct{}
	done    chan struct{}
	// stopped is closed when the dispatch goroutine exits.
	stopped chan struct{}
	// dispatcher is the id of the dispatch goroutine, so close can tell
	// whether it is called from a listener.
	dispatcher atomic.Uint64
}

func newStateUpdates() *stateUpdates {
	return &stateUpdates{
		pending: make(chan struct{}, 1),
		done:    make(chan struct{}),
		stopped: make(chan struct{}),
	}
}

func (s *stateUpdates) add(fn func()) (remove func()) {
	if fn == nil {
		return func() {}
	}

	l := &stateUpdateListener{fn: fn}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return func() {}
	}

	s.listeners = append(s.listeners, l)
	if !s.started {
		s.started = true
		go s.run()
	}

	return func() {
		if l.removed.Swap(true) {
			return
		}
		s.mu.Lock()
		defer s.mu.Unlock()
		for i, other := range s.listeners {
			if other == l {
				s.listeners = append(s.listeners[:i:i], s.listeners[i+1:]...)
				break
			}
		}
	}
}

// publish reports a change. It never blocks.
func (s *stateUpdates) publish() {
	s.mu.Lock()
	hasListeners := !s.closed && len(s.listeners) > 0
	s.mu.Unlock()

	if !hasListeners {
		return
	}

	select {
	case s.pending <- struct{}{}:
	default:
		// a notification is already pending
	}
}

// close stops dispatching and waits for a listener that is already running.
// Called from a listener, it returns without waiting; the listeners after it
// are not called.
func (s *stateUpdates) close() {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return
	}
	s.closed = true
	s.listeners = nil
	started := s.started
	close(s.done)
	// unlock before waiting: a running listener may add or remove listeners
	s.mu.Unlock()

	if started && goroutineID() != s.dispatcher.Load() {
		<-s.stopped
	}
}

func (s *stateUpdates) run() {
	s.dispatcher.Store(goroutineID())
	defer close(s.stopped)

	for {
		select {
		case <-s.done:
			return
		case <-s.pending:
		}

		s.mu.Lock()
		listeners := slices.Clone(s.listeners)
		s.mu.Unlock()

		for _, l := range listeners {
			if s.isClosed() {
				return
			}
			if l.removed.Load() {
				continue
			}
			callListener(l.fn)
		}
	}
}

func (s *stateUpdates) isClosed() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.closed
}

// goroutineID returns the id of the calling goroutine, parsed from the
// "goroutine N [" header of its stack trace.
func goroutineID() uint64 {
	var buf [64]byte
	n := runtime.Stack(buf[:], false)
	fields := bytes.Fields(buf[:n])
	if len(fields) < 2 {
		return 0
	}
	id, _ := strconv.ParseUint(string(fields[1]), 10, 64)
	return id
}

func callListener(fn func()) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("flipt: state update listener panicked", "panic", r)
		}
	}()
	fn()
}
