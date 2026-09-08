package metaengine

type scanConfig struct {
	filters  []FilterSpec
	orGroups [][]FilterSpec
	sort     *SortSpec
	sortCols []SortColumn
	cursor   any
	limit    int
	ranges   []RangeSpec
	inSpecs  []InSpec
}

// RangeSpec declares a range filter (SQL BETWEEN) on a column.
type RangeSpec struct {
	Column string
	Low    any
	High   any
}

// InSpec declares an IN filter (SQL WHERE col IN (...)) on a column.
type InSpec struct {
	Column string
	Values []any
}

// ScanOption tunes a TypedReader.Scan call.
type ScanOption func(*scanConfig)

// WithFilter adds a comparison filter on a column.
func WithFilter(column string, op FilterOp, value any) ScanOption {
	return func(c *scanConfig) {
		c.filters = append(c.filters, FilterSpec{Column: column, Op: op, Value: value})
	}
}

// WithRange adds a range filter (low <= column <= high) on a column.
// On pushdown engines this generates SQL BETWEEN; on closure-based engines it
// generates two comparison predicates.
func WithRange(column string, low, high any) ScanOption {
	return func(c *scanConfig) {
		c.ranges = append(c.ranges, RangeSpec{Column: column, Low: low, High: high})
	}
}

// WithIn adds an IN filter (column IN values) on a column.
// On pushdown engines this generates SQL WHERE col IN (...); on closure-based
// engines it generates a membership predicate.
func WithIn(column string, values []any) ScanOption {
	return func(c *scanConfig) {
		c.inSpecs = append(c.inSpecs, InSpec{Column: column, Values: values})
	}
}

// WithSort sets the sort column and direction.
func WithSort(column string, desc bool) ScanOption {
	return func(c *scanConfig) {
		c.sort = &SortSpec{Column: column, Desc: desc}
	}
}

// SortColumn declares one column in a compound sort.
type SortColumn struct {
	Column string
	Desc   bool
}

// WithSortColumns sets a compound sort (multi-column ORDER BY). When set,
// single-column WithSort is ignored. Pushdown engines use only the first
// column; the full multi-column sort is applied in the closure fallback path.
func WithSortColumns(cols ...SortColumn) ScanOption {
	return func(c *scanConfig) {
		c.sortCols = cols
		if len(cols) > 0 {
			c.sort = &SortSpec{Column: cols[0].Column, Desc: cols[0].Desc}
		}
	}
}

// WithOr adds an OR group of filters. Each call adds one parenthesized OR
// group: at least one filter in the group must match. Multiple WithOr calls
// are ANDed together (and with WithFilter conditions).
//
// On pushdown engines, OR groups generate SQL: AND (cond1 OR cond2 ...).
// On closure-based engines, they are evaluated in Go.
func WithOr(filters ...FilterSpec) ScanOption {
	return func(c *scanConfig) {
		if len(filters) > 0 {
			c.orGroups = append(c.orGroups, filters)
		}
	}
}

// WithLimit sets the maximum number of results.
func WithLimit(n int) ScanOption {
	return func(c *scanConfig) { c.limit = n }
}

// WithCursor sets the keyset pagination cursor (the last sort key from the
// previous page). Pass cursor.Value from a previous ScanPage result, or use
// WithCursorString to pass an encoded cursor string.
func WithCursor(v any) ScanOption {
	return func(c *scanConfig) { c.cursor = v }
}

// WithCursorString sets the keyset pagination cursor from an encoded cursor
// string produced by [Cursor.Encode]. This is the HTTP-safe counterpart to
// [WithCursor]: callers receive an opaque cursor string from ScanPage, pass
// it back via WithCursorString, and the PrefetchCache lookup matches because
// both trimAndCache and the cache check normalize through Cursor.Encode.
func WithCursorString(s string) ScanOption {
	return func(c *scanConfig) {
		cursor, err := ParseCursor(s)
		if err != nil || cursor == nil {
			return
		}

		c.cursor = cursor
	}
}

// ScanPage performs a scan and returns both the results and the next-page
// cursor for keyset pagination. The cursor is derived from the sort field of
// the last returned item. Pass it back via WithCursor(cursor.Value) on the
// next call, or use WithCursorString(cursor.Encode()) for an HTTP-safe
// opaque cursor string that round-trips through the PrefetchCache.
//
// When a PrefetchCache is attached, ScanPage auto-populates it: extra rows
// beyond the limit are cached so the next page request is served from cache
// instead of hitting the engine.
//
// Returns (items, nil cursor, nil) when there are no more pages.
