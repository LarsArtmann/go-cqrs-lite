package search_test

import (
	"context"
	"errors"
	"net/url"
	"reflect"
	"strconv"
	"testing"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/query/v4"
	"github.com/larsartmann/go-cqrs-lite/search/v4"
)

// cappedSource is a Source that stops at a limit of its own, below the
// spec's scan limit, and says so.
type cappedSource struct {
	search.Slice[Note]

	cap    int
	called bool
	got    url.Values
	limit  int
}

func (s *cappedSource) List(ctx context.Context, filters url.Values, limit int) (search.Listing[Note], error) {
	s.called, s.got, s.limit = true, filters, limit
	if limit > s.cap {
		limit = s.cap
	}
	listing, err := s.Slice.List(ctx, filters, limit)
	if len(s.Items) > s.cap {
		listing.Truncated = true
	}

	return listing, err
}

// failingSource is a Source whose List fails.
type failingSource struct {
	search.Slice[Note]

	err error
}

func (s failingSource) List(context.Context, url.Values, int) (search.Listing[Note], error) {
	return search.Listing[Note]{}, s.err
}

// notes is the corpus of the index tests.
func notes() search.Slice[Note] {
	return search.Slice[Note]{
		Items: []Note{
			{ID: "n3", Title: "Standup", Body: "Talk about the big invoice", Status: "open"},
			{ID: "n1", Title: "Big invoice", Body: "The March invoice for Acme", Status: "open"},
			{ID: "n2", Title: "Holiday", Body: "Nothing to do", Status: "done"},
			{ID: "n4", Title: "Small invoice", Body: "April", Status: "done"},
		},
		Identify: func(n Note) string { return n.ID },
		Match: func(n Note, filters url.Values) bool {
			return filters.Get("status") == "" || filters.Get("status") == n.Status
		},
	}
}

func newNotesIndex(t *testing.T, mutate func(*search.Spec[Note]), eng search.Engine) *search.Index[Note] {
	t.Helper()

	spec := testSpec()
	if mutate != nil {
		mutate(&spec)
	}
	index, err := search.New(spec, eng, notes())
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	return index
}

func ids(hits []search.Hit[Note]) []string {
	out := make([]string, len(hits))
	for i, hit := range hits {
		out[i] = hit.ID
	}

	return out
}

func TestNewRejectsMissingPieces(t *testing.T) {
	t.Parallel()

	if _, err := search.New(testSpec(), nil, notes()); err == nil {
		t.Error("New accepted a nil engine")
	}
	if _, err := search.New(testSpec(), search.NoEngine(), nil); err == nil {
		t.Error("New accepted a nil source")
	}
	if _, err := search.New(search.Spec[Note]{}, search.NoEngine(), notes()); !errors.Is(err, search.ErrNoFields) {
		t.Errorf("New with no fields = %v, want ErrNoFields", err)
	}
}

func TestIndexSpecIsACopy(t *testing.T) {
	t.Parallel()

	fields := testSpec().Fields
	index, err := search.New(search.Spec[Note]{Fields: fields}, search.NoEngine(), notes())
	if err != nil {
		t.Fatal(err)
	}
	fields[0].Weight = 99
	if got := index.Spec().Fields[0].Weight; got != 1 {
		t.Fatalf("changing the caller's spec changed the index: weight %v", got)
	}
}

func TestIndexEmptyQueryIsTheUnfilteredList(t *testing.T) {
	t.Parallel()

	result, err := newNotesIndex(t, nil, search.NoEngine()).Search(context.Background(), search.Request{})
	if err != nil {
		t.Fatal(err)
	}
	if result.Via != search.ViaNone || result.TotalCount != 4 || result.Page != 1 || result.PageSize != 20 {
		t.Fatalf(
			"Via %v Total %d Page %d/%d; want none, 4, 1/20",
			result.Via,
			result.TotalCount,
			result.Page,
			result.PageSize,
		)
	}
	if got := ids(result.Data); !reflect.DeepEqual(got, []string{"n3", "n1", "n2", "n4"}) {
		t.Fatalf("empty query lost the source order: %v", got)
	}
	for _, hit := range result.Data {
		if hit.Snippet != nil || hit.Score != 0 {
			t.Fatalf("unfiltered hit %s carries a snippet or a score", hit.ID)
		}
	}
	if result.Truncated || result.Degraded || result.HasNext() {
		t.Fatalf("flags: truncated %v degraded %v next %v", result.Truncated, result.Degraded, result.HasNext())
	}
}

