package memory_test

import (
	"context"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4/eventtest"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4/idtest"
	"github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
)

func TestMemoryStore_Closed(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	ctx := context.Background()

	_ = store.Close()

	streamID := idtest.ParseStreamID(t, "01HK1540X0841Y0A6BSX1VKR95")
	evt := eventtest.QuickEvent("UserCreated", streamID, "User", 1, nil)

	err := store.Save(
		ctx,
		id.NewStreamRef(id.StreamType("User"), streamID),
		[]event.Event{evt},
		0,
	)
	if err == nil {
		t.Error("expected store closed error")
	}
}

func TestMemoryStore_ClosedLoad(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	ctx := context.Background()
	_ = store.Close()

	_, err := store.Load(
		ctx,
		id.NewStreamRef(
			id.StreamType("User"),
			idtest.ParseStreamID(t, "01HK1540X0841Y0A6BSX1VKR95"),
		),
	)
	if err == nil {
		t.Error("expected store closed error on Load")
	}
}

func TestMemoryStore_ClosedLoadFromVersion(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	ctx := context.Background()
	_ = store.Close()

	streamID := idtest.ParseStreamID(t, "01HK1540X0841Y0A6BSX1VKR95")

	_, err := store.LoadFromVersion(
		ctx,
		id.NewStreamRef(id.StreamType("User"), streamID),
		0,
	)
	if err == nil {
		t.Error("expected store closed error on LoadFromVersion")
	}
}

func TestMemoryStore_LoadToVersion_Closed(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	_ = store.Close()

	_, err := store.LoadToVersion(
		context.Background(),
		id.NewStreamRef(id.StreamType("User"), id.NewStreamID()),
		1,
	)
	if err == nil {
		t.Fatal("expected error for closed store")
	}
}

func TestMemoryStore_LoadToTimestamp_Closed(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	_ = store.Close()

	_, err := store.LoadToTimestamp(
		context.Background(),
		id.NewStreamRef(id.StreamType("User"), id.NewStreamID()),
		time.Now(),
	)
	if err == nil {
		t.Fatal("expected error for closed store")
	}
}

func TestMemoryStore_ReadFrom_Closed(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	_ = store.Close()

	_, err := store.ReadFrom(context.Background(), id.EventID{}, 10)
	if err == nil {
		t.Fatal("expected error for closed store")
	}
}

func TestMemoryStore_LoadBackwards_Closed(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	_ = store.Close()

	backwardsLoader := event.BackwardsSource(store)
	_, err := backwardsLoader.LoadBackwards(context.Background(), id.StreamRef{})
	if err == nil {
		t.Fatal("expected error for closed store")
	}
}
