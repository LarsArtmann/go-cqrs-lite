package pebble

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/cockroachdb/pebble"

	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
	"github.com/larsartmann/go-cqrs-lite/snapshot/v3"
)

func newSnapshotStore(t *testing.T) *SnapshotStore {
	t.Helper()

	dir := t.TempDir()
	db, err := pebble.Open(dir, &pebble.Options{})
	if err != nil {
		t.Fatalf("open pebble: %v", err)
	}

	t.Cleanup(func() { _ = db.Close() })

	store, err := NewSnapshotStore(db, slog.Default())
	if err != nil {
		t.Fatal(err)
	}

	return store
}

func testSnapshot(
	t *testing.T,
	aggID id.AggregateID,
	version int,
	state string,
) snapshot.Snapshot {
	t.Helper()

	return snapshot.Snapshot{
		AggregateID:   aggID,
		AggregateType: "Order",
		Version:       event.Version(version),
		State:         []byte(state),
		CreatedAt:     time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC),
	}
}

func TestSnapshotStore_SaveAndLoad(t *testing.T) {
	t.Parallel()

	store := newSnapshotStore(t)
	ctx := context.Background()
	aggID := id.NewAggregateID()
	ref := id.NewAggregateRef("Order", aggID)
	snap := testSnapshot(t, aggID, 5, `{"status":"shipped"}`)

	if err := store.Save(ctx, snap); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := store.Load(ctx, ref)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if loaded.AggregateID != aggID {
		t.Errorf("AggregateID = %s, want %s", loaded.AggregateID, aggID)
	}

	if loaded.Version.Int() != 5 {
		t.Errorf("Version = %d, want 5", loaded.Version.Int())
	}

	if string(loaded.State) != `{"status":"shipped"}` {
		t.Errorf("State = %q, want {\"status\":\"shipped\"}", string(loaded.State))
	}

	if !loaded.CreatedAt.Equal(snap.CreatedAt) {
		t.Errorf("CreatedAt = %v, want %v", loaded.CreatedAt, snap.CreatedAt)
	}
}

func TestSnapshotStore_Load_NotFound(t *testing.T) {
	t.Parallel()

	store := newSnapshotStore(t)
	ref := id.NewAggregateRef("Order", id.NewAggregateID())

	_, err := store.Load(context.Background(), ref)
	if !errors.Is(err, snapshot.ErrSnapshotNotFound) {
		t.Fatalf("expected ErrSnapshotNotFound, got: %v", err)
	}
}

func TestSnapshotStore_LoadAtVersion_NotFound(t *testing.T) {
	t.Parallel()

	t.Run("no snapshot at all", func(t *testing.T) {
		t.Parallel()

		store := newSnapshotStore(t)
		ref := id.NewAggregateRef("Order", id.NewAggregateID())

		_, err := store.LoadAtVersion(context.Background(), ref, event.Version(10))
		if !errors.Is(err, snapshot.ErrSnapshotNotFound) {
			t.Fatalf("expected ErrSnapshotNotFound, got: %v", err)
		}
	})

	t.Run("snapshot version too high", func(t *testing.T) {
		t.Parallel()

		store := newSnapshotStore(t)
		ctx := context.Background()
		aggID := id.NewAggregateID()
		ref := id.NewAggregateRef("Order", aggID)

		snap := testSnapshot(t, aggID, 5, `{"status":"shipped"}`)
		if err := store.Save(ctx, snap); err != nil {
			t.Fatalf("Save: %v", err)
		}

		_, err := store.LoadAtVersion(ctx, ref, event.Version(3))
		if !errors.Is(err, snapshot.ErrSnapshotNotFound) {
			t.Fatalf("expected ErrSnapshotNotFound for version < snapshot, got: %v", err)
		}
	})
}

func TestSnapshotStore_LoadAtVersion_AtOrAfter(t *testing.T) {
	t.Parallel()

	store := newSnapshotStore(t)
	ctx := context.Background()
	aggID := id.NewAggregateID()
	ref := id.NewAggregateRef("Order", aggID)

	if err := store.Save(ctx, testSnapshot(t, aggID, 5, `{"status":"shipped"}`)); err != nil {
		t.Fatalf("Save: %v", err)
	}

	tests := []struct {
		name        string
		loadVersion event.Version
		wantVersion int
	}{
		{name: "exact version", loadVersion: event.Version(5), wantVersion: 5},
		{name: "after version", loadVersion: event.Version(10), wantVersion: 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			loaded, err := store.LoadAtVersion(ctx, ref, tt.loadVersion)
			if err != nil {
				t.Fatalf("LoadAtVersion: %v", err)
			}

			if loaded.Version.Int() != tt.wantVersion {
				t.Errorf("Version = %d, want %d", loaded.Version.Int(), tt.wantVersion)
			}
		})
	}
}

