package queue

import (
	"context"
	"encoding/json/v2"
	"time"

	"github.com/larsartmann/go-cqrs-lite/queue/v4/facts"
	"github.com/larsartmann/go-cqrs-lite/queue/v4/task"
)

// Store is the persistence boundary for tasks and their facts. Engines
// (queue/sqlite, queue/postgres, queue/mysql) implement this contract and
// are held to identical semantics by the shared conformance suite
// (queue/conformance).
//
// Two invariants bind every implementation:
//
//   - Facts in the same transaction: every method that mutates task state
//     appends its fact(s) inside the SAME transaction — a state change
//     without its fact did not happen, and the journal can never disagree
//     with the task table.
//   - Claims are exclusive by lease: ClaimDue grants a lease until a
//     deadline; a second claimer can take the task only after the lease
//     expires (crash reclaim) or the holder releases it (Complete, Fail,
//     Requeue, CancelOwned).
type Store[T any] interface {
	// Enqueue persists a new task (ID and defaults assigned here) and
	// records the facts.Enqueued fact. When task.New.DedupKey is set
	// and a task with that key already exists — in ANY status — the
	// stored task is returned unchanged: no duplicate row, no duplicate
	// fact. An empty Type is refused with ErrEmptyType.
	Enqueue(ctx context.Context, n task.New[T]) (task.Task[T], error)

	// ClaimDue atomically claims at most one due task for owner: pending
	// with NotBefore passed, or running with an expired lease (crash
	// reclaim), with every dependency completed, ordered by effective
	// priority (stored priority + bounded age bonus, see
	// PriorityAgingDaysPerPoint) then age. Sets Running + lease and
	// records the facts.Claimed fact (a reclaim first records
	// facts.Released for the previous owner). Returns ErrNoTaskDue when
	// nothing is claimable.
	ClaimDue(ctx context.Context, owner string, lease time.Duration) (Claim[T], error)

	// Complete marks a Running task Completed (lease must be held) and
	// records the facts.Completed fact, carrying the result when
	// non-empty.
	Complete(ctx context.Context, id task.ID, owner string, result []byte) error

	// Fail records a failed attempt. When attempts remain, the task
	// returns to Pending with NotBefore = now + backoff; otherwise it is
	// dead-lettered. Facts: facts.Failed (attempt number, error,
	// evidence in Detail) plus facts.DeadLettered with class
	// "exhausted" on the final attempt.
	Fail(
		ctx context.Context,
		id task.ID,
		owner string,
		errText string,
		backoff time.Duration,
		evidence []byte,
	) error

	// FailPermanent dead-letters a Running task immediately, regardless
	// of the attempt budget: the error class makes retrying pointless.
	// The attempt is still counted. Facts: facts.Failed (carrying
	// evidence) + facts.DeadLettered with class "permanent".
	FailPermanent(
		ctx context.Context,
		id task.ID,
		owner string,
		errText string,
		evidence []byte,
	) error

	// Requeue returns a claimed task to Pending WITHOUT counting an
	// attempt; it becomes claimable again after delay. For preflight
	// refusals: the environment was not ready, not the task. Fact:
	// facts.Requeued carrying facts.RequeueEvidence.
	Requeue(
		ctx context.Context,
		id task.ID,
		owner string,
		errText string,
		delay time.Duration,
	) error

	// Heartbeat extends the lease of a Running task held by owner. An
	// expired or foreign lease affects zero rows and returns
	// ErrLeaseNotHeld — an expired claim cannot be resurrected because
	// another worker may already be processing.
	Heartbeat(ctx context.Context, id task.ID, owner string, extend time.Duration) error

	// Cancel withdraws a Pending task. A non-empty reason is stored in
	// the facts.Cancelled fact's Detail ("reason" key).
	Cancel(ctx context.Context, id task.ID, reason string) error

	// CancelRunning records a cooperative cancel request for a Running
	// task: the facts.CancelRequested fact is the flag. The executing
	// worker observes it (CancelRequested), stops the execution, and
	// finalizes with CancelOwned; an expired lease finalizes it at
	// reclaim. A non-empty reason rides the request fact's Detail and is
	// carried onto the final Cancelled fact. Idempotent — a second
	// request appends nothing.
	CancelRunning(ctx context.Context, id task.ID, reason string) error

	// CancelRequested reports whether a cooperative cancel request is
	// pending for the task — the worker's heartbeat observation query.
	CancelRequested(ctx context.Context, id task.ID) (bool, error)

	// CancelOwned finalizes a cooperative cancel: Running → Cancelled,
	// recorded by the lease-holding worker after it stopped the
	// execution.
	CancelOwned(ctx context.Context, id task.ID, owner string) error

	// MarkOrphaned appends a facts.Orphaned fact for every Running task
	// whose lease expired before the cutoff and that has no Orphaned
	// fact yet (idempotent). It changes no state — orphans stay Running
	// until a reclaim — it records WHY the task is stranded so the
	// journal can explain it. Returns how many facts were appended.
	MarkOrphaned(ctx context.Context, cutoff time.Time) (int, error)

	// RescueDead re-queues a Dead task with a fresh attempt budget
	// (maxAttempts <= 0 means task.DefaultMaxAttempts) and records an
	// Enqueued fact carrying a rescue marker.
	RescueDead(ctx context.Context, id task.ID, maxAttempts int) error

	// DismissDead cancels a Dead task with a recorded reason (DLQ
	// dismiss): the facts.Cancelled fact's Detail carries the reason
	// and who dismissed it. Dead is terminal otherwise; facts are never
	// deleted.
	DismissDead(ctx context.Context, id task.ID, reason string, by string) error

	// UpdatePendingPriority changes a PENDING task's priority and records
	// the facts.Reprioritized fact (old/new, source, reason) in the
	// SAME transaction. Running/terminal tasks are refused with
	// ErrInvalidTransition — priority is enqueue-time truth for anything
	// already claimed or finished. A same-value update is a no-op: no
	// error, no fact (idempotency: reruns and racing re-prioritizers
	// never spam the journal).
	UpdatePendingPriority(
		ctx context.Context,
		id task.ID,
		newPriority int,
		source string,
		reason string,
	) error

	// Get returns the current task record.
	Get(ctx context.Context, id task.ID) (task.Task[T], error)

	// List returns tasks matching the filter, default-ordered by stored
	// priority descending then creation age ascending (the queue's
	// fairness order, without the aging bonus — aging is scheduling, not
	// display state).
	List(ctx context.Context, f Filter) ([]task.Task[T], error)

	// CountTasks counts the tasks matching the filter — the COUNT(*)
	// pushdown behind pagination.
	CountTasks(ctx context.Context, f Filter) (int, error)

	// StatusCounts counts tasks per status — the GROUP BY behind
	// dashboard counters.
	StatusCounts(ctx context.Context) (map[task.Status]int, error)

	// Facts exposes the journal: facts with Seq strictly greater than
	// after, in Seq order. limit bounds the result when > 0; 0 means
	// unbounded (bulk exports).
	Facts(ctx context.Context, after int64, limit int) ([]facts.Fact, error)

	// FactsForTask returns one task's facts in Seq order, bounded to the
	// most recent limit when > 0 (0 = unbounded).
	FactsForTask(ctx context.Context, id task.ID, limit int) ([]facts.Fact, error)

	// HeadSeq returns the current highest fact Seq (0 when the journal is
	// empty): the O(1) watermark for tailers, bridges and resume points.
	HeadSeq(ctx context.Context) (int64, error)

	// Watermark returns the persisted read cursor for a journal consumer
	// and whether the consumer ever checkpointed (seq 0 is a valid
	// cursor: "consumed nothing yet"): the resume point for bridges and
	// sweepers after a restart.
	Watermark(ctx context.Context, consumer string) (seq int64, exists bool, err error)

	// SaveWatermark checkpoints a consumer cursor as a monotonic upsert
	// (never regresses). It records consumer progress, not task state, so
	// no fact is appended.
	SaveWatermark(ctx context.Context, consumer string, seq int64) error

	// Close releases resources.
	Close() error
}

