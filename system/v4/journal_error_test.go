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

// failingJournalBackend wraps a memory engine and fails JournalReadAll.
type failingJournalBackend struct {
	metaengine.StreamLogBackend

	failErr error
}

func (b *failingJournalBackend) JournalReadAll(
	_ context.Context, _ string,
) ([]any, error) {
	return nil, b.failErr
}

// compile-time: the wrapper still satisfies the backend the adapter needs.
var _ metaengine.StreamLogBackend = (*failingJournalBackend)(nil)

// TestEventAdapter_ReadFrom_PropagatesLookupError is the regression test for
// silent cursor fallback: when the journal scan needed to resolve a resume
// cursor fails, ReadFrom must return the error instead of silently resuming
// from position 0 (which would replay already-processed events).
func TestEventAdapter_ReadFrom_PropagatesLookupError(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	boom := errors.New("journal scan failed")

	eng := metaengine.NewMemoryEngine().(metaengine.StreamLogBackend)
	backend := &failingJournalBackend{StreamLogBackend: eng, failErr: boom}
	adapter := system.NewEventAdapter(backend, "events")

	unknown := id.NewEventID()

	_, err := adapter.ReadFrom(ctx, unknown, 10)
	if !errors.Is(err, boom) {
		t.Fatalf("expected journal scan error to propagate, got %v", err)
	}
}

// TestEventAdapter_ReadFrom_ZeroCursorSkipsScan pins the cold-start contract:
// a zero afterEventID must not trigger any journal scan at all.
func TestEventAdapter_ReadFrom_ZeroCursorSkipsScan(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	boom := errors.New("journal scan must not run for zero cursor")

	eng := metaengine.NewMemoryEngine().(metaengine.StreamLogBackend)
	backend := &failingJournalBackend{StreamLogBackend: eng, failErr: boom}
	adapter := system.NewEventAdapter(backend, "events")

	if _, err := adapter.ReadFrom(ctx, id.EventID{}, 10); err != nil {
		t.Fatalf("zero cursor must skip the journal scan, got %v", err)
	}
}

// TestSystem_ScreamReport_SurfacesWarnings pins the WARN surfacing contract:
// a memory source-of-truth deployment must expose its WARN+OVERRIDE
// diagnostic through System.ScreamReport() after construction.
func TestSystem_ScreamReport_SurfacesWarnings(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	sys, err := system.New(ctx, system.DomainConfig{}, system.DeploymentConfig{
		Engines: map[string]system.EngineConfig{"primary": {Driver: "memory"}},
		Instances: []system.InstanceConfig{
			{Role: system.RoleSourceOfTruth, Engine: "primary"},
		},
	})
	if err != nil {
		t.Fatalf("system.New: %v", err)
	}
	defer sys.Close()

	report := sys.ScreamReport()
	if report == nil {
		t.Fatal("expected non-nil ScreamReport")
	}

	if !report.HasWarnings() {
		t.Fatal("expected WARN diagnostics on memory source-of-truth to be surfaced")
	}

	for _, d := range report.Diagnostics {
		if d.Tier == system.TierScream {
			t.Fatalf("construction succeeded, but SCREAM diagnostic present: %+v", d)
		}
	}
}

// TestEventAdapter_SeqCacheBounded documents the bounded seq cache: writing
// more distinct event IDs than the cache capacity must not grow memory
// without limit (otter evicts at MaximumSize).
func TestEventAdapter_SeqCacheBounded(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	adapter := system.NewEventAdapter(
		metaengine.NewMemoryEngine().(metaengine.StreamLogBackend), "events",
	)

	ref := id.NewStreamRef("Bounded", id.NewStreamID())

	for i := 1; i <= 32; i++ {
		evt := eventtest.NewEvent(t, "bounded.event", ref.ID, ref.Type, event.Version(i), nil)
		if err := adapter.Save(ctx, ref, []event.Event{evt}, event.Version(i-1)); err != nil {
			t.Fatalf("Save %d: %v", i, err)
		}
	}

	events, err := adapter.ReadAll(ctx)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if len(events) != 32 {
		t.Fatalf("expected 32 events, got %d", len(events))
	}
}
