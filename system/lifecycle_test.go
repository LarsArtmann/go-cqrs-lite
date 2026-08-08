package system

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/projectionhost/v4"
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

func TestSystem_Close_ProjectionHostError(t *testing.T) {
	t.Parallel()

	stopErr := errors.New("projection host crashed")

	var mu sync.Mutex
	engineClosed := []string{}

	sys := &System{
		engines: []namedEngine{
			{
				engine: &closeOrderEngine{name: "engine-a", order: &engineClosed, mu: &mu},
				name:   "engine-a",
			},
		},
		projHost: &failingProjHost{stopErr: stopErr},
	}

	err := sys.Close()
	if err == nil {
		t.Fatal("expected Close to return error when projection host Stop fails")
	}

	if !errors.Is(err, stopErr) {
		t.Fatalf("expected error to wrap projection host stop error, got: %v", err)
	}

	// Engine close must still run even when projection host Stop fails.
	mu.Lock()
	defer mu.Unlock()

	if len(engineClosed) != 1 || engineClosed[0] != "engine-a" {
		t.Fatalf("engine should still close after projection host error, got: %v", engineClosed)
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

// ── HealthCheckDetailed tests ──

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

func TestSystem_HealthCheckDetailed_MultipleEnginesMixed(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	sys := &System{
		engines: []namedEngine{
			{engine: &healthyEngine{profile: metaengine.EngineProfile{Name: "good"}}, name: "good"},
			{
				engine: &unhealthyEngine{profile: metaengine.EngineProfile{Name: "bad"}},
				name:   "bad",
			},
			{engine: &closeOrderEngine{name: "no-hc"}, name: "no-hc"},
		},
	}

	results := sys.HealthCheckDetailed(ctx)
	if len(results) != 2 {
		t.Fatalf("expected 2 results (healthy + unhealthy, skip non-HC), got %d", len(results))
	}

	byName := make(map[string]EngineHealth, len(results))
	for _, r := range results {
		byName[r.Name] = r
	}

	good, ok := byName["good"]
	if !ok {
		t.Fatal("expected result for healthy engine 'good'")
	}

	if good.Error != nil {
		t.Errorf("expected nil error for healthy engine, got: %v", good.Error)
	}

	bad, ok := byName["bad"]
	if !ok {
		t.Fatal("expected result for unhealthy engine 'bad'")
	}

	if bad.Error == nil {
		t.Error("expected error for unhealthy engine 'bad'")
	}

	// non-HealthChecker engine should be absent.
	if _, ok := byName["no-hc"]; ok {
		t.Error("non-HealthChecker engine should not appear in results")
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

// healthyEngine is a minimal Engine that implements HealthChecker and always
// returns nil (healthy).
type healthyEngine struct {
	profile metaengine.EngineProfile
}

func (e *healthyEngine) Profile() metaengine.EngineProfile { return e.profile }
func (e *healthyEngine) Close() error                      { return nil }
func (e *healthyEngine) HealthCheck(_ context.Context) error { return nil }

// failingProjHost is a mock projectionHostLifecycle whose Stop returns a
// configurable error. All other methods are no-ops.
type failingProjHost struct {
	stopErr error
}

func (f *failingProjHost) Start(_ context.Context) error                  { return nil }
func (f *failingProjHost) Stop() error                                    { return f.stopErr }
func (f *failingProjHost) Status() []projectionhost.WorkerState           { return nil }
func (f *failingProjHost) LagPerProjection() map[string]time.Duration     { return nil }
func (f *failingProjHost) LagDuration() time.Duration                     { return 0 }
func (f *failingProjHost) Reset(_ context.Context, _ string, _ ...projectionhost.ResetOption) error {
	return nil
}
