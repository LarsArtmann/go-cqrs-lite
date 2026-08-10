// Package listing provides CQRS read model capabilities for event-sourced streams.
//
// It offers:
//   - Stream listing with cursor pagination
//   - Event-type-based deletion detection (ADR-0114)
//   - In-memory fallback readers for testing
//   - Bus middleware for cache invalidation after publish
//
// The listing module is the read model. It never writes events.
// It queries via Journal (cross-stream) or StreamReader (stream listings).
//
// Deletion is detected by event type, not metadata stamps (ADR-0114).
// Configure the reader with delete event types:
//
//	reader := listing.NewInMemoryStreamReader(journal,
//	    listing.WithDeleteTypes("user.deleted", "order.cancelled"),
//	)
//
// Usage:
//
//	// List active users (in-memory, for testing)
//	page, err := listing.NewListBuilder(reader).OfType("User").PageSize(20).List(ctx)
//
//	// List with status (includes deletion state)
//	statusPage, err := listing.NewListBuilder(reader).
//	    OfType("User").IncludeDeleted().ListWithStatus(ctx)
package listing
