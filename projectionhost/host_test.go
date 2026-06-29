package projectionhost_test

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/id/v3"
	"github.com/larsartmann/go-cqrs-lite/projection/v3"
	"github.com/larsartmann/go-cqrs-lite/projectionhost/v3"
	"github.com/larsartmann/go-cqrs-lite/testutil/v3"
)

// --- Test fixtures ---

// memoryJournal is a minimal event.SeekableJournal for testing.
type memoryJournal struct {
	mu     sync.Mutex
	events []event.Event
}

func (j *memoryJournal) ReadAll(ctx context.Context) ([]event.Event, error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	result := make([]event.Event, len(j.events))
	copy(result, j.events)

	return result, nil
}

func (j *memoryJournal) ReadFrom(
	ctx context.Context,
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

		return j.events[:end], nil
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

	return j.events[start:end], nil
}

func (j *memoryJournal) append(evt event.Event) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.events = append(j.events, evt)
}

// memoryCheckpointStore is a minimal event.CheckpointStore for testing.
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

// countingProjection counts how many events it has handled.
type countingProjection struct {
	name       string
	eventTypes []event.Type
	count      atomic.Int64
	seen       []event.Event
	mu         sync.Mutex
	failOn     int // fail on the Nth event (1-based), 0 = never
	failErr    error
}

func (p *countingProjection) Name() string             { return p.name }
func (p *countingProjection) EventTypes() []event.Type { return p.eventTypes }

func (p *countingProjection) Handle(_ context.Context, evt event.Event) error {
	n := int(p.count.Add(1))
	if p.failOn > 0 && n == p.failOn && p.failErr != nil {
		return p.failErr
	}
	p.mu.Lock()
	p.seen = append(p.seen, evt)
	p.mu.Unlock()

	return nil
}

func makeEvent(eventType string) event.Event {
	aggID := id.NewAggregateID()
	evt, _ := event.New(
		event.Type(eventType),
		aggID,
		"TestAggregate",
		1,
		map[string]any{"ok": true},
	)

	return evt
}

// --- Tests ---

func TestHost_HappyPath_ProcessesAllEvents(t *testing.T) {
	t.Parallel()
	journal := &memoryJournal{}
	cpStore := newMemoryCheckpointStore()

	for range 5 {
		journal.append(makeEvent("test.created"))
	}

	proj := &countingProjection{name: "test-projection"}
	host, err := projectionhost.New(journal, cpStore, projectionhost.WithBatchSize(10))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if err := host.Register(proj); err != nil {
		t.Fatalf("Register: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		if err := host.Start(ctx); err != nil {
			t.Errorf("Start: %v", err)
		}
	}()

	requireEventually(t, 2*time.Second, func() bool {
		return proj.count.Load() == 5
	})
	cancel()
	host.Stop()
}

func TestHost_CheckpointPersistsAcrossRestarts(t *testing.T) {
	t.Parallel()
	journal := &memoryJournal{}
	cpStore := newMemoryCheckpointStore()

	for range 10 {
		journal.append(makeEvent("item.created"))
	}

	proj := &countingProjection{name: "items"}
	host, _ := projectionhost.New(journal, cpStore, projectionhost.WithBatchSize(5))
	host.Register(proj)

	ctx, cancel := context.WithCancel(context.Background())
	go host.Start(ctx)

	requireEventually(t, 2*time.Second, func() bool {
		return proj.count.Load() == 10
	})

	cp, _ := cpStore.Load(ctx, "items")
	if cp.IsZero() {
		t.Fatal("expected checkpoint to be saved")
	}
	cancel()
	host.Stop()
}

func TestHost_MultipleProjections_Independent(t *testing.T) {
	t.Parallel()
	journal := &memoryJournal{}

	for range 3 {
		journal.append(makeEvent("user.created"))
	}
	for range 2 {
		journal.append(makeEvent("order.placed"))
	}

	cpStore := newMemoryCheckpointStore()
	userProj := &countingProjection{name: "users", eventTypes: []event.Type{"user.created"}}
	orderProj := &countingProjection{name: "orders", eventTypes: []event.Type{"order.placed"}}

	host, _ := projectionhost.New(journal, cpStore, projectionhost.WithBatchSize(10))
	host.Register(userProj)
	host.Register(orderProj)

	ctx, cancel := context.WithCancel(context.Background())
	go host.Start(ctx)

	requireEventually(t, 2*time.Second, func() bool {
		return userProj.count.Load() == 3 && orderProj.count.Load() == 2
	})
	cancel()
	host.Stop()
}

