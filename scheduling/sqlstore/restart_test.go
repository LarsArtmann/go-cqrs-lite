package sqlstore_test

import (
	"context"
	"database/sql"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	"github.com/larsartmann/go-cqrs-lite/scheduling/sqlstore/v4"
	"github.com/larsartmann/go-cqrs-lite/scheduling/v4"
)

// TestSQLiteTimerStore_SurvivesRestart is the M44 durability test. It proves
// that a timer scheduled before a simulated crash (db close + reopen) is still
// present and fires correctly after the process restarts.
//
// This is the production-critical property of durable deadline timers: an
// "cancel order after 30 min unpaid" timer MUST survive a process crash. If it
// doesn't, crashed processes leak orders that never time out.
func TestSQLiteTimerStore_SurvivesRestart(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dbPath := filepath.Join(dir, "timers.db")
	dsn := "file:" + dbPath + "?_pragma=busy_timeout(5000)"
	ctx := context.Background()

	// --- Phase 1: "first process" schedules a timer, then crashes. ---
	db1, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatalf("open db1: %v", err)
	}

	store1, err := sqlstore.NewSQLiteStore[testPayload](ctx, db1)
	if err != nil {
		t.Fatalf("NewSQLiteStore 1: %v", err)
	}

	fireAt := time.Now().Add(50 * time.Millisecond)

	if err := store1.Schedule(ctx, scheduling.Timer[testPayload]{
		ID:      "order-123-timeout",
		FireAt:  fireAt,
		Payload: testPayload{Action: "cancel", Amount: 42},
	}); err != nil {
		t.Fatalf("Schedule: %v", err)
	}

	// Simulate a crash: close the DB WITHOUT marking the timer as fired.
	// In a real crash, MarkFired never runs.
	_ = db1.Close()

	// Wait until after the fire time, simulating downtime.
	time.Sleep(100 * time.Millisecond)

	// --- Phase 2: "second process" starts and recovers the timer. ---
	db2, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatalf("open db2: %v", err)
	}

	t.Cleanup(func() { _ = db2.Close() })

	store2, err := sqlstore.NewSQLiteStore[testPayload](ctx, db2)
	if err != nil {
		t.Fatalf("NewSQLiteStore 2: %v", err)
	}

	// The timer MUST be present and due — crash did not lose it.
	due, err := store2.Due(ctx, time.Now())
	if err != nil {
		t.Fatalf("Due after restart: %v", err)
	}

	if len(due) != 1 {
		t.Fatalf("expected 1 timer survived restart, got %d", len(due))
	}

	if due[0].ID != "order-123-timeout" {
		t.Fatalf("timer ID: got %q, want %q", due[0].ID, "order-123-timeout")
	}

	if due[0].Payload.Action != "cancel" || due[0].Payload.Amount != 42 {
		t.Fatalf(
			"payload corrupted across restart: got %+v",
			due[0].Payload,
		)
	}

	// After dispatching, the second process marks the timer as fired.
	if err := store2.MarkFired(ctx, "order-123-timeout"); err != nil {
		t.Fatalf("MarkFired: %v", err)
	}

	// No more timers remain.
	due, err = store2.Due(ctx, time.Now().Add(1*time.Hour))
	if err != nil {
		t.Fatalf("Due after MarkFired: %v", err)
	}

	if len(due) != 0 {
		t.Fatalf("expected 0 timers after MarkFired, got %d", len(due))
	}
}

// TestSQLiteTimerStore_SchedulerIntegration_Recovery proves the full
// scheduler + SQL store loop recovers overdue timers after a restart. This is
// the end-to-end version of SurvivesRestart: not just the store, but the
// Scheduler itself picks up the persisted timer and dispatches it.
func TestSQLiteTimerStore_SchedulerIntegration_Recovery(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dbPath := filepath.Join(dir, "timers.db")
	dsn := "file:" + dbPath + "?_pragma=busy_timeout(5000)"
	ctx := context.Background()

	// Phase 1: schedule a timer with a very short deadline, then "crash".
	db1, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatalf("open db1: %v", err)
	}

	store1, err := sqlstore.NewSQLiteStore[testPayload](ctx, db1)
	if err != nil {
		t.Fatalf("NewSQLiteStore 1: %v", err)
	}

	deadline := time.Now().Add(20 * time.Millisecond)

	if err := store1.Schedule(ctx, scheduling.Timer[testPayload]{
		ID:      "timeout-1",
		FireAt:  deadline,
		Payload: testPayload{Action: "expire"},
	}); err != nil {
		t.Fatalf("Schedule: %v", err)
	}

	// "Crash" — the timer is persisted but not fired.
	_ = db1.Close()

	// Simulate 50ms of downtime — timer is now overdue.
	time.Sleep(50 * time.Millisecond)

	// Phase 2: restart with a fresh scheduler that recovers the overdue timer.
	db2, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatalf("open db2: %v", err)
	}

	t.Cleanup(func() { _ = db2.Close() })

	store2, err := sqlstore.NewSQLiteStore[testPayload](ctx, db2)
	if err != nil {
		t.Fatalf("NewSQLiteStore 2: %v", err)
	}

	var dispatched atomic.Int64

	sched := scheduling.New[testPayload](
		store2,
		func(_ context.Context, tm scheduling.Timer[testPayload]) error {
			dispatched.Add(1)

			return nil
		},
		scheduling.WithPollInterval(10*time.Millisecond),
		scheduling.WithMaxRetries(1),
	)

	schedCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	_ = sched.Start(schedCtx)

	// The scheduler must dispatch the overdue timer.
	requireEventually(t, 1*time.Second, func() bool {
		return dispatched.Load() == 1
	})

	cancel()

	if dispatched.Load() != 1 {
		t.Fatalf("expected 1 dispatch after restart, got %d", dispatched.Load())
	}

	// The timer must have been marked as fired (removed from the store).
	due, err := store2.Due(ctx, time.Now().Add(1*time.Hour))
	if err != nil {
		t.Fatalf("Due after scheduler: %v", err)
	}

	if len(due) != 0 {
		t.Fatalf("expected 0 remaining timers, got %d", len(due))
	}
}

func requireEventually(t *testing.T, timeout time.Duration, cond func() bool) {
	t.Helper()

	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		if cond() {
			return
		}

		time.Sleep(5 * time.Millisecond)
	}

	t.Fatalf("condition not met within %s", timeout)
}
