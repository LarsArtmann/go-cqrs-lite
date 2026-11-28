package search

import "github.com/larsartmann/go-cqrs-lite/query/v4"

// Segment is one run of highlighted text. Highlighting is data, never HTML:
// a template renders Text escaped and wraps Marked runs in the tag it
// chooses.
type Segment struct {
	Text   string
	Marked bool
}

// Plain is a snippet made of one unmarked segment.
func Plain(text string) []Segment {
	if text == "" {
		return nil
	}
	return []Segment{{Text: text}}
}

// Hit is one ranked record.
type Hit[T any] struct {
	// ID is the record's identifier from [Source.ID]. It breaks score ties
	// and gives templates a stable list key.
	ID string
	// Item is the record itself.
	Item T
	// Score orders the hits. The order is the contract; the number is a
	// hint and must not be shown to users or compared across queries.
	Score float64
	// Snippet marks the terms in the text around the first match.
	Snippet []Segment
}

// Result is the page a search returned. It embeds the pagination of
// [query.PaginatedResult] and adds what the search decided.
type Result[T any] struct {
	query.PaginatedResult[Hit[T]]
	// Via names the tier that ran.
	Via Via
	// Truncated is true when the scan tier stopped at its limit before it
	// had seen every record: more matches may exist beyond the first page.
	Truncated bool
	// Degraded is true when the search ran on a lower tier than the
	// deployment could have served, because an embedder failed or the tier
	// had no executor. The results are complete for the tier that ran.
	Degraded bool
}

// IsEmpty reports whether the page carries no hits.
func (r Result[T]) IsEmpty() bool {
	return len(r.Data) == 0
}
