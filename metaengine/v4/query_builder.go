package metaengine

import "context"

// QueryBuilder provides a fluent, chainable API on top of TypedReader for
// building scan queries with filters, sort, and pagination. Each method
// returns the builder for chaining; call Execute to run the scan.
//
//	reader := metaengine.NewReader[TaskView](store, "task_views")
//	tasks, err := metaengine.NewQueryBuilder(reader).
//	    Where("status", metaengine.FilterEq, "active").
//	    Where("priority", metaengine.FilterGe, 5).
//	    SortBy("title", false).
//	    Limit(50).
//	    Execute(ctx)
//
// This is a convenience layer — TypedReader.Scan accepts the same options
// directly. QueryBuilder eliminates the `[]ScanOption` slice boilerplate and
// makes query construction read top-to-bottom.
type QueryBuilder[V any] struct {
	reader *TypedReader[V]
	opts   []ScanOption
}

// NewQueryBuilder creates a fluent query builder wrapping a TypedReader.
func NewQueryBuilder[V any](reader *TypedReader[V]) *QueryBuilder[V] {
	return &QueryBuilder[V]{reader: reader}
}

// Where adds a filter predicate (column op value). Multiple calls are
// AND-combined. On pushdown engines (SQLite, Pebble), filters generate
// SQL WHERE clauses; on memory engines, they are evaluated in Go.
func (b *QueryBuilder[V]) Where(column string, op FilterOp, value any) *QueryBuilder[V] {
	b.opts = append(b.opts, WithFilter(column, op, value))

	return b
}

// WhereIn adds an IN filter (column IN values).
func (b *QueryBuilder[V]) WhereIn(column string, values []any) *QueryBuilder[V] {
	b.opts = append(b.opts, WithIn(column, values))

	return b
}

// WhereRange adds a range filter (low <= column <= high).
func (b *QueryBuilder[V]) WhereRange(column string, low, high any) *QueryBuilder[V] {
	b.opts = append(b.opts, WithRange(column, low, high))

	return b
}

// SortBy sets the sort column and direction. desc=true for descending.
func (b *QueryBuilder[V]) SortBy(column string, desc bool) *QueryBuilder[V] {
	b.opts = append(b.opts, WithSort(column, desc))

	return b
}

// Limit sets the maximum number of results.
func (b *QueryBuilder[V]) Limit(n int) *QueryBuilder[V] {
	b.opts = append(b.opts, WithLimit(n))

	return b
}

// Cursor sets the keyset pagination cursor for offset-free paging.
func (b *QueryBuilder[V]) Cursor(c any) *QueryBuilder[V] {
	b.opts = append(b.opts, WithCursor(c))

	return b
}

// Execute runs the scan with all accumulated options and returns the results.
func (b *QueryBuilder[V]) Execute(ctx context.Context) ([]V, error) {
	return b.reader.Scan(ctx, b.opts...)
}

// Get performs a point lookup by key, bypassing the builder chain.
// This is a convenience for the common "build a reader, then get one item"
// pattern.
func (b *QueryBuilder[V]) Get(ctx context.Context, key any) (V, bool, error) {
	return b.reader.Get(ctx, key)
}
