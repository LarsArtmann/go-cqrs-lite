package metaengine

import (
	"bytes"
	"fmt"
	"sort"
	"testing"
)

// spPair mirrors the engine-side pair shape (badger/pebble/bbolt map their own
// pair types in via accessors); values are float64 to exercise the common
// numeric sort path.
type spPair struct {
	key []byte
	val any
}

func spKey(p spPair) []byte { return p.key }

func spVal(p spPair) any { return p.val }

// spLess is the tri-state numeric comparator the KV engines pass for
// SortOnField queries.
func spLess(a, b any) int {
	x, y := a.(float64), b.(float64)

	switch {
	case x < y:
		return -1
	case x > y:
		return 1
	default:
		return 0
	}
}

func spPairs(n int) []spPair {
	pairs := make([]spPair, 0, n)
	for i := range n {
		pairs = append(pairs, spPair{
			key: []byte(fmt.Sprintf("k%04d", i%100)),
			val: float64(i % 7),
		})
	}

	return pairs
}

// TestSortPaginate_SortValueTiebreakByKey pins the ordering contract: primary
// sort by valueOf, deterministic byte-key tiebreak for equal values.
func TestSortPaginate_SortValueTiebreakByKey(t *testing.T) {
	t.Parallel()

	pairs := []spPair{
		{key: []byte("c"), val: float64(2)},
		{key: []byte("a"), val: float64(1)},
		{key: []byte("b"), val: float64(1)},
		{key: []byte("d"), val: float64(0)},
	}

	got := SortPaginate(pairs, spKey, spVal, spLess, nil, 0)

	wantOrder := []string{"d", "a", "b", "c"}
	for i, want := range wantOrder {
		if string(got[i].key) != want {
			t.Fatalf("position %d: key %q, want %q (full: %v)", i, got[i].key, want, got)
		}
	}
}

// TestSortPaginate_CursorSkipsSeen pins keyset pagination: items where
// sortFn(item, cursor) <= 0 are dropped, items strictly after the cursor stay.
func TestSortPaginate_CursorSkipsSeen(t *testing.T) {
	t.Parallel()

	pairs := spPairs(21) // values 0..6, three keys per value

	got := SortPaginate(pairs, spKey, spVal, spLess, float64(3), 0)

	for _, p := range got {
		if p.val.(float64) <= 3 {
			t.Fatalf("cursor pagination kept seen value %v (key %s)", p.val, p.key)
		}
	}

	if len(got) != 9 {
		t.Fatalf("kept %d pairs, want 9 (values 4,5,6 x 3 keys)", len(got))
	}
}

// TestSortPaginate_LimitPlusOne pins the has-more contract: the result is
// truncated at limit+1 so callers can detect another page.
func TestSortPaginate_LimitPlusOne(t *testing.T) {
	t.Parallel()

	pairs := spPairs(21)

	got := SortPaginate(pairs, spKey, spVal, spLess, nil, 5)
	if len(got) != 6 {
		t.Fatalf("limit 5 must truncate at 6 (limit+1), got %d", len(got))
	}

	exact := SortPaginate(pairs, spKey, spVal, spLess, nil, 21)
	if len(exact) != 21 {
		t.Fatalf("limit >= len must keep all pairs, got %d", len(exact))
	}
}

// TestSortPaginate_NilSortFn pins the no-sort contract: nil sortFn means no
// sorting and no cursor filtering — only the limit truncation runs.
func TestSortPaginate_NilSortFn(t *testing.T) {
	t.Parallel()

	pairs := []spPair{
		{key: []byte("z"), val: float64(9)},
		{key: []byte("a"), val: float64(0)},
		{key: []byte("m"), val: float64(5)},
		{key: []byte("b"), val: float64(1)},
	}

	got := SortPaginate(pairs, spKey, spVal, nil, float64(9), 2)
	if len(got) != 3 {
		t.Fatalf("limit 2 must truncate at 3 (limit+1), got %d", len(got))
	}

	if string(got[0].key) != "z" {
		t.Fatalf("input order must be preserved with nil sortFn, got %s first", got[0].key)
	}
}

