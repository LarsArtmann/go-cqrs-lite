package system

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

// ── GracefulClose tests ──

// slowDrainer sleeps for the specified duration before completing.
type slowDrainer struct {
	delay time.Duration
}

func (d *slowDrainer) Drain(ctx context.Context) error {
	select {
	case <-time.After(d.delay):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func TestSystem_GracefulClose_DrainTimeout(t *testing.T) {
	t.Parallel()

	sys := &System{
		drainers: []Drainer{&slowDrainer{delay: 200 * time.Millisecond}},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := sys.GracefulClose(ctx)
	if err == nil {
		t.Fatal("expected GracefulClose to fail with context deadline exceeded")
	}

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected DeadlineExceeded, got: %v", err)
	}
}

func TestSystem_GracefulClose_CloseTimeout(t *testing.T) {
	t.Parallel()

	// No drainers, but a slow engine close. Context expires during Close.
	sys := &System{
		engines: []namedEngine{
			{engine: &slowCloseEngine{delay: 200 * time.Millisecond}, name: "slow"},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := sys.GracefulClose(ctx)
	if err == nil {
		t.Fatal("expected GracefulClose to fail with context deadline exceeded")
	}

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected DeadlineExceeded, got: %v", err)
	}
}

func TestSystem_GracefulClose_NoDrainers(t *testing.T) {
	t.Parallel()

	sys := &System{
		engines: []namedEngine{
			{engine: &failingEngine{name: "a", err: nil}, name: "a"},
		},
	}

	err := sys.GracefulClose(context.Background())
	if err != nil {
		t.Fatalf("GracefulClose with no drainers should succeed: %v", err)
	}
}

func TestSystem_GracefulClose_MultipleDrainers(t *testing.T) {
	t.Parallel()

	var mu sync.Mutex
	order := []string{}

	sys := &System{
		drainers: []Drainer{
			&recordingDrainer{name: "first", order: &order, mu: &mu},
			&recordingDrainer{name: "second", order: &order, mu: &mu},
			&recordingDrainer{name: "third", order: &order, mu: &mu},
		},
	}

	err := sys.GracefulClose(context.Background())
	if err != nil {
		t.Fatalf("GracefulClose: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()

	if len(order) != 3 {
		t.Fatalf("expected 3 drainer calls, got %d", len(order))
	}

	expected := []string{"first", "second", "third"}
	for i, want := range expected {
		if order[i] != want {
			t.Errorf("drainer %d: expected %s, got %s", i, want, order[i])
		}
	}
}

// ── Test helpers ──

type recordingDrainer struct {
	name  string
	order *[]string
	mu    *sync.Mutex
}

func (d *recordingDrainer) Drain(_ context.Context) error {
	d.mu.Lock()
	*d.order = append(*d.order, d.name)
	d.mu.Unlock()
	return nil
}

// errorDrainer always returns the configured error from Drain.
type errorDrainer struct {
	err error
}

func (d *errorDrainer) Drain(_ context.Context) error { return d.err }

// ── Drain standalone tests ──

func TestSystem_Drain(t *testing.T) {
	t.Parallel()

	var mu sync.Mutex
	order := []string{}

	sys := &System{
		drainers: []Drainer{
			&recordingDrainer{name: "a", order: &order, mu: &mu},
			&recordingDrainer{name: "b", order: &order, mu: &mu},
		},
	}

	if err := sys.Drain(context.Background()); err != nil {
		t.Fatalf("Drain: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()

	if len(order) != 2 {
		t.Fatalf("expected 2 drainer calls, got %d", len(order))
	}

	// System is not closed — Close should still work.
	if err := sys.Close(); err != nil {
		t.Fatalf("Close after Drain: %v", err)
	}
}

func TestSystem_Drain_Error(t *testing.T) {
	t.Parallel()

	drainErr := errors.New("drain failed")

	sys := &System{
		drainers: []Drainer{
			&errorDrainer{err: drainErr},
		},
	}

	err := sys.Drain(context.Background())
	if err == nil {
		t.Fatal("expected Drain to return error from failing drainer")
	}

	if !errors.Is(err, drainErr) {
		t.Fatalf("expected error to wrap drain failure, got: %v", err)
	}

	// Drain error should NOT prevent a subsequent Close.
	if err := sys.Close(); err != nil {
		t.Fatalf("Close after Drain error should succeed: %v", err)
	}
}

func TestSystem_Drain_ContextExpired(t *testing.T) {
	t.Parallel()

	sys := &System{
		drainers: []Drainer{
			&slowDrainer{delay: 200 * time.Millisecond},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := sys.Drain(ctx)
	if err == nil {
		t.Fatal("expected Drain to fail with context deadline exceeded")
	}

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected DeadlineExceeded, got: %v", err)
	}
}