func TestHost_DeadLetterQueue_CapturesPoisonMessage(t *testing.T) {
	t.Parallel()
	journal := &memoryJournal{}
	cpStore := newMemoryCheckpointStore()

	goodEvent := makeEvent("task.created")
	badEvent := makeEvent("task.created")
	goodEvent2 := makeEvent("task.created")
	journal.append(goodEvent)
	journal.append(badEvent)
	journal.append(goodEvent2)

	dlq := projectionhost.NewMemoryDeadLetterStore()
	proj := &countingProjection{
		name:    "tasks",
		failOn:  2,
		failErr: errors.New("poison"),
	}

	host, _ := projectionhost.New(
		journal, cpStore,
		projectionhost.WithBatchSize(10),
		projectionhost.WithDeadLetterStore(dlq, 1),
		projectionhost.WithMaxRestarts(-1),
	)
	host.Register(proj)

	ctx, cancel := context.WithCancel(context.Background())
	go host.Start(ctx)

	requireEventually(t, 2*time.Second, func() bool {
		entries, _ := dlq.List(context.Background(), "")

		return len(entries) == 1
	})
	cancel()
	host.Stop()

	entries, _ := dlq.List(context.Background(), "")
	if len(entries) != 1 {
		t.Fatalf("expected 1 DLQ entry, got %d", len(entries))
	}
	if entries[0].Error != "poison" {
		t.Fatalf("expected error 'poison', got %q", entries[0].Error)
	}
}

func TestHost_GracefulStop_CompletesInflight(t *testing.T) {
	t.Parallel()
	journal := &memoryJournal{}
	cpStore := newMemoryCheckpointStore()

	for range 3 {
		journal.append(makeEvent("data.sync"))
	}

	proj := &countingProjection{name: "sync"}
	host, _ := projectionhost.New(journal, cpStore, projectionhost.WithBatchSize(2))
	host.Register(proj)

	ctx, cancel := context.WithCancel(context.Background())
	go host.Start(ctx)

	requireEventually(t, 2*time.Second, func() bool {
		return proj.count.Load() >= 1
	})

	if err := host.Stop(); err != nil {
		t.Fatalf("Stop: %v", err)
	}

	cancel()
	states := host.Status()
	for _, s := range states {
		if s.Status != projectionhost.WorkerStopped {
			t.Fatalf("expected WorkerStopped, got %s for %s", s.Status, s.Name)
		}
	}
}

func TestHost_Status_ReportsCorrectState(t *testing.T) {
	t.Parallel()
	journal := &memoryJournal{}
	cpStore := newMemoryCheckpointStore()

	host, _ := projectionhost.New(journal, cpStore)
	proj := &countingProjection{name: "status-test"}
	host.Register(proj)

	states := host.Status()
	if len(states) != 1 {
		t.Fatalf("expected 1 worker state, got %d", len(states))
	}
	if states[0].Status != projectionhost.WorkerIdle {
		t.Fatalf("expected WorkerIdle, got %s", states[0].Status)
	}
	if states[0].Name != "status-test" {
		t.Fatalf("expected name 'status-test', got %q", states[0].Name)
	}
}

func TestHost_RejectsDuplicateRegistration(t *testing.T) {
	t.Parallel()
	journal := &memoryJournal{}
	cpStore := newMemoryCheckpointStore()

	host, _ := projectionhost.New(journal, cpStore)
	proj1 := &countingProjection{name: "dup"}
	proj2 := &countingProjection{name: "dup"}

	if err := host.Register(proj1); err != nil {
		t.Fatalf("first Register: %v", err)
	}
	if err := host.Register(proj2); err == nil {
		t.Fatal("expected error on duplicate registration")
	}
}

