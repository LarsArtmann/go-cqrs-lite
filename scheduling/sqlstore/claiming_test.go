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
		t.Fatalf("second claimer got %d timers while leases fresh, want 0 (double fire)", len(second))
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
	if _, err := sqlstore.NewClaimingMySQLStore[struct{}](nil, nil, 0); err == nil {
		t.Fatal("NewClaimingMySQLStore succeeded, want rejection")
	}
}
