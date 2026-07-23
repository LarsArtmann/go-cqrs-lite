package stack

import (
	"context"
	"errors"
	"io"
	"sync"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

type testCloser struct {
	name     string
	closeErr error
	closed   bool
}

func (tc *testCloser) Close() error {
	tc.closed = true

	return tc.closeErr
}

var _ io.Closer = (*testCloser)(nil)

// orderTracker is a thread-safe closer that records the order in which
// Close is called relative to other orderTrackers sharing the same log.
type orderTracker struct {
	name string
	log  *orderLog
}

type orderLog struct {
	mu  sync.Mutex
	seq []string
}

func (l *orderLog) record(name string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.seq = append(l.seq, name)
}

func (t *orderTracker) Close() error {
	t.log.record(t.name)

	return nil
}

func TestHealthCheck_NoResources(t *testing.T) {
	t.Parallel()

	b := &Bundle{}
	if err := b.HealthCheck(context.Background()); err != nil {
		t.Fatalf("empty bundle should be healthy: %v", err)
	}
}

type testHealthChecker struct {
	healthy bool
}

func (h *testHealthChecker) Close() error { return nil }

func (h *testHealthChecker) HealthCheck(_ context.Context) error {
	if !h.healthy {
		return errors.New("unhealthy")
	}

	return nil
}

func TestHealthCheck_HealthyResource(t *testing.T) {
	t.Parallel()

	hc := &testHealthChecker{healthy: true}
	b := &Bundle{closers: []io.Closer{hc}}
	if err := b.HealthCheck(context.Background()); err != nil {
		t.Fatalf("healthy resource should pass: %v", err)
	}
}

func TestHealthCheck_UnhealthyResource(t *testing.T) {
	t.Parallel()

	hc := &testHealthChecker{healthy: false}
	b := &Bundle{closers: []io.Closer{hc}}
	if err := b.HealthCheck(context.Background()); err == nil {
		t.Fatal("unhealthy resource should fail")
	}
}

func TestShutdown_Ordered(t *testing.T) {
	t.Parallel()

	// Register in order: A, B, C
	// Declare: B must close before A (reverse of registration)
	a := &testCloser{name: "A"}
	b_ := &testCloser{name: "B"}
	c := &testCloser{name: "C"}

	bundle := &Bundle{
		closers:      []io.Closer{a, b_, c},
		shutdownDeps: []shutdownEdge{{before: b_, after: a}},
	}

	_ = bundle.Close()

	// Check that B closed before A
	// (both are closed, but we verify the ordering by checking close state during execution)
	// Since we can't inspect order after all are closed, we verify the orderedClosers output
	// by checking a fresh bundle with the same deps.
	a2 := &testCloser{name: "A"}
	b2 := &testCloser{name: "B"}
	c2 := &testCloser{name: "C"}
	bundle2 := &Bundle{
		closers:      []io.Closer{a2, b2, c2},
		shutdownDeps: []shutdownEdge{{before: b2, after: a2}},
	}

	ordered := bundle2.orderedClosers()

	// Find positions of B and A
	bPos, aPos := -1, -1
	for i, c := range ordered {
		if c == b2 {
			bPos = i
		}

		if c == a2 {
			aPos = i
		}
	}

	if bPos >= aPos {
		t.Fatalf("B (pos %d) should close before A (pos %d)", bPos, aPos)
	}
}

func TestShutdown_NoDeps_RegistrationOrder(t *testing.T) {
	t.Parallel()

	a := &testCloser{name: "A"}
	b_ := &testCloser{name: "B"}

	bundle := &Bundle{closers: []io.Closer{a, b_}}
	ordered := bundle.orderedClosers()

	if len(ordered) != 2 || ordered[0] != a || ordered[1] != b_ {
		t.Fatal("without deps, should keep registration order")
	}
}

func TestShutdown_CloseErrorReturned(t *testing.T) {
	t.Parallel()

	fail := &testCloser{name: "fail", closeErr: errors.New("boom")}
	bundle := &Bundle{closers: []io.Closer{fail}}

	err := bundle.Close()
	if err == nil {
		t.Fatal("close error should be returned")
	}
}

// TestShutdown_ThroughNewConstructor verifies that WithShutdownDependency
// works when wired through the real stack.New() constructor path — not
// struct literals. This is the integration test: it proves the option
// function is invoked, the closer pointers registered by WithCloser match
// the pointers passed to WithShutdownDependency, and Close() respects the
// declared ordering.
func TestShutdown_ThroughNewConstructor(t *testing.T) {
	t.Parallel()

	log := &orderLog{}

	storeA := &orderTracker{name: "eventstore", log: log}
	projectionHost := &orderTracker{name: "projectionhost", log: log}

	// Register in registration order: eventstore first, projectionhost second.
	// Declare: projectionhost must close BEFORE eventstore (reverse of registration).
	// This is the real-world pattern: projections drain before the store they read from.
	bundle, err := New(
		WithCloser(storeA),
		WithCloser(projectionHost),
		WithShutdownDependency(projectionHost, storeA),
		WithEventSink(nilEventSink{}), // satisfy validate() — at least one capability
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if err := bundle.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	if len(log.seq) != 2 {
		t.Fatalf("expected 2 closes, got %d: %v", len(log.seq), log.seq)
	}

	if log.seq[0] != "projectionhost" {
		t.Fatalf("expected projectionhost to close first, got %q in position 0", log.seq[0])
	}

	if log.seq[1] != "eventstore" {
		t.Fatalf("expected eventstore to close second, got %q in position 1", log.seq[1])
	}
}

// TestShutdown_ThroughNewConstructor_NoDeps verifies that without
// WithShutdownDependency, registration order is preserved through the
// real stack.New() constructor path.
func TestShutdown_ThroughNewConstructor_NoDeps(t *testing.T) {
	t.Parallel()

	log := &orderLog{}

	first := &orderTracker{name: "first", log: log}
	second := &orderTracker{name: "second", log: log}

	bundle, err := New(
		WithCloser(first),
		WithCloser(second),
		WithEventSink(nilEventSink{}),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if err := bundle.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	if len(log.seq) != 2 {
		t.Fatalf("expected 2 closes, got %d", len(log.seq))
	}

	if log.seq[0] != "first" || log.seq[1] != "second" {
		t.Fatalf("expected [first second], got %v", log.seq)
	}
}

// nilEventSink satisfies event.EventSink so Bundle.validate() passes.
// It is a no-op store that is never actually used in shutdown tests.
type nilEventSink struct{}

func (nilEventSink) Save(
	_ context.Context,
	_ id.StreamRef,
	_ []event.Event,
	_ event.Version,
) error {
	return nil
}

func (nilEventSink) AppendBatch(_ context.Context, _ id.StreamRef, _ []event.Event) error {
	return nil
}