// Filter selects tasks for List and CountTasks. The zero filter matches
// everything (bounded by Limit when set).
type Filter struct {
	Project *string
	Status  *task.Status
	Type    *string
	// Query is a case-insensitive substring search over id, type,
	// project, payload, lease owner and last error — pushed into SQL
	// LIKE, not a post-filter.
	Query string
	Limit int
	// Offset skips the first Offset matches (pagination); applied after
	// ordering. Meaningful together with Limit.
	Offset int
	// Since restricts the listing to tasks created at or after this time
	// (inclusive).
	Since *time.Time
	// Parked restricts the listing to parked tasks: pending with
	// not_before in the future (the one-glance "waiting on rate limits"
	// view). false or nil leaves the filter off.
	Parked *bool
	// PriorityMin/PriorityMax bound the listing to a STORED-priority
	// range. nil leaves the bound open. Aging is scheduling, not state:
	// the range sees the stored value, not the effective rank.
	PriorityMin *int
	PriorityMax *int
}

// Codec serializes task payloads for storage. Engines default to
// JSONCodec; a custom codec is how an application pins its wire format
// (e.g. CBOR) without the queue knowing the domain type.
type Codec[T any] struct {
	Encode func(T) ([]byte, error)
	Decode func([]byte) (T, error)
}

// JSONCodec returns the default payload codec: encoding/json round-trips.
// A zero-length payload column decodes to T's zero value, so tasks
// enqueued without a payload read back cleanly.
func JSONCodec[T any]() Codec[T] {
	return Codec[T]{
		Encode: func(v T) ([]byte, error) { return json.Marshal(v) },
		Decode: func(b []byte) (T, error) {
			var zero T
			if len(b) == 0 {
				return zero, nil
			}

			var v T
			if err := json.Unmarshal(b, &v); err != nil {
				return zero, err
			}

			return v, nil
		},
	}
}
