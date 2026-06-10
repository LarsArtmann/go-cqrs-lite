package memory_test

import (
	"context"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
	"github.com/larsartmann/go-cqrs-lite/memory/v2"
	"github.com/larsartmann/go-cqrs-lite/snapshot/v2"
)

func parseAggID(s string) id.AggregateID {
	v, err := id.ParseAggregateID(s)
	if err != nil {
		panic(err)
	}
	return v
}

func newTestSnapshot(
	tb testing.TB,
	aggregateID id.AggregateID,
	version int,
	status string,
) snapshot.Snapshot {
	tb.Helper()

	return snapshot.Snapshot{
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

	aggID := parseAggID("01HK1540X0841Y0A6BSX1VKR95")
	snapshot := newTestSnapshot(t, aggID, 5, "shipped")

	err := store.Save(ctx, snapshot)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	loaded, err := store.Load(ctx, event.NewAggregateRef(event.AggregateType("Order"), aggID))
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

	aggID := parseAggID("01HK154ME034FVHK95R554AKSE")

	_, err := store.Load(
		context.Background(),
		event.NewAggregateRef(event.AggregateType("Order"), aggID),
	)
	if err == nil {
		t.Error("expected snapshot not found error")
	}
}

func TestMemorySnapshotStore_Save_IgnoresOlderVersion(t *testing.T) {
	t.Parallel()

	store := memory.NewMemorySnapshotStore()
	ctx := context.Background()

	aggID := parseAggID("01HK154V8RH53JQZ4XRXR7XYJB")
	snapshotV5 := newTestSnapshot(t, aggID, 5, "shipped")

	err := store.Save(ctx, snapshotV5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	snapshotV3 := newTestSnapshot(t, aggID, 3, "placed")

	err = store.Save(ctx, snapshotV3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	loaded, err := store.Load(ctx, event.NewAggregateRef(event.AggregateType("Order"), aggID))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertVersion(t, loaded.Version, 5, "should not downgrade")
}

func TestMemorySnapshotStore_Save_UpdatesNewerVersion(t *testing.T) {
	t.Parallel()

	store := memory.NewMemorySnapshotStore()
	ctx := context.Background()

	aggID := parseAggID("01HK154W80KZSKN04HJMMDCJDW")
	snapshotV3 := newTestSnapshot(t, aggID, 3, "placed")

	err := store.Save(ctx, snapshotV3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	snapshotV7 := newTestSnapshot(t, aggID, 7, "delivered")

	err = store.Save(ctx, snapshotV7)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	loaded, err := store.Load(ctx, event.NewAggregateRef(event.AggregateType("Order"), aggID))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertVersion(t, loaded.Version, 7)
}

func TestMemorySnapshotStore_LoadAtVersion(t *testing.T) {
	t.Parallel()

	store := memory.NewMemorySnapshotStore()
	ctx := context.Background()

	aggID := parseAggID("01HK154X784RCKJT5QZC6MNJTS")
	snapshot := newTestSnapshot(t, aggID, 5, "shipped")

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
			{
				name:        "at exact version",
				loadVersion: event.Version(5),
				expectVers:  event.Version(5),
			},
			{
				name:        "after snapshot version",
				loadVersion: event.Version(10),
				expectVers:  event.Version(5),
			},
		}

		orderID := parseAggID("01HK154X784RCKJT5QZC6MNJTS")

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				loaded, err := store.LoadAtVersion(
					ctx,
					event.NewAggregateRef(event.AggregateType("Order"), orderID),
					tt.loadVersion,
				)
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
			event.NewAggregateRef(event.AggregateType("Order"), aggID),
			event.Version(3),
		)
		if err == nil {
			t.Error("expected snapshot not found for version before snapshot")
		}
	})
}

func TestMemorySnapshotStore_Save_Closed(t *testing.T) {
	t.Parallel()

	store := memory.NewMemorySnapshotStore()
	_ = store.Close()

	snapshot := newTestSnapshot(t, id.NewAggregateID(), 1, "test")

	err := store.Save(context.Background(), snapshot)
	if err == nil {
		t.Error("expected error when saving to closed store")
	}
}

func TestMemorySnapshotStore_Load_Closed(t *testing.T) {
	t.Parallel()

	store := memory.NewMemorySnapshotStore()
	_ = store.Close()

	_, err := store.Load(
		context.Background(),
		event.NewAggregateRef(event.AggregateType("Order"), id.NewAggregateID()),
	)
	if err == nil {
		t.Error("expected error when loading from closed store")
	}
}

