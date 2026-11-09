package memory_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4/idtest"
	"github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
)

func TestMemoryStore_SaveAndLoad(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	ctx := context.Background()

	streamID := idtest.ParseStreamID(t, "01HK1540X0841Y0A6BSX1VKR95")
	evt1 := eventtest.QuickEvent("UserCreated", streamID, "User", 1, nil)
	evt2 := eventtest.QuickEvent("UserUpdated", streamID, "User", 1, nil)

	err := store.Save(
		ctx,
		id.NewStreamRef(id.StreamType("User"), streamID),
		[]event.Event{evt1},
		0,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = store.Save(
		ctx,
		id.NewStreamRef(id.StreamType("User"), streamID),
		[]event.Event{evt2},
		1,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	events, err := store.Load(ctx, id.NewStreamRef(id.StreamType("User"), streamID))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	eventtest.AssertLen(t, "events", events, 2)
}

func TestMemoryStore_VersionConflict(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	ctx := context.Background()

	streamID := idtest.ParseStreamID(t, "01HK1540X0841Y0A6BSX1VKR95")
	evt := eventtest.QuickEvent("UserCreated", streamID, "User", 1, nil)

	err := store.Save(
		ctx,
		id.NewStreamRef(id.StreamType("User"), streamID),
		[]event.Event{evt},
		5,
	)
	if err == nil {
		t.Error("expected version conflict error")
	}
}

func TestMemoryStore_StreamNotFound(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	ctx := context.Background()

	_, err := store.Load(
		ctx,
		id.NewStreamRef(
			id.StreamType("User"),
			idtest.ParseStreamID(t, "01HK154KER4E8AJ20Q4JD5TJ1E"),
		),
	)
	if err == nil {
		t.Error("expected stream not found error")
	}
}

func seedTestEvents(
	t *testing.T,
	store *memory.MemoryStore,
	ctx context.Context,
	streamID id.StreamID,
	n int,
) {
	t.Helper()

	for i := range n {
		evt := eventtest.QuickEvent("Created", streamID, "User", event.Version(i+1), nil)
		_ = store.AppendBatch(
			ctx,
			id.NewStreamRef(id.StreamType("User"), streamID),
			[]event.Event{evt},
		)
	}
}
