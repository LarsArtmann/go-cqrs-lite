package projectionadapter_test

import (
	"context"
	"encoding/json/v2"
	"sync"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/projectionadapter/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/projectionhost/v4"
)

// ─── Domain types for the integration test ───

type CounterID string

type CounterIncremented struct {
	ID     CounterID
	Amount int
}

type CounterState struct {
	Value int
}

type FindCounter struct {
	ID CounterID
}

// ─── Minimal journal + checkpoint for the projection host ───

type memoryJournal struct {
	mu     sync.Mutex
	events []event.Event
}

func (j *memoryJournal) ReadAll(ctx context.Context) ([]event.Event, error) {
	j.mu.Lock()
	defer j.mu.Unlock()

	return append([]event.Event{}, j.events...), nil
}

func (j *memoryJournal) ReadFrom(
	_ context.Context,
	after id.EventID,
	limit int,
) ([]event.Event, error) {
	j.mu.Lock()
	defer j.mu.Unlock()

	if limit <= 0 {
		limit = len(j.events)
	}

	if after.IsZero() {
		end := min(limit, len(j.events))

		return append([]event.Event{}, j.events[:end]...), nil
	}

	start := -1
	for i, e := range j.events {
		if e.ID() == after {
			start = i + 1

			break
		}
	}

	if start < 0 {
		return nil, nil
	}

	end := min(start+limit, len(j.events))

	return append([]event.Event{}, j.events[start:end]...), nil
}

func (j *memoryJournal) append(evt event.Event) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.events = append(j.events, evt)
}

type memoryCheckpointStore struct {
	mu   sync.Mutex
	data map[string]event.Checkpoint
}

func newMemoryCheckpointStore() *memoryCheckpointStore {
	return &memoryCheckpointStore{data: make(map[string]event.Checkpoint)}
}

func (s *memoryCheckpointStore) Save(_ context.Context, name string, cp event.Checkpoint) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[name] = cp

	return nil
}

func (s *memoryCheckpointStore) Load(_ context.Context, name string) (event.Checkpoint, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.data[name], nil
}

// ─── Helpers ───

func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}

	return b
}

func waitForProcessed(t *testing.T, host *projectionhost.Host, name string, want int) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)

	for time.Now().Before(deadline) {
		for _, s := range host.Status() {
			if s.Name == name && s.Processed >= int64(want) {
				return
			}
		}
		time.Sleep(20 * time.Millisecond)
	}

	t.Fatalf("timeout waiting for projection %q to process %d events", name, want)
}

// ─── Tests ───

func TestAdapter_ProjectionHostIntegration(t *testing.T) {
	t.Parallel()

	// Build a metaengine Store with a simple Map query that counts increments.
	counterQuery := metaengine.Query[FindCounter, CounterState](
		"counter",
		metaengine.On(CounterIncremented{}, func(e CounterIncremented) (CounterID, CounterState) {
			return e.ID, CounterState{Value: e.Amount}
		}),
		metaengine.On(
			CounterIncremented{},
			func(e CounterIncremented, prev CounterState) CounterState {
				prev.Value += e.Amount

				return prev
			},
		),
	)

	store, err := metaengine.Plan(
		[]metaengine.Engine{metaengine.NewMemoryEngine()},
		counterQuery,
	)
	if err != nil {
		t.Fatalf("metaengine.Plan: %v", err)
	}

	// Verify EventTypes works before any processing.
	types := store.EventTypes()
	if len(types) != 1 || types[0] != "CounterIncremented" {
		t.Fatalf("EventTypes() = %v, want [CounterIncremented]", types)
	}

	// A PayloadDecoder is required because metaengine fold handlers are
	// reflection-based and expect typed structs, not map[string]any.
	decoder := func(eventType string, payload []byte) (any, error) {
		var e CounterIncremented
		if err := json.Unmarshal(payload, &e); err != nil {
			return nil, err
		}

		return e, nil
	}

	adapter := projectionadapter.New("counters", store, decoder)

	if adapter.Name() != "counters" {
		t.Fatalf("Name() = %q, want %q", adapter.Name(), "counters")
	}

	// Create events.
	streamID := id.NewStreamID()

	evt1, err := event.NewEvent(
		"CounterIncremented", streamID, "Counter", event.Version(1),
		mustJSON(t, CounterIncremented{ID: "counter-1", Amount: 5}),
	)
	if err != nil {
		t.Fatalf("event.NewEvent evt1: %v", err)
	}

	evt2, err := event.NewEvent(
		"CounterIncremented", streamID, "Counter", event.Version(2),
		mustJSON(t, CounterIncremented{ID: "counter-1", Amount: 3}),
	)
	if err != nil {
		t.Fatalf("event.NewEvent evt2: %v", err)
	}

	journal := &memoryJournal{}
	journal.append(evt1)
	journal.append(evt2)

	cpStore := newMemoryCheckpointStore()

	// Create and start the projection host.
	host, err := projectionhost.New(
		journal, cpStore,
		projectionhost.WithBatchSize(10),
	)
	if err != nil {
		t.Fatalf("projectionhost.New: %v", err)
	}

	if err := host.Register(adapter); err != nil {
		t.Fatalf("host.Register: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	go func() { _ = host.Start(ctx) }()
	defer func() { _ = host.Stop() }()

	// Wait for the worker to process both events.
	waitForProcessed(t, host, "counters", 2)

	// Verify no errors.
	for _, s := range host.Status() {
		if s.Name == "counters" && s.Errors != 0 {
			t.Fatalf("Errors = %d, want 0", s.Errors)
		}
	}

	// Verify the metaengine store processed the data correctly.
	result, err := store.Execute(FindCounter{ID: "counter-1"})
	if err != nil {
		t.Fatalf("store.Execute: %v", err)
	}

	state, ok := result.(CounterState)
	if !ok {
		t.Fatalf("Execute returned %T, want CounterState", result)
	}

	// 5 + 3 = 8. The first fold inserts {Value: 5}, the second adds 3.
	if state.Value != 8 {
		t.Fatalf("Counter value = %d, want 8", state.Value)
	}
}

func TestAdapter_NameAndTypes(t *testing.T) {
	t.Parallel()

	counterQuery := metaengine.Query[FindCounter, CounterState](
		"counter",
		metaengine.On(CounterIncremented{}, func(e CounterIncremented) (CounterID, CounterState) {
			return e.ID, CounterState{Value: e.Amount}
		}),
	)

	store, err := metaengine.Plan(
		[]metaengine.Engine{metaengine.NewMemoryEngine()},
		counterQuery,
	)
	if err != nil {
		t.Fatalf("metaengine.Plan: %v", err)
	}

	adapter := projectionadapter.New("test-projection", store, nil)

	if adapter.Name() != "test-projection" {
		t.Fatalf("Name() = %q, want %q", adapter.Name(), "test-projection")
	}

	types := adapter.EventTypes()
	if len(types) != 1 || types[0] != event.Type("CounterIncremented") {
		t.Fatalf("EventTypes() = %v, want [CounterIncremented]", types)
	}
}
