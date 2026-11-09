package system

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// unhealthyEngine is a minimal Engine that implements HealthChecker and always
// returns an error from HealthCheck.
type unhealthyEngine struct {
	profile metaengine.EngineProfile
}

func (e *unhealthyEngine) Profile() metaengine.EngineProfile { return e.profile }
func (e *unhealthyEngine) Close() error                      { return nil }
func (e *unhealthyEngine) HealthCheck(_ context.Context) error {
	return errors.New("simulated engine failure")
}

// closeOrderEngine is a minimal Engine that records its close order into a
// shared slice. Used to verify orderedEngines output.
type closeOrderEngine struct {
	name  string
	order *[]string
	mu    *sync.Mutex
}

func (e *closeOrderEngine) Profile() metaengine.EngineProfile {
	return metaengine.EngineProfile{Name: e.name}
}

func (e *closeOrderEngine) Close() error {
	e.mu.Lock()
	*e.order = append(*e.order, e.name)
	e.mu.Unlock()

	return nil
}

// failingEngine is a minimal Engine that returns an error from Close.
type failingEngine struct {
	name string
	err  error
}

func (e *failingEngine) Profile() metaengine.EngineProfile {
	return metaengine.EngineProfile{Name: e.name}
}

func (e *failingEngine) Close() error { return e.err }

