package listing

import (
	"context"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
)

// CacheInvalidator clears the cached stream index after successful publish.
// Implemented by *InMemoryStreamReader.
type CacheInvalidator interface {
	InvalidateCache()
}

// CacheInvalidationMiddleware returns a PublishMiddleware that invalidates the
// reader's cache after each successful publish, ensuring subsequent List calls
// reflect the latest events.
//
// Usage:
//
//	reader := listing.NewInMemoryStreamReader(journal)
//	bus.UsePublish(listing.CacheInvalidationMiddleware(reader))
func CacheInvalidationMiddleware(reader CacheInvalidator) event.PublishMiddleware {
	return func(next event.Publisher) event.Publisher {
		return event.PublisherFunc(func(ctx context.Context, events ...event.Event) error {
			err := next.Publish(ctx, events...)
			if err != nil {
				return errorfamily.WrapInfrastructure(err,
					"listing.publish_events_failed",
					"publish events")
			}

			reader.InvalidateCache()

			return nil
		})
	}
}
