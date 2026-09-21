package system_test

import (
	"context"
	"errors"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

// racySaveBackend hides the memory engine's AtomicAppender capability so the
// EventAdapter's fallback tier (neither AtomicAppender nor Transactional) is
// reachable in tests. Embedding the narrow interfaces forwards their methods
// while the wider capabilities stay absent.
type racySaveBackend struct {
	metaengine.Engine
	metaengine.StreamLogBackend
}

func newRacySaveBackend() *racySaveBackend {
	eng := metaengine.NewMemoryEngine()

	return &racySaveBackend{
		Engine:           eng,
		StreamLogBackend: eng.(metaengine.StreamLogBackend),
	}
}

var (
	_ metaengine.Engine           = (*racySaveBackend)(nil)
	_ metaengine.StreamLogBackend = (*racySaveBackend)(nil)
)

// TestEventAdapter_Save_FailsClosedOnRacyBackend pins the fail-closed
// contract: a backend with neither AtomicAppender nor Transactional refuses
// Save instead of silently running the racy check-then-append fallback, and
// the refused save writes nothing.
func TestEventAdapter_Save_FailsClosedOnRacyBackend(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	backend := newRacySaveBackend()

	if _, ok := any(backend).(metaengine.AtomicAppender); ok {
		t.Fatal("racySaveBackend must not implement AtomicAppender")
	}

	if _, ok := any(backend).(metaengine.Transactional); ok {
		t.Fatal("racySaveBackend must not implement Transactional")
	}

	adapter := system.NewEventAdapter(backend, "events")

	ref := id.NewStreamRef("Racy", id.NewStreamID())
	evt := eventtest.NewEvent(t, "racy.event", ref.ID, ref.Type, event.Version(1), nil)

	err := adapter.Save(ctx, ref, []event.Event{evt}, event.Version(0))
	if !errors.Is(err, system.ErrRacySaveRefused) {
		t.Fatalf("expected ErrRacySaveRefused, got %v", err)
	}

	events, readErr := adapter.Load(ctx, ref)
	if readErr != nil {
		t.Fatalf("Load: %v", readErr)
	}

	if len(events) != 0 {
		t.Fatalf("refused Save must not write, got %d events", len(events))
	}
}

// TestEventAdapter_Save_RacyOptInAppends pins the opt-in contract:
// WithRacySave re-enables the single-threaded fallback and its version
// conflict check.
func TestEventAdapter_Save_RacyOptInAppends(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	adapter := system.NewEventAdapter(newRacySaveBackend(), "events", system.WithRacySave())

	ref := id.NewStreamRef("Racy", id.NewStreamID())

	evt := eventtest.NewEvent(t, "racy.event", ref.ID, ref.Type, event.Version(1), nil)
	if err := adapter.Save(ctx, ref, []event.Event{evt}, event.Version(0)); err != nil {
		t.Fatalf("opt-in Save: %v", err)
	}

	stale := eventtest.NewEvent(t, "racy.event", ref.ID, ref.Type, event.Version(2), nil)
	if err := adapter.Save(ctx, ref, []event.Event{stale}, event.Version(0)); !errors.Is(err, event.ErrVersionConflict) {
		t.Fatalf("expected version conflict on stale expectedVersion, got %v", err)
	}

	events, err := adapter.Load(ctx, ref)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(events) != 1 {
		t.Fatalf("expected exactly 1 event, got %d", len(events))
	}
}

// TestSystem_New_RejectsRacySourceOfTruthEngine pins the construction-time
// rejection: an engine without AtomicAppender or Transactional cannot serve
// the source-of-truth role.
func TestSystem_New_RejectsRacySourceOfTruthEngine(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	metaengine.RegisterDriver("racysave-test", func(
		_ context.Context, _ metaengine.DriverConfig,
	) (metaengine.Engine, error) {
		return newRacySaveBackend(), nil
	})

	_, err := system.New(ctx, system.DomainConfig{}, system.DeploymentConfig{
		Engines: map[string]system.EngineConfig{"racy": {Driver: "racysave-test"}},
		Instances: []system.InstanceConfig{
			{Role: system.RoleSourceOfTruth, Engine: "racy"},
		},
	})
	if !errors.Is(err, system.ErrEventSaveNotAtomic) {
		t.Fatalf("expected ErrEventSaveNotAtomic, got %v", err)
	}
}