func TestSystem_HealthCheck_EngineUnhealthy(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	sys, err := New(ctx, DomainConfig{}, DeploymentConfig{
		Engines: map[string]EngineConfig{"primary": {Driver: "memory"}},
		Instances: []InstanceConfig{
			{Role: RoleSourceOfTruth, Engine: "primary"},
		},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer sys.Close()

	// Inject an unhealthy engine into the system's engine list.
	sys.mu.Lock()
	sys.engines = append(sys.engines, namedEngine{
		engine: &unhealthyEngine{
			profile: metaengine.EngineProfile{Name: "unhealthy-mock"},
		},
		name: "unhealthy-mock",
	})
	sys.mu.Unlock()

	err = sys.HealthCheck(ctx)
	if err == nil {
		t.Fatal("expected HealthCheck to return error for unhealthy engine")
	}

	if !strings.Contains(err.Error(), "unhealthy-mock") {
		t.Fatalf("expected error to name the unhealthy engine, got: %v", err)
	}

	if !strings.Contains(err.Error(), "simulated engine failure") {
		t.Fatalf("expected error to contain the engine's error message, got: %v", err)
	}
}

func TestOrderedEngines_NoDeps(t *testing.T) {
	t.Parallel()

	sys := &System{
		engines: []namedEngine{
			{engine: &closeOrderEngine{name: "a"}, name: "a"},
			{engine: &closeOrderEngine{name: "b"}, name: "b"},
			{engine: &closeOrderEngine{name: "c"}, name: "c"},
		},
	}

	result := sys.orderedEngines()
	if len(result) != 3 {
		t.Fatalf("expected 3 engines, got %d", len(result))
	}

	// With no deps, creation order is preserved.
	for i, want := range []string{"a", "b", "c"} {
		if result[i].Profile().Name != want {
			t.Fatalf("engine %d: expected %s, got %s", i, want, result[i].Profile().Name)
		}
	}
}

func TestOrderedEngines_BasicOrdering(t *testing.T) {
	t.Parallel()

	// Declare "b must close before a" — so a should close AFTER b.
	sys := &System{
		engines: []namedEngine{
			{engine: &closeOrderEngine{name: "a"}, name: "a"},
			{engine: &closeOrderEngine{name: "b"}, name: "b"},
		},
		shutdownDeps: []shutdownEdge{
			{before: "b", after: "a"},
		},
	}

	result := sys.orderedEngines()
	if len(result) != 2 {
		t.Fatalf("expected 2 engines, got %d", len(result))
	}

	// b should be first (closes before a), a should be second.
	if result[0].Profile().Name != "b" {
		t.Fatalf("expected b first, got %s", result[0].Profile().Name)
	}

	if result[1].Profile().Name != "a" {
		t.Fatalf("expected a second, got %s", result[1].Profile().Name)
	}
}

func TestOrderedEngines_CycleFallback(t *testing.T) {
	t.Parallel()

	// a→b and b→a creates a cycle. Both should fall back to creation order.
	sys := &System{
		engines: []namedEngine{
			{engine: &closeOrderEngine{name: "a"}, name: "a"},
			{engine: &closeOrderEngine{name: "b"}, name: "b"},
			{engine: &closeOrderEngine{name: "c"}, name: "c"}, // c is not in the cycle
		},
		shutdownDeps: []shutdownEdge{
			{before: "a", after: "b"},
			{before: "b", after: "a"},
		},
	}

	result := sys.orderedEngines()
	if len(result) != 3 {
		t.Fatalf("expected 3 engines, got %d", len(result))
	}

	// c has inDegree 0, so it's processed first by Kahn's algorithm.
	// a and b are in a cycle, so they fall back to creation order (a then b).
	if result[0].Profile().Name != "c" {
		t.Fatalf("expected c first (no deps), got %s", result[0].Profile().Name)
	}

	// Remaining a and b in creation order.
	if result[1].Profile().Name != "a" {
		t.Fatalf("expected a second (creation order fallback), got %s", result[1].Profile().Name)
	}

	if result[2].Profile().Name != "b" {
		t.Fatalf("expected b third (creation order fallback), got %s", result[2].Profile().Name)
	}
}

func TestOrderedEngines_UnknownNames(t *testing.T) {
	t.Parallel()

	// Edges referencing unknown engine names should be silently ignored.
	sys := &System{
		engines: []namedEngine{
			{engine: &closeOrderEngine{name: "a"}, name: "a"},
			{engine: &closeOrderEngine{name: "b"}, name: "b"},
		},
		shutdownDeps: []shutdownEdge{
			{before: "nonexistent", after: "a"},
			{before: "b", after: "also-missing"},
		},
	}

	result := sys.orderedEngines()
	if len(result) != 2 {
		t.Fatalf("expected 2 engines, got %d", len(result))
	}

	// Both edges ignored, so creation order is preserved.
	if result[0].Profile().Name != "a" || result[1].Profile().Name != "b" {
		t.Fatalf("expected creation order a,b, got %s,%s",
			result[0].Profile().Name, result[1].Profile().Name)
	}
}

func TestSystem_Close_ErrorJoining(t *testing.T) {
	t.Parallel()

	errA := errors.New("engine a failed")
	errB := errors.New("engine b failed")

	sys := &System{
		engines: []namedEngine{
			{engine: &failingEngine{name: "a", err: errA}, name: "a"},
			{engine: &failingEngine{name: "b", err: errB}, name: "b"},
		},
	}

	err := sys.Close()
	if err == nil {
		t.Fatal("expected Close to return joined error")
	}

	// errors.Join wraps both errors — verify both are present.
	if !errors.Is(err, errA) {
		t.Fatalf("expected error to contain errA, got: %v", err)
	}

	if !errors.Is(err, errB) {
		t.Fatalf("expected error to contain errB, got: %v", err)
	}

	// Double close is a no-op.
	if err := sys.Close(); err != nil {
		t.Fatalf("second Close should be nil, got: %v", err)
	}
}

func TestSystem_Close_OrderMatchesOrderedEngines(t *testing.T) {
	t.Parallel()

	var mu sync.Mutex
	var closeOrder []string

	sys := &System{
		engines: []namedEngine{
			{engine: &closeOrderEngine{name: "a", order: &closeOrder, mu: &mu}, name: "a"},
			{engine: &closeOrderEngine{name: "b", order: &closeOrder, mu: &mu}, name: "b"},
			{engine: &closeOrderEngine{name: "c", order: &closeOrder, mu: &mu}, name: "c"},
		},
		shutdownDeps: []shutdownEdge{
			{before: "c", after: "a"}, // c closes before a
			{before: "c", after: "b"}, // c closes before b
		},
	}

	if err := sys.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()

	// c should close first, then a and b in creation order.
	if len(closeOrder) != 3 {
		t.Fatalf("expected 3 closes, got %d: %v", len(closeOrder), closeOrder)
	}

	if closeOrder[0] != "c" {
		t.Fatalf("expected c first, got %s", closeOrder[0])
	}

	// a and b have no edge between them, so they keep creation order.
	if closeOrder[1] != "a" || closeOrder[2] != "b" {
		t.Fatalf("expected a,b after c, got %s,%s", closeOrder[1], closeOrder[2])
	}
}
