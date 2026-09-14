package queue

import "time"

// Claim is the lease capability returned by ClaimDue: the claimed task
// plus the lease deadline the holder must renew (Heartbeat) or finish
// before. After LeaseUntil, with no renewal, the task becomes claimable
// by any other worker — crash reclaim is the lease predicate, not a
// supervisor.
//
// The claim is currently carried by the (task ID, owner) pair, exactly as
// in the donor. A token field (minted per claim, presented by finalize
// calls, theft-detecting) is the planned ADR-0134 upgrade; the struct is
// the seam so engines can add it without changing call sites.
type Claim[T any] struct {
	Task       Task[T]
	LeaseUntil time.Time
}

// ID returns the claimed task's ID — the handle every finalize call
// (Complete, Fail, Heartbeat, Requeue, CancelOwned) takes.
func (c Claim[T]) ID() ID { return c.Task.ID }
