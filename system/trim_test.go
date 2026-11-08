package system_test

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/snapshot/v4"
	"github.com/larsartmann/go-cqrs-lite/system/v4"
)

// trimDeployment boots one sqlite engine as source-of-truth + projections.
func trimDeployment(dsn string) system.DeploymentConfig {
	return system.DeploymentConfig{
		Engines: map[string]system.EngineConfig{
			"primary": {Driver: "sqlite", DSN: dsn},
		},
		Instances: []system.InstanceConfig{
			{Role: system.RoleSourceOfTruth, Engine: "primary"},
			{Role: system.RoleProjections, Engine: "primary"},
		},
	}
}

func trimEvent(t *testing.T, streamID id.StreamID, version event.Version, label string) event.Event {
	t.Helper()

	evt, err := event.New(
		event.Type("trim.probe."+label),
		streamID,
		"TrimProbe",
		version,
		map[string]string{"label": label},
	)
	if err != nil {
		t.Fatalf("event.New: %v", err)
	}

	return evt
}

// TestSystem_TrimStream_ErasesStreamAndSnapshot pins the GDPR-erasure
// primitive end-to-end (#118): after TrimStream the journal rows are GONE
// (Load answers stream-not-found, the journal no longer carries them), the
// snapshot is co-erased (a stale snapshot would resurrect trimmed state on
// replay), and the next append at expected version 0 succeeds as a fresh
// stream.
func TestSystem_TrimStream_ErasesStreamAndSnapshot(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dsn := "file:" + filepath.Join(t.TempDir(), "trim.db")

	sys, err := system.New(ctx, system.DomainConfig{}, trimDeployment(dsn))
	if err != nil {
		t.Fatalf("system.New: %v", err)
	}

	defer func() { _ = sys.Close() }()

	streamA := id.NewStreamID()
	refA := id.NewStreamRef("TrimProbe", streamA)
	refB := id.NewStreamRef("TrimProbe", id.NewStreamID())

	if err := sys.EventStore().Save(ctx, refA, []event.Event{
		trimEvent(t, streamA, 1, "a1"),
		trimEvent(t, streamA, 2, "a2"),
	}, 0); err != nil {
		t.Fatalf("save A: %v", err)
	}

	if err := sys.EventStore().Save(ctx, refB, []event.Event{
		trimEvent(t, refB.ID, 1, "b1"),
	}, 0); err != nil {
		t.Fatalf("save B: %v", err)
	}

	if sys.SnapshotStore() != nil {
		if err := sys.SnapshotStore().Save(ctx, snapshot.Snapshot{
			StreamID:   streamA,
			StreamType: "TrimProbe",
			Version:    2,
			State:      []byte(`{"state":"a"}`),
		}); err != nil {
			t.Fatalf("snapshot save A: %v", err)
		}
	}

	removed, err := sys.TrimStream(ctx, refA)
	if err != nil {
		t.Fatalf("TrimStream: %v", err)
	}

	if removed != 2 {
		t.Fatalf("TrimStream removed %d rows, want 2", removed)
	}

	// The adapter maps a missing stream to an EMPTY load (the decider
	// repository's fresh-state contract), so the erased stream's observable
	// shape is zero events — not an error.
	if loaded, err := sys.EventStore().Load(ctx, refA); err != nil || len(loaded) != 0 {
		t.Fatalf("Load after trim: (%d events, %v), want (0, <nil>)", len(loaded), err)
	}

	journalReader, ok := sys.EventStore().(event.Journal)
	if !ok {
		t.Fatal("event store must implement event.Journal")
	}

	journal, err := journalReader.ReadAll(ctx)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}

	for _, evt := range journal {
		if evt.StreamID() == streamA {
			t.Fatalf("journal still carries a trimmed row: %s v%d", evt.Type(), evt.Version())
		}
	}

	if snap := sys.SnapshotStore(); snap != nil {
		if got, err := snap.Load(ctx, refA); err != nil || got != nil {
			t.Fatalf("snapshot load after trim: (%v, %v), want (nil, nil)", got, err)
		}
	}

	// The erased stream accepts a fresh append at version 0 — GDPR erasure
	// must leave a re-usable key, not a poisoned one.
	fresh := trimEvent(t, streamA, 1, "a-reborn")
	if err := sys.EventStore().Save(ctx, refA, []event.Event{fresh}, 0); err != nil {
		t.Fatalf("fresh append after trim: %v", err)
	}

	loaded, err := sys.EventStore().Load(ctx, refA)
	if err != nil {
		t.Fatalf("Load after re-append: %v", err)
	}

	if len(loaded) != 1 || loaded[0].Version() != 1 {
		t.Fatalf("re-appended stream = %d events (want 1, v1): %+v", len(loaded), loaded)
	}
}

