package stack

import (
	"context"
	"fmt"
	"log/slog"
	"slices"

	cqrsevent "github.com/larsartmann/go-cqrs-lite/event/v3"
	cqrsprojection "github.com/larsartmann/go-cqrs-lite/projection/v3"
	cqrswatermill "github.com/larsartmann/go-cqrs-lite/watermill/v3"
)

// RunProjections runs the given projections against the Bundle's event stream.
// It replays historical events from the journal, then subscribes to live
// events, dispatching each to every registered projection. Blocks until ctx
// is cancelled.
//
// This is the one-call replacement for the manual CatchUpSubscriber + channel
// consumption + message decoding + projection dispatch boilerplate (~20 lines
// in the deployer-first example). All projections share one checkpoint and
// process events sequentially in the same goroutine — ordering is guaranteed.
//
// Each projection's Handle is called for every event whose type matches
// proj.EventTypes() (nil EventTypes = handle all types). If any projection
// returns an error, the event is NOT acked — the message will be redelivered.
// This provides at-least-once delivery semantics.
//
// Usage:
//
//	err := bundle.RunProjections(ctx, mat, relationalProj, graphProj)
//	// blocks until ctx.Done()
func (b *Bundle) RunProjections(
	ctx context.Context,
	projections ...cqrsprojection.Projection,
) error {
	if len(projections) == 0 {
		return ErrMissingProjection
	}

	catchUp, err := b.CatchUpSubscriber()
	if err != nil {
		return fmt.Errorf("run projections: %w", err)
	}

	msgs, err := catchUp.Subscribe(ctx, cqrswatermill.DefaultEventBusTopic)
	if err != nil {
		return fmt.Errorf("run projections: subscribe: %w", err)
	}

	defer func() { _ = catchUp.Close() }()

	for {
		select {
		case msg, ok := <-msgs:
			if !ok {
				return nil // channel closed
			}

			evt, decodeErr := cqrswatermill.MessageToEvent(
				msg.Metadata.Get("event_type"), msg,
			)
			if decodeErr != nil {
				return fmt.Errorf("run projections: decode message: %w", decodeErr)
			}

			for _, proj := range projections {
				if !shouldHandle(proj, evt) {
					continue
				}

				if handleErr := proj.Handle(ctx, evt); handleErr != nil {
					slog.Warn(
						"projection handler error",
						"projection", proj.Name(),
						"event_id", evt.ID().String(),
						"event_type", string(evt.Type()),
						"error", handleErr,
					)

					return fmt.Errorf("run projections: %s handle %s: %w",
						proj.Name(), evt.ID().String(), handleErr)
				}
			}

			msg.Ack()

		case <-ctx.Done():
			return fmt.Errorf("run projections: %w", ctx.Err())
		}
	}
}

// shouldHandle returns true if the projection should process this event type.
// A nil EventTypes slice means "handle all types" (Materialize's default).
func shouldHandle(proj cqrsprojection.Projection, evt cqrsevent.Event) bool {
	types := proj.EventTypes()
	if len(types) == 0 {
		return true
	}

	return slices.Contains(types, evt.Type())
}

// ErrMissingProjection is returned when RunProjections is called with no projections.
var ErrMissingProjection = missingError("no projections provided")

type missingError string

func (e missingError) Error() string { return "stack: " + string(e) }