func TestIndexOperatorTokensOnlyIsFacetsOnly(t *testing.T) {
	t.Parallel()

	index := newNotesIndex(t, func(s *search.Spec[Note]) { s.Keys = []string{"status"} }, search.NoEngine())
	result, err := index.Search(context.Background(), search.Request{Query: "status:done"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Via != search.ViaNone || result.TotalCount != 4 {
		t.Fatalf(
			"operator-only query: Via %v total %d; the token is the product's business, not a text search",
			result.Via,
			result.TotalCount,
		)
	}
}

func TestIndexRanksTitleAboveBody(t *testing.T) {
	t.Parallel()

	result, err := newNotesIndex(
		t,
		nil,
		search.NoEngine(),
	).Search(context.Background(), search.Request{Query: "big invoice"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Via != search.ViaScan || result.TotalCount != 2 {
		t.Fatalf("Via %v total %d; want scan, 2", result.Via, result.TotalCount)
	}
	if got := ids(result.Data); !reflect.DeepEqual(got, []string{"n1", "n3"}) {
		t.Fatalf("order %v; want the title match first", got)
	}
	if got := result.Data[0].Snippet; marked(got) != "*Big* *invoice*" {
		t.Fatalf("snippet %+v; want the title field with both terms marked", got)
	}
	if result.Data[0].Score <= result.Data[1].Score {
		t.Fatalf("scores %v, %v; want descending", result.Data[0].Score, result.Data[1].Score)
	}
}

func TestIndexPaging(t *testing.T) {
	t.Parallel()

	index := newNotesIndex(t, nil, search.NoEngine())
	result, err := index.Search(context.Background(), search.Request{Page: query.Pagination{Page: 2, PageSize: 3}})
	if err != nil {
		t.Fatal(err)
	}
	if result.TotalCount != 4 || result.TotalPages != 2 || len(result.Data) != 1 || result.Data[0].ID != "n4" {
		t.Fatalf("page 2 of size 3: %+v", result.PaginatedResult)
	}
	empty, err := index.Search(context.Background(), search.Request{Page: query.Pagination{Page: 9, PageSize: 3}})
	if err != nil || len(empty.Data) != 0 || empty.TotalCount != 4 {
		t.Fatalf("page 9: %v, %d hits, total %d", err, len(empty.Data), empty.TotalCount)
	}
}

func TestIndexReportsTruncatedScan(t *testing.T) {
	t.Parallel()

	source := &cappedSource{Slice: notes(), cap: 2}
	index, err := search.New(testSpec(), search.NoEngine(), source)
	if err != nil {
		t.Fatal(err)
	}
	result, err := index.Search(context.Background(), search.Request{})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Truncated || result.TotalCount != 2 || source.limit != search.DefaultScanLimit {
		t.Fatalf(
			"Truncated %v total %d limit %d; want true, 2, %d",
			result.Truncated,
			result.TotalCount,
			source.limit,
			search.DefaultScanLimit,
		)
	}
}

func TestIndexPassesFacetsAndLimitUntouched(t *testing.T) {
	t.Parallel()

	source := &cappedSource{Slice: notes(), cap: 100}
	index, err := search.New(search.Spec[Note]{Fields: testSpec().Fields, ScanLimit: 50}, search.NoEngine(), source)
	if err != nil {
		t.Fatal(err)
	}
	filters := url.Values{"status": {"done"}, "tag": {"a", "b"}}
	result, err := index.Search(context.Background(), search.Request{Query: "invoice", Filters: filters})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(source.got, filters) || source.limit != 50 {
		t.Fatalf(
			"source saw %v with limit %d; want the facets untouched and the spec's limit",
			source.got,
			source.limit,
		)
	}
	if got := ids(result.Data); !reflect.DeepEqual(got, []string{"n4"}) {
		t.Fatalf("facets did not narrow: %v", got)
	}
}

func TestIndexSort(t *testing.T) {
	t.Parallel()

	byTitle := func(a, b Note) int { return stringCompare(a.Title, b.Title) }
	index := newNotesIndex(
		t,
		func(s *search.Spec[Note]) { s.Sorts = map[string]func(a, b Note) int{"title": byTitle} },
		search.NoEngine(),
	)

	sorted, err := index.Search(context.Background(), search.Request{Sort: "title"})
	if err != nil {
		t.Fatal(err)
	}
	if got := ids(sorted.Data); !reflect.DeepEqual(got, []string{"n1", "n2", "n4", "n3"}) {
		t.Fatalf("ascending %v; want Big invoice, Holiday, Small invoice, Standup", got)
	}
	descending, _ := index.Search(context.Background(), search.Request{Sort: "-title"})
	if got := ids(descending.Data); !reflect.DeepEqual(got, []string{"n3", "n4", "n2", "n1"}) {
		t.Fatalf("descending %v", got)
	}
	if _, err := index.Search(
		context.Background(),
		search.Request{Sort: "nope"},
	); !errors.Is(
		err,
		search.ErrUnknownSort,
	) {
		t.Fatalf("unknown sort = %v, want ErrUnknownSort", err)
	}
}

func stringCompare(a, b string) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	default:
		return 0
	}
}

func TestIndexRejectsBadRequests(t *testing.T) {
	t.Parallel()

	index := newNotesIndex(t, func(s *search.Spec[Note]) { s.MaxQueryRunes = 10 }, search.NoEngine())
	cases := map[string]search.Request{
		"too long": {Query: "abcdefghijk"},
		"control":  {Query: "a\x01b"},
	}
	for name, req := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if _, err := index.Search(context.Background(), req); err == nil {
				t.Fatal("Search accepted an invalid request")
			}
		})
	}
}

