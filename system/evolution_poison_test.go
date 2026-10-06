package system_test

import (
	"context"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/projectionhost/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

// ── Poison-event test types (idea 251) ──

type PoisonCreated struct {
	ID    string
	Title string
}

type PoisonRenamed struct {
	ID string
}

type PoisonView struct {
	ID    string
	Title string
}

// poisonPrevEngine wraps a memory engine and hands out a schema-mismatched
// previous value for one (collection, key): the shape a SQL engine holds when
// the result type evolved away from stored state. It exposes only MapBackend
// (no MapUpdater/VersionedWriter), so update folds take the MapGet → invoke →
// MapSet path where reifyTo runs.
type poisonPrevEngine struct {
	metaengine.Engine

	inner      metaengine.MapBackend
	collection string
	key        string
	badPrev    any
}

func (e *poisonPrevEngine) MapSet(
	ctx context.Context, collection string, key, value any,
) error {
	return e.inner.MapSet(ctx, collection, key, value)
}

func (e *poisonPrevEngine) MapGet(
	ctx context.Context, collection string, key any,
) (any, bool, error) {
	if collection == e.collection && key == e.key {
		return e.badPrev, true, nil
	}

	return e.inner.MapGet(ctx, collection, key)
}

func (e *poisonPrevEngine) MapDelete(ctx context.Context, collection string, key any) error {
	return e.inner.MapDelete(ctx, collection, key)
}

// TestSystem_EvolutionPoisonEvent_LandsInDLQ_WorkerSurvives drives a poison
// event (stored state that cannot reify into the result type) through the
// full projection pipeline: journal drain → projectionhost worker → fold
// reifyTo failure → collection poison → dead-letter. The poison entry must
// carry the Corruption family and the reify code, and the worker must keep
// processing later events for healthy collections.
func TestSystem_EvolutionPoisonEvent_LandsInDLQ_WorkerSurvives(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	metaengine.RegisterDriver("poisonprev-evolutions-test", func(
		_ context.Context, _ metaengine.DriverConfig,
	) (metaengine.Engine, error) {
		base := metaengine.NewMemoryEngine()
		inner, ok := base.(metaengine.MapBackend)
		if !ok {
			panic(
				"memory engine must implement MapBackend",
			)
		}

		return &poisonPrevEngine{
			Engine:     base,
			inner:      inner,
			collection: "poison_views",
			key:        "poison-1",
			// Title as a number: JSON cannot unmarshal it into PoisonView's
			// string field — the schema-mismatch shape.
			badPrev: map[string]any{"ID": "poison-1", "Title": 123},
		}, nil
	})

	dlq := projectionhost.NewMemoryDeadLetterStore()

	domain := system.DomainConfig{
		Evolutions: []system.EvolutionSpec{
			system.OnEvolution(
				system.Evolve[PoisonView]("poison_evo").
					On("poison.created", PoisonCreated{}),
				"poison.renamed", PoisonRenamed{},
				func(_ PoisonRenamed, v *PoisonView) { v.Title = "renamed" },
			).Done(),
		},
		Projections: []system.ProjectionDeclaration{
			system.Lookup[PoisonView]("poison_views").Done(),
			system.Lookup[PoisonView]("healthy_views").Done(),
		},
		ProjectionHostOptions: []projectionhost.HostOption{
			projectionhost.WithDeadLetterStore(dlq, 2),
		},
	}

	deployment := system.DeploymentConfig{
		Engines: map[string]system.EngineConfig{
			"primary":  {Driver: "memory"},
			"poisoned": {Driver: "poisonprev-evolutions-test"},
		},
		Instances: []system.InstanceConfig{
			{Role: system.RoleSourceOfTruth, Engine: "primary"},
			{Role: system.RoleProjections, Engine: "poisoned"},
		},
	}

	sys, err := system.New(ctx, domain, deployment)
	if err != nil {
		t.Fatalf("system.New: %v", err)
	}

	defer sys.Close()

	seedPoisonEvents(t, ctx, sys)

	if err := sys.Start(ctx); err != nil {
		t.Fatalf("system.Start: %v", err)
	}

	waitForPoisonDrain(t, ctx, sys, dlq)

	assertPoisonDLQEntry(t, ctx, dlq)
	assertPoisonWorkerSurvived(t, ctx, sys)
}

// seedPoisonEvents writes the three-event scenario directly to the journal:
// a good insert, the poison rename (schema-mismatched prev), and a healthy
// insert that must still project after the poison event was dead-lettered.
func seedPoisonEvents(t *testing.T, ctx context.Context, sys *system.System) {
	t.Helper()

	streamID := id.NewStreamID()
	ref := id.NewStreamRef("Poison", streamID)

	events := []event.Event{
		mustEvent(event.New("poison.created", streamID, "Poison", event.Version(1),
			PoisonCreated{ID: "poison-1", Title: "Original"})),
		mustEvent(event.New("poison.renamed", streamID, "Poison", event.Version(2),
			PoisonRenamed{ID: "poison-1"})),
		mustEvent(event.New("poison.created", streamID, "Poison", event.Version(3),
			PoisonCreated{ID: "healthy-1", Title: "Healthy"})),
	}

	if err := sys.EventStore().Save(ctx, ref, events, event.Version(0)); err != nil {
		t.Fatalf("seed events: %v", err)
	}
}

// waitForPoisonDrain blocks until the healthy projection caught up (the
// post-poison insert landed) and the dead-letter store recorded the poison
// event, or the deadline expires.
func waitForPoisonDrain(
	t *testing.T,
	ctx context.Context,
	sys *system.System,
	dlq *projectionhost.MemoryDeadLetterStore,
) {
	t.Helper()

	deadline := loadScaledDeadline(10 * time.Second)

	for time.Now().Before(deadline) {
		entries, err := dlq.List(ctx, "")
		if err != nil {
			t.Fatalf("dlq.List: %v", err)
		}

		if len(entries) > 0 {
			if _, getErr := system.Get[PoisonView](
				ctx,
				sys,
				"healthy_views",
				"healthy-1",
			); getErr == nil {
				return
			}
		}

		time.Sleep(50 * time.Millisecond)
	}

	t.Fatal("projection host did not drain the poison scenario within timeout")
}

// assertPoisonDLQEntry pins the DLQ entry's shape: the poison event type, the
// Corruption family (non-retryable → dead-lettered instead of retried), and
// the machine-readable reify code from the error chain.
func assertPoisonDLQEntry(
	t *testing.T, ctx context.Context, dlq *projectionhost.MemoryDeadLetterStore,
) {
	t.Helper()

	entries, err := dlq.List(ctx, "")
	if err != nil {
		t.Fatalf("dlq.List: %v", err)
	}

	if len(entries) == 0 {
		t.Fatal("expected at least one dead-letter entry")
	}

	for _, entry := range entries {
		if entry.EventType != "poison.renamed" {
			continue
		}

		if entry.ErrorFamily != "corruption" {
			t.Errorf("ErrorFamily: want corruption, got %q (entry: %+v)", entry.ErrorFamily, entry)
		}

		if entry.ErrorCode != "system.evolution.reify_failed" {
			t.Errorf("ErrorCode: want system.evolution.reify_failed, got %q", entry.ErrorCode)
		}

		return
	}

	t.Fatalf("no dead-letter entry for poison.renamed among %d entries", len(entries))
}

// assertPoisonWorkerSurvived verifies the post-poison state: the poisoned
// collection is marked poisoned (reads refuse), the worker is still active,
// and the healthy collection served the post-poison event.
func assertPoisonWorkerSurvived(t *testing.T, ctx context.Context, sys *system.System) {
	t.Helper()

	if err := sys.MetaEngine().IsPoisoned("poison_views"); err == nil {
		t.Error("poison_views collection should be poisoned after the reify failure")
	}

	for _, state := range sys.ProjectionHost().Status() {
		if state.Status == projectionhost.WorkerFailed {
			t.Errorf("worker %q failed after the poison event: %+v", state.Name, state)
		}
	}

	healthy, err := system.Get[PoisonView](ctx, sys, "healthy_views", "healthy-1")
	if err != nil {
		t.Fatalf("healthy projection read after poison: %v", err)
	}

	if healthy.Title != "Healthy" {
		t.Fatalf("healthy projection: want Title %q, got %+v", "Healthy", healthy)
	}
}
