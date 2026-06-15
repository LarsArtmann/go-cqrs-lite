// Package query provides query dispatch with typed results, pagination,
// and middleware chains for CQRS applications.
//
// Queries represent requests for information. They never modify state.
// Handlers return typed results, with optional pagination support.
//
// # Quick Start
//
//	queries := query.NewDispatcher()
//	queries.Register("user.get", func(ctx context.Context, q query.Query) (any, error) {
//	    return getUser(q)
//	})
//	result, err := queries.Dispatch(ctx, q)
//
// # Typed Results
//
// Use DispatchTyped for type-safe result extraction without manual assertions:
//
//	result, err := query.DispatchTyped[*GetUserResult](ctx, queries, q)
//
// # Pagination
//
//	 page := query.NewPagination(1, 20)
//		result := query.NewPaginatedResult(items, total, page)
//		if result.HasNext() { ... }
//
// # Query Persistence (Audit Trail)
//
// PersistedQuery captures every received query with full audit metadata
// (type, payload, received-at timestamp). Use a QueryStore to save and load
// queries for audit — "who queried what data and when?":
//
//	store := memory.NewMemoryQueryStore()
//	q, _ := query.NewPersistedQuery("user.search", payload)
//	store.SaveQuery(ctx, q)
//	recent, _ := store.LoadQueries(ctx, cutoffTime)
//
// For cross-cutting audit, use the QueryJournal interface:
//
//	all, _ := store.ReadAllQueries(ctx)            // all queries, ordered by received_at
//	page, _ := store.ReadQueriesFrom(ctx, lastID, 100) // position-based pagination
package query
