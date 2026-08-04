package sqlstore_test

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	"github.com/larsartmann/go-cqrs-lite/scheduling/sqlstore/v4"
	"github.com/larsartmann/go-cqrs-lite/scheduling/v4"
)

type testPayload struct {
	Action string `json:"action"`
	Amount int    `json:"amount"`
}

func newSQLiteStore[P any](t *testing.T) (*sqlstore.SQLTimerStore[P], *sql.DB) {
	t.Helper()

	dir := t.TempDir()
	dsn := "file:" + filepath.Join(dir, "timers.db") + "?_pragma=busy_timeout(5000)"

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	t.Cleanup(func() { _ = db.Close() })

	store, err := sqlstore.NewSQLiteStore[P](context.Background(), db)
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}

	return store, db
}

func TestSQLiteTimerStore_ScheduleAndDue(t *testing.T) {
	t.Parallel()

	store, _ := newSQLiteStore[testPayload](t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Millisecond)

	timers := []scheduling.Timer[testPayload]{
		{ID: "t3", FireAt: now.Add(30 * time.Second), Payload: testPayload{Action: "c", Amount: 3}},
		{ID: "t1", FireAt: now.Add(5 * time.Second), Payload: testPayload{Action: "a", Amount: 1}},
		{ID: "t2", FireAt: now.Add(10 * time.Second), Payload: testPayload{Action: "b", Amount: 2}},
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

	if len(due) != 1 {
		t.Fatalf("expected 1 due timer, got %d", len(due))
	}

	if due[0].ID != "t1" {
		t.Fatalf("expected t1, got %s", due[0].ID)
	}

	if due[0].Payload.Action != "a" {
		t.Fatalf("expected action 'a', got %s", due[0].Payload.Action)
	}

	// All three are due at now+60s, ordered by FireAt ascending.
	due, err = store.Due(ctx, now.Add(60*time.Second))
	if err != nil {
		t.Fatalf("Due all: %v", err)
	}

	if len(due) != 3 {
		t.Fatalf("expected 3 due timers, got %d", len(due))
	}

	if due[0].ID != "t1" || due[1].ID != "t2" || due[2].ID != "t3" {
		t.Fatalf("expected order t1,t2,t3 got %s,%s,%s", due[0].ID, due[1].ID, due[2].ID)
	}
}

func TestSQLiteTimerStore_ScheduleIsIdempotent(t *testing.T) {
	t.Parallel()

	store, _ := newSQLiteStore[testPayload](t)
	ctx := context.Background()

	timer := scheduling.Timer[testPayload]{
		ID:      "order-timeout",
		FireAt:  time.Now().Add(1 * time.Hour),
		Payload: testPayload{Action: "cancel", Amount: 100},
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
		t.Fatalf("idempotent schedule should yield 1 timer, got %d", len(due))
	}

	if due[0].Payload.Amount != 100 {
		t.Fatalf(
			"idempotent schedule must keep original payload; got amount %d",
			due[0].Payload.Amount,
		)
	}
}

func TestSQLiteTimerStore_MarkFired(t *testing.T) {
	t.Parallel()

	store, _ := newSQLiteStore[testPayload](t)
	ctx := context.Background()

	timer := scheduling.Timer[testPayload]{
		ID:      "fire-me",
		FireAt:  time.Now().Add(1 * time.Minute),
		Payload: testPayload{Action: "go"},
	}

	_ = store.Schedule(ctx, timer)

	if err := store.MarkFired(ctx, "fire-me"); err != nil {
		t.Fatalf("MarkFired: %v", err)
	}

	due, err := store.Due(ctx, time.Now().Add(2*time.Hour))
	if err != nil {
		t.Fatalf("Due: %v", err)
	}

	if len(due) != 0 {
		t.Fatalf("expected 0 timers after MarkFired, got %d", len(due))
	}
}

func TestSQLiteTimerStore_Cancel(t *testing.T) {
	t.Parallel()

	store, _ := newSQLiteStore[testPayload](t)
	ctx := context.Background()

	timer := scheduling.Timer[testPayload]{
		ID:      "cancel-me",
		FireAt:  time.Now().Add(1 * time.Minute),
		Payload: testPayload{Action: "go"},
	}

	_ = store.Schedule(ctx, timer)

	if err := store.Cancel(ctx, "cancel-me"); err != nil {
		t.Fatalf("Cancel: %v", err)
	}

	due, _ := store.Due(ctx, time.Now().Add(2*time.Hour))
	if len(due) != 0 {
		t.Fatalf("expected 0 timers after Cancel, got %d", len(due))
	}
}

func TestSQLiteTimerStore_EmptyDue(t *testing.T) {
	t.Parallel()

	store, _ := newSQLiteStore[testPayload](t)

	due, err := store.Due(context.Background(), time.Now())
	if err != nil {
		t.Fatalf("Due on empty store: %v", err)
	}

	if due != nil {
		t.Fatalf("expected nil slice for empty Due, got %v", due)
	}
}

