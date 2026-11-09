package queue

import (
	"time"

	"github.com/larsartmann/go-cqrs-lite/queue/v4/task"
)

// Claim is the lease capability returned by ClaimDue: the claimed task,
// the lease deadline the holder must renew (Heartbeat) or finish before,
// and the claim Token — the unguessable holder proof every finalize call
// (Complete, Fail, FailPermanent, Requeue, Heartbeat, CancelOwned)
// presents. After LeaseUntil, with no renewal, the task becomes claimable
// by any other worker — crash reclaim is the lease predicate, not a
// supervisor — and the reclaim mints a fresh token, so a lapsed holder's
// finalize fails with ErrLeaseNotHeld instead of racing the new owner
// (ADR-0134 fencing tokens; the theft detector IS the finalize path).
type Claim[T any] struct {
	Task       task.Task[T]
	LeaseUntil time.Time
	Token      string
}

// ID returns the claimed task's ID — the handle every finalize call
// (Complete, Fail, Heartbeat, Requeue, CancelOwned) takes, alongside the
// claim's Token.
func (c Claim[T]) ID() task.ID { return c.Task.ID }
