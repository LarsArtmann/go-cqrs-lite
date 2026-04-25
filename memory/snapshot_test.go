package memory_test

import (
	"context"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	"github.com/larsartmann/go-cqrs-lite/memory"
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

func assertVersion(t *testing.T, got event.Version, want int, msg ...string) {
	t.Helper()

	if got.Int() != want {
		if len(msg) > 0 {
			t.Errorf("version mismatch: %s (got %d, want %d)", msg[0], got.Int(), want)
		} else {
			t.Errorf("expected version %d, got %d", want, got.Int())
		}
	}
}

func TestMemorySnapshotStore_SaveAndLoad(t *testing.T) {
	t.Parallel()

	store := memory.NewMemorySnapshotStore()
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

	assertVersion(t, loaded.Version, 5)

	if string(loaded.State) != `{"status":"shipped"}` {
		t.Errorf("unexpected state: %s", string(loaded.State))
	}
}

func TestMemorySnapshotStore_Load_NotFound(t *testing.T) {
	t.Parallel()

	store := memory.NewMemorySnapshotStore()

	_, err := store.Load(context.Background(), "Order", id.MustParseAggregateID("nonexistent"))
	if err == nil {
		t.Error("expected snapshot not found error")
	}
}

func TestMemorySnapshotStore_Save_IgnoresOlderVersion(t *testing.T) {
	t.Parallel()

	store := memory.NewMemorySnapshotStore()
	ctx := context.Background()

	snapshotV5 := newTestSnapshot(t, id.MustParseAggregateID("order-2"), 5, "shipped")

	err := store.Save(ctx, snapshotV5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	snapshotV3 := newTestSnapshot(t, id.MustParseAggregateID("order-2"), 3, "placed")

	err = store.Save(ctx, snapshotV3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	loaded, err := store.Load(ctx, "Order", id.MustParseAggregateID("order-2"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertVersion(t, loaded.Version, 5, "should not downgrade")
}

func TestMemorySnapshotStore_Save_UpdatesNewerVersion(t *testing.T) {
	t.Parallel()

	store := memory.NewMemorySnapshotStore()
	ctx := context.Background()

	snapshotV3 := newTestSnapshot(t, id.MustParseAggregateID("order-3"), 3, "placed")

	err := store.Save(ctx, snapshotV3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	snapshotV7 := newTestSnapshot(t, id.MustParseAggregateID("order-3"), 7, "delivered")

	err = store.Save(ctx, snapshotV7)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	loaded, err := store.Load(ctx, "Order", id.MustParseAggregateID("order-3"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertVersion(t, loaded.Version, 7)
}

func TestMemorySnapshotStore_LoadAtVersion(t *testing.T) {
	t.Parallel()

	store := memory.NewMemorySnapshotStore()
	ctx := context.Background()

	snapshot := newTestSnapshot(t, id.MustParseAggregateID("order-4"), 5, "shipped")

	err := store.Save(ctx, snapshot)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	t.Run("load at version returns correct version", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name        string
			loadVersion event.Version
			expectVers  event.Version
		}{
			{name: "at exact version", loadVersion: event.Version(5), expectVers: event.Version(5)},
			{
				name:        "after snapshot version",
				loadVersion: event.Version(10),
				expectVers:  event.Version(5),
			},
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

	store := memory.NewMemorySnapshotStore()
	ctx := context.Background()

	snapshot := newTestSnapshot(t, id.MustParseAggregateID("order-5"), 1, "")

	err := store.Save(ctx, snapshot)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = store.Delete(ctx, "Order", id.MustParseAggregateID("order-5"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = store.Load(ctx, "Order", id.MustParseAggregateID("order-5"))
	if err == nil {
		t.Error("expected snapshot not found after delete")
	}
}
