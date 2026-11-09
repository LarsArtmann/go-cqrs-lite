package metaengine

import (
	"context"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"
)

// ErrClaimLeaseNotHeld is returned by [DueClaimer.RenewLease] and
// [FactSink] mutations when the caller no longer owns the row's claim: the
// row completed or disappeared, or the lease expired and another claimer may
// have re-taken it. An expired claim cannot be resurrected — another worker
// may already be processing. Classified as Orchestration — a
// distributed-coordination race, not a caller bug. Mirrors claiming.ErrLeaseNotHeld
// (ADR-0142); SQL runtimes translate between the two.
var ErrClaimLeaseNotHeld error = errorfamily.NewOrchestration(
	"metaengine.claim_lease_not_held",
	"no live claim for this key (completed, canceled, or lease expired)",
)

// DefaultClaimLease is how long a claim lasts when a ClaimDueRequest names no
// duration. A claimed row becomes claimable again only after the lease
// expires, so the lease bounds how long a crashed worker delays the row — and
// how long concurrent workers are guaranteed not to double-process it.
const DefaultClaimLease = time.Minute

// DefaultClaimLimit bounds a ClaimDueRequest with Limit <= 0. Claimers that
// want everything due loop until a batch comes back short.
const DefaultClaimLimit = 512

// DueClaim is one claimable item as returned by [DueClaimer.ClaimDue]: its
// identity, payload, scheduling time, and — after a claim — the lease fence.
// Payload is opaque bytes; the codec is the caller's business (the scheduling
// facade JSON-encodes timers, the queue facade task envelopes).
type DueClaim struct {
	// Key uniquely identifies the item within its collection.
	Key string

	// DueAt is when the item became claimable (NotBefore gating: ClaimDue
	// never returns items whose DueAt is after Now).
	DueAt time.Time

	// Payload is the caller's opaque value bytes, stored as given.
	Payload []byte

	// LeaseUntil is the claim fence set by ClaimDue; zero on an unclaimed
	// item. While after Now, no other claimer can take the item.
	LeaseUntil time.Time
}

// ClaimDueRequest parameterizes [DueClaimer.ClaimDue]. A struct (not
// positional args) so the surface grows additively.
type ClaimDueRequest struct {
	// Collection scopes the claim (the claim keyspace's name).
	Collection string

	// Owner identifies the claimer stamped into the claim; only the owner
	// may renew. Empty is allowed (anonymous claims) but forfeits
	// owner-checked renewal.
	Owner string

	// Lease is how long the claim fences other claimers. <=0 means
	// [DefaultClaimLease].
	Lease time.Duration

	// Limit caps the number of items claimed in one call. <=0 means
	// [DefaultClaimLimit].
	Limit int

	// Now is the claim's reference time (due gating and lease-expiry
	// reclaim). Zero means time.Now(). Tests inject a fixed clock here.
	Now time.Time
}

func (r ClaimDueRequest) resolvedNow() time.Time {
	if r.Now.IsZero() {
		return time.Now()
	}

	return r.Now
}

func (r ClaimDueRequest) resolvedLease() time.Duration {
	if r.Lease <= 0 {
		return DefaultClaimLease
	}

	return r.Lease
}

func (r ClaimDueRequest) resolvedLimit() int {
	if r.Limit <= 0 {
		return DefaultClaimLimit
	}

	return r.Limit
}

// DueClaimer is an optional engine capability (ADR-0142): atomic,
// lease-fenced claiming of due items — the primitive timers and task queues
// are built on. It merges claim/lease and time-ordered due-claiming because
// they are one operation in every real consumer: a due-claim IS a claim gated
// on a timestamp with lease fencing.
//
// Contract (pinned by adttest.AssertDueClaimer):
//
//   - ClaimInsert is idempotent by key: an existing not-yet-completed item is
//     left untouched.
//   - ClaimDue returns only items whose DueAt <= Now and whose lease (if any)
//     has lapsed, ordered by DueAt ascending (key as tie-breaker), and stamps
//     a fresh lease on exactly the returned items. Two concurrent claimers
//     never receive the same item while its lease is fresh; a crashed
//     claimer's items become claimable again after lease expiry
//     (at-least-once).
//   - RenewLease extends the lease only while held by owner; otherwise
//     [ErrClaimLeaseNotHeld].
//
// Semantics source of truth: queue.Store.ClaimDue and the claiming/ SQL core
// (production-proven). SQL engines get them via metaengine/claimkit;
// map-shaped engines via [MapDueClaimer] (degraded O(N) scan, declared in
// DegradedADTs).
type DueClaimer interface {
	// ClaimInsert schedules a claimable item (due at dueAt). If an item with
	// the same key already exists and has not been deleted, this is a no-op.
	ClaimInsert(ctx context.Context, collection, key string, dueAt time.Time, payload []byte) error

	// ClaimDue atomically claims due items per the request.
	ClaimDue(ctx context.Context, req ClaimDueRequest) ([]DueClaim, error)

	// RenewLease extends the claim's lease by extend from now. Returns
	// [ErrClaimLeaseNotHeld] when the claim is gone, expired, or owned by
	// another owner.
	RenewLease(
		ctx context.Context,
		collection, key, owner string,
		extend time.Duration,
		now time.Time,
	) error

	// ClaimDelete removes an item unconditionally and idempotently (absent
	// is success). Covers timer Cancel and queue Complete-style finalization
	// after a claim.
	ClaimDelete(ctx context.Context, collection, key string) error

	// ClaimDeleteIfDue removes an item only when it still carries the given
	// dueAt — the epoch guard that kills the re-schedule race: a stale
	// MarkFired for generation N cannot delete a re-scheduled generation
	// N+1 under the same key. Returns ErrClaimLeaseNotHeld-free semantics:
	// a no-op delete (wrong epoch or absent) is NOT an error.
	ClaimDeleteIfDue(ctx context.Context, collection, key string, dueAt time.Time) error
}

// ClaimFact is one journal fact that rides a claim transition atomically when
// the engine implements [FactSink]: a state change without its fact did not
// happen (the queue contract's invariant #1).
type ClaimFact struct {
	// Type names the fact ("claimed", "completed", "dead-lettered").
	Type string

	// Payload is the fact's opaque detail bytes.
	Payload []byte
}

// FactSink is an optional [DueClaimer] extension for engines whose claim
// transitions can carry facts in the SAME transaction. Implemented by the
// claimkit SQL runtime (T14); map-shaped engines do not offer it (two
// engine-method calls cannot share one storage transaction) and say so by not
// implementing the interface.
type FactSink interface {
	// ClaimDueFacts claims due items per req and, inside the same
	// transaction, appends factFor(claim)'s facts for each claimed item.
	// factFor may return nil to record nothing for an item.
	ClaimDueFacts(
		ctx context.Context,
		req ClaimDueRequest,
		factFor func(DueClaim) []ClaimFact,
	) ([]DueClaim, error)

	// ClaimDeleteFacts performs the epoch-guarded delete of
	// [DueClaimer.ClaimDeleteIfDue] and appends facts inside the same
	// transaction. Returns whether the delete matched (false = wrong epoch
	// or absent; NOT an error, same as ClaimDeleteIfDue).
	ClaimDeleteFacts(
		ctx context.Context,
		collection, key string,
		dueAt time.Time,
		facts ...ClaimFact,
	) (bool, error)
}

// SupportsDueClaims reports whether the engine implements [DueClaimer].
// The profile's ADTDueClaim entry declares the capability for routing and
// diagnostics; this probe is the runtime truth.
func SupportsDueClaims(eng Engine) bool {
	_, ok := eng.(DueClaimer)

	return ok
}