// TestSystem_TrimStream_UnsupportedBackendNamesTheClass: the memory engine
// implements StreamTrimmer, so this pins the error SHAPE with a nil-harness —
// instead the capability split is pinned by asserting a memory system's trim
// works while a sweeper-less path errors precisely. The memory boot covers
// the in-process contract; the JournalSweeper capability is sqlite-only.
func TestSystem_TrimStream_MemoryEngineTrims(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	sys, err := system.New(ctx, system.DomainConfig{}, system.DeploymentConfig{
		Engines: map[string]system.EngineConfig{"primary": {Driver: "memory"}},
		Instances: []system.InstanceConfig{
			{Role: system.RoleSourceOfTruth, Engine: "primary"},
		},
	})
	if err != nil {
		t.Fatalf("system.New: %v", err)
	}

	defer func() { _ = sys.Close() }()

	ref := id.NewStreamRef("TrimProbe", id.NewStreamID())

	if err := sys.EventStore().Save(ctx, ref, []event.Event{
		trimEvent(t, ref.ID, 1, "m1"),
		trimEvent(t, ref.ID, 2, "m2"),
	}, 0); err != nil {
		t.Fatalf("save: %v", err)
	}

	removed, err := sys.TrimStream(ctx, ref)
	if err != nil {
		t.Fatalf("TrimStream on memory: %v", err)
	}

	if removed != 2 {
		t.Fatalf("memory TrimStream removed %d, want 2", removed)
	}

	if loaded, err := sys.EventStore().Load(ctx, ref); err != nil || len(loaded) != 0 {
		t.Fatalf("Load after memory trim: (%d events, %v), want (0, <nil>)", len(loaded), err)
	}

	if _, err := sys.TrimJournalBefore(ctx, time.Now()); !errors.Is(err, system.ErrTrimUnavailable) {
		t.Fatalf("TrimJournalBefore on memory: err=%v, want ErrTrimUnavailable", err)
	}
}

// TestSystem_TrimJournalBefore_SweepsOldRowsAndCoTrimsSnapshots pins the
// age-sweep boundary through the system surface: a cutoff in the future
// sweeps every typed row AND the co-trimmed snapshot; a cutoff in the past
// removes nothing. The mixed-age and NULL-timestamp precision cases are
// pinned engine-side (sqliteengine/stream_trim_test.go), where occurred_at
// is directly controllable.
func TestSystem_TrimJournalBefore_SweepsOldRowsAndCoTrimsSnapshots(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dsn := "file:" + filepath.Join(t.TempDir(), "sweep.db")

	sys, err := system.New(ctx, system.DomainConfig{}, trimDeployment(dsn))
	if err != nil {
		t.Fatalf("system.New: %v", err)
	}

	defer func() { _ = sys.Close() }()

	ref := id.NewStreamRef("TrimProbe", id.NewStreamID())

	if err := sys.EventStore().Save(ctx, ref, []event.Event{
		trimEvent(t, ref.ID, 1, "s1"),
		trimEvent(t, ref.ID, 2, "s2"),
	}, 0); err != nil {
		t.Fatalf("save: %v", err)
	}

	if sys.SnapshotStore() != nil {
		if err := sys.SnapshotStore().Save(ctx, snapshot.Snapshot{
			StreamID:   ref.ID,
			StreamType: "TrimProbe",
			Version:    2,
			State:      []byte(`{"state":"s"}`),
		}); err != nil {
			t.Fatalf("snapshot save: %v", err)
		}
	}

	removed, err := sys.TrimJournalBefore(ctx, time.Now().Add(-time.Hour))
	if err != nil {
		t.Fatalf("TrimJournalBefore (past cutoff): %v", err)
	}

	if removed != 0 {
		t.Fatalf("past cutoff removed %d rows, want 0", removed)
	}

	removed, err = sys.TrimJournalBefore(ctx, time.Now().Add(time.Hour))
	if err != nil {
		t.Fatalf("TrimJournalBefore (future cutoff): %v", err)
	}

	if removed != 2 {
		t.Fatalf("future cutoff removed %d rows, want 2", removed)
	}

	if loaded, err := sys.EventStore().Load(ctx, ref); err != nil || len(loaded) != 0 {
		t.Fatalf("Load after sweep: (%d events, %v), want (0, <nil>)", len(loaded), err)
	}

	if snap := sys.SnapshotStore(); snap != nil {
		if got, err := snap.Load(ctx, ref); err != nil || got != nil {
			t.Fatalf("snapshot load after sweep: (%v, %v), want (nil, nil)", got, err)
		}
	}
}
