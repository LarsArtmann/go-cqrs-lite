// Package stream provides CQRS read model capabilities for event-sourced aggregates.
//
// It offers:
//   - Aggregate listing with cursor pagination
//   - Tombstone (soft-delete) detection and filtering
//   - Projection-backed SQL readers for production
//   - In-memory fallback readers for testing
//   - Bus middleware for automatic tombstone/rebirth marking
//
// The stream module is the read model. It never writes events.
// It queries via Journal (cross-aggregate) or AggregateReader (aggregate listings).
//
// Usage:
//
//	// Setup: auto-mark tombstones and rebirths on publish
//	bus.UsePublish(stream.StatusMiddleware(
//	    []event.Type{"user.deleted"},
//	    []event.Type{"user.reactivated"},
//	))
//
//	// List active users (in-memory, for testing)
//	page, err := stream.NewListBuilder(
//	    stream.NewInMemoryAggregateReader(journal),
//	).OfType("User").PageSize(20).List(ctx)
//
//	// List with status (includes tombstone state)
//	statusPage, err := stream.NewListBuilder(
//	    stream.NewInMemoryAggregateReader(journal),
//	).OfType("User").IncludeDeleted().ListWithStatus(ctx)
//
//	// SQL-backed (production)
//	reader := stream.NewSQLAggregateReader(db, "cqrs_")
//	page, err := stream.NewListBuilder(reader).OfType("User").List(ctx)
package stream