func TestMemorySnapshotStore_LoadAtVersion_Closed(t *testing.T) {
	t.Parallel()

	store := memory.NewMemorySnapshotStore()
	_ = store.Close()

	_, err := store.LoadAtVersion(
		context.Background(),
		event.NewAggregateRef(event.AggregateType("Order"), id.NewAggregateID()),
		1,
	)
	if err == nil {
		t.Error("expected error when loading from closed store")
	}
}

func TestMemorySnapshotStore_Delete_Closed(t *testing.T) {
	t.Parallel()

	store := memory.NewMemorySnapshotStore()
	_ = store.Close()

	err := store.Delete(
		context.Background(),
		event.NewAggregateRef(event.AggregateType("Order"), id.NewAggregateID()),
	)
	if err == nil {
		t.Error("expected error when deleting from closed store")
	}
}

func TestMemorySnapshotStore_Delete(t *testing.T) {
	t.Parallel()

	store := memory.NewMemorySnapshotStore()
	ctx := context.Background()

	aggID := parseAggID("01HK154V8RH53JQZ4XRXR7XYJB")
	snapshot := newTestSnapshot(t, aggID, 1, "")

	err := store.Save(ctx, snapshot)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = store.Delete(ctx, event.NewAggregateRef(event.AggregateType("Order"), aggID))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = store.Load(ctx, event.NewAggregateRef(event.AggregateType("Order"), aggID))
	if err == nil {
		t.Error("expected snapshot not found after delete")
	}
}

func TestMemorySnapshotStore_Load_DeepCopy(t *testing.T) {
	t.Parallel()

	store := memory.NewMemorySnapshotStore()
	ctx := context.Background()

	aggID := id.NewAggregateID()
	originalState := []byte(`{"status":"placed"}`)
	snapshot := snapshot.Snapshot{
		AggregateID:   aggID,
		AggregateType: "Order",
		Version:       1,
		State:         originalState,
		CreatedAt:     time.Now(),
	}

	err := store.Save(ctx, snapshot)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	loaded, err := store.Load(ctx, event.NewAggregateRef(event.AggregateType("Order"), aggID))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Modify the loaded snapshot's state
	loaded.State[10] = 'x'

	// Reload and verify original is unchanged
	reloaded, err := store.Load(ctx, event.NewAggregateRef(event.AggregateType("Order"), aggID))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if string(reloaded.State) != string(originalState) {
		t.Errorf(
			"original state corrupted after modifying loaded copy: got %q, want %q",
			reloaded.State,
			originalState,
		)
	}
}

func TestMemorySnapshotStore_Load_NilState(t *testing.T) {
	t.Parallel()

	store := memory.NewMemorySnapshotStore()
	ctx := context.Background()

	aggID := id.NewAggregateID()
	snapshot := snapshot.Snapshot{
		AggregateID:   aggID,
		AggregateType: "Order",
		Version:       1,
		State:         nil,
		CreatedAt:     time.Now(),
	}

	err := store.Save(ctx, snapshot)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	loaded, err := store.Load(ctx, event.NewAggregateRef(event.AggregateType("Order"), aggID))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if loaded.State != nil {
		t.Errorf("expected nil state, got %v", loaded.State)
	}
}

func TestMemorySnapshotStore_LoadAtVersion_DeepCopy(t *testing.T) {
	t.Parallel()

	store := memory.NewMemorySnapshotStore()
	ctx := context.Background()

	aggID := id.NewAggregateID()
	originalState := []byte(`{"status":"shipped"}`)
	snapshot := snapshot.Snapshot{
		AggregateID:   aggID,
		AggregateType: "Order",
		Version:       5,
		State:         originalState,
		CreatedAt:     time.Now(),
	}

	err := store.Save(ctx, snapshot)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	loaded, err := store.LoadAtVersion(
		ctx,
		event.NewAggregateRef(event.AggregateType("Order"), aggID),
		5,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Modify the loaded snapshot's state
	loaded.State[10] = 'x'

	// Reload and verify original is unchanged
	reloaded, err := store.LoadAtVersion(
		ctx,
		event.NewAggregateRef(event.AggregateType("Order"), aggID),
		5,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if string(reloaded.State) != string(originalState) {
		t.Errorf(
			"original state corrupted after modifying loaded copy: got %q, want %q",
			reloaded.State,
			originalState,
		)
	}
}
