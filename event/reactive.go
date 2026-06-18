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
//
// For bridging replay→live dedup (where you need to seed the seen-set with
// IDs from a prior phase), use DistinctByEventIDWith instead.
func DistinctByEventID() func(ro.Observable[Event]) ro.Observable[Event] {
	return DistinctByEventIDWith(nil)
}

// DistinctByEventIDWith returns an operator that suppresses duplicate events
// by ID, pre-seeded with already-seen IDs. Pass the event IDs from a prior
// phase (e.g. journal replay) so the live stream skips them automatically:
//
//	deduped := ro.Pipe1(live, event.DistinctByEventIDWith(replayIDs))
//
// A nil seed map is equivalent to DistinctByEventID().
func DistinctByEventIDWith(
	seen map[id.EventID]struct{},
) func(ro.Observable[Event]) ro.Observable[Event] {
	return func(source ro.Observable[Event]) ro.Observable[Event] {
		return ro.NewUnsafeObservableWithContext(
			func(subscriberCtx context.Context, destination ro.Observer[Event]) ro.Teardown {
				localSeen := make(map[id.EventID]struct{}, len(seen))

				for k := range seen {
					localSeen[k] = struct{}{}
				}

				sub := source.SubscribeWithContext(
					subscriberCtx,
					ro.NewObserverWithContext(
						func(ctx context.Context, evt Event) {
							eid := evt.ID()

							if _, ok := localSeen[eid]; ok {
								return
							}

							localSeen[eid] = struct{}{}

							destination.NextWithContext(ctx, evt)
						},
						destination.ErrorWithContext,
						destination.CompleteWithContext,
					),
				)

				return sub.Unsubscribe
			},
		)
	}
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

// SubscriberToObservable adapts a callback-based Subscriber into a reactive
// Observable. Each subscription calls SubscribeAll internally; events are
// forwarded to the observer via NextWithContext.
//
// When the subscription context is cancelled (or the subscription is
// unsubscribed), the internal handler becomes a no-op: it checks ctx.Err()
// and the observer's IsClosed() before forwarding.
//
// Note: the underlying Subscriber (e.g. memory.MemoryBus) does not support
// handler removal, so the handler closure stays registered for the lifetime
// of the Subscriber. This is acceptable for testing and single-process
// deployments. Production message buses should implement their own
// Observable adapter with proper cleanup.
func SubscriberToObservable(sub Subscriber) ro.Observable[Event] {
	return ro.NewObservableWithContext(
		func(ctx context.Context, dest ro.Observer[Event]) ro.Teardown {
			handler := func(handlerCtx context.Context, evt Event) error {
				if ctx.Err() != nil {
					return nil
				}

				if !dest.IsClosed() {
					dest.NextWithContext(handlerCtx, evt)
				}

				return nil
			}

			err := sub.SubscribeAll(handler)
			if err != nil {
				dest.ErrorWithContext(ctx, err)
			}

			return nil
		},
	)
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
