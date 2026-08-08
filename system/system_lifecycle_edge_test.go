package system

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/projectionhost/v4"
)

// ── Close idempotency ──

func TestSystem_Close_Idempotent(t *testing.T) {
	t.Parallel()

	var mu sync.Mutex
	var closeCount int

	sys := &System{
		engines: []namedEngine{
			{
				engine: &countingCloseEngine{name: "engine-a", count: &closeCount, mu: &mu},
				name:   "engine-a",
			},
		},
	}

	if err := sys.Close(); err != nil {
		t.Fatalf("first Close: %v", err)
	}

	if err := sys.Close(); err != nil {
		t.Fatalf("second Close should return nil, got: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()

	if closeCount != 1 {
		t.Fatalf("engine should be closed exactly once, got %d", closeCount)
	}
}

// ── GracefulClose idempotency ──

func TestSystem_GracefulClose_Idempotent(t *testing.T) {
	t.Parallel()

	var mu sync.Mutex
	var closeCount int

	sys := &System{
		engines: []namedEngine{
			{
				engine: &countingCloseEngine{name: "engine-a", count: &closeCount, mu: &mu},
				name:   "engine-a",
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := sys.GracefulClose(ctx); err != nil {
		t.Fatalf("first GracefulClose: %v", err)
	}

	if err := sys.GracefulClose(ctx); err != nil {
		t.Fatalf("second GracefulClose should return nil, got: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()

	if closeCount != 1 {
		t.Fatalf("engine should be closed exactly once, got %d", closeCount)
	}
}

// ── Start with projection host error ──

func TestSystem_Start_ProjectionHostError(t *testing.T) {
	t.Parallel()

	startErr := errors.New("projection host failed to start")

	sys := &System{
		projHost: &failingStartProjHost{err: startErr},
	}

	err := sys.Start(context.Background())
	if err == nil {
		t.Fatal("expected Start to return error when projection host Start fails")
	}

	if !errors.Is(err, startErr) {
		t.Fatalf("expected error to wrap projection host start error, got: %v", err)
	}

	// started flag is set before calling projHost.Start — a second Start
	// should return ErrAlreadyStarted, not re-attempt the projection host.
	err = sys.Start(context.Background())
	if !errors.Is(err, ErrAlreadyStarted) {
		t.Fatalf("second Start should return ErrAlreadyStarted, got: %v", err)
	}
}

// ── RegisterCloser after Close ──

func TestSystem_RegisterCloser_AfterClose(t *testing.T) {
	t.Parallel()

	sys := &System{}

	if err := sys.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	var mu sync.Mutex
	var closed bool

	// Register a closer AFTER the system is already closed.
	rc := &trackingCloser{name: "late", closed: &closed, mu: &mu}
	sys.RegisterCloser("late-closer", rc)

	// Second Close is a no-op (stopped=true) — the late closer must never fire.
	if err := sys.Close(); err != nil {
		t.Fatalf("second Close: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()

	if closed {
		t.Fatal("closer registered after Close should never be invoked")
	}
}

// ── HealthCheckDetailed with a failed projection ──

func TestSystem_HealthCheckDetailed_WithFailedProjection(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	projName := "orders"
	sys := &System{
		engines: []namedEngine{
			{
				engine: &healthyEngine{profile: metaengine.EngineProfile{Name: "primary"}},
				name:   "primary",
			},
		},
		projHost: &failedStatusProjHost{
			failedWorker: projectionhost.WorkerState{
				Name:      projName,
				Status:    projectionhost.WorkerFailed,
				LastError: "handler panic: nil pointer dereference",
			},
		},
	}

	results := sys.HealthCheckDetailed(ctx)

	// Expect 2 entries: the healthy engine + the failed projection.
	if len(results) != 2 {
		t.Fatalf("expected 2 results (engine + projection), got %d: %+v", len(results), results)
	}

	var foundProj bool

	for _, r := range results {
		if r.Name == "projection:"+projName {
			foundProj = true

			if r.Error == nil {
				t.Error("expected error for failed projection")
			}

			continue
		}

		// The engine entry should be healthy.
		if r.Name == "primary" && r.Error != nil {
			t.Errorf("expected healthy engine, got error: %v", r.Error)
		}
	}

	if !foundProj {
		t.Fatalf("failed projection %q not found in results: %+v", "projection:"+projName, results)
	}

	// HealthCheck (non-detailed) should also surface the failure.
	if err := sys.HealthCheck(ctx); err == nil {
		t.Error("HealthCheck should return error when a projection has failed")
	}
}

// ── Test helpers ──

// countingCloseEngine is a minimal Engine that increments a shared counter
// on Close. Used to verify idempotent close behavior.
type countingCloseEngine struct {
	name  string
	count *int
	mu    *sync.Mutex
}

func (e *countingCloseEngine) Profile() metaengine.EngineProfile {
	return metaengine.EngineProfile{Name: e.name}
}

func (e *countingCloseEngine) Close() error {
	e.mu.Lock()
	*e.count++
	e.mu.Unlock()
	return nil
}

// trackingCloser is an io.Closer that records whether Close was called.
type trackingCloser struct {
	name   string
	closed *bool
	mu     *sync.Mutex
}

func (c *trackingCloser) Close() error {
	c.mu.Lock()
	*c.closed = true
	c.mu.Unlock()
	return nil
}

// failingStartProjHost is a mock projectionHostLifecycle whose Start returns
// a configurable error. All other methods are no-ops.
type failingStartProjHost struct {
	err error
}

func (f *failingStartProjHost) Start(_ context.Context) error              { return f.err }
func (f *failingStartProjHost) Stop() error                                { return nil }
func (f *failingStartProjHost) Status() []projectionhost.WorkerState       { return nil }
func (f *failingStartProjHost) LagPerProjection() map[string]time.Duration { return nil }
func (f *failingStartProjHost) LagDuration() time.Duration                 { return 0 }

func (f *failingStartProjHost) Reset(
	_ context.Context,
	_ string,
	_ ...projectionhost.ResetOption,
) error {
	return nil
}

// failedStatusProjHost is a mock projectionHostLifecycle that returns a
// configurable failed worker from Status. All lifecycle methods are no-ops.
type failedStatusProjHost struct {
	failedWorker projectionhost.WorkerState
}

func (f *failedStatusProjHost) Start(_ context.Context) error { return nil }
func (f *failedStatusProjHost) Stop() error                   { return nil }

func (f *failedStatusProjHost) Status() []projectionhost.WorkerState {
	return []projectionhost.WorkerState{f.failedWorker}
}

func (f *failedStatusProjHost) LagPerProjection() map[string]time.Duration { return nil }
func (f *failedStatusProjHost) LagDuration() time.Duration                 { return 0 }

func (f *failedStatusProjHost) Reset(
	_ context.Context,
	_ string,
	_ ...projectionhost.ResetOption,
) error {
	return nil
}
