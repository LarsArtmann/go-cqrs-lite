package memory_test

import (
	"context"
	"sync"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	"github.com/larsartmann/go-cqrs-lite/core/pkg/id"
	"github.com/larsartmann/go-cqrs-lite/memory"
)

func newTestEvent(version int, payload []byte) (event.Event, error) {
	return event.NewEvent(
		event.Type("test.event"),
		id.NewAggregateID(),
		event.AggregateType("Test"),
		event.Version(version),
		payload,
	)
}

func TestConcurrent_BusPublish(t *testing.T) {
	t.Parallel()

	bus := memory.NewMemoryBus()
	var mutex sync.Mutex
	received := 0

	handler := func(_ context.Context, _ event.Event) error {
		mutex.Lock()
		received++
		mutex.Unlock()

		return nil
	}

	err := bus.Subscribe(event.Type("test.event"), handler)
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}

	var wg sync.WaitGroup

	for i := range 100 {
		wg.Go(func() {
			evt, _ := newTestEvent(i+1, []byte(`{"i":0}`))

			_ = bus.Publish(context.Background(), evt)
		})
	}

	wg.Wait()

	mutex.Lock()
	count := received
	mutex.Unlock()

	if count != 100 {
		t.Errorf("expected 100 events, got %d", count)
	}
}

func TestConcurrent_StoreSaveLoad(t *testing.T) {
	t.Parallel()

	store := memory.NewMemoryStore()
	aggType := event.AggregateType("Test")

	var wg sync.WaitGroup

	for range 50 {
		wg.Go(func() {
			aggID := id.NewAggregateID()
			evt, _ := event.NewEvent(
				event.Type("test.event"),
				aggID,
				aggType,
				event.Version(1),
				[]byte(`{}`),
			)

			_ = store.Save(
				context.Background(),
				aggType,
				aggID,
				[]event.Event{evt},
				event.Version(0),
			)
		})
	}

	wg.Wait()
}

func TestConcurrent_OutboxAppendPollAck(t *testing.T) {
	t.Parallel()

	outbox := memory.NewMemoryOutboxStore()
	var wg sync.WaitGroup

	for i := range 20 {
		wg.Go(func() {
			evt, _ := newTestEvent(i+1, []byte(`{}`))

			_ = outbox.Append(context.Background(), []event.Event{evt})
		})
	}

	wg.Wait()

	entries, err := outbox.PollPending(context.Background(), 100)
	if err != nil {
		t.Fatalf("poll: %v", err)
	}

	if len(entries) != 20 {
		t.Errorf("expected 20 entries, got %d", len(entries))
	}

	ids := make([]event.OutboxID, 0, len(entries))
	for _, e := range entries {
		ids = append(ids, e.ID)
	}

	err = outbox.Ack(context.Background(), ids)
	if err != nil {
		t.Fatalf("ack: %v", err)
	}

	remaining, _ := outbox.PollPending(context.Background(), 100)
	if len(remaining) != 0 {
		t.Errorf("expected 0 remaining, got %d", len(remaining))
	}
}

func TestConcurrent_CheckpointSaveLoad(t *testing.T) {
	t.Parallel()

	checkpoint := memory.NewMemoryCheckpointStore()
	var wg sync.WaitGroup

	names := []string{"proj-1", "proj-2", "proj-3", "proj-4", "proj-5"}

	for _, name := range names {
		wg.Add(1)

		go func(n string) {
			defer wg.Done()

			for range 10 {
				eid := id.NewEventID()
				_ = checkpoint.Save(context.Background(), n, eid)
			}
		}(name)
	}

	wg.Wait()

	for _, name := range names {
		result, err := checkpoint.Load(context.Background(), name)
		if err != nil {
			t.Errorf("load %s: %v", name, err)
		}

		if result.IsZero() {
			t.Errorf("expected non-zero checkpoint for %s", name)
		}
	}
}

func TestConcurrent_SnapshotSaveLoad(t *testing.T) {
	t.Parallel()

	store := memory.NewMemorySnapshotStore()
	aggID := id.NewAggregateID()
	aggType := event.AggregateType("Test")

	var wg sync.WaitGroup

	for i := range 10 {
		wg.Add(1)

		go func(version int) {
			defer wg.Done()

			snap := event.Snapshot{
				AggregateID:   aggID,
				AggregateType: aggType,
				Version:       event.Version(version + 1),
				State:         []byte(`{"v":0}`),
			}

			_ = store.Save(context.Background(), snap)
		}(i)
	}

	wg.Wait()

	snap, err := store.Load(context.Background(), aggType, aggID)
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	if snap == nil {
		t.Fatal("expected snapshot, got nil")
	}
}
