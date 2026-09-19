package main

import (
	"context"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/projection/v4"
)

// ──────────────────────────────────────────────────────────────────────────
// Deriver — event→command reactions.
//
// When a task is created, the deriver schedules its automatic assignment to
// the default team lead — as a DURABLE job on the engine-backed work queue
// (queue/sqlite, ADR-0142), not a fire-and-forget goroutine: the assignment
// survives crashes, retries with backoff, and dead-letters when exhausted.
// The enqueue is a quick SQL write on the queue database, so it cannot
// deadlock the projection pipeline the way a synchronous dispatch would
// (BlockPublishUntilSubscriberAck=true).
// ──────────────────────────────────────────────────────────────────────────

const defaultAssignee = "team-lead"

// newDeriverProjection creates a projection that auto-assigns new tasks via
// the durable work queue; the worker loop dispatches the command.
//
//nolint:ireturn // factory returning interface is intentional for projection registration
func newDeriverProjection(wq *WorkQueue) projection.Projection {
	//cqrs-lint:ignore(C004) library code or intentional pattern
	return projection.NewProjection(
		"auto-assign",
		func(ctx context.Context, evt event.Event) error {
			return wq.enqueueAssignment(ctx, AssignmentJob{
				TaskID:     evt.StreamID().String(),
				AssigneeID: defaultAssignee,
			})
		},
		[]event.Type{evtTaskCreated},
	)
}
