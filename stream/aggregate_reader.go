package stream

import "context"

// AggregateReader queries aggregate streams.
// Implementations may query projected tables, the events table,
// or enumerate via Journal.
type AggregateReader interface {
	// List returns a page of aggregate references.
	// Tombstoned aggregates are excluded by default (TombstoneExclude).
	List(ctx context.Context, opts ListOptions) (*Page[AggregateRef], error)

	// ListWithStatus returns aggregates with their computed tombstone status.
	// Use this when you need to know which aggregates are tombstoned.
	ListWithStatus(ctx context.Context, opts ListOptions) (*Page[AggregateStatus], error)
}
