package stack

import (
	"context"
	"errors"
	"io"
	"testing"
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

var _ io.Closer = (*testCloser)(nil)
