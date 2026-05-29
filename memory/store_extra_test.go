package memory_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	"github.com/larsartmann/go-cqrs-lite/memory"
	"github.com/larsartmann/go-cqrs-lite/testhelpers"
)

func TestMemoryCheckpointStore_Close(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryCheckpointStore()

	err := store.Close()
	if err != nil {
		t.Fatalf("close: %v", err)
	}
}

func TestMemoryOutboxStore_Close(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryOutboxStore()

	err := store.Close()
	if err != nil {
		t.Fatalf("close: %v", err)
	}
}

func TestMemoryStore_ReadFrom(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	ctx := context.Background()

	aggID1 := id.NewAggregateID()
	aggID2 := id.NewAggregateID()

	evt1, evt2, evt3 := testhelpers.MakeThreeTimelineEvents(t, "User", aggID1, "Order", aggID2)

	_ = store.AppendBatch(ctx, event.NewAggregateRef(event.AggregateType("User"), aggID1), evt1)
	_ = store.AppendBatch(ctx, event.NewAggregateRef(event.AggregateType("Order"), aggID2), evt2)
	_ = store.AppendBatch(ctx, event.NewAggregateRef(event.AggregateType("User"), aggID1), evt3)

	all, err := store.ReadAll(ctx)
	testhelpers.AssertNoError(t, err, "ReadAll")
	testhelpers.AssertLen(t, "all events", all, 3)

	fromPos, err := store.ReadFrom(ctx, evt1[0].ID(), 1)
	testhelpers.AssertNoError(t, err, "ReadFrom")
	testhelpers.AssertLen(t, "from position", fromPos, 1)
}