// TestSortPaginate_ZeroLimitNoTruncation pins limit == 0 as "no truncation".
func TestSortPaginate_ZeroLimitNoTruncation(t *testing.T) {
	t.Parallel()

	pairs := spPairs(50)

	got := SortPaginate(pairs, spKey, spVal, spLess, nil, 0)
	if len(got) != 50 {
		t.Fatalf("limit 0 must keep all pairs, got %d", len(got))
	}
}

// TestSortPaginate_AllocBudget pins the zero-alloc closure contract: with
// non-capturing accessors, the filter+truncate path allocates nothing and the
// sort path stays within a small constant budget (upper bounds, not exact
// pins — see AGENTS on cross-graph alloc drift). Serial: AllocsPerRun forbids
// parallel tests.
func TestSortPaginate_AllocBudget(t *testing.T) {
	const size = 1000

	base := spPairs(size)

	filterPairs := make([]spPair, size)
	copy(filterPairs, base)

	var cursor any = float64(3)

	filterAllocs := testing.AllocsPerRun(50, func() {
		SortPaginate(filterPairs, spKey, spVal, spLess, cursor, 0)
	})
	if filterAllocs > 1 {
		t.Errorf("filter+truncate path allocates %.1f allocs, want <= 1 (cursor boxing)", filterAllocs)
	}

	sortPairs := make([]spPair, size)
	copy(sortPairs, base)

	sortAllocs := testing.AllocsPerRun(50, func() {
		SortPaginate(sortPairs, spKey, spVal, spLess, nil, 0)
	})
	if sortAllocs > 3 {
		t.Errorf("sort path allocates %.1f allocs, want <= 3", sortAllocs)
	}
}

func BenchmarkSortPaginate_1K(b *testing.B) {
	base := spPairs(1000)

	b.Run("sort", func(b *testing.B) {
		pairs := make([]spPair, len(base))

		for b.Loop() {
			copy(pairs, base)
			SortPaginate(pairs, spKey, spVal, spLess, nil, 0)
		}
	})

	b.Run("filter-truncate", func(b *testing.B) {
		pairs := make([]spPair, len(base))

		for b.Loop() {
			copy(pairs, base)
			SortPaginate(pairs, spKey, spVal, spLess, float64(3), 50)
		}
	})
}

// sortPaginateReference is the pre-extraction inlined algorithm, kept here to
// prove the shared implementation is behaviorally identical on the benchmark
// workload (same result slice contents).
func sortPaginateReference(pairs []spPair, cursor any, limit int) []spPair {
	sort.Slice(pairs, func(i, j int) bool {
		if c := spLess(spVal(pairs[i]), spVal(pairs[j])); c != 0 {
			return c < 0
		}

		return bytes.Compare(spKey(pairs[i]), spKey(pairs[j])) < 0
	})

	if cursor != nil {
		filtered := pairs[:0]

		for _, p := range pairs {
			if spLess(spVal(p), cursor) <= 0 {
				continue
			}

			filtered = append(filtered, p)
		}

		pairs = filtered
	}

	if limit > 0 && len(pairs) > limit+1 {
		pairs = pairs[:limit+1]
	}

	return pairs
}

// TestSortPaginate_MatchesInlinedReference guards the extraction: the shared
// function produces byte-identical output to the inlined twin it replaced.
func TestSortPaginate_MatchesInlinedReference(t *testing.T) {
	t.Parallel()

	for _, limit := range []int{0, 3, 21} {
		a := SortPaginate(spPairs(42), spKey, spVal, spLess, float64(2), limit)
		b := sortPaginateReference(spPairs(42), float64(2), limit)

		if len(a) != len(b) {
			t.Fatalf("limit %d: shared %d pairs, reference %d", limit, len(a), len(b))
		}

		for i := range a {
			if string(a[i].key) != string(b[i].key) {
				t.Fatalf("limit %d: position %d key %q, reference %q", limit, i, a[i].key, b[i].key)
			}
		}
	}
}
