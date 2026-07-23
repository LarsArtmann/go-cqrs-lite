package main

import (
	"context"
	"log/slog"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/projection/v4"
)

// ──────────────────────────────────────────────────────────────────────────
// Deriver — event→command reactions.
//
// When a task is created, the deriver automatically assigns it to the
// default team lead. This demonstrates the reactive CQRS pattern:
// events trigger commands that modify other aggregates (or the same one).
//
// The dispatch is ASYNCHRONOUS to avoid deadlocking the projection
// pipeline (BlockPublishUntilSubscriberAck=true means a synchronous
// dispatch inside a handler would block forever waiting for its own ack).
// ──────────────────────────────────────────────────────────────────────────

const defaultAssignee = "team-lead"

// newDeriverProjection creates a projection that auto-assigns new tasks.
// The command dispatch runs in a goroutine to avoid blocking the event pipeline.
func newDeriverProjection(disp *command.Dispatcher, logger *slog.Logger) projection.Projection {
	return projection.NewProjection(
		"auto-assign",
		func(ctx context.Context, evt event.Event) error {
			bc, err := command.New(cmdAssignTask, evt.StreamID())
			if err != nil {
				return err
			}

			cmd := AssignTaskCmd{
				BasicCommand: bc,
				AssigneeID:   defaultAssignee,
			}

			// Async dispatch: fire-and-forget to avoid deadlock with
			// BlockPublishUntilSubscriberAck=true on the bus.
			go func() {
				if dErr := disp.Dispatch(ctx, cmd); dErr != nil {
					logger.Error("deriver: auto-assign failed",
						"taskID", evt.StreamID(), "error", dErr)
				}
			}()

			return nil
		},
		[]event.Type{evtTaskCreated},
	)
}
