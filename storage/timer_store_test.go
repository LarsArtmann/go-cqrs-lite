package storage_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	_ "modernc.org/sqlite" // pure-Go SQLite driver

	"github.com/larsartmann/go-cqrs-lite/scheduling/v3"
	"github.com/larsartmann/go-cqrs-lite/storage/v3"
)

func newTestTimerStore(t *testing.T) *storage.SQLTimerStore[string] {
	t.Helper()

	db, err := storage.OpenSQLiteInMemory()
	if err != nil {
		t.Fatalf("OpenSQLiteInMemory: %v", err)
	}

	t.Cleanup(func() { _ = db.Close() })

	if err := storage.SQLiteInitSchema(context.Background(), db); err != nil {
		t.Fatalf("SQLiteInitSchema: %v", err)
	}

	store, err := storage.NewSQLiteTimerStore[string](db)
	if err != nil {
		t.Fatalf("NewSQLiteTimerStore: %v", err)
	}

	return store
}

func TestSQLTimerStore_ScheduleAndDue(t *testing.T) {
	t.Parallel()

	store := newTestTimerStore(t)
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Second)

	if err := store.Schedule(ctx, scheduling.Timer[string]{
		ID: "a", FireAt: now.Add(-1 * time.Minute), Payload: "early",
	}); err != nil {
		t.Fatalf("Schedule early: %v", err)
	}

	if err := store.Schedule(ctx, scheduling.Timer[string]{
		ID: "b", FireAt: now.Add(1 * time.Hour), Payload: "late",
	}); err != nil {
		t.Fatalf("Schedule late: %v", err)
	}

	due, err := store.Due(ctx, now)
	if err != nil {
		t.Fatalf("Due: %v", err)
	}

	if len(due) != 1 {
		t.Fatalf("expected 1 due timer, got %d", len(due))
	}

	if due[0].ID != "a" {
		t.Fatalf("expected 'a', got %q", due[0].ID)
	}

	if due[0].Payload != "early" {
		t.Fatalf("expected payload 'early', got %q", due[0].Payload)
	}

	// FireAt should round-trip within a second (SQLite TEXT storage).
	if d := due[0].FireAt.Sub(now.Add(-1 * time.Minute)); d > time.Second || d < -time.Second {
		t.Fatalf("FireAt drift: %v", d)
	}
}

func TestSQLTimerStore_ScheduleIsIdempotent(t *testing.T) {
	t.Parallel()

	store := newTestTimerStore(t)
	ctx := context.Background()

	fireAt := time.Now().Add(-1 * time.Minute)

	for range 3 {
		if err := store.Schedule(ctx, scheduling.Timer[string]{
			ID: "dup", FireAt: fireAt, Payload: "once",
		}); err != nil {
			t.Fatalf("Schedule: %v", err)
		}
	}

	due, _ := store.Due(ctx, time.Now())
	if len(due) != 1 {
		t.Fatalf("expected 1 timer (idempotent), got %d", len(due))
	}
}

func TestSQLTimerStore_DueOrdersByFireAtAscending(t *testing.T) {
	t.Parallel()

	store := newTestTimerStore(t)
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Second)

	for _, tc := range []struct {
		id     string
		offset time.Duration
	}{
		{"middle", -10 * time.Minute},
		{"newest", -1 * time.Minute},
		{"oldest", -60 * time.Minute},
	} {
		if err := store.Schedule(ctx, scheduling.Timer[string]{
			ID: tc.id, FireAt: now.Add(tc.offset), Payload: tc.id,
		}); err != nil {
			t.Fatalf("Schedule %s: %v", tc.id, err)
		}
	}

	due, err := store.Due(ctx, now)
	if err != nil {
		t.Fatalf("Due: %v", err)
	}

	if len(due) != 3 {
		t.Fatalf("expected 3 due timers, got %d", len(due))
	}

	want := []string{"oldest", "middle", "newest"}
	for i, w := range want {
		if due[i].ID != w {
			t.Fatalf("position %d: want %q, got %q", i, w, due[i].ID)
		}
	}
}

func TestSQLTimerStore_MarkFired(t *testing.T) {
	t.Parallel()

	store := newTestTimerStore(t)
	ctx := context.Background()

	if err := store.Schedule(ctx, scheduling.Timer[string]{
		ID: "fire", FireAt: time.Now().Add(-1 * time.Minute), Payload: "x",
	}); err != nil {
		t.Fatalf("Schedule: %v", err)
	}

	if err := store.MarkFired(ctx, "fire"); err != nil {
		t.Fatalf("MarkFired: %v", err)
	}

	due, _ := store.Due(ctx, time.Now())
	if len(due) != 0 {
		t.Fatalf("expected 0 after mark fired, got %d", len(due))
	}
}

