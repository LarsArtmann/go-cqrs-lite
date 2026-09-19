package metaengine

import (
	"context"
	"time"
)

// DedupStore is an optional engine capability (ADR-0142): a
// check-and-set deduplication window with expiry — the shape
// idempotency/sqlstore proved in production (Seen/Record/CheckAndRecord/
// Sweep), promoted to an engine ADT so dedup keys ride engines like every
// other collection.
//
// Semantics (pinned by adttest.AssertDedupStore):
//
//   - DedupCheckAndRecord is the atomic CAS: it returns true (already seen)
//     exactly when a live, unexpired record for the key existed before the
//     call; otherwise it records the key with a fresh ttl window and returns
//     false. Two concurrent calls for one key see exactly one true.
//   - DedupSeen is a read: expired records read as unseen.
//   - DedupSweep deletes expired records and returns how many.
//
// SQL engines implement it via metaengine/claimkit; map-shaped engines via
// [MapDedupStore] (degraded, declared in DegradedADTs).
type DedupStore interface {
	// DedupCheckAndRecord atomically checks whether key is recorded in the
	// collection's dedup window and records it (expiring at now+ttl) when
	// not. Returns true when the key was ALREADY seen (the duplicate
	// signal); false for the first recording and when a previous window
	// expired (an expired window may be re-claimed).
	DedupCheckAndRecord(
		ctx context.Context,
		collection, key string,
		ttl time.Duration,
		now time.Time,
	) (bool, error)

	// DedupSeen reports whether key is currently recorded and unexpired.
	DedupSeen(ctx context.Context, collection, key string, now time.Time) (bool, error)

	// DedupSweep deletes every expired record in the collection and returns
	// the number removed.
	DedupSweep(ctx context.Context, collection string, now time.Time) (int, error)
}

// SupportsDedup reports whether the engine implements [DedupStore] — the
// runtime truth behind the profile's ADTDedup entry.
func SupportsDedup(eng Engine) bool {
	_, ok := eng.(DedupStore)

	return ok
}