func TestIndexZeroPageDefaults(t *testing.T) {
	t.Parallel()

	index := newNotesIndex(t, nil, search.NoEngine())
	result, err := index.Search(context.Background(), search.Request{
		Query: "invoice",
		Page:  query.Pagination{},
	})
	if err != nil || result.Page != 1 || result.PageSize != 20 {
		t.Fatalf("a zero page should default, got %v page %d/%d", err, result.Page, result.PageSize)
	}
}

func TestIndexSourceErrorWraps(t *testing.T) {
	t.Parallel()

	boom := errors.New("disk on fire")
	index, err := search.New(testSpec(), search.NoEngine(), failingSource{err: boom})
	if err != nil {
		t.Fatal(err)
	}
	_, err = index.Search(context.Background(), search.Request{Query: "x"})
	if !errors.Is(err, boom) {
		t.Fatalf("Search error = %v; the cause must stay reachable", err)
	}
	if errorfamily.Classify(err) != errorfamily.Transient || errorfamily.Code(err) != "search.source_list" {
		t.Fatalf(
			"family %v code %q; want Transient search.source_list",
			errorfamily.Classify(err),
			errorfamily.Code(err),
		)
	}
}

func TestIndexStopsOnCancelledContext(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	index := newNotesIndex(t, nil, search.NoEngine())
	if _, err := index.Search(ctx, search.Request{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("Search on a cancelled context = %v; want context.Canceled", err)
	}
}

func TestIndexStepsDownWhenATierHasNoExecutor(t *testing.T) {
	t.Parallel()

	engine := fakeEngine{caps: everythin, embedder: hashEmbedder{}}
	spec := testSpec()

	auto, err := search.New(spec, engine, notes())
	if err != nil {
		t.Fatal(err)
	}
	result, err := auto.Search(context.Background(), search.Request{Query: "unpaid invoices"})
	if err != nil || result.Via != search.ViaScan || !result.Degraded {
		t.Fatalf("Auto: Via %v degraded %v err %v; want scan, degraded, nil", result.Via, result.Degraded, err)
	}

	lexical, _ := search.New(search.Spec[Note]{Fields: spec.Fields, Mode: search.ModeLexical}, engine, notes())
	result, err = lexical.Search(context.Background(), search.Request{Query: "unpaid invoices"})
	if err != nil || result.Via != search.ViaScan || !result.Degraded {
		t.Fatalf("Lexical: Via %v degraded %v err %v; want scan, degraded, nil", result.Via, result.Degraded, err)
	}

	for _, mode := range []search.Mode{search.ModeFullText, search.ModeHybrid} {
		strict, _ := search.New(search.Spec[Note]{Fields: spec.Fields, Mode: mode}, engine, notes())
		if _, err := strict.Search(
			context.Background(),
			search.Request{Query: "unpaid invoices"},
		); !errors.Is(
			err,
			search.ErrModeUnavailable,
		) {
			t.Fatalf("%s without an executor = %v; want ErrModeUnavailable, never a silent step down", mode, err)
		}
	}
}

func TestSliceSource(t *testing.T) {
	t.Parallel()

	source := notes()
	if source.ID(Note{ID: "x"}) != "x" {
		t.Fatal("ID")
	}
	byIDs, _ := source.ByIDs(context.Background(), []string{"n4", "n1", "gone"})
	if got := idsOf(byIDs); !reflect.DeepEqual(got, []string{"n1", "n4"}) {
		t.Fatalf("ByIDs = %v", got)
	}
	listing, _ := source.List(context.Background(), nil, 3)
	if len(listing.Items) != 3 || !listing.Truncated {
		t.Fatalf("List with a limit: %d items truncated %v", len(listing.Items), listing.Truncated)
	}
	free := search.Slice[Note]{Items: notes().Items}
	listing, _ = free.List(context.Background(), url.Values{"status": {"done"}}, 10)
	if len(listing.Items) != 4 {
		t.Fatalf("a Slice without Match ignored its facets instead of matching all: %d", len(listing.Items))
	}
}

func idsOf(items []Note) []string {
	out := make([]string, len(items))
	for i, item := range items {
		out[i] = item.ID
	}

	return out
}

func TestIndexIsSafeForConcurrentUse(t *testing.T) {
	t.Parallel()

	index := newNotesIndex(t, nil, search.NoEngine())
	const routines = 16
	errs := make(chan error, routines)
	for i := range routines {
		go func() {
			for j := range 50 {
				result, err := index.Search(
					context.Background(),
					search.Request{Query: "invoice", Page: query.Pagination{Page: uint(i%2) + 1, PageSize: 2}},
				)
				if err != nil || result.TotalCount != 3 {
					errs <- errors.New("routine " + strconv.Itoa(i) + " step " + strconv.Itoa(j))

					return
				}
			}
			errs <- nil
		}()
	}
	for range routines {
		if err := <-errs; err != nil {
			t.Fatal(err)
		}
	}
}
