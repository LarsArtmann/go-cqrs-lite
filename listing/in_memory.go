package listing

import (
	"context"
	"slices"
	"strings"
	"sync"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// InMemoryAggregateReader implements AggregateReader using a Journal.
// Caches the aggregate index and only rebuilds when the event count changes.
// Suitable for testing, development, and single-process deployments.
type InMemoryAggregateReader struct {
	journal event.Journal

	mu     sync.RWMutex
	cached []AggregateStatus
}

var _ AggregateReader = (*InMemoryAggregateReader)(nil)

// NewInMemoryAggregateReader creates a reader that enumerates via Journal.ReadAll.
func NewInMemoryAggregateReader(journal event.Journal) *InMemoryAggregateReader {
	return &InMemoryAggregateReader{ //nolint:exhaustruct // mu and cached zero-initialized
		journal: journal,
	}
}

func (r *InMemoryAggregateReader) List(
	ctx context.Context,
	opts ListOptions,
) (*Page[AggregateListing], error) {
	return ListRefsFromStatus(r, ctx, opts)
}

func (r *InMemoryAggregateReader) ListWithStatus(
	ctx context.Context,
	opts ListOptions,
) (*Page[AggregateStatus], error) {
	refs := r.getRefsUnsorted()
	if refs == nil {
		var err error

		refs, err = r.rebuildCache(ctx)
		if err != nil {
			return nil, err
		}
	}

	if opts.Type != "" {
		refs = filterByType(refs, opts.Type)
	}

	refs = applyTombstonePolicy(refs, opts.Tombstone)

	refs = applyCursor(refs, opts.After)

	return paginateStatus(refs, opts.Limit), nil
}

func (r *InMemoryAggregateReader) getRefsUnsorted() []AggregateStatus {
	r.mu.RLock()
	cached := r.cached
	r.mu.RUnlock()

	return slices.Clone(cached)
}

func (r *InMemoryAggregateReader) rebuildCache(ctx context.Context) ([]AggregateStatus, error) {
	all, err := r.journal.ReadAll(ctx)
	if err != nil {
		return nil, errorfamily.WrapInfrastructure(
			err,
			"listing.in_memory_list",
			"stream in-memory list",
		)
	}

	refs := buildRefs(all)

	slices.SortFunc(refs, func(a, b AggregateStatus) int {
		if a.Ref.Type != b.Ref.Type {
			return strings.Compare(string(a.Ref.Type), string(b.Ref.Type))
		}

		return strings.Compare(a.Ref.ID.String(), b.Ref.ID.String())
	})

	r.mu.Lock()
	r.cached = refs
	r.mu.Unlock()

	return refs, nil
}

// InvalidateCache clears the cached aggregate index.
// Call this after new events are saved to the store.
func (r *InMemoryAggregateReader) InvalidateCache() {
	r.mu.Lock()
	r.cached = nil
	r.mu.Unlock()
}

func buildRefs(events []event.Event) []AggregateStatus {
	type streamKey struct {
		aggType id.AggregateType
		aggID   id.AggregateID
	}

	type streamBuilder struct {
		ref       AggregateListing
		lastEvent event.Event
	}

	builders := make(map[streamKey]*streamBuilder)

	for _, evt := range events {
		key := streamKey{aggType: evt.AggregateType(), aggID: evt.AggregateID()}

		b, ok := builders[key]
		if !ok {
			b = &streamBuilder{ //nolint:exhaustruct // fields populated incrementally below
				ref: AggregateListing{ //nolint:exhaustruct // ID+Type set; Version/EventCount/LastEventAt added in loop
					ID:   evt.AggregateID(),
					Type: evt.AggregateType(),
				},
			}
			builders[key] = b
		}

		b.ref.Version = evt.Version()
		b.ref.EventCount++
		b.ref.LastEventAt = evt.OccurredAt()
		b.lastEvent = evt
	}

	result := make([]AggregateStatus, 0, len(builders))

	for _, b := range builders {
		result = append(result, AggregateStatus{
			Ref:    b.ref,
			Status: event.DetectTombstone([]event.Event{b.lastEvent}),
		})
	}

	return result
}

func filterByType(refs []AggregateStatus, aggregateType id.AggregateType) []AggregateStatus {
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
		if policy == TombstoneExclude && !r.Status.IsTombstoned() {
			filtered = append(filtered, r)
		} else if policy == TombstoneOnly && r.Status.IsTombstoned() {
			filtered = append(filtered, r)
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
