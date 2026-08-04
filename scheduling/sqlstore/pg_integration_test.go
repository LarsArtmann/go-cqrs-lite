//go:build integration

package sqlstore_test

// PostgreSQL integration tests for scheduling/sqlstore. Run with:
//
//	nix run .#integration-pg -- go test ./scheduling/sqlstore/... -tags=integration
//
// These tests verify that the Postgres-backed timer store correctly persists
// timers across process restarts using native TIMESTAMP WITH TIME ZONE, BYTEA
// payloads, and $N placeholders — the dialect path that is impossible to
// exercise with SQLite alone.

import (
	"context"
	"database/sql"
	"sync/atomic"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/scheduling/sqlstore/v4"
	"github.com/larsartmann/go-cqrs-lite/scheduling/v4"
)

// pgOpen opens a Postgres connection, drops any existing timers table for
// per-test isolation (the DSN is shared across tests when ephemeral-pg.sh sets
// DATABASE_URL), and pings the server. NewPostgresStore recreates the table.
func pgOpen(t *testing.T) *sql.DB {
	t.Helper()

	url := pgTestDSN(t)

	db, err := sql.Open("pgx", url)
	if err != nil {
		t.Fatalf("open pg: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("ping pg: %v", err)
	}

	if _, err := db.ExecContext(ctx, "DROP TABLE IF EXISTS timers"); err != nil {
		t.Fatalf("drop timers table: %v", err)
	}

	return db
}

// TestIntegration_PostgresTimerStore_ScheduleAndDue verifies basic CRUD
// against real PostgreSQL: native time.Time scanning, BYTEA payload round-trip,
// $N placeholder substitution, and ascending FireAt ordering.
func TestIntegration_PostgresTimerStore_ScheduleAndDue(t *testing.T) {
	ctx := context.Background()
	db := pgOpen(t)
	defer func() { _ = db.Close() }()

	store, err := sqlstore.NewPostgresStore[testPayload](ctx, db)
	if err != nil {
		t.Fatalf("NewPostgresStore: %v", err)
	}

	now := time.Now().UTC().Truncate(time.Millisecond)

	timers := []scheduling.Timer[testPayload]{
		{ID: "pg-t3", FireAt: now.Add(30 * time.Second), Payload: testPayload{Action: "c", Amount: 3}},
		{ID: "pg-t1", FireAt: now.Add(5 * time.Second), Payload: testPayload{Action: "a", Amount: 1}},
		{ID: "pg-t2", FireAt: now.Add(10 * time.Second), Payload: testPayload{Action: "b", Amount: 2}},
	}

	for _, tm := range timers {
		if err := store.Schedule(ctx, tm); err != nil {
			t.Fatalf("Schedule %s: %v", tm.ID, err)
		}
	}

	// Only t1 is due at now+6s.
	due, err := store.Due(ctx, now.Add(6*time.Second))
	if err != nil {
		t.Fatalf("Due: %v", err)
	}

	if len(due) != 1 || due[0].ID != "pg-t1" {
		t.Fatalf("expected only pg-t1 due, got %+v", due)
	}

	if due[0].Payload.Action != "a" {
		t.Fatalf("payload action: got %q, want %q", due[0].Payload.Action, "a")
	}

	// All three due at now+60s, ordered ascending.
	due, err = store.Due(ctx, now.Add(60*time.Second))
	if err != nil {
		t.Fatalf("Due all: %v", err)
	}

	if len(due) != 3 {
		t.Fatalf("expected 3 due timers, got %d", len(due))
	}

	if due[0].ID != "pg-t1" || due[1].ID != "pg-t2" || due[2].ID != "pg-t3" {
		t.Fatalf("ordering wrong: got %s,%s,%s", due[0].ID, due[1].ID, due[2].ID)
	}
}

// TestIntegration_PostgresTimerStore_IdempotentSchedule verifies ON CONFLICT
// DO NOTHING works on PostgreSQL — re-scheduling the same ID is a no-op.
func TestIntegration_PostgresTimerStore_IdempotentSchedule(t *testing.T) {
	ctx := context.Background()
	db := pgOpen(t)
	defer func() { _ = db.Close() }()

	store, err := sqlstore.NewPostgresStore[testPayload](ctx, db)
	if err != nil {
		t.Fatalf("NewPostgresStore: %v", err)
	}

	timer := scheduling.Timer[testPayload]{
		ID:      "pg-idempotent",
		FireAt:  time.Now().Add(1 * time.Hour),
		Payload: testPayload{Action: "original", Amount: 42},
	}

	if err := store.Schedule(ctx, timer); err != nil {
		t.Fatalf("first Schedule: %v", err)
	}

	// Re-schedule with different payload — must be a no-op.
	timer.Payload.Amount = 999
	if err := store.Schedule(ctx, timer); err != nil {
		t.Fatalf("second Schedule: %v", err)
	}

	due, err := store.Due(ctx, time.Now().Add(2*time.Hour))
	if err != nil {
		t.Fatalf("Due: %v", err)
	}

	if len(due) != 1 {
		t.Fatalf("expected 1 timer, got %d", len(due))
	}

	if due[0].Payload.Amount != 42 {
		t.Fatalf("idempotent schedule must keep original payload; got %d", due[0].Payload.Amount)
	}
}

// TestIntegration_PostgresTimerStore_SurvivesRestart is the core PG durability
// test: a timer scheduled before a simulated crash (connection close + reopen)
// must still be present and due after reopening a fresh connection to the same
// database.
func TestIntegration_PostgresTimerStore_SurvivesRestart(t *testing.T) {
	ctx := context.Background()

	// Phase 1: "first process" schedules a timer, then crashes.
	db1 := pgOpen(t)

	store1, err := sqlstore.NewPostgresStore[testPayload](ctx, db1)
	if err != nil {
		t.Fatalf("NewPostgresStore 1: %v", err)
	}

	fireAt := time.Now().Add(50 * time.Millisecond)

	if err := store1.Schedule(ctx, scheduling.Timer[testPayload]{
		ID:      "pg-order-timeout",
		FireAt:  fireAt,
		Payload: testPayload{Action: "cancel", Amount: 42},
	}); err != nil {
		t.Fatalf("Schedule: %v", err)
	}

	// Simulate a crash: close the connection WITHOUT marking the timer fired.
	_ = db1.Close()

	// Wait until after the fire time, simulating downtime.
	time.Sleep(100 * time.Millisecond)

	// Phase 2: "second process" opens a fresh connection and recovers.
	db2, err := sql.Open("pgx", pgTestDSN(t))
	if err != nil {
		t.Fatalf("open db2: %v", err)
	}

	defer func() { _ = db2.Close() }()

	store2, err := sqlstore.NewPostgresStore[testPayload](ctx, db2)
	if err != nil {
		t.Fatalf("NewPostgresStore 2: %v", err)
	}

	// The timer MUST be present and due — crash did not lose it.
	due, err := store2.Due(ctx, time.Now())
	if err != nil {
		t.Fatalf("Due after restart: %v", err)
	}

	if len(due) != 1 {
		t.Fatalf("expected 1 timer survived restart, got %d", len(due))
	}

	if due[0].ID != "pg-order-timeout" {
		t.Fatalf("timer ID: got %q, want %q", due[0].ID, "pg-order-timeout")
	}

	if due[0].Payload.Action != "cancel" || due[0].Payload.Amount != 42 {
		t.Fatalf("payload corrupted across restart: got %+v", due[0].Payload)
	}

	// After dispatching, mark fired and confirm it is gone.
	if err := store2.MarkFired(ctx, "pg-order-timeout"); err != nil {
		t.Fatalf("MarkFired: %v", err)
	}

	due, err = store2.Due(ctx, time.Now().Add(1*time.Hour))
	if err != nil {
		t.Fatalf("Due after MarkFired: %v", err)
	}

	if len(due) != 0 {
		t.Fatalf("expected 0 timers after MarkFired, got %d", len(due))
	}
}

// TestIntegration_PostgresTimerStore_SchedulerIntegration_Recovery proves the
// full scheduler + Postgres store loop recovers overdue timers after a restart.
func TestIntegration_PostgresTimerStore_SchedulerIntegration_Recovery(t *testing.T) {
	ctx := context.Background()

	// Phase 1: schedule a timer with a very short deadline, then "crash".
	db1 := pgOpen(t)

	store1, err := sqlstore.NewPostgresStore[testPayload](ctx, db1)
	if err != nil {
		t.Fatalf("NewPostgresStore 1: %v", err)
	}

	deadline := time.Now().Add(20 * time.Millisecond)

	if err := store1.Schedule(ctx, scheduling.Timer[testPayload]{
		ID:      "pg-timeout-1",
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
	db2, err := sql.Open("pgx", pgTestDSN(t))
	if err != nil {
		t.Fatalf("open db2: %v", err)
	}

	defer func() { _ = db2.Close() }()

	store2, err := sqlstore.NewPostgresStore[testPayload](ctx, db2)
	if err != nil {
		t.Fatalf("NewPostgresStore 2: %v", err)
	}

	var dispatched atomic.Int64

	sched := scheduling.New[testPayload](
		store2,
		func(_ context.Context, _ scheduling.Timer[testPayload]) error {
			dispatched.Add(1)
			return nil
		},
		scheduling.WithPollInterval(10*time.Millisecond),
		scheduling.WithMaxRetries(1),
	)

	schedCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	_ = sched.Start(schedCtx)

	// The scheduler must dispatch the overdue timer.
	requireEventually(t, 2*time.Second, func() bool {
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
