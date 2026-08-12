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

// InMemoryStreamReader implements StreamReader using a Journal.
// Caches the stream index and only rebuilds when the event count changes.
// Suitable for testing, development, and single-process deployments.
type InMemoryStreamReader struct {
	journal event.Journal

	mu     sync.RWMutex
	cached []StreamStatus
}

var _ StreamReader = (*InMemoryStreamReader)(nil)

// NewInMemoryStreamReader creates a reader that enumerates via Journal.ReadAll.
func NewInMemoryStreamReader(journal event.Journal) *InMemoryStreamReader {
	return &InMemoryStreamReader{ //nolint:exhaustruct // mu and cached zero-initialized
		journal: journal,
	}
}

// Deprecated: use InMemoryStreamReader.
type InMemoryAggregateReader = InMemoryStreamReader

// Deprecated: use NewInMemoryStreamReader.
func NewInMemoryAggregateReader(journal event.Journal) *InMemoryStreamReader {
	return NewInMemoryStreamReader(journal)
}

func (r *InMemoryStreamReader) List(
	ctx context.Context,
	opts ListOptions,
) (*Page[StreamListing], error) {
	return ListRefsFromStatus(r, ctx, opts)
}

func (r *InMemoryStreamReader) ListWithStatus(
	ctx context.Context,
	opts ListOptions,
) (*Page[StreamStatus], error) {
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

func (r *InMemoryStreamReader) getRefsUnsorted() []StreamStatus {
	r.mu.RLock()
	cached := r.cached
	r.mu.RUnlock()

	return slices.Clone(cached)
}

func (r *InMemoryStreamReader) rebuildCache(ctx context.Context) ([]StreamStatus, error) {
	all, err := r.journal.ReadAll(ctx)
	if err != nil {
		return nil, errorfamily.WrapInfrastructure(
			err,
			"listing.in_memory_list",
			"stream in-memory list",
		)
	}

	refs := buildRefs(all)

	slices.SortFunc(refs, func(a, b StreamStatus) int {
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

// InvalidateCache clears the cached stream index.
// Call this after new events are saved to the store.
func (r *InMemoryStreamReader) InvalidateCache() {
	r.mu.Lock()
	r.cached = nil
	r.mu.Unlock()
}

func buildRefs(events []event.Event) []StreamStatus {
	type streamKey struct {
		streamType id.StreamType
		streamID   id.StreamID
	}

	type streamBuilder struct {
		ref       StreamListing
		lastEvent event.Event
	}

	builders := make(map[streamKey]*streamBuilder)

	for _, evt := range events {
		key := streamKey{streamType: evt.StreamType(), streamID: evt.StreamID()}

		b, ok := builders[key]
		if !ok {
			b = &streamBuilder{ //nolint:exhaustruct // fields populated incrementally below
				ref: StreamListing{ //nolint:exhaustruct // ID+Type set; Version/EventCount/LastEventAt added in loop
					ID:   evt.StreamID(),
					Type: evt.StreamType(),
				},
			}
			builders[key] = b
		}

		b.ref.Version = evt.Version()
		b.ref.EventCount++
		b.ref.LastEventAt = evt.OccurredAt()
		b.lastEvent = evt
	}

	result := make([]StreamStatus, 0, len(builders))

	for _, b := range builders {
		result = append(result, StreamStatus{
			Ref:    b.ref,
			Status: event.DetectTombstone([]event.Event{b.lastEvent}),
		})
	}

	return result
}

func filterByType(refs []StreamStatus, streamType id.StreamType) []StreamStatus {
	filtered := make([]StreamStatus, 0, len(refs))

	for _, r := range refs {
		if r.Ref.Type == streamType {
			filtered = append(filtered, r)
		}
	}

	return filtered
}

func applyTombstonePolicy(refs []StreamStatus, policy TombstonePolicy) []StreamStatus {
	if policy == TombstoneInclude {
		return refs
	}

	filtered := make([]StreamStatus, 0, len(refs))

	for _, r := range refs {
		if policy == TombstoneExclude && !r.Status.IsTombstoned() {
			filtered = append(filtered, r)
		} else if policy == TombstoneOnly && r.Status.IsTombstoned() {
			filtered = append(filtered, r)
		}
	}

	return filtered
}

func applyCursor(refs []StreamStatus, after id.StreamID) []StreamStatus {
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

func paginateStatus(refs []StreamStatus, limit uint) *Page[StreamStatus] {
	if limit == 0 {
		limit = defaultPageSize
	}

	if uint(len(refs)) <= limit {
		return &Page[StreamStatus]{Items: refs, HasMore: false}
	}

	return &Page[StreamStatus]{Items: refs[:limit], HasMore: true}
}
