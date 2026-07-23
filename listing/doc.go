// Package listing provides CQRS read model capabilities for event-sourced aggregates.
//
// It offers:
//   - Aggregate listing with cursor pagination
//   - Tombstone (soft-delete) detection and filtering
//   - In-memory fallback readers for testing
//   - Bus middleware for automatic tombstone/rebirth marking
//
// The listing module is the read model. It never writes events.
// It queries via Journal (cross-aggregate) or StreamReader (aggregate listings).
//
// Usage:
//
//	// Setup: auto-mark tombstones and rebirths on publish
//	bus.UsePublish(listing.StatusMiddleware(
//	    []event.Type{"user.deleted"},
//	    []event.Type{"user.reactivated"},
//	))
//
//	// List active users (in-memory, for testing)
//	page, err := listing.NewListBuilder(
//	    listing.NewInMemoryStreamReader(journal),
//	).OfType("User").PageSize(20).List(ctx)
//
//	// List with status (includes tombstone state)
//	statusPage, err := listing.NewListBuilder(
//	    listing.NewInMemoryStreamReader(journal),
//	).OfType("User").IncludeDeleted().ListWithStatus(ctx)
package listing
