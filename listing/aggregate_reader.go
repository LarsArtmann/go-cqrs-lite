package listing

import (
	"context"

	errorfamily "github.com/larsartmann/go-error-family"
)

// AggregateReader queries aggregate streams.
// Implementations may query projected tables, the events table,
// or enumerate via Journal.
type AggregateReader interface {
	// List returns a page of aggregate references.
	// Tombstoned aggregates are excluded by default (TombstoneExclude).
	List(ctx context.Context, opts ListOptions) (*Page[AggregateListing], error)

	// ListWithStatus returns aggregates with their computed tombstone status.
	// Use this when you need to know which aggregates are tombstoned.
	ListWithStatus(ctx context.Context, opts ListOptions) (*Page[AggregateStatus], error)
}

// ListRefsFromStatus delegates to ListWithStatus and strips the status,
// returning only the AggregateListing page. Both InMemoryAggregateReader
// and SQLAggregateReader use this for their List implementation.
func ListRefsFromStatus(
	r AggregateReader,
	ctx context.Context,
	opts ListOptions,
) (*Page[AggregateListing], error) {
	statusPage, err := r.ListWithStatus(ctx, opts)
	if err != nil {
		return nil, errorfamily.WrapInfrastructure(err, "listing.list_with_status",
			"list with status")
	}

	refs := make([]AggregateListing, len(statusPage.Items))
	for i, s := range statusPage.Items {
		refs[i] = s.Ref
	}

	return &Page[AggregateListing]{Items: refs, HasMore: statusPage.HasMore}, nil
}
