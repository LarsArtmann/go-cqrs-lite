package system

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// ── Shutdown ordering tests ──

func TestOrderedEngines_ComplexDAG(t *testing.T) {
	t.Parallel()

	// 5-engine DAG: a→b→c, a→d, e is independent.
	// Shutdown order must respect: a closes before b and d, b before c.
	sys := &System{
		engines: []namedEngine{
			{engine: &closeOrderEngine{name: "a"}, name: "a"},
			{engine: &closeOrderEngine{name: "b"}, name: "b"},
			{engine: &closeOrderEngine{name: "c"}, name: "c"},
			{engine: &closeOrderEngine{name: "d"}, name: "d"},
			{engine: &closeOrderEngine{name: "e"}, name: "e"},
		},
		shutdownDeps: []shutdownEdge{
			{before: "a", after: "b"},
			{before: "b", after: "c"},
			{before: "a", after: "d"},
		},
	}

	result := sys.orderedEngines()
	if len(result) != 5 {
		t.Fatalf("expected 5 engines, got %d", len(result))
	}

	names := make([]string, len(result))
	for i, eng := range result {
		names[i] = eng.Profile().Name
	}

	// a must come before b and d; b must come before c.
	pos := make(map[string]int)
	for i, n := range names {
		pos[n] = i
	}

	if pos["a"] > pos["b"] {
		t.Errorf("a should close before b: order=%v", names)
	}

	if pos["a"] > pos["d"] {
		t.Errorf("a should close before d: order=%v", names)
	}

	if pos["b"] > pos["c"] {
		t.Errorf("b should close before c: order=%v", names)
	}
}

func TestOrderedEngines_SelfLoop(t *testing.T) {
	t.Parallel()

	// A self-loop {before: "a", after: "a"} should be silently ignored.
	sys := &System{
		engines: []namedEngine{
			{engine: &closeOrderEngine{name: "a"}, name: "a"},
			{engine: &closeOrderEngine{name: "b"}, name: "b"},
		},
		shutdownDeps: []shutdownEdge{
			{before: "a", after: "a"},
			{before: "b", after: "a"},
		},
	}

	result := sys.orderedEngines()
	if len(result) != 2 {
		t.Fatalf("expected 2 engines, got %d", len(result))
	}

	// b should close before a (self-loop on a is ignored).
	if result[0].Profile().Name != "b" {
		t.Fatalf("expected b first, got %s", result[0].Profile().Name)
	}
}

func TestOrderedEngines_DuplicateEdges(t *testing.T) {
	t.Parallel()

	// Duplicate edges should not prevent correct topological ordering. The
	// adjacency list stores duplicates, so inDegree is decremented the same
	// number of times, eventually reaching 0.
	sys := &System{
		engines: []namedEngine{
			{engine: &closeOrderEngine{name: "a"}, name: "a"},
			{engine: &closeOrderEngine{name: "b"}, name: "b"},
		},
		shutdownDeps: []shutdownEdge{
			{before: "a", after: "b"},
			{before: "a", after: "b"},
			{before: "a", after: "b"},
		},
	}

	result := sys.orderedEngines()
	if len(result) != 2 {
		t.Fatalf("expected 2 engines, got %d", len(result))
	}

	// a should close before b (Kahn's algorithm handles duplicate edges
	// because the adjacency list also has triplicate entries, so inDegree
	// is decremented correctly).
	if result[0].Profile().Name != "a" {
		t.Fatalf("expected a first, got %s", result[0].Profile().Name)
	}

	if result[1].Profile().Name != "b" {
		t.Fatalf("expected b second, got %s", result[1].Profile().Name)
	}
}

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

// ── Close edge cases ──

func TestSystem_Close_NoEngines(t *testing.T) {
	t.Parallel()

	sys := &System{}

	if err := sys.Close(); err != nil {
		t.Fatalf("Close with no engines should return nil: %v", err)
	}
}

