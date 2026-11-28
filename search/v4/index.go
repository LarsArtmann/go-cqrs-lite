package search

import (
	"context"
	"sort"
	"strings"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/query/v4"
)

// executor runs one tier above the scan and returns its ranked hits. The
// engines that serve the full-text and vector tiers register one; the scan
// tier needs none.
type executor[T any] func(ctx context.Context, req Request, terms []string) ([]Hit[T], error)

// Index serves searches over one record type. Build it with [New]; a zero
// Index is not usable. An Index is safe for concurrent use.
type Index[T any] struct {
	spec   Spec[T]
	engine Engine
	source Source[T]
	tiers  map[Via]executor[T]
}

// New validates the spec and binds an engine and a source. The returned
// Index keeps a private copy of the spec, so a Spec changed after New does
// not change an existing Index.
func New[T any](spec Spec[T], eng Engine, src Source[T]) (*Index[T], error) {
	if err := spec.Validate(); err != nil {
		return nil, err
	}
	if eng == nil || src == nil {
		return nil, errorfamily.NewRejection("search.no_engine_or_source",
			"an index needs an engine and a source; use search.NoEngine for a deployment that only scans")
	}
	return &Index[T]{
		spec:   spec.clone(),
		engine: eng,
		source: src,
		tiers:  map[Via]executor[T]{},
	}, nil
}

// Spec returns the spec the Index was built with.
func (ix *Index[T]) Spec() Spec[T] {
	return ix.spec.clone()
}

// Search runs one request through the ladder.
//
// An empty query is the unfiltered list in source order, never an error.
// Auto never fails for a missing capability: it steps down to the scan and
// says so in [Result.Via] and [Result.Degraded].
func (ix *Index[T]) Search(ctx context.Context, req Request) (Result[T], error) {
	normalized, err := req.Normalize(ix.spec.MaxQueryRunes)
	if err != nil {
		return Result[T]{}, err
	}
	sortKey, descending, err := ix.resolveSort(normalized.Sort)
	if err != nil {
		return Result[T]{}, err
	}
	terms, _ := ix.spec.Tokens(normalized.Query)
	page := query.NewPagination(normalized.Page.Page, normalized.Page.PageSize)
	caps := ix.engine.Capabilities()
	via, err := Tier(ix.spec.Mode, terms, caps, ix.engine.Embedder() != nil)
	if err != nil {
		return Result[T]{}, err
	}

	if via > ViaScan {
		if run := ix.tiers[via]; run != nil {
			return ix.runTier(ctx, normalized, terms, via, run, page, sortKey, descending)
		}
		if ix.spec.Mode != ModeAuto && ix.spec.Mode != ModeLexical {
			return Result[T]{}, modeUnavailable(ix.spec.Mode, "no executor is registered for "+via.String())
		}
		return ix.scan(ctx, normalized, terms, ViaScan, true, page, sortKey, descending)
	}
	return ix.scan(ctx, normalized, terms, ViaScan, false, page, sortKey, descending)
}

// runTier runs a tier above the scan, hydrates its hits and pages them.
func (ix *Index[T]) runTier(
	ctx context.Context,
	req Request,
	terms []string,
	via Via,
	run executor[T],
	page query.Pagination,
	sortKey string,
	descending bool,
) (Result[T], error) {
	hits, err := run(ctx, req, terms)
	if err != nil {
		return Result[T]{}, err
	}
	RankHits(hits)
	applySort(hits, sortKey, descending, ix.spec.Sorts)
	return ix.paginate(hits, via, false, false, page), nil
}

// scan is tier 1: list with facets through the source, score in Go, rank.
func (ix *Index[T]) scan(
	ctx context.Context,
	req Request,
	terms []string,
	via Via,
	degraded bool,
	page query.Pagination,
	sortKey string,
	descending bool,
) (Result[T], error) {
	listing, err := ix.source.List(ctx, req.Filters, ix.spec.scanLimit())
	if err != nil {
		return Result[T]{}, errorfamily.WrapOnce(err, errorfamily.Transient,
			"search.source_list", "the records could not be listed for this search; nothing changed, retry")
	}

	folded := make([]string, len(terms))
	for i, term := range terms {
		folded[i] = Fold(term)
	}

	hits := make([]Hit[T], 0, len(listing.Items))
	for i, item := range listing.Items {
		if i%checkEvery == 0 {
			if err := ctx.Err(); err != nil {
				return Result[T]{}, errorfamily.WrapOnce(err, errorfamily.Transient,
					"search.cancelled", "the search was cancelled before it finished; nothing changed, retry")
			}
		}
		if len(terms) > 0 {
			score, matched := ScoreTerms(ix.spec, item, folded)
			if !matched {
				continue
			}
			hits = append(hits, ix.hit(item, score, folded))
			continue
		}
		hits = append(hits, Hit[T]{ID: ix.source.ID(item), Item: item})
	}
	if len(terms) > 0 {
		RankHits(hits)
	}
	applySort(hits, sortKey, descending, ix.spec.Sorts)
	if len(terms) == 0 {
		via = ViaNone
	}
	return ix.paginate(hits, via, listing.Truncated, degraded, page), nil
}

// checkEvery is how many records the scan reads between cancellation checks.
const checkEvery = 256

// hit builds a scored hit with a snippet from the first matching field.
func (ix *Index[T]) hit(item T, score float64, terms []string) Hit[T] {
	return Hit[T]{
		ID:      ix.source.ID(item),
		Item:    item,
		Score:   score,
		Snippet: ix.snippet(item, terms),
	}
}

// snippet returns the window of the first field that matches a term.
func (ix *Index[T]) snippet(item T, terms []string) []Segment {
	for _, field := range ix.spec.Fields {
		text := field.Value(item)
		if _, first, _ := markTerms([]rune(text), terms); first >= 0 {
			return SnippetTerms(text, terms, 0)
		}
	}
	return nil
}

// paginate slices the ranked hits into the requested page.
func (ix *Index[T]) paginate(
	hits []Hit[T],
	via Via,
	truncated, degraded bool,
	page query.Pagination,
) Result[T] {
	total := uint(len(hits))
	from := min(int(page.Offset()), len(hits))
	to := min(from+int(page.PageSize), len(hits))
	return Result[T]{
		PaginatedResult: query.NewPaginatedResult(hits[from:to], total, page),
		Via:             via,
		Truncated:       truncated,
		Degraded:        degraded,
	}
}

// resolveSort validates the sort key and reports its direction. An empty key
// means relevance. A leading minus selects descending order.
func (ix *Index[T]) resolveSort(sortParam string) (key string, descending bool, err error) {
	if sortParam == "" {
		return "", false, nil
	}
	key = sortParam
	if strings.HasPrefix(key, "-") {
		key, descending = key[1:], true
	}
	if _, ok := ix.spec.Sorts[key]; !ok {
		return "", false, rejectionWith(ErrUnknownSort,
			"the sort key "+sortParam+" is not declared by the spec; sort by relevance or by a declared key")
	}
	return key, descending, nil
}

// applySort orders hits by a declared comparator, stably, so relevance
// breaks ties.
func applySort[T any](hits []Hit[T], key string, descending bool, sorts map[string]func(a, b T) int) {
	if key == "" {
		return
	}
	less := sorts[key]
	sort.SliceStable(hits, func(i, j int) bool {
		order := less(hits[i].Item, hits[j].Item)
		if descending {
			return order > 0
		}
		return order < 0
	})
}
