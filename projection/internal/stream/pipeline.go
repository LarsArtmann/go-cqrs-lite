package stream

import (
	"context"
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/core/event"
	ro "github.com/samber/ro"
)

// ProcessAll runs events through a ro.Pipe pipeline with filtering and
// invokes the handler for each event. Returns the first error encountered.
func ProcessAll(
	ctx context.Context,
	events []event.Event,
	types []event.Type,
	handler func(context.Context, event.Event) error,
) error {
	if len(events) == 0 {
		return nil
	}

	var firstErr error

	ops := []any{}

	if len(types) > 0 {
		ops = append(ops, FilterByType(types...))
	}

	ops = append(ops, ro.Tap[event.Event](
		func(evt event.Event) {
			if firstErr != nil {
				return
			}

			err := handler(ctx, evt)
			if err != nil {
				firstErr = fmt.Errorf("handle event %s: %w", evt.Type(), err)
			}
		},
		nil,
		nil,
	))

	pipeline := ro.Pipe[event.Event, event.Event](
		ro.FromSlice(events),
		ops...,
	)

	sub := pipeline.Subscribe(ro.OnNext(func(_ event.Event) {}))
	sub.Wait()

	return firstErr
}
