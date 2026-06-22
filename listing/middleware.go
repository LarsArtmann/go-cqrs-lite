package listing

import (
	"context"

	"github.com/larsartmann/go-cqrs-lite/event/v3"
)

// StatusMiddleware returns PublishMiddleware that marks tombstone/rebirth metadata
// on events whose type is in the configured sets.
//
// Usage:
//
//	bus.UsePublish(listing.StatusMiddleware(
//	    []event.Type{"user.deleted", "order.cancelled"},   // tombstone types
//	    []event.Type{"user.reactivated", "order.restored"}, // rebirth types
//	))
func StatusMiddleware(deleteTypes, rebirthTypes []event.Type) event.PublishMiddleware {
	deletes := makeTypeSet(deleteTypes)
	rebirths := makeTypeSet(rebirthTypes)

	return func(next event.Publisher) event.Publisher {
		return event.PublisherFunc(func(ctx context.Context, events ...event.Event) error {
			marked := make([]event.Event, 0, len(events))

			for _, evt := range events {
				_, isDelete := deletes[evt.Type()]
				_, isRebirth := rebirths[evt.Type()]

				switch {
				case isDelete:
					m, err := event.MarkTombstone(evt)
					if err != nil {
						return event.WrapInfrastructure(
							err,
							"listing.tombstone",
							"status middleware tombstone "+string(evt.Type()),
						)
					}

					marked = append(marked, m)
				case isRebirth:
					m, err := event.MarkRebirth(evt)
					if err != nil {
						return event.WrapInfrastructure(
							err,
							"listing.rebirth",
							"status middleware rebirth "+string(evt.Type()),
						)
					}

					marked = append(marked, m)
				default:
					marked = append(marked, evt)
				}
			}

			return next.Publish(ctx, marked...)
		})
	}
}

// CacheInvalidator clears the cached aggregate index after successful publish.
// Implemented by *InMemoryAggregateReader.
type CacheInvalidator interface {
	InvalidateCache()
}

// CacheInvalidationMiddleware returns a PublishMiddleware that invalidates the
// reader's cache after each successful publish, ensuring subsequent List calls
// reflect the latest events.
//
// Usage:
//
//	reader := listing.NewInMemoryAggregateReader(journal)
//	bus.UsePublish(listing.CacheInvalidationMiddleware(reader))
func CacheInvalidationMiddleware(reader CacheInvalidator) event.PublishMiddleware {
	return func(next event.Publisher) event.Publisher {
		return event.PublisherFunc(func(ctx context.Context, events ...event.Event) error {
			err := next.Publish(ctx, events...)
			if err != nil {
				return event.WrapInfrastructure(err,
					"listing.publish_events_failed",
					"publish events")
			}

			reader.InvalidateCache()

			return nil
		})
	}
}

func makeTypeSet(types []event.Type) map[event.Type]struct{} {
	set := make(map[event.Type]struct{}, len(types))

	for _, t := range types {
		set[t] = struct{}{}
	}

	return set
}
