package sqlstore_test

import (
	"context"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/scheduling/sqlstore/v4"
	"github.com/larsartmann/go-cqrs-lite/scheduling/v4"
)

// TestClaimingSQLite_NoDoubleFireAcrossClaimers pins the claiming contract on
// SQLite (single-writer serialization stands in for SKIP LOCKED): timers Due
// returns to one claimer are NOT returned to another while the lease is
// fresh, and MarkFired still deletes after dispatch.
func TestClaimingSQLite_NoDoubleFireAcrossClaimers(t *testing.T) {
	_, db := newSQLiteStore[struct{}](t)

	ctx := context.Background()

	store, err := sqlstore.NewClaimingSQLiteStore[struct{}](ctx, db, time.Minute)
	if err != nil {
		t.Fatalf("NewClaimingSQLiteStore: %v", err)
	}

	now := time.Now().UTC()

	for _, id := range []string{"timeout-a", "timeout-b"} {
		timer := scheduling.Timer[struct{}]{
			ID:     scheduling.MustParseTimerID(id),
			FireAt: now.Add(-time.Second),
		}

		if err := store.Schedule(ctx, timer); err != nil {
			t.Fatalf("Schedule %s: %v", id, err)
		}
	}

	first, err := store.Due(ctx, now)
	if err != nil {
		t.Fatalf("Due (first claimer): %v", err)
	}

	if len(first) != 2 {
		t.Fatalf("first claimer got %d timers, want 2", len(first))
	}

	second, err := store.Due(ctx, now)
	if err != nil {
		t.Fatalf("Due (second claimer): %v", err)
	}

	if len(second) != 0 {
		t.Fatalf(
			"second claimer got %d timers while leases fresh, want 0 (double fire)",
			len(second),
		)
	}

	// Fired timers are deleted even though they were only leased.
	if err := store.MarkFired(ctx, first[0].ID); err != nil {
		t.Fatalf("MarkFired: %v", err)
	}

	again, err := store.Due(ctx, now.Add(2*time.Minute))
	if err != nil {
		t.Fatalf("Due (after lease expiry): %v", err)
	}

	for _, timer := range again {
		if timer.ID == first[0].ID {
			t.Errorf("fired timer %s reappeared after lease expiry", timer.ID)
		}
	}
}

// TestClaimingSQLite_LeaseExpiryReclaims pins the crash-recovery half: a
// timer claimed by a dispatcher that crashed (never MarkFired) becomes
// claimable again once the lease expires — at-least-once, not at-most-once.
func TestClaimingSQLite_LeaseExpiryReclaims(t *testing.T) {
	_, db := newSQLiteStore[struct{}](t)

	ctx := context.Background()

	store, err := sqlstore.NewClaimingSQLiteStore[struct{}](ctx, db, time.Minute)
	if err != nil {
		t.Fatalf("NewClaimingSQLiteStore: %v", err)
	}

	now := time.Now().UTC()
	timer := scheduling.Timer[struct{}]{
		ID:     scheduling.MustParseTimerID("lease-reclaim"),
		FireAt: now.Add(-time.Second),
	}

	if err := store.Schedule(ctx, timer); err != nil {
		t.Fatalf("Schedule: %v", err)
	}

	// Crashed dispatcher claims the timer, then dies without MarkFired.
	if _, err := store.Due(ctx, now); err != nil {
		t.Fatalf("Due (crashed claimer): %v", err)
	}

	// Within the lease: nobody else may fire it.
	inLease, err := store.Due(ctx, now.Add(time.Second))
	if err != nil {
		t.Fatalf("Due (within lease): %v", err)
	}

	if len(inLease) != 0 {
		t.Fatalf("timer reclaimed while lease fresh, want fenced")
	}

	// After the lease: reclaimable.
	after, err := store.Due(ctx, now.Add(2*time.Minute))
	if err != nil {
		t.Fatalf("Due (after lease): %v", err)
	}

	if len(after) != 1 || after[0].ID != timer.ID {
		t.Fatalf("timer not reclaimed after lease expiry: got %v", after)
	}
}

