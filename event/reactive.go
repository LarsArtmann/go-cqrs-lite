package event

import (
	"context"

	ro "github.com/samber/ro"

	"github.com/larsartmann/go-cqrs-lite/id/v2"
)

// EventBus is a reactive subject for event streams.
// Use NewEventBus() to create one. Subscribe with ro.Observer, emit with Next.
type EventBus = ro.Subject[Event]

// NewEventBus creates a new PublishSubject-backed EventBus for broadcasting events to multiple subscribers.
func NewEventBus() ro.Subject[Event] {
	return ro.NewPublishSubject[Event]()
}

// NewReplayEventBus creates a new ReplaySubject-backed EventBus that replays the last n events to new subscribers.
func NewReplayEventBus(n int) ro.Subject[Event] {
	return ro.NewReplaySubject[Event](n)
}

// NewBehaviorEventBus creates a new BehaviorSubject-backed EventBus that replays the latest event to new subscribers.
func NewBehaviorEventBus(initial Event) ro.Subject[Event] {
	return ro.NewBehaviorSubject(initial)
}

// FilterEventType returns an operator that filters an Observable[Event] to only events of the given type.
func FilterEventType(eventType Type) func(ro.Observable[Event]) ro.Observable[Event] {
	return ro.Filter(func(e Event) bool {
		return e.Type() == eventType
	})
}

// FilterEventTypes returns an operator that filters an Observable[Event] to only events of the given types.
func FilterEventTypes(eventTypes ...Type) func(ro.Observable[Event]) ro.Observable[Event] {
	types := newTypeSet(eventTypes)

	return ro.Filter(func(e Event) bool {
		return types.has(e.Type())
	})
}

// ReplayFilter returns an operator that filters an Observable[Event] to only events after the given checkpoint
// and matching the given event types. Used by projection replay.
//
// Not goroutine-safe: the returned operator captures mutable state (checkpoint position) in a closure.
// Each subscription must use its own ReplayFilter instance. For concurrent use, wrap with ro.Serialize.
func ReplayFilter(
	types []Type,
	checkpoint Checkpoint,
) func(ro.Observable[Event]) ro.Observable[Event] {
	typeSet := newTypeSet(types)
	pastCheckpoint := checkpoint.IsZero()

	return ro.Filter(func(e Event) bool {
		if !pastCheckpoint {
			if e.ID() == checkpoint.EventID {
				pastCheckpoint = true
			}

			return false
		}

		if len(types) > 0 && !typeSet.has(e.Type()) {
			return false
		}

		return true
	})
}

// DistinctByEventID returns an operator that suppresses duplicate events by ID.
// When the same event is emitted through multiple paths (e.g. journal replay +
// live bus), this prevents processing the same event twice per subscription.
func DistinctByEventID() func(ro.Observable[Event]) ro.Observable[Event] {
	return ro.DistinctBy(func(e Event) id.EventID { return e.ID() })
}

// DistinctByAggregateID returns an operator that emits only the first event per
// aggregate ID within a subscription. Useful for "latest state per aggregate"
// projections where only the most recent event matters.
func DistinctByAggregateID() func(ro.Observable[Event]) ro.Observable[Event] {
	return ro.DistinctBy(func(e Event) string { return e.AggregateID().String() })
}

// HandlerToObserver converts an event.Handler into a ro.Observer[Event].
// The handler receives the context from the stream (via NextWithContext/SubscribeWithContext).
// If the handler returns an error, the error is forwarded through the observer's error channel
// (ErrorWithContext), terminating this observer's subscription.
func HandlerToObserver(handler Handler) ro.Observer[Event] {
	var obs ro.Observer[Event]
	obs = ro.NewObserverWithContext(
		func(ctx context.Context, e Event) {
			if err := handler(ctx, e); err != nil {
				obs.ErrorWithContext(ctx, err)
			}
		},
		func(_ context.Context, _ error) {},
		func(_ context.Context) {},
	)

	return obs
}

// HandlerToObserverWithContext converts an event.Handler into a ro.Observer[Event]
// using an explicit context for all handler invocations instead of the stream's context.
// Use this when you need a fixed deadline, cancellation signal, or trace context.
// If the handler returns an error, the error is forwarded through the observer's error channel.
func HandlerToObserverWithContext(ctx context.Context, handler Handler) ro.Observer[Event] {
	var obs ro.Observer[Event]
	obs = ro.NewObserverWithContext(
		func(_ context.Context, e Event) {
			if err := handler(ctx, e); err != nil {
				obs.ErrorWithContext(ctx, err)
			}
		},
		func(_ context.Context, _ error) {},
		func(_ context.Context) {},
	)

	return obs
}

// Observable is a named type for event observables, improving discoverability
// over the raw ro.Observable[Event].
type Observable = ro.Observable[Event]

type typeSet map[Type]struct{}

func newTypeSet(types []Type) typeSet {
	if len(types) == 0 {
		return nil
	}

	s := make(typeSet, len(types))
	for _, t := range types {
		s[t] = struct{}{}
	}

	return s
}

func (s typeSet) has(t Type) bool {
	if s == nil {
		return true
	}

	_, ok := s[t]

	return ok
}
