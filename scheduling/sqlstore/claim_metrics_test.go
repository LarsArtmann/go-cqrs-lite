package sqlstore_test

import (
	"context"
	"encoding/json/v2"
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

	if err := store.RenewLease(
		ctx,
		scheduling.MustParseTimerID("metrics-a"),
		time.Minute,
	); err != nil {
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

	if err := store.RenewLease(
		ctx,
		scheduling.MustParseTimerID("metrics-a"),
		time.Minute,
	); err == nil {
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

	if err := store.RenewLease(
		ctx,
		scheduling.MustParseTimerID("metrics-nil"),
		time.Minute,
	); err != nil {
		t.Fatalf("RenewLease with nil metrics: %v", err)
	}
}

// TestClaimingSQLite_MetricsSnapshot pins the built-in counter surface: the
// store maintains claim activity itself (no hook wiring), Metrics reports it,
// and empty polls count as batches (liveness heartbeat).
func TestClaimingSQLite_MetricsSnapshot(t *testing.T) {
	_, db := newSQLiteStore[struct{}](t)

	ctx := context.Background()

	store, err := sqlstore.NewClaimingSQLiteStore[struct{}](ctx, db, time.Minute)
	if err != nil {
		t.Fatalf("NewClaimingSQLiteStore: %v", err)
	}

	if got := store.Metrics(); got != (sqlstore.ClaimMetricsSnapshot{}) {
		t.Fatalf("fresh store Metrics = %+v, want zero", got)
	}

	now := time.Now().UTC()

	for _, id := range []string{"metrics-s1", "metrics-s2"} {
		if err := store.Schedule(ctx, scheduling.Timer[struct{}]{
			ID:     scheduling.MustParseTimerID(id),
			FireAt: now.Add(-time.Second),
		}); err != nil {
			t.Fatalf("Schedule %s: %v", id, err)
		}
	}

	if _, err := store.Due(ctx, now); err != nil {
		t.Fatalf("Due: %v", err)
	}

	if _, err := store.Due(ctx, now); err != nil {
		t.Fatalf("empty Due: %v", err)
	}

	got := store.Metrics()
	if got.ClaimedBatches != 2 || got.ClaimedTimers != 2 {
		t.Fatalf("Metrics after two polls = %+v, want 2 batches / 2 timers", got)
	}

	if err := store.RenewLease(
		ctx,
		scheduling.MustParseTimerID("metrics-s1"),
		time.Minute,
	); err != nil {
		t.Fatalf("RenewLease: %v", err)
	}

	if err := store.MarkFired(ctx, scheduling.MustParseTimerID("metrics-s1")); err != nil {
		t.Fatalf("MarkFired: %v", err)
	}

	if err := store.RenewLease(
		ctx,
		scheduling.MustParseTimerID("metrics-s1"),
		time.Minute,
	); err == nil {
		t.Fatal("RenewLease on fired timer must fail")
	}

	got = store.Metrics()
	if got.Renewed != 1 || got.RenewRejected != 1 {
		t.Fatalf("Metrics after renewals = %+v, want 1 renewed / 1 rejected", got)
	}
}

// TestClaimMetricsSnapshot_JSONTagsAreStable pins the wire shape of the
// snapshot: the camelCase tags are public API for any /status endpoint or
// dashboard consuming the marshaled snapshot (field order follows the struct).
func TestClaimMetricsSnapshot_JSONTagsAreStable(t *testing.T) {
	data, err := json.Marshal(sqlstore.ClaimMetricsSnapshot{
		ClaimedBatches: 2,
		ClaimedTimers:  3,
		Renewed:        1,
		RenewRejected:  1,
	})
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	const want = `{"claimedBatches":2,"claimedTimers":3,"renewed":1,"renewRejected":1}`
	if string(data) != want {
		t.Fatalf("JSON = %s, want %s", data, want)
	}
}
