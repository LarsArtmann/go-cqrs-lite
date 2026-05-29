package event

import (
	"context"

	ro "github.com/samber/ro"
)

type EventBus = ro.Subject[Event]

func NewEventBus() ro.Subject[Event] {
	return ro.NewPublishSubject[Event]()
}

func FilterEventType(eventType Type) func(ro.Observable[Event]) ro.Observable[Event] {
	return ro.Filter(func(e Event) bool {
		return e.Type() == eventType
	})
}

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

func HandlerToObserver(handler Handler) ro.Observer[Event] {
	return ro.NewObserver(
		func(e Event) { _ = handler(context.Background(), e) },
		func(_ error) {},
		func() {},
	)
}

func HandlerToObserverWithContext(ctx context.Context, handler Handler) ro.Observer[Event] {
	return ro.NewObserver(
		func(e Event) { _ = handler(ctx, e) },
		func(_ error) {},
		func() {},
	)
}