// TestNewClaimingMySQLStore_Rejects pins the honest rejection: MySQL/MariaDB
// has no SKIP LOCKED, so a claiming store there would lie about its
// guarantee — construction fails loudly instead.
func TestNewClaimingMySQLStore_Rejects(t *testing.T) {
	if _, err := sqlstore.NewClaimingMySQLStore[struct{}](context.TODO(), nil, 0); err == nil {
		t.Fatal("NewClaimingMySQLStore succeeded, want rejection")
	}
}

// TestClaimingSQLite_RenewLease pins the handler-outlives-lease contract:
// renewal extends a live claim, a renewed timer stays invisible to another
// claimer, renewal after expiry fails ErrLeaseNotHeld (expired claims
// cannot be resurrected), and a fired timer's renewal fails too.
func TestClaimingSQLite_RenewLease(t *testing.T) {
	_, db := newSQLiteStore[struct{}](t)

	ctx := context.Background()

	store, err := sqlstore.NewClaimingSQLiteStore[struct{}](ctx, db, time.Minute)
	if err != nil {
		t.Fatalf("NewClaimingSQLiteStore: %v", err)
	}

	now := time.Now().UTC()

	timer := scheduling.Timer[struct{}]{
		ID:     scheduling.MustParseTimerID("long-handler"),
		FireAt: now.Add(-time.Second),
	}

	if err := store.Schedule(ctx, timer); err != nil {
		t.Fatalf("Schedule: %v", err)
	}

	claimed, err := store.Due(ctx, now)
	if err != nil || len(claimed) != 1 {
		t.Fatalf("Due: %v (%d timers)", err, len(claimed))
	}

	// Renewal while the lease is live succeeds.
	if err := store.RenewLease(ctx, timer.ID, 5*time.Minute); err != nil {
		t.Fatalf("RenewLease (live lease): %v", err)
	}

	// Another claimer gets nothing while the renewed lease holds.
	second, err := store.Due(ctx, now)
	if err != nil {
		t.Fatalf("Due (second claimer after renewal): %v", err)
	}

	if len(second) != 0 {
		t.Fatalf("renewed timer re-claimed while lease live (double fire)")
	}

	// Renewal of a missing (fired/canceled) timer fails: no claim exists.
	if err := store.RenewLease(ctx, scheduling.MustParseTimerID("gone"), time.Minute); err == nil {
		t.Fatal("renewing a missing timer must fail")
	}

	// A fired timer's renewal fails too: the row is gone.
	if err := store.MarkFired(ctx, timer.ID); err != nil {
		t.Fatalf("MarkFired: %v", err)
	}

	err = store.RenewLease(ctx, timer.ID, time.Minute)
	if err == nil {
		t.Fatal("renewing a fired timer must fail")
	}
}

// TestClaimingSQLite_ExpiredLeaseReclaimable pins the expiry half of the
// lease contract that RenewLease builds on: an expired claim cannot be
// renewed (cannot resurrect), and after expiry the timer IS claimable again
// by another poller.
func TestClaimingSQLite_ExpiredLeaseReclaimable(t *testing.T) {
	_, db := newSQLiteStore[struct{}](t)

	ctx := context.Background()

	store, err := sqlstore.NewClaimingSQLiteStore[struct{}](ctx, db, time.Millisecond)
	if err != nil {
		t.Fatalf("NewClaimingSQLiteStore: %v", err)
	}

	now := time.Now().UTC()

	timer := scheduling.Timer[struct{}]{
		ID:     scheduling.MustParseTimerID("crashed"),
		FireAt: now.Add(-time.Second),
	}

	if err := store.Schedule(ctx, timer); err != nil {
		t.Fatalf("Schedule: %v", err)
	}

	if _, err := store.Due(ctx, now); err != nil {
		t.Fatalf("Due (first): %v", err)
	}

	// The 20ms sleep dwarfs the 1ms lease, so real-now is deterministically
	// past lease_until: the lapsed claim cannot be renewed.
	time.Sleep(20 * time.Millisecond)

	if err := store.RenewLease(ctx, timer.ID, time.Minute); err == nil {
		t.Fatal("renewing an expired claim must fail with ErrLeaseNotHeld")
	}

	// After expiry the timer IS claimable again by another poller.
	reclaimed, err := store.Due(ctx, now.Add(10*time.Millisecond))
	if err != nil {
		t.Fatalf("Due (after expiry): %v", err)
	}

	if len(reclaimed) != 1 {
		t.Fatalf("expired lease must be reclaimable, got %d timers", len(reclaimed))
	}
}