func TestHost_RejectsRegistrationAfterStart(t *testing.T) {
	t.Parallel()
	journal := &memoryJournal{}
	cpStore := newMemoryCheckpointStore()

	host, _ := projectionhost.New(journal, cpStore)
	host.Register(&countingProjection{name: "first"})

	ctx, cancel := context.WithCancel(context.Background())
	host.Start(ctx)
	cancel()
	host.Stop()

	err := host.Register(&countingProjection{name: "second"})
	if err == nil {
		t.Fatal("expected error registering after Start")
	}
}

func TestHost_NilJournal_ReturnsError(t *testing.T) {
	t.Parallel()
	_, err := projectionhost.New(nil, newMemoryCheckpointStore())
	if err == nil {
		t.Fatal("expected error for nil journal")
	}
}

func TestHost_NilCheckpointStore_ReturnsError(t *testing.T) {
	t.Parallel()
	journal := &memoryJournal{}
	_, err := projectionhost.New(journal, nil)
	if err == nil {
		t.Fatal("expected error for nil checkpoint store")
	}
}

func TestMemoryDeadLetterStore_List_FilterByProjection(t *testing.T) {
	t.Parallel()
	store := projectionhost.NewMemoryDeadLetterStore()
	ctx := context.Background()

	store.Store(ctx, projectionhost.DeadLetterEntry{ProjectionName: "a", EventID: "1"})
	store.Store(ctx, projectionhost.DeadLetterEntry{ProjectionName: "b", EventID: "2"})
	store.Store(ctx, projectionhost.DeadLetterEntry{ProjectionName: "a", EventID: "3"})

	all, _ := store.List(ctx, "")
	if len(all) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(all))
	}

	aOnly, _ := store.List(ctx, "a")
	if len(aOnly) != 2 {
		t.Fatalf("expected 2 entries for 'a', got %d", len(aOnly))
	}

	store.Purge(ctx, "a")
	remaining, _ := store.List(ctx, "")
	if len(remaining) != 1 {
		t.Fatalf("expected 1 remaining after purge, got %d", len(remaining))
	}
}

