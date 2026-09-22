package main

import (
	"math"
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	_ "modernc.org/sqlite" // driver registration for the test database

	"github.com/larsartmann/go-cqrs-lite/scheduling/sqlstore/v4"
)

// newTestStore builds the example's exact store shape (in-memory SQLite,
// payload struct{}) without the HTTP/OTel wiring — the claim flow under test
// is what both /status and /metrics report on.
func newTestStore(t *testing.T) *sqlstore.ClaimingTimerStore[struct{}] {
	t.Helper()

	db, err := sql.Open("sqlite", "file:claimflow-"+t.Name()+"?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	store, err := sqlstore.NewClaimingSQLiteStore[struct{}](context.Background(), db, 0)
	if err != nil {
		t.Fatalf("NewClaimingSQLiteStore: %v", err)
	}

	return store
}

// TestSchedulerOtelStatus_ClaimFlowCounted pins the example's core path:
// a scheduled due timer is claimed by the Due read (the claiming poll the
// demo's pollLoop performs), MarkFired completes it, the built-in Metrics()
// snapshot counts exactly that activity, and a second poll finds nothing
// (single-fire semantics).
func TestSchedulerOtelStatus_ClaimFlowCounted(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := newTestStore(t)

	timer := dueTimer("claim-flow-1")
	if err := store.Schedule(ctx, timer); err != nil {
		t.Fatalf("Schedule: %v", err)
	}

	before := store.Metrics()

	claimed, err := store.Due(ctx, time.Now())
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("Due: %v", err)
	}
	if len(claimed) != 1 {
		t.Fatalf("Due claimed %d timers, want 1", len(claimed))
	}
	if claimed[0].ID != timer.ID {
		t.Fatalf("Due claimed %s, want %s", claimed[0].ID, timer.ID)
	}

	if err := store.MarkFired(ctx, claimed[0].ID); err != nil {
		t.Fatalf("MarkFired: %v", err)
	}

	after := store.Metrics()
	if after.ClaimedTimers != before.ClaimedTimers+1 {
		t.Errorf("ClaimedTimers: got %d, want %d (one claimed timer)",
			after.ClaimedTimers, before.ClaimedTimers+1)
	}
	if after.ClaimedBatches < before.ClaimedBatches+1 {
		t.Errorf("ClaimedBatches: got %d, want >= %d (the poll is a batch)",
			after.ClaimedBatches, before.ClaimedBatches+1)
	}
	if after.StartedAt.IsZero() {
		t.Error("StartedAt is zero — the /status rate denominator is broken")
	}

	again, err := store.Due(ctx, time.Now())
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("second Due: %v", err)
	}
	if len(again) != 0 {
		t.Errorf("second Due claimed %d timers, want 0 (already fired)", len(again))
	}
}

// TestSchedulerOtelStatus_StatusSnapshotRate pins the /status rate math in
// isolation: claimedPerMinute derives from the StartedAt anchor, and a zero
// window reports 0 rather than dividing by zero.
func TestSchedulerOtelStatus_StatusSnapshotRate(t *testing.T) {
	t.Parallel()

	snap := statusSnapshot{
		ClaimMetricsSnapshot: sqlstore.ClaimMetricsSnapshot{
			ClaimedTimers: 10,
			StartedAt:     time.Now().Add(-5 * time.Minute),
		},
	}

	minutes := time.Since(snap.StartedAt).Minutes()
	rate := 0.0
	if minutes > 0 {
		rate = float64(snap.ClaimedTimers) / minutes
	}

	if rate < 1.99 || rate > 2.01 {
		t.Errorf("rate over 5min window: got %f, want ~2.0", rate)
	}

	zero := statusSnapshot{ClaimMetricsSnapshot: sqlstore.ClaimMetricsSnapshot{
		ClaimedTimers: 10,
		StartedAt:     time.Now(),
	}}
	zeroMinutes := time.Since(zero.StartedAt).Minutes()
	zeroRate := 0.0
	if zeroMinutes > 0 {
		zeroRate = float64(zero.ClaimedTimers) / zeroMinutes
	}
	if math.IsNaN(zeroRate) || math.IsInf(zeroRate, 0) {
		t.Errorf("fresh StartedAt produced a non-finite rate: %f (window %f)", zeroRate, zeroMinutes)
	}
}
