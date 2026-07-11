package listing

import (
	"context"

	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

const (
	defaultPageSize = 20
	maxPageSize     = 100
)

// ListBuilder provides a fluent API for aggregate listings.
type ListBuilder struct {
	reader AggregateReader
	opts   ListOptions
}

// NewListBuilder creates a builder for aggregate listings.
func NewListBuilder(reader AggregateReader) *ListBuilder {
	return &ListBuilder{
		reader: reader,
		opts: ListOptions{ //nolint:exhaustruct // builder pattern: Type and After set via methods
			Limit:     defaultPageSize,
			Tombstone: TombstoneExclude,
		},
	}
}

// OfType filters to a specific aggregate type.
func (b *ListBuilder) OfType(t id.AggregateType) *ListBuilder {
	b.opts.Type = t

	return b
}

// After sets the cursor for the next page.
// Pass the last AggregateListing.ID from the previous Page.
func (b *ListBuilder) After(id id.AggregateID) *ListBuilder {
	b.opts.After = id

	return b
}

// PageSize sets the page size. Clamped to [1, maxPageSize].
func (b *ListBuilder) PageSize(n uint) *ListBuilder {
	switch {
	case n == 0:
		b.opts.Limit = defaultPageSize
	case n > maxPageSize:
		b.opts.Limit = maxPageSize
	default:
		b.opts.Limit = n
	}

	return b
}

// IncludeDeleted shows all aggregates, including tombstoned ones.
func (b *ListBuilder) IncludeDeleted() *ListBuilder {
	b.opts.Tombstone = TombstoneInclude

	return b
}

// OnlyDeleted shows only tombstoned aggregates.
func (b *ListBuilder) OnlyDeleted() *ListBuilder {
	b.opts.Tombstone = TombstoneOnly

	return b
}

// List executes the query and returns a page of aggregate references.
func (b *ListBuilder) List(ctx context.Context) (*Page[AggregateListing], error) {
	return b.reader.List(ctx, b.opts) //nolint:wrapcheck // transparent proxy to reader
}

// ListWithStatus executes the query and returns aggregates with tombstone status.
func (b *ListBuilder) ListWithStatus(ctx context.Context) (*Page[AggregateStatus], error) {
	return b.reader.ListWithStatus(ctx, b.opts) //nolint:wrapcheck // transparent proxy to reader
}
