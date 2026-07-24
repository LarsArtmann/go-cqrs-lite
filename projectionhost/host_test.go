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

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/projection/v4"
	"github.com/larsartmann/go-cqrs-lite/projectionhost/v4"
	"github.com/larsartmann/go-cqrs-lite/testutil/v4"
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
	streamID := id.NewStreamID()
	evt, _ := event.New(
		event.Type(eventType),
		streamID,
		"TestStream",
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

// --- Live subscriber test fixtures ---

// channelSubscriber is a minimal event.Subscriber that delivers events from
// a Go channel. Used to simulate live event delivery after journal drain.
type channelSubscriber struct {
	ch chan event.Event
}

func newChannelSubscriber() *channelSubscriber {
	return &channelSubscriber{ch: make(chan event.Event, 64)}
}

func (s *channelSubscriber) Subscribe(_ event.Type, _ event.Handler) error {
	return errors.New("channelSubscriber only supports SubscribeAll")
}

func (s *channelSubscriber) SubscribeAll(handler event.Handler) error {
	for evt := range s.ch {
		if err := handler(context.Background(), evt); err != nil {
			return err
		}
	}

	return nil
}

func (s *channelSubscriber) send(evt event.Event) { s.ch <- evt }
func (s *channelSubscriber) close()               { close(s.ch) }

// --- Live-mode tests ---

func TestHost_WithSubscriber_DrainsThenLive(t *testing.T) {
	t.Parallel()

	journal := &memoryJournal{}
	cpStore := newMemoryCheckpointStore()
	liveSub := newChannelSubscriber()

	// Seed two events in the journal (replay phase).
	evt1 := makeEvent("task.created")
	evt2 := makeEvent("task.created")
	journal.append(evt1)
	journal.append(evt2)

	proj := &countingProjection{name: "replay+live", eventTypes: []event.Type{"task.created"}}

	host, err := projectionhost.New(
		journal, cpStore,
		projectionhost.WithSubscriber(liveSub),
		projectionhost.WithBatchSize(10),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if err := host.Register(proj); err != nil {
		t.Fatalf("Register: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := host.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}

	// Wait for replay events to be processed.
	requireEventually(t, 2*time.Second, func() bool {
		return proj.count.Load() >= 2
	})

	// Send a live event — it should be processed too.
	evt3 := makeEvent("task.created")
	liveSub.send(evt3)

	requireEventually(t, 2*time.Second, func() bool {
		return proj.count.Load() >= 3
	})

	cancel()
	liveSub.close()
	_ = host.Stop()
}

func TestHost_WithSubscriber_DedupOverlap(t *testing.T) {
	t.Parallel()

	journal := &memoryJournal{}
	cpStore := newMemoryCheckpointStore()
	liveSub := newChannelSubscriber()

	// Seed one event in the journal.
	evt1 := makeEvent("task.created")
	journal.append(evt1)

	// The same event will also arrive via the live subscriber — it must be
	// deduped, not double-processed.
	proj := &countingProjection{name: "dedup", eventTypes: []event.Type{"task.created"}}

	host, err := projectionhost.New(
		journal, cpStore,
		projectionhost.WithSubscriber(liveSub),
		projectionhost.WithBatchSize(10),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if err := host.Register(proj); err != nil {
		t.Fatalf("Register: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := host.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}

	// Wait for replay to process the journal event.
	requireEventually(t, 2*time.Second, func() bool {
		return proj.count.Load() >= 1
	})

	// Send the SAME event ID via live — must be deduped.
	liveSub.send(evt1)

	// Send a new event — must be processed.
	evt2 := makeEvent("task.created")
	liveSub.send(evt2)

	requireEventually(t, 2*time.Second, func() bool {
		return proj.count.Load() >= 2
	})

	// Give a brief moment for any potential dedup failure to show up.
	time.Sleep(100 * time.Millisecond)

	if got := proj.count.Load(); got != 2 {
		t.Fatalf("expected count=2 (deduped overlap), got %d", got)
	}

	cancel()
	liveSub.close()
	_ = host.Stop()
}

func TestHost_LastProcessedAt_ReportsTime(t *testing.T) {
	t.Parallel()

	journal := &memoryJournal{}
	cpStore := newMemoryCheckpointStore()

	evt := makeEvent("task.created")
	journal.append(evt)

	proj := &countingProjection{name: "ts", eventTypes: []event.Type{"task.created"}}
	host, _ := projectionhost.New(journal, cpStore)
	_ = host.Register(proj)

	before := time.Now()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_ = host.Start(ctx)

	requireEventually(t, 2*time.Second, func() bool {
		return proj.count.Load() >= 1
	})

	last := host.LastProcessedAt()
	if last.IsZero() {
		t.Fatal("expected non-zero LastProcessedAt after processing events")
	}

	if !last.After(before) {
		t.Fatal("expected LastProcessedAt to be after start time")
	}

	cancel()
	_ = host.Stop()
}

// --- New feature tests (M1, M6, M7, M8) ---

// failingCheckpointStore wraps a memoryCheckpointStore and fails on the Nth Save.
type failingCheckpointStore struct {
	*memoryCheckpointStore

	failAfter int // fail after this many successful saves (0 = never)
	saves     atomic.Int64
}

func newFailingCheckpointStore(failAfter int) *failingCheckpointStore {
	return &failingCheckpointStore{
		memoryCheckpointStore: newMemoryCheckpointStore(),
		failAfter:             failAfter,
	}
}

func (s *failingCheckpointStore) Save(ctx context.Context, name string, cp event.Checkpoint) error {
	if s.failAfter > 0 && s.saves.Add(1) > int64(s.failAfter) {
		return errors.New("simulated checkpoint store failure")
	}

	return s.memoryCheckpointStore.Save(ctx, name, cp)
}

// resettableCountingProjection extends countingProjection with Reset support.
type resettableCountingProjection struct {
	countingProjection

	resetCount atomic.Int64
}

func (p *resettableCountingProjection) Reset(_ context.Context) error {
	p.resetCount.Add(1)
	p.count.Store(0)
	p.mu.Lock()
	p.seen = nil
	p.mu.Unlock()

	return nil
}

// alwaysFailingProjection returns failErr on every Handle call.
type alwaysFailingProjection struct {
	name    string
	calls   atomic.Int64
	failErr error
}

func (p *alwaysFailingProjection) Name() string             { return p.name }
func (p *alwaysFailingProjection) EventTypes() []event.Type { return nil }

func (p *alwaysFailingProjection) Handle(_ context.Context, _ event.Event) error {
	p.calls.Add(1)

	return p.failErr
}

func TestHost_OnFailed_FiresOnExhaustedRestarts(t *testing.T) {
	t.Parallel()

	journal := &memoryJournal{}
	cpStore := newMemoryCheckpointStore()
	journal.append(makeEvent("data.fail"))

	var failedName string
	var failedErr string
	var failedMu sync.Mutex

	proj := &alwaysFailingProjection{
		name:    "always-fail",
		failErr: errors.New("permanent handler failure"),
	}

	host, _ := projectionhost.New(
		journal, cpStore,
		projectionhost.WithMaxRestarts(1),
		projectionhost.WithBackoff(time.Millisecond, 5*time.Millisecond),
		projectionhost.WithOnFailed(func(name, lastErr string) {
			failedMu.Lock()
			failedName = name
			failedErr = lastErr
			failedMu.Unlock()
		}),
	)
	_ = host.Register(proj)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_ = host.Start(ctx)

	requireEventually(t, 5*time.Second, func() bool {
		failedMu.Lock()
		defer failedMu.Unlock()

		return failedName != ""
	})

	failedMu.Lock()
	defer failedMu.Unlock()

	if failedName != "always-fail" {
		t.Fatalf("expected OnFailed for 'always-fail', got %q", failedName)
	}

	if failedErr == "" {
		t.Fatal("expected non-empty lastError in OnFailed callback")
	}
}

func TestHost_WorkerFailedMetric(t *testing.T) {
	t.Parallel()

	journal := &memoryJournal{}
	cpStore := newMemoryCheckpointStore()
	journal.append(makeEvent("data.fail"))

	recorder := &capturingMetricsRecorder{}

	proj := &alwaysFailingProjection{
		name:    "metric-fail",
		failErr: errors.New("permanent handler failure"),
	}

	host, _ := projectionhost.New(
		journal, cpStore,
		projectionhost.WithMaxRestarts(1),
		projectionhost.WithBackoff(time.Millisecond, 5*time.Millisecond),
		projectionhost.WithMetrics(recorder),
	)
	_ = host.Register(proj)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_ = host.Start(ctx)

	requireEventually(t, 5*time.Second, func() bool {
		return recorder.failed.Load() >= 1
	})

	if recorder.failed.Load() != 1 {
		t.Fatalf("expected 1 WorkerFailed call, got %d", recorder.failed.Load())
	}

	_ = host.Stop()
}

func TestHost_Reset_DropsCheckpointAndReplaysFromZero(t *testing.T) {
	t.Parallel()

	journal := &memoryJournal{}
	cpStore := newMemoryCheckpointStore()

	for range 3 {
		journal.append(makeEvent("data.reset"))
	}

	proj := &countingProjection{name: "reset-test"}
	host, _ := projectionhost.New(journal, cpStore, projectionhost.WithBatchSize(10))
	_ = host.Register(proj)

	ctx, cancel := context.WithCancel(context.Background())
	go host.Start(ctx)

	requireEventually(t, 2*time.Second, func() bool {
		return proj.count.Load() == 3
	})

	cancel()
	_ = host.Stop()

	if err := host.Reset(context.Background(), "reset-test"); err != nil {
		t.Fatalf("Reset: %v", err)
	}

	if proj.count.Load() != 3 {
		t.Fatal("Reset should not affect projection state when not Resettable")
	}

	ctx2, cancel2 := context.WithCancel(context.Background())
	defer cancel2()
	host2, _ := projectionhost.New(journal, cpStore, projectionhost.WithBatchSize(10))
	_ = host2.Register(proj)
	go host2.Start(ctx2)

	requireEventually(t, 2*time.Second, func() bool {
		return proj.count.Load() == 6 // 3 original + 3 replayed
	})

	cancel2()
	_ = host2.Stop()
}

func TestHost_Reset_CallsResettableProjection(t *testing.T) {
	t.Parallel()

	journal := &memoryJournal{}
	cpStore := newMemoryCheckpointStore()

	for range 3 {
		journal.append(makeEvent("data.rst"))
	}

	proj := &resettableCountingProjection{
		countingProjection: countingProjection{name: "rst-proj"},
	}
	host, _ := projectionhost.New(journal, cpStore, projectionhost.WithBatchSize(10))
	_ = host.Register(proj)

	ctx, cancel := context.WithCancel(context.Background())
	go host.Start(ctx)

	requireEventually(t, 2*time.Second, func() bool {
		return proj.count.Load() == 3
	})

	cancel()
	_ = host.Stop()

	if err := host.Reset(context.Background(), "rst-proj"); err != nil {
		t.Fatalf("Reset: %v", err)
	}

	if proj.resetCount.Load() != 1 {
		t.Fatalf("expected Resettable.Reset called once, got %d", proj.resetCount.Load())
	}

	if proj.count.Load() != 0 {
		t.Fatalf("expected count reset to 0, got %d", proj.count.Load())
	}

	cp, _ := cpStore.Load(context.Background(), "rst-proj")
	if !cp.IsZero() {
		t.Fatal("expected checkpoint cleared after Reset")
	}
}

func TestHost_Reset_UnknownProjection_ReturnsError(t *testing.T) {
	t.Parallel()

	journal := &memoryJournal{}
	cpStore := newMemoryCheckpointStore()

	host, _ := projectionhost.New(journal, cpStore)

	err := host.Reset(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error for unknown projection")
	}
}

func TestHost_Reset_WhileRunning_ReturnsError(t *testing.T) {
	t.Parallel()

	journal := &memoryJournal{}
	cpStore := newMemoryCheckpointStore()
	journal.append(makeEvent("data.run"))

	proj := &countingProjection{name: "running"}
	host, _ := projectionhost.New(journal, cpStore, projectionhost.WithBatchSize(1))
	_ = host.Register(proj)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_ = host.Start(ctx)

	requireEventually(t, 2*time.Second, func() bool {
		return proj.count.Load() >= 1
	})

	err := host.Reset(context.Background(), "running")
	if err == nil {
		t.Fatal("expected error when resetting while running")
	}

	cancel()
	_ = host.Stop()
}

func TestHost_LiveCheckpointFailure_StopsWorker(t *testing.T) {
	t.Parallel()

	journal := &memoryJournal{}
	cpStore := newFailingCheckpointStore(1) // succeed once (drain), fail in live
	liveSub := newChannelSubscriber()

	// Seed one event in journal for drain phase.
	journal.append(makeEvent("task.live"))

	proj := &countingProjection{name: "live-cp-fail", eventTypes: []event.Type{"task.live"}}
	host, _ := projectionhost.New(
		journal, cpStore,
		projectionhost.WithSubscriber(liveSub),
		projectionhost.WithBatchSize(10),
		projectionhost.WithMaxRestarts(0), // don't restart — we want to see the failure
	)
	_ = host.Register(proj)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_ = host.Start(ctx)

	// Wait for drain to complete (1 event processed).
	requireEventually(t, 2*time.Second, func() bool {
		return proj.count.Load() == 1
	})

	// Send a live event — checkpoint save should fail and worker should stop.
	liveSub.send(makeEvent("task.live"))

	requireEventually(t, 3*time.Second, func() bool {
		states := host.Status()
		for _, s := range states {
			if s.Status == projectionhost.WorkerFailed || s.Status == projectionhost.WorkerStopped {
				return true
			}
		}

		return false
	})

	liveSub.close()
	cancel()
	_ = host.Stop()
}

func TestHost_WorkerDraining_StatusDuringShutdown(t *testing.T) {
	t.Parallel()

	journal := &memoryJournal{}
	cpStore := newMemoryCheckpointStore()

	// Add enough events that the worker is still processing when Stop fires.
	for range 100 {
		journal.append(makeEvent("data.drain"))
	}

	proj := &countingProjection{name: "drain-test"}
	host, _ := projectionhost.New(journal, cpStore, projectionhost.WithBatchSize(5))
	_ = host.Register(proj)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_ = host.Start(ctx)

	requireEventually(t, 2*time.Second, func() bool {
		return proj.count.Load() >= 1
	})

	_ = host.Stop()

	states := host.Status()
	for _, s := range states {
		if s.Status != projectionhost.WorkerStopped {
			t.Fatalf("expected WorkerStopped after Stop, got %s", s.Status)
		}
	}
}

func TestHost_WithShutdownTimeout_CustomValue(t *testing.T) {
	t.Parallel()

	journal := &memoryJournal{}
	cpStore := newMemoryCheckpointStore()
	journal.append(makeEvent("data.timeout"))

	proj := &countingProjection{name: "timeout-test"}
	host, _ := projectionhost.New(
		journal, cpStore,
		projectionhost.WithShutdownTimeout(5*time.Second),
	)
	_ = host.Register(proj)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_ = host.Start(ctx)

	requireEventually(t, 2*time.Second, func() bool {
		return proj.count.Load() >= 1
	})

	// Stop should succeed within the custom timeout.
	start := time.Now()
	err := host.Stop()
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("Stop: %v", err)
	}

	if elapsed > 5*time.Second {
		t.Fatalf("Stop took %s, expected < 5s", elapsed)
	}
}

func TestHost_LagPerProjection_NoEvents(t *testing.T) {
	t.Parallel()

	journal := &memoryJournal{}
	cpStore := newMemoryCheckpointStore()
	host, _ := projectionhost.New(journal, cpStore)
	_ = host.Register(&countingProjection{name: "lag-a"})
	_ = host.Register(&countingProjection{name: "lag-b"})

	lags := host.LagPerProjection()
	if len(lags) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(lags))
	}
	if lags["lag-a"] != 0 {
		t.Fatalf("expected 0 lag for unstarted worker, got %v", lags["lag-a"])
	}
	if lags["lag-b"] != 0 {
		t.Fatalf("expected 0 lag for unstarted worker, got %v", lags["lag-b"])
	}
}

func TestHost_LagPerProjection_AfterProcessing(t *testing.T) {
	t.Parallel()

	journal := &memoryJournal{}
	cpStore := newMemoryCheckpointStore()
	journal.append(makeEvent("data.lag"))

	proj := &countingProjection{name: "lag-proj"}
	host, _ := projectionhost.New(journal, cpStore, projectionhost.WithBatchSize(10))
	_ = host.Register(proj)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go host.Start(ctx)

	requireEventually(t, 2*time.Second, func() bool {
		return proj.count.Load() == 1
	})

	cancel()
	_ = host.Stop()

	lags := host.LagPerProjection()
	lag, ok := lags["lag-proj"]
	if !ok {
		t.Fatal("expected 'lag-proj' in lag map")
	}
	// After processing + stop, lag may be tiny or 0 — just verify it's non-negative.
	if lag < 0 {
		t.Fatalf("lag should be non-negative, got %v", lag)
	}
}

func TestHost_Status_WorkerStateLag(t *testing.T) {
	t.Parallel()

	journal := &memoryJournal{}
	cpStore := newMemoryCheckpointStore()
	journal.append(makeEvent("data.stlag"))

	proj := &countingProjection{name: "stlag-proj"}
	host, _ := projectionhost.New(journal, cpStore, projectionhost.WithBatchSize(10))
	_ = host.Register(proj)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go host.Start(ctx)

	requireEventually(t, 2*time.Second, func() bool {
		return proj.count.Load() == 1
	})

	cancel()
	_ = host.Stop()

	states := host.Status()
	if len(states) != 1 {
		t.Fatalf("expected 1 state, got %d", len(states))
	}
	// After processing, Lag should be populated (non-zero since some time has passed).
	if states[0].Lag < 0 {
		t.Fatalf("expected non-negative lag, got %v", states[0].Lag)
	}
}

func TestHost_LagDuration_MatchesMaxLag(t *testing.T) {
	t.Parallel()

	journal := &memoryJournal{}
	cpStore := newMemoryCheckpointStore()
	host, _ := projectionhost.New(journal, cpStore)
	_ = host.Register(&countingProjection{name: "only"})

	// No events processed: both should return 0.
	if d := host.LagDuration(); d != 0 {
		t.Fatalf("expected 0 lag before processing, got %v", d)
	}
	lags := host.LagPerProjection()
	if lags["only"] != 0 {
		t.Fatalf("expected 0 per-projection lag before processing, got %v", lags["only"])
	}
}

func TestHost_Reset_PurgesDeadLetters(t *testing.T) {
	t.Parallel()

	journal := &memoryJournal{}
	cpStore := newMemoryCheckpointStore()
	dlq := projectionhost.NewMemoryDeadLetterStore()

	// Pre-populate DLQ with entries for two projections.
	_ = dlq.Store(context.Background(), projectionhost.DeadLetterEntry{
		ProjectionName: "purge-me",
		EventID:        "evt-1",
		EventType:      "test.created",
		Error:          "boom",
	})
	_ = dlq.Store(context.Background(), projectionhost.DeadLetterEntry{
		ProjectionName: "keep-me",
		EventID:        "evt-2",
		EventType:      "test.created",
		Error:          "boom",
	})

	proj := &resettableCountingProjection{
		countingProjection: countingProjection{name: "purge-me"},
	}
	host, _ := projectionhost.New(
		journal, cpStore,
		projectionhost.WithDeadLetterStore(dlq, 3),
	)
	_ = host.Register(proj)
	_ = host.Register(&countingProjection{name: "keep-me"})

	if err := host.Reset(
		context.Background(), "purge-me",
		projectionhost.WithPurgeDeadLetters(),
	); err != nil {
		t.Fatalf("Reset with purge: %v", err)
	}

	entries, _ := dlq.List(context.Background(), "purge-me")
	if len(entries) != 0 {
		t.Fatalf("expected 0 entries for purge-me after purge, got %d", len(entries))
	}

	entries, _ = dlq.List(context.Background(), "keep-me")
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry for keep-me (untouched), got %d", len(entries))
	}
}

func TestHost_Reset_WithoutPurge_KeepsDeadLetters(t *testing.T) {
	t.Parallel()

	journal := &memoryJournal{}
	cpStore := newMemoryCheckpointStore()
	dlq := projectionhost.NewMemoryDeadLetterStore()

	_ = dlq.Store(context.Background(), projectionhost.DeadLetterEntry{
		ProjectionName: "keep-dlq",
		EventID:        "evt-1",
		EventType:      "test.created",
		Error:          "boom",
	})

	proj := &resettableCountingProjection{
		countingProjection: countingProjection{name: "keep-dlq"},
	}
	host, _ := projectionhost.New(
		journal, cpStore,
		projectionhost.WithDeadLetterStore(dlq, 3),
	)
	_ = host.Register(proj)

	if err := host.Reset(context.Background(), "keep-dlq"); err != nil {
		t.Fatalf("Reset: %v", err)
	}

	entries, _ := dlq.List(context.Background(), "keep-dlq")
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry preserved (no purge flag), got %d", len(entries))
	}
}

// TestHost_StaggerShutdownNoLeak is a regression test for a goroutine-leak bug
// where workers cancelled during their stagger delay (before run() was called)
// would return without decrementing the WaitGroup, causing Stop() to block for
// the full shutdownTimeout. With many projections in a single host and immediate
// Stop(), the stagger window (N workers × 10ms) guarantees some workers are
// still in their delay when Stop() cancels the context. Stop() should return
// promptly instead of blocking for the full shutdown timeout.
func TestHost_StaggerShutdownNoLeak(t *testing.T) {
	t.Parallel()

	journal := &memoryJournal{}
	cpStore := newMemoryCheckpointStore()

	host, err := projectionhost.New(
		journal, cpStore,
		projectionhost.WithBatchSize(100),
		projectionhost.WithShutdownTimeout(5*time.Second),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	// Register enough projections so the total stagger window is significant.
	// 20 workers × 10ms = 200ms stagger window — guaranteed to overlap with
	// an immediate Stop().
	const numProjections = 20
	for i := range numProjections {
		if err := host.Register(&countingProjection{
			name: fmt.Sprintf("proj-%d", i),
		}); err != nil {
			t.Fatalf("Register: %v", err)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	if err := host.Start(ctx); err != nil {
		cancel()
		t.Fatalf("Start: %v", err)
	}

	// Cancel immediately — some workers are still in their stagger delay.
	cancel()

	start := time.Now()
	if err := host.Stop(); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	elapsed := time.Since(start)

	// With the bug, Stop() blocked for the full shutdownTimeout (5s here).
	// With the fix, all stagger goroutines decrement the WaitGroup on return.
	// 2s is generous; the stagger window for 20 workers is ~200ms.
	if elapsed > 2*time.Second {
		t.Fatalf("Stop took %v — goroutine leak in stagger shutdown (expected <2s)", elapsed)
	}
}

// Ensure unused imports are referenced.
var (
	_ = fmt.Sprintf
	_ = id.NewStreamID
)

func TestHost_Status_ReturnsSortedByName(t *testing.T) {
	t.Parallel()

	journal := &memoryJournal{}
	cpStore := newMemoryCheckpointStore()

	host, _ := projectionhost.New(journal, cpStore)

	names := []string{
		"discord-zebra",
		"discord-alpha",
		"discord-mike",
		"discord-charlie",
		"discord-bravo",
	}
	for _, name := range names {
		if err := host.Register(&countingProjection{name: name}); err != nil {
			t.Fatalf("Register(%s): %v", name, err)
		}
	}

	states := host.Status()
	if len(states) != len(names) {
		t.Fatalf("expected %d states, got %d", len(names), len(states))
	}

	sorted := []string{
		"discord-alpha",
		"discord-bravo",
		"discord-charlie",
		"discord-mike",
		"discord-zebra",
	}
	for i, s := range states {
		if s.Name != sorted[i] {
			t.Fatalf("Status() position %d: expected %q, got %q — output must be sorted by name",
				i, sorted[i], s.Name)
		}
	}
}
