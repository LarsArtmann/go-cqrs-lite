package event_test

import (
	"context"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event"
	"github.com/larsartmann/go-cqrs-lite/pkg/id"
)

func newTestSnapshot(
	tb testing.TB,
	aggregateID id.AggregateID,
	version int,
	status string,
) event.Snapshot {
	tb.Helper()

	return event.Snapshot{
		AggregateID:   aggregateID,
		AggregateType: "Order",
		Version:       event.Version(version),
		State:         []byte(`{"status":"` + status + `"}`),
		CreatedAt:     time.Now(),
	}
}

func TestMemorySnapshotStore_SaveAndLoad(t *testing.T) {
	t.Parallel()

	store := event.NewMemorySnapshotStore()
	ctx := context.Background()

	snapshot := newTestSnapshot(t, id.MustParseAggregateID("order-1"), 5, "shipped")

	err := store.Save(ctx, snapshot)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	loaded, err := store.Load(ctx, "Order", id.MustParseAggregateID("order-1"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if loaded.AggregateID != snapshot.AggregateID {
		t.Errorf("expected aggregate ID %s, got %s", snapshot.AggregateID, loaded.AggregateID)
	}

	if loaded.Version.Int() != 5 {
		t.Errorf("expected version 5, got %d", loaded.Version.Int())
	}

	if string(loaded.State) != `{"status":"shipped"}` {
		t.Errorf("unexpected state: %s", string(loaded.State))
	}
}

func TestMemorySnapshotStore_Load_NotFound(t *testing.T) {
	t.Parallel()

	store := event.NewMemorySnapshotStore()

	_, err := store.Load(context.Background(), "Order", id.MustParseAggregateID("nonexistent"))
	if err == nil {
		t.Error("expected snapshot not found error")
	}
}

func TestMemorySnapshotStore_Save_IgnoresOlderVersion(t *testing.T) {
	t.Parallel()

	store := event.NewMemorySnapshotStore()
	ctx := context.Background()

	snapshotV5 := newTestSnapshot(t, id.MustParseAggregateID("order-2"), 5, "shipped")

	if err := store.Save(ctx, snapshotV5); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	snapshotV3 := newTestSnapshot(t, id.MustParseAggregateID("order-2"), 3, "placed")

	if err := store.Save(ctx, snapshotV3); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	loaded, err := store.Load(ctx, "Order", id.MustParseAggregateID("order-2"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if loaded.Version.Int() != 5 {
		t.Errorf("expected version 5, got %d (should not downgrade)", loaded.Version.Int())
	}
}

func TestMemorySnapshotStore_Save_UpdatesNewerVersion(t *testing.T) {
	t.Parallel()

	store := event.NewMemorySnapshotStore()
	ctx := context.Background()

	snapshotV3 := newTestSnapshot(t, id.MustParseAggregateID("order-3"), 3, "placed")

	if err := store.Save(ctx, snapshotV3); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	snapshotV7 := newTestSnapshot(t, id.MustParseAggregateID("order-3"), 7, "delivered")

	if err := store.Save(ctx, snapshotV7); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	loaded, err := store.Load(ctx, "Order", id.MustParseAggregateID("order-3"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if loaded.Version.Int() != 7 {
		t.Errorf("expected version 7, got %d", loaded.Version.Int())
	}
}

func TestMemorySnapshotStore_LoadAtVersion(t *testing.T) {
	t.Parallel()

	store := event.NewMemorySnapshotStore()
	ctx := context.Background()

	snapshot := newTestSnapshot(t, id.MustParseAggregateID("order-4"), 5, "shipped")

	err := store.Save(ctx, snapshot)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	t.Run("load at version returns correct version", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name         string
			loadVersion  event.Version
			expectVers  event.Version
		}{
			{name: "at exact version", loadVersion: event.Version(5), expectVers: event.Version(5)},
			{name: "after snapshot version", loadVersion: event.Version(10), expectVers: event.Version(5)},
		}

		orderID := id.MustParseAggregateID("order-4")
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				loaded, err := store.LoadAtVersion(ctx, "Order", orderID, tt.loadVersion)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				if loaded.Version.Int() != int(tt.expectVers) {
					t.Errorf("expected version %d, got %d", tt.expectVers, loaded.Version.Int())
				}
			})
		}
	})

	t.Run("before snapshot version", func(t *testing.T) {
		t.Parallel()

		_, err := store.LoadAtVersion(
			ctx,
			"Order",
			id.MustParseAggregateID("order-4"),
			event.Version(3),
		)
		if err == nil {
			t.Error("expected snapshot not found for version before snapshot")
		}
	})
}

func TestMemorySnapshotStore_Delete(t *testing.T) {
	t.Parallel()

	store := event.NewMemorySnapshotStore()
	ctx := context.Background()

	snapshot := newTestSnapshot(t, id.MustParseAggregateID("order-5"), 1, "")

	if err := store.Save(ctx, snapshot); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := store.Delete(ctx, "Order", id.MustParseAggregateID("order-5")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err := store.Load(ctx, "Order", id.MustParseAggregateID("order-5"))
	if err == nil {
		t.Error("expected snapshot not found after delete")
	}
}
