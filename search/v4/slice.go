package search

import (
	"context"
	"net/url"
)

// Slice is a [Source] over records held in memory. It serves examples, tests
// and small products that have no store behind their search.
type Slice[T any] struct {
	// Items are the records, in the source's own order.
	Items []T
	// Identify returns a record's identifier.
	Identify func(T) string
	// Match reports whether a record satisfies the facets. Nil means every
	// record matches, so the zero-facet request lists everything.
	Match func(item T, filters url.Values) bool
}

// List returns up to limit records that satisfy the facets.
func (s Slice[T]) List(_ context.Context, filters url.Values, limit int) (Listing[T], error) {
	var items []T
	truncated := false
	for _, item := range s.Items {
		if s.Match != nil && !s.Match(item, filters) {
			continue
		}
		if len(items) == limit {
			truncated = true
			break
		}
		items = append(items, item)
	}
	return Listing[T]{Items: items, Truncated: truncated}, nil
}

// ByIDs returns the records with the given identifiers.
func (s Slice[T]) ByIDs(_ context.Context, ids []string) ([]T, error) {
	wanted := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		wanted[id] = struct{}{}
	}
	var items []T
	for _, item := range s.Items {
		if _, ok := wanted[s.Identify(item)]; ok {
			items = append(items, item)
		}
	}
	return items, nil
}

// ID returns the identifier of a record.
func (s Slice[T]) ID(item T) string { return s.Identify(item) }