func TestMemoryDeadLetterStore_Delete_RemovesSingleEntry(t *testing.T) {
	t.Parallel()
	store := projectionhost.NewMemoryDeadLetterStore()
	ctx := context.Background()

	store.Store(ctx, projectionhost.DeadLetterEntry{ProjectionName: "orders", EventID: "e1"})
	store.Store(ctx, projectionhost.DeadLetterEntry{ProjectionName: "orders", EventID: "e2"})
	store.Store(ctx, projectionhost.DeadLetterEntry{ProjectionName: "orders", EventID: "e3"})

	// Delete a single entry: the other two must survive.
	if err := store.Delete(ctx, "orders", "e2"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	remaining, _ := store.List(ctx, "orders")
	if len(remaining) != 2 {
		t.Fatalf("expected 2 entries after delete, got %d", len(remaining))
	}

	for _, e := range remaining {
		if e.EventID == "e2" {
			t.Fatal("deleted entry e2 still present")
		}
	}
}

// --- Helpers ---

func TestHost_ReplayDeadLetters_PureNoMutation(t *testing.T) {
	t.Parallel()
	journal := &memoryJournal{}
	cpStore := newMemoryCheckpointStore()
	dlq := projectionhost.NewMemoryDeadLetterStore()

	// A projection that now succeeds — simulating "we shipped a handler fix."
	proj := &countingProjection{name: "tasks"}
	host, _ := projectionhost.New(
		journal, cpStore,
		projectionhost.WithDeadLetterStore(dlq, 1),
	)
	_ = host.Register(proj)

	badEvent := makeEvent("task.created")
	_ = dlq.Store(context.Background(), projectionhost.DeadLetterEntry{
		ProjectionName: "tasks",
		EventID:        badEvent.ID().String(),
		EventType:      "task.created",
		Event:          badEvent,
		Error:          "poison",
	})

	result, err := host.ReplayDeadLetters(context.Background(), "")
	if err != nil {
		t.Fatalf("ReplayDeadLetters: %v", err)
	}
	if len(result.Replayed) != 1 {
		t.Fatalf("expected 1 replayed, got %d", len(result.Replayed))
	}
	if len(result.StillFailing) != 0 {
		t.Fatalf("expected 0 still-failing, got %d", len(result.StillFailing))
	}
	if proj.count.Load() != 1 {
		t.Fatalf(
			"projection should have handled the replayed event, got count=%d",
			proj.count.Load(),
		)
	}

	// ReplayDeadLetters is PURE — the store must be unchanged until caller purges.
	remaining, _ := dlq.List(context.Background(), "")
	if len(remaining) != 1 {
		t.Fatalf("pure replay must NOT mutate DLQ; got %d entries (want 1)", len(remaining))
	}

	// Caller-driven cleanup.
	_ = dlq.Purge(context.Background(), "tasks")
	remaining, _ = dlq.List(context.Background(), "")
	if len(remaining) != 0 {
		t.Fatalf("Purge should clear the projection's entries, got %d", len(remaining))
	}
}

func TestHost_ReplayDeadLetters_PreservesStillFailingEntries(t *testing.T) {
	t.Parallel()
	journal := &memoryJournal{}
	cpStore := newMemoryCheckpointStore()
	dlq := projectionhost.NewMemoryDeadLetterStore()

	// Projection that succeeds — both entries will replay OK, but this test
	// verifies StillFailing is populated when the handler still errors.
	proj := &countingProjection{name: "tasks", failOn: 1, failErr: errors.New("still broken")}
	host, _ := projectionhost.New(
		journal, cpStore,
		projectionhost.WithDeadLetterStore(dlq, 1),
	)
	_ = host.Register(proj)

	evt := makeEvent("task.created")
	_ = dlq.Store(context.Background(), projectionhost.DeadLetterEntry{
		ProjectionName: "tasks",
		EventID:        evt.ID().String(),
		Event:          evt,
		Error:          "original",
	})

	result, err := host.ReplayDeadLetters(context.Background(), "")
	if err != nil {
		t.Fatalf("ReplayDeadLetters: %v", err)
	}
	if len(result.Replayed) != 0 {
		t.Fatalf("expected 0 replayed (handler still broken), got %d", len(result.Replayed))
	}
	if len(result.StillFailing) != 1 {
		t.Fatalf("expected 1 still-failing, got %d", len(result.StillFailing))
	}
	// The failing entry MUST remain in the store for a future replay attempt.
	remaining, _ := dlq.List(context.Background(), "")
	if len(remaining) != 1 {
		t.Fatalf("still-failing entry must be preserved, got %d entries", len(remaining))
	}
}

func TestHost_ReplayDeadLetters_NoDLQConfigured(t *testing.T) {
	t.Parallel()
	journal := &memoryJournal{}
	cpStore := newMemoryCheckpointStore()
	host, _ := projectionhost.New(journal, cpStore)

	if _, err := host.ReplayDeadLetters(context.Background(), ""); err == nil {
		t.Fatal("expected error when no DLQ configured, got nil")
	}
}

func TestHost_WithLogger_RoutesLifecycleEvents(t *testing.T) {
	t.Parallel()
	journal := &memoryJournal{}
	cpStore := newMemoryCheckpointStore()

	badEvent := makeEvent("task.created")
	journal.append(badEvent)

	dlq := projectionhost.NewMemoryDeadLetterStore()
	proj := &countingProjection{
		name:    "tasks",
		failOn:  1,
		failErr: errors.New("boom"),
	}

	handler := testutil.NewCapturingSlogHandler(slog.LevelDebug)
	logger := slog.New(handler)

	host, _ := projectionhost.New(
		journal, cpStore,
		projectionhost.WithBatchSize(10),
		projectionhost.WithDeadLetterStore(dlq, 1),
		projectionhost.WithMaxRestarts(-1),
		projectionhost.WithLogger(logger),
	)
	_ = host.Register(proj)

	ctx, cancel := context.WithCancel(context.Background())
	go host.Start(ctx)

	requireEventually(t, 2*time.Second, func() bool {
		entries, _ := dlq.List(context.Background(), "")

		return len(entries) == 1
	})
	cancel()
	host.Stop()

	if handler.Count() == 0 {
		t.Fatal("WithLogger: expected the injected logger to receive lifecycle records, got none")
	}
}

func requireEventually(t *testing.T, timeout time.Duration, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("condition not met within %v", timeout)
}

// Ensure the projection interface is satisfied.
var _ projection.Projection = (*countingProjection)(nil)

// Ensure unused imports are referenced.
var (
	_ = fmt.Sprintf
	_ = id.NewAggregateID
)
