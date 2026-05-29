package listing

import (
	"context"
	"slices"
	"strings"

	"github.com/larsartmann/go-cqrs-lite/event"
	"github.com/larsartmann/go-cqrs-lite/id"
)

// InMemoryAggregateReader implements AggregateReader using a Journal.
// Loads ALL events and filters in-memory. Suitable for testing
// and small datasets.
type InMemoryAggregateReader struct {
	journal event.Journal
}

var _ AggregateReader = (*InMemoryAggregateReader)(nil)

// NewInMemoryAggregateReader creates a reader that enumerates via Journal.ReadAll.
func NewInMemoryAggregateReader(journal event.Journal) *InMemoryAggregateReader {
	return &InMemoryAggregateReader{journal: journal}
}

func (r *InMemoryAggregateReader) List(
	ctx context.Context,
	opts ListOptions,
) (*Page[AggregateRef], error) {
	return listRefsFromStatus(r, ctx, opts)
}

func (r *InMemoryAggregateReader) ListWithStatus(
	ctx context.Context,
	opts ListOptions,
) (*Page[AggregateStatus], error) {
	all, err := r.journal.ReadAll(ctx)
	if err != nil {
		return nil, event.WrapInfrastructure(
			err,
			"listing.in_memory_list",
			"stream in-memory list",
		)
	}

	refs := buildRefs(all)

	if opts.Type != "" {
		refs = filterByType(refs, opts.Type)
	}

	refs = applyTombstonePolicy(refs, opts.Tombstone)

	slices.SortFunc(refs, func(a, b AggregateStatus) int {
		if a.Ref.Type != b.Ref.Type {
			return strings.Compare(string(a.Ref.Type), string(b.Ref.Type))
		}

		return strings.Compare(a.Ref.ID.String(), b.Ref.ID.String())
	})

	refs = applyCursor(refs, opts.After)

	return paginateStatus(refs, opts.Limit), nil
}

func buildRefs(events []event.Event) []AggregateStatus {
	type streamKey struct {
		aggType event.AggregateType
		aggID   id.AggregateID
	}

	type streamBuilder struct {
		ref        AggregateRef
		lastEvents []event.Event
	}

	builders := make(map[streamKey]*streamBuilder)

	for _, evt := range events {
		key := streamKey{aggType: evt.AggregateType(), aggID: evt.AggregateID()}

		b, ok := builders[key]
		if !ok {
			b = &streamBuilder{
				ref: AggregateRef{
					ID:   evt.AggregateID(),
					Type: evt.AggregateType(),
				},
			}
			builders[key] = b
		}

		b.ref.Version = evt.Version()
		b.ref.EventCount++
		b.ref.LastEventAt = evt.OccurredAt()
		b.lastEvents = append(b.lastEvents, evt)
	}

	result := make([]AggregateStatus, 0, len(builders))

	for _, b := range builders {
		result = append(result, AggregateStatus{
			Ref:    b.ref,
			Status: event.DetectTombstone(b.lastEvents),
		})
	}

	return result
}

func filterByType(refs []AggregateStatus, aggregateType event.AggregateType) []AggregateStatus {
	filtered := make([]AggregateStatus, 0, len(refs))

	for _, r := range refs {
		if r.Ref.Type == aggregateType {
			filtered = append(filtered, r)
		}
	}

	return filtered
}

func applyTombstonePolicy(refs []AggregateStatus, policy TombstonePolicy) []AggregateStatus {
	if policy == TombstoneInclude {
		return refs
	}

	filtered := make([]AggregateStatus, 0, len(refs))

	for _, r := range refs {
		switch policy {
		case TombstoneExclude:
			if !r.Status.IsTombstoned() {
				filtered = append(filtered, r)
			}
		case TombstoneOnly:
			if r.Status.IsTombstoned() {
				filtered = append(filtered, r)
			}
		}
	}

	return filtered
}

func applyCursor(refs []AggregateStatus, after id.AggregateID) []AggregateStatus {
	if after.IsZero() {
		return refs
	}

	for i, r := range refs {
		if r.Ref.ID.String() == after.String() {
			if i+1 < len(refs) {
				return refs[i+1:]
			}

			return nil
		}
	}

	return refs
}

func paginateStatus(refs []AggregateStatus, limit uint) *Page[AggregateStatus] {
	if limit == 0 {
		limit = defaultPageSize
	}

	if uint(len(refs)) <= limit {
		return &Page[AggregateStatus]{Items: refs, HasMore: false}
	}

	return &Page[AggregateStatus]{Items: refs[:limit], HasMore: true}
}
