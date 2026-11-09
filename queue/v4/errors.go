package queue

import (
	"errors"

	errorfamily "github.com/larsartmann/go-error-family"
)

var (
	// ErrNotFound is returned when a task ID does not exist.
	ErrNotFound = errors.New("queue: task not found")
	// ErrNoTaskDue is returned by ClaimDue when nothing is claimable
	// right now. It is a poll result, not a failure: workers loop on it.
	ErrNoTaskDue = errors.New("queue: no due task")
	// ErrEmptyType is returned by Enqueue when New.Type is empty
	// (Normalize cannot invent a type; callers must choose an executor).
	ErrEmptyType = errors.New("queue: task type must not be empty")
	// ErrInvalidTransition is returned when a method's precondition on
	// the current status does not hold (e.g. Cancel on a completed task).
	ErrInvalidTransition = errors.New("queue: invalid status transition")
	// ErrDuplicateID is returned when enqueueing a task whose ID already
	// exists.
	ErrDuplicateID = errors.New("queue: duplicate id")
)

// ErrDanglingDep is returned by Enqueue when New.Deps references a task
// ID that does not exist: every dependency must already be enqueued.
// This validation is also the cycle guard — deps are fixed at enqueue and
// a fresh store-minted task ID cannot be referenced by any existing task,
// so a dependency cycle can never close (its last edge would have to
// point at a task that did not exist yet, which this error rejects).
// Classified as Rejection — the caller's template is invalid.
var ErrDanglingDep error = errorfamily.NewRejection(
	"queue.dangling_dep",
	"task depends on an unknown task id",
)

// ErrLeaseNotHeld is returned by Complete/Fail/Heartbeat/CancelOwned when
// the caller no longer owns the task's lease: the task completed or
// disappeared, or the lease expired and another worker may have
// re-claimed it. Classified as Orchestration — a distributed-coordination
// race, not a caller bug (same classification as claiming.ErrLeaseNotHeld).
var ErrLeaseNotHeld error = errorfamily.NewOrchestration(
	"queue.lease_not_held",
	"no live lease for this task (completed, cancelled, or lease expired)",
)