func TestSQLTimerStore_Cancel(t *testing.T) {
	t.Parallel()

	store := newTestTimerStore(t)
	ctx := context.Background()

	if err := store.Schedule(ctx, scheduling.Timer[string]{
		ID: "cancel-me", FireAt: time.Now().Add(-1 * time.Minute), Payload: "x",
	}); err != nil {
		t.Fatalf("Schedule: %v", err)
	}

	if err := store.Cancel(ctx, "cancel-me"); err != nil {
		t.Fatalf("Cancel: %v", err)
	}

	due, _ := store.Due(ctx, time.Now())
	if len(due) != 0 {
		t.Fatalf("expected 0 after cancel, got %d", len(due))
	}
}

func TestSQLTimerStore_NilDB(t *testing.T) {
	t.Parallel()

	_, err := storage.NewSQLiteTimerStore[string](nil)
	if err == nil {
		t.Fatal("expected error for nil DB")
	}
}

func TestSQLTimerStore_StructPayload(t *testing.T) {
	t.Parallel()

	type cancelOrder struct {
		OrderID string `json:"order_id"`
		Reason  string `json:"reason"`
	}

	db, err := storage.OpenSQLiteInMemory()
	if err != nil {
		t.Fatalf("OpenSQLiteInMemory: %v", err)
	}

	t.Cleanup(func() { _ = db.Close() })

	if err := storage.SQLiteInitSchema(context.Background(), db); err != nil {
		t.Fatalf("SQLiteInitSchema: %v", err)
	}

	store, err := storage.NewSQLiteTimerStore[cancelOrder](db)
	if err != nil {
		t.Fatalf("NewSQLiteTimerStore: %v", err)
	}

	ctx := context.Background()
	payload := cancelOrder{OrderID: "order-123", Reason: "timeout"}

	if err := store.Schedule(ctx, scheduling.Timer[cancelOrder]{
		ID: "order-123-timeout", FireAt: time.Now().Add(-1 * time.Minute), Payload: payload,
	}); err != nil {
		t.Fatalf("Schedule: %v", err)
	}

	due, err := store.Due(ctx, time.Now())
	if err != nil {
		t.Fatalf("Due: %v", err)
	}

	if len(due) != 1 {
		t.Fatalf("expected 1, got %d", len(due))
	}

	if due[0].Payload.OrderID != "order-123" {
		t.Fatalf("OrderID: got %q", due[0].Payload.OrderID)
	}

	if due[0].Payload.Reason != "timeout" {
		t.Fatalf("Reason: got %q", due[0].Payload.Reason)
	}
}

func TestSQLTimerStore_IntegrationWithScheduler(t *testing.T) {
	t.Parallel()

	db, err := storage.OpenSQLiteInMemory()
	if err != nil {
		t.Fatalf("OpenSQLiteInMemory: %v", err)
	}

	t.Cleanup(func() { _ = db.Close() })

	if err := storage.SQLiteInitSchema(context.Background(), db); err != nil {
		t.Fatalf("SQLiteInitSchema: %v", err)
	}

	store, err := storage.NewSQLiteTimerStore[string](db)
	if err != nil {
		t.Fatalf("NewSQLiteTimerStore: %v", err)
	}

	ctx := context.Background()

	if err := store.Schedule(ctx, scheduling.Timer[string]{
		ID: "sql-fired", FireAt: time.Now().Add(-1 * time.Second), Payload: "run-me",
	}); err != nil {
		t.Fatalf("Schedule: %v", err)
	}

	var (
		dispatched  atomic.Int64
		dispatchErr = errors.New("boom")
	)

	// First attempt fails, second succeeds — exercises MarkFired path.
	var attempts atomic.Int64

	sched := scheduling.New(
		store,
		func(_ context.Context, timer scheduling.Timer[string]) error {
			attempts.Add(1)
			if attempts.Load() == 1 {
				return dispatchErr
			}
			dispatched.Add(1)

			return nil
		},
		scheduling.WithPollInterval(20*time.Millisecond),
		scheduling.WithMaxRetries(3),
		scheduling.WithRetryDelay(5*time.Millisecond),
	)

	runCtx, cancel := context.WithCancel(ctx)
	go sched.Start(runCtx)

	waitForTimerDispatch(t, 3*time.Second, func() bool { return dispatched.Load() == 1 })
	cancel()

	if dispatched.Load() != 1 {
		t.Fatalf("expected 1 dispatch, got %d", dispatched.Load())
	}

	// The timer should have been removed after firing.
	due, _ := store.Due(ctx, time.Now())
	if len(due) != 0 {
		t.Fatalf("expected 0 timers after fire, got %d", len(due))
	}
}

func waitForTimerDispatch(t *testing.T, timeout time.Duration, cond func() bool) {
	t.Helper()

	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		if cond() {
			return
		}

		time.Sleep(5 * time.Millisecond)
	}
}
