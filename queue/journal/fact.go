// Package journal defines the append-only fact log that records everything
// that ever happened to tasks, exactly as in the donor (go-taskqueue
// internal/journal). All derived views (queue, DLQ, stats) are projections
// over these facts.
package journal

import "time"

// FactType enumerates the kinds of facts the journal records. One fact
// type per lifecycle event; the mutating Store method that emits each is
// named in its doc.
type FactType string

const (
	// Enqueued records a task entering the queue (Enqueue; also RescueDead
	// re-queues carry it with a rescue marker in the detail).
	Enqueued FactType = "task.enqueued"
	// Claimed records a lease being granted (ClaimDue).
	Claimed FactType = "task.claimed"
	// Completed records a successful finish (Complete).
	Completed FactType = "task.completed"
	// Failed records a failed attempt (Fail, FailPermanent) — the fact
	// carries the attempt number, error text and, when supplied, the
	// failure evidence in its Detail.
	Failed FactType = "task.failed"
	// DeadLettered records the move to the dead-letter queue (Fail,
	// FailPermanent); its Detail classifies the death ("exhausted" or
	// "permanent").
	DeadLettered FactType = "task.dead-lettered"
	// Cancelled records a withdrawal (Cancel, CancelOwned, DismissDead,
	// and the reclaim finalize of a cooperatively-cancelled task).
	Cancelled FactType = "task.cancelled"
	// CancelRequested records an operator's request to stop a Running
	// task (CancelRunning). The fact IS the flag — no task-row column
	// mirrors it; the executing worker observes it and finalizes with
	// CancelOwned, an expired lease finalizes it at reclaim.
	CancelRequested FactType = "task.cancel-requested"
	// Released records an expired lease being taken over by a new claim
	// (ClaimDue reclaim path) so the journal explains the owner change.
	Released FactType = "task.released"
	// Requeued records a claim returned to Pending WITHOUT counting an
	// attempt (Requeue — preflight refusals).
	Requeued FactType = "task.requeued"
	// Orphaned records that a Running task's lease expired and NO worker
	// reclaimed it (MarkOrphaned). It is an observation, not a state
	// change: the task stays Running until a ClaimDue reclaim or a human.
	Orphaned FactType = "task.orphaned"
	// Reprioritized records that a PENDING task's priority changed
	// (UpdatePendingPriority), carrying old/new priority, source and
	// reason.
	Reprioritized FactType = "task.reprioritized"
)

// Fact is one immutable observation about one task, appended in the same
// transaction as the state change it records.
type Fact struct {
	Seq     int64     `json:"seq"`
	Time    time.Time `json:"time"`
	TaskID  string    `json:"taskId"`
	Type    FactType  `json:"type"`
	Owner   string    `json:"owner,omitempty"`
	Attempt int       `json:"attempt,omitempty"`
	Error   string    `json:"error,omitempty"`
	// Detail is optional structured evidence (e.g. failure evidence,
	// reprioritize provenance, cancel reason) — opaque bytes the journal
	// stores and returns verbatim.
	Detail []byte `json:"detail,omitempty"`
}

// RequeueEvidence is the structured Detail on Requeued facts: why the
// executor refused to start and how long the task waits before it
// becomes claimable again.
type RequeueEvidence struct {
	Reason  string `json:"reason"`
	RetryIn int64  `json:"retry_in_ms"`
}

// ReprioritizeEvidence is the structured Detail on Reprioritized facts:
// what the priority was, what it became, which source decided, and why.
type ReprioritizeEvidence struct {
	OldPriority int    `json:"old_priority"`
	NewPriority int    `json:"new_priority"`
	Source      string `json:"source"`
	Reason      string `json:"reason,omitempty"`
}
