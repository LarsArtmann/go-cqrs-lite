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
	return ro.Filter[Event](func(e Event) bool {
		return e.Type() == eventType
	})
}

func FilterEventTypes(eventTypes ...Type) func(ro.Observable[Event]) ro.Observable[Event] {
	types := make(map[Type]struct{}, len(eventTypes))
	for _, t := range eventTypes {
		types[t] = struct{}{}
	}

	return ro.Filter[Event](func(e Event) bool {
		_, ok := types[e.Type()]
		return ok
	})
}

type EventHandler func(ctx context.Context, event Event) error

func HandlerToObserver(handler EventHandler) ro.Observer[Event] {
	return ro.NewObserver[Event](
		func(e Event) { _ = handler(context.Background(), e) },
		func(err error) {},
		func() {},
	)
}

func HandlerToObserverWithContext(ctx context.Context, handler EventHandler) ro.Observer[Event] {
	return ro.NewObserver[Event](
		func(e Event) { _ = handler(ctx, e) },
		func(err error) {},
		func() {},
	)
}
