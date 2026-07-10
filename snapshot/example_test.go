package snapshot_test

import (
	"context"
	"fmt"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
	"github.com/larsartmann/go-cqrs-lite/snapshot/v3"
)

func ExampleEveryNEvents() {
	strategy, err := snapshot.EveryNEvents(100)
	if err != nil {
		fmt.Println("error:", err)

		return
	}

	fmt.Println(strategy != nil)

	// Output:
	// true
}

func ExampleSnapshotStore() {
	store := newFakeStore()

	aggID := id.NewAggregateID()
	ref := id.NewAggregateRef("User", aggID)
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	snap := snapshot.Snapshot{
		AggregateID:   aggID,
		AggregateType: "User",
		Version:       event.Version(10),
		State:         []byte(`{"name":"Alice"}`),
		CreatedAt:     now,
	}

	err := store.Save(context.Background(), snap)
	if err != nil {
		fmt.Println("error:", err)

		return
	}

	loaded, err := store.Load(context.Background(), ref)
	if err != nil {
		fmt.Println("error:", err)

		return
	}

	fmt.Println(loaded.AggregateType, loaded.Version.Int())

	// Output:
	// User 10
}
