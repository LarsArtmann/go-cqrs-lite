package sqlstore_test

import (
	"context"
	"testing"
	"time"

	sqlstore "github.com/larsartmann/go-cqrs-lite/scheduling/sqlstore/v4"
	"github.com/larsartmann/go-cqrs-lite/scheduling/v4"
)

// TestClaimingSQLite_MetricsHooks pins the zero-dep observability surface:
// Claimed fires once per Due poll with the claimed batch size, Renewed fires
// on a successful lease extension, and RenewRejected fires when the lease is
// not held.
func TestClaimingSQLite_MetricsHooks(t *testing.T) {
	_, db := newSQLiteStore[struct{}](t)

	ctx := context.Background()

	var claimedBatches []int

	renewed, rejected := 0, 0

	store, err := sqlstore.NewClaimingSQLiteStore[struct{}](ctx, db, time.Minute,
		sqlstore.WithClaimMetrics[struct{}](sqlstore.ClaimMetrics{
			Claimed:       func(count int) { claimedBatches = append(claimedBatches, count) },
			Renewed:       func() { renewed++ },
			RenewRejected: func() { rejected++ },
		}))
	if err != nil {
		t.Fatalf("NewClaimingSQLiteStore: %v", err)
	}

	now := time.Now().UTC()

	if err := store.Schedule(ctx, scheduling.Timer[struct{}]{
		ID:     scheduling.MustParseTimerID("metrics-a"),
		FireAt: now.Add(-time.Second),
	}); err != nil {
		t.Fatalf("Schedule: %v", err)
	}

	if _, err := store.Due(ctx, now); err != nil {
		t.Fatalf("Due: %v", err)
	}

	if len(claimedBatches) != 1 || claimedBatches[0] != 1 {
		t.Fatalf("Claimed batches = %v, want [1]", claimedBatches)
	}

	if err := store.RenewLease(ctx, scheduling.MustParseTimerID("metrics-a"), time.Minute); err != nil {
		t.Fatalf("RenewLease: %v", err)
	}

	if renewed != 1 {
		t.Errorf("Renewed fired %d times, want 1", renewed)
	}

	if rejected != 0 {
		t.Errorf("RenewRejected fired %d times, want 0", rejected)
	}

	// MarkFired removes the timer: renewal is now rejected.
	if err := store.MarkFired(ctx, scheduling.MustParseTimerID("metrics-a")); err != nil {
		t.Fatalf("MarkFired: %v", err)
	}

	if err := store.RenewLease(ctx, scheduling.MustParseTimerID("metrics-a"), time.Minute); err == nil {
		t.Fatal("RenewLease on fired timer must fail")
	} else if rejected != 1 {
		t.Errorf("RenewRejected fired %d times, want 1", rejected)
	}
}

// TestClaimingSQLite_NilMetrics verifies the opt-in contract: without
// WithClaimMetrics the store runs unobserved and every hook stays nil-safe.
func TestClaimingSQLite_NilMetrics(t *testing.T) {
	_, db := newSQLiteStore[struct{}](t)

	ctx := context.Background()

	store, err := sqlstore.NewClaimingSQLiteStore[struct{}](ctx, db, time.Minute)
	if err != nil {
		t.Fatalf("NewClaimingSQLiteStore: %v", err)
	}

	now := time.Now().UTC()

	if err := store.Schedule(ctx, scheduling.Timer[struct{}]{
		ID:     scheduling.MustParseTimerID("metrics-nil"),
		FireAt: now.Add(-time.Second),
	}); err != nil {
		t.Fatalf("Schedule: %v", err)
	}

	if _, err := store.Due(ctx, now); err != nil {
		t.Fatalf("Due with nil metrics: %v", err)
	}

	if err := store.RenewLease(ctx, scheduling.MustParseTimerID("metrics-nil"), time.Minute); err != nil {
		t.Fatalf("RenewLease with nil metrics: %v", err)
	}
}