func TestSystem_RegisterCloser(t *testing.T) {
	t.Parallel()

	var mu sync.Mutex
	var engineClosed []string
	closedNames := []string{}

	sys := &System{
		engines: []namedEngine{
			{
				engine: &closeOrderEngine{name: "engine-a", order: &engineClosed, mu: &mu},
				name:   "engine-a",
			},
		},
	}

	sys.RegisterCloser("ext-1", &recordingCloser{name: "ext-1", names: &closedNames, mu: &mu})
	sys.RegisterCloser("ext-2", &recordingCloser{name: "ext-2", names: &closedNames, mu: &mu})

	if err := sys.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()

	if len(closedNames) != 2 {
		t.Fatalf("expected 2 closers called, got %d: %v", len(closedNames), closedNames)
	}

	// Engines close first, then registered closers.
	if closedNames[0] != "ext-1" || closedNames[1] != "ext-2" {
		t.Fatalf("expected ext-1,ext-2 order, got %v", closedNames)
	}
}

// ── Test helpers ──

type slowCloseEngine struct {
	metaengine.Engine
	delay time.Duration
}

func (e *slowCloseEngine) Profile() metaengine.EngineProfile {
	return metaengine.EngineProfile{Name: "slow"}
}

func (e *slowCloseEngine) Close() error {
	time.Sleep(e.delay)
	return nil
}

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

type recordingCloser struct {
	name  string
	names *[]string
	mu    *sync.Mutex
}

func (c *recordingCloser) Close() error {
	c.mu.Lock()
	*c.names = append(*c.names, c.name)
	c.mu.Unlock()
	return nil
}

// ── Drain standalone test ──

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

// ── EngineNames test ──

func TestSystem_EngineNames(t *testing.T) {
	t.Parallel()

	sys := &System{
		engines: []namedEngine{
			{engine: &closeOrderEngine{name: "alpha"}, name: "alpha"},
			{engine: &closeOrderEngine{name: "beta"}, name: "beta"},
			{engine: &closeOrderEngine{name: "gamma"}, name: "gamma"},
		},
	}

	names := sys.EngineNames()
	if len(names) != 3 {
		t.Fatalf("expected 3 names, got %d", len(names))
	}

	expected := []string{"alpha", "beta", "gamma"}
	for i, want := range expected {
		if names[i] != want {
			t.Errorf("name %d: expected %s, got %s", i, want, names[i])
		}
	}
}

// ── ShutdownOrder test ──

func TestSystem_ShutdownOrder(t *testing.T) {
	t.Parallel()

	sys := &System{
		engines: []namedEngine{
			{engine: &closeOrderEngine{name: "a"}, name: "a"},
			{engine: &closeOrderEngine{name: "b"}, name: "b"},
			{engine: &closeOrderEngine{name: "c"}, name: "c"},
		},
		shutdownDeps: []shutdownEdge{
			{before: "c", after: "a"},
			{before: "c", after: "b"},
		},
	}

	order := sys.ShutdownOrder()
	if len(order) != 3 {
		t.Fatalf("expected 3 names, got %d", len(order))
	}

	if order[0] != "c" {
		t.Fatalf("expected c first, got %s", order[0])
	}
}

// ── HealthCheckDetailed test ──

func TestSystem_HealthCheckDetailed(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	sys := &System{
		engines: []namedEngine{
			{
				engine: &unhealthyEngine{profile: metaengine.EngineProfile{Name: "broken"}},
				name:   "broken",
			},
		},
	}

	// broken engine always returns an error from HealthCheck.
	results := sys.HealthCheckDetailed(ctx)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	if results[0].Name != "broken" {
		t.Errorf("expected name broken, got %s", results[0].Name)
	}

	if results[0].Error == nil {
		t.Error("expected error from unhealthy engine")
	}

	if !strings.Contains(results[0].Error.Error(), "simulated") {
		t.Errorf("expected simulated error, got: %v", results[0].Error)
	}
}

func TestSystem_HealthCheckDetailed_AllHealthy(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	sys := &System{
		engines: []namedEngine{
			{engine: &closeOrderEngine{name: "no-hc"}, name: "no-hc"},
		},
	}

	// closeOrderEngine does NOT implement HealthChecker — should be skipped.
	results := sys.HealthCheckDetailed(ctx)
	if len(results) != 0 {
		t.Fatalf("expected 0 results for non-HealthChecker engine, got %d", len(results))
	}
}