func TestSnapshotStore_Save_VersionPrecedence(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		first       int
		firstState  string
		second      int
		secondState string
		wantVersion int
		wantState   string
	}{
		{
			name:  "newer overwrites older",
			first: 3, firstState: `{"status":"placed"}`,
			second: 7, secondState: `{"status":"delivered"}`,
			wantVersion: 7, wantState: `{"status":"delivered"}`,
		},
		{
			name:  "older is ignored",
			first: 5, firstState: `{"status":"shipped"}`,
			second: 3, secondState: `{"status":"placed"}`,
			wantVersion: 5, wantState: `{"status":"shipped"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			store := newSnapshotStore(t)
			ctx := context.Background()
			aggID := id.NewAggregateID()
			ref := id.NewAggregateRef("Order", aggID)

			if err := store.Save(ctx, testSnapshot(t, aggID, tt.first, tt.firstState)); err != nil {
				t.Fatalf("Save first: %v", err)
			}

			if err := store.Save(
				ctx,
				testSnapshot(t, aggID, tt.second, tt.secondState),
			); err != nil {
				t.Fatalf("Save second: %v", err)
			}

			loaded, err := store.Load(ctx, ref)
			if err != nil {
				t.Fatalf("Load: %v", err)
			}

			if loaded.Version.Int() != tt.wantVersion {
				t.Errorf("Version = %d, want %d", loaded.Version.Int(), tt.wantVersion)
			}

			if string(loaded.State) != tt.wantState {
				t.Errorf("State = %q, want %q", string(loaded.State), tt.wantState)
			}
		})
	}
}

func TestSnapshotStore_Delete(t *testing.T) {
	t.Parallel()

	store := newSnapshotStore(t)
	ctx := context.Background()
	aggID := id.NewAggregateID()
	ref := id.NewAggregateRef("Order", aggID)

	if err := store.Save(ctx, testSnapshot(t, aggID, 5, `{"status":"shipped"}`)); err != nil {
		t.Fatalf("Save: %v", err)
	}

	if err := store.Delete(ctx, ref); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	_, err := store.Load(ctx, ref)
	if !errors.Is(err, snapshot.ErrSnapshotNotFound) {
		t.Fatalf("expected ErrSnapshotNotFound after Delete, got: %v", err)
	}
}

func TestSnapshotStore_Delete_Idempotent(t *testing.T) {
	t.Parallel()

	store := newSnapshotStore(t)
	ref := id.NewAggregateRef("Order", id.NewAggregateID())

	err := store.Delete(context.Background(), ref)
	if err != nil {
		t.Fatalf("Delete on missing snapshot should be no-op, got: %v", err)
	}
}

func TestSnapshotStore_DistinctAggregates(t *testing.T) {
	t.Parallel()

	store := newSnapshotStore(t)
	ctx := context.Background()
	agg1 := id.NewAggregateID()
	agg2 := id.NewAggregateID()

	if err := store.Save(ctx, testSnapshot(t, agg1, 3, `{"a":1}`)); err != nil {
		t.Fatalf("Save agg1: %v", err)
	}

	if err := store.Save(ctx, testSnapshot(t, agg2, 5, `{"b":2}`)); err != nil {
		t.Fatalf("Save agg2: %v", err)
	}

	loaded1, err := store.Load(ctx, id.NewAggregateRef("Order", agg1))
	if err != nil {
		t.Fatalf("Load agg1: %v", err)
	}

	loaded2, err := store.Load(ctx, id.NewAggregateRef("Order", agg2))
	if err != nil {
		t.Fatalf("Load agg2: %v", err)
	}

	if loaded1.Version.Int() != 3 || loaded2.Version.Int() != 5 {
		t.Errorf("versions = (%d, %d), want (3, 5)", loaded1.Version.Int(), loaded2.Version.Int())
	}
}

func TestSnapshotStore_SharedDB_WithEventStore(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	db, err := pebble.Open(dir, &pebble.Options{})
	if err != nil {
		t.Fatalf("open pebble: %v", err)
	}

	t.Cleanup(func() { _ = db.Close() })

	eventStore, err := NewStore(db, slog.Default())
	if err != nil {
		t.Fatal(err)
	}
	snapStore, err := NewSnapshotStore(db, slog.Default())
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	aggID := id.NewAggregateID()
	ref := id.NewAggregateRef("Issue", aggID)
	cfg := issueStoreConfig()

	evt := cfg.NewTestEvent(t, aggID, 1)
	if err := eventStore.Save(ctx, ref, []event.Event{evt}, event.Version(0)); err != nil {
		t.Fatalf("event Save: %v", err)
	}

	// Same aggregate type + ID but snapshot prefix is disjoint from event prefix.
	snapRef := id.NewAggregateRef("Issue", aggID)
	snap := snapshot.Snapshot{
		AggregateID:   aggID,
		AggregateType: "Issue",
		Version:       event.Version(1),
		State:         []byte(`{"title":"v1"}`),
		CreatedAt:     time.Now(),
	}

	if err := snapStore.Save(ctx, snap); err != nil {
		t.Fatalf("snapshot Save: %v", err)
	}

	loaded, err := snapStore.Load(ctx, snapRef)
	if err != nil {
		t.Fatalf("snapshot Load: %v", err)
	}

	if loaded.Version.Int() != 1 {
		t.Errorf("snapshot Version = %d, want 1", loaded.Version.Int())
	}

	events, err := eventStore.Load(ctx, ref)
	if err != nil {
		t.Fatalf("event Load: %v", err)
	}

	if len(events) != 1 {
		t.Errorf("events = %d, want 1 (snapshot should not pollute event keys)", len(events))
	}
}

func TestSnapshotStore_NewStore_NilDB(t *testing.T) {
	t.Parallel()

	_, err := NewSnapshotStore(nil, slog.Default())
	if !errors.Is(err, ErrNilDatabase) {
		t.Fatalf("expected ErrNilDatabase, got: %v", err)
	}
}

func TestSnapshotStore_Close_NoOp(t *testing.T) {
	t.Parallel()

	store := newSnapshotStore(t)

	if err := store.Close(); err != nil {
		t.Fatalf("Close should be no-op: %v", err)
	}

	// Verify still usable after Close (DB lifetime is caller-owned).
	ctx := context.Background()
	aggID := id.NewAggregateID()

	err := store.Save(ctx, testSnapshot(t, aggID, 1, "{}"))
	if err != nil {
		t.Fatalf("Save after Close: %v", err)
	}
}
