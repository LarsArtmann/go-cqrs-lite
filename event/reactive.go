package event

import (
	"context"

	ro "github.com/samber/ro"
)

// EventBus is a reactive subject for event streams.
// Use NewEventBus() to create one. Subscribe with ro.Observer, emit with Next.
type EventBus = ro.Subject[Event]

// NewEventBus creates a new PublishSubject-backed EventBus for broadcasting events to multiple subscribers.
func NewEventBus() ro.Subject[Event] {
	return ro.NewPublishSubject[Event]()
}

// FilterEventType returns an operator that filters an Observable[Event] to only events of the given type.
func FilterEventType(eventType Type) func(ro.Observable[Event]) ro.Observable[Event] {
	return ro.Filter(func(e Event) bool {
		return e.Type() == eventType
	})
}

// FilterEventTypes returns an operator that filters an Observable[Event] to only events of the given types.
func FilterEventTypes(eventTypes ...Type) func(ro.Observable[Event]) ro.Observable[Event] {
	types := make(map[Type]struct{}, len(eventTypes))
	for _, t := range eventTypes {
		types[t] = struct{}{}
	}

	return ro.Filter(func(e Event) bool {
		_, ok := types[e.Type()]

		return ok
	})
}

// HandlerToObserver converts an event.Handler into a ro.Observer[Event].
// The handler's error return is forwarded to the onError callback — never silently dropped.
// The event's own Context() is passed to the handler (not context.Background()),
// preserving correlation IDs, tracing spans, and deadlines.
func HandlerToObserver(handler Handler, onError func(error)) ro.Observer[Event] {
	return ro.NewObserver(
		func(e Event) {
			if err := handler(e.Context(), e); err != nil {
				onError(err)
			}
		},
		func(err error) { onError(err) },
		func() {},
	)
}

// HandlerToObserverWithContext converts an event.Handler into a ro.Observer[Event]
// using an explicit context instead of the event's own context.
// Use this when you need to override the context (e.g., a fixed timeout or cancellation signal).
// The handler's error return is forwarded to onError.
func HandlerToObserverWithContext(ctx context.Context, handler Handler, onError func(error)) ro.Observer[Event] {
	return ro.NewObserver(
		func(e Event) {
			if err := handler(ctx, e); err != nil {
				onError(err)
			}
		},
		func(err error) { onError(err) },
		func() {},
	)
}
