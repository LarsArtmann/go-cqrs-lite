package metaengine_test

import (
	"context"
	"fmt"
	"slices"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// buildTiePairs builds the tie-heavy dataset: 6 distinct sort values × 5 keys
// = 30 items. A fresh copy per page mirrors how engines re-scan the collection
// for every MapScan call (SortPaginate filters its input slice in place).
type tiePair struct {
	Sort int
	Key  string
}

func buildTiePairs() []tiePair {
	pairs := make([]tiePair, 0, 30)

	for sortVal := 0; sortVal < 6; sortVal++ {
		for n := 0; n < 5; n++ {
			pairs = append(pairs, tiePair{Sort: sortVal, Key: fmt.Sprintf("k%d-%02d", sortVal, n)})
		}
	}

	return pairs
}

func tiePairKey(p tiePair) []byte { return []byte(p.Key) }

func tiePairValue(p tiePair) any { return p }

func tiePairSortFn(a, b any) int {
	av, aok := a.(tiePair)
	bv, bok := b.(tiePair)
	if !aok || !bok {
		return 0
	}

	return cmpInts(av.Sort, bv.Sort)
}

func cmpInts(a, b int) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	default:
		return 0
	}
}

func cmpStrings(a, b string) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	default:
		return 0
	}
}

// expectedTieOrder returns the dataset's IDs in (Sort, Key) order — the global
// order every correctly paginating walk must serve.
func expectedTieOrder() []string {
	sorted := slices.SortedFunc(
		slices.Values(buildTiePairs()),
		func(a, b tiePair) int {
			if c := tiePairSortFn(a, b); c != 0 {
				return c
			}

			return cmpStrings(a.Key, b.Key)
		},
	)

	ids := make([]string, len(sorted))
	for i, p := range sorted {
		ids[i] = p.Key
	}

	return ids
}

// paginateTiePairs walks the dataset page by page through the SortPaginate +
// PairsToScanResult core, rebuilding the pair slice per page (engine re-scan)
// and deriving each next cursor from the page's last returned item.
func paginateTiePairs(t *testing.T, limit int, mkCursor func(last tiePair) any) []string {
	t.Helper()

	var keys []string

	var cursor any

	for page := 0; page < 100; page++ {
		sorted := metaengine.SortPaginate(
			buildTiePairs(),
			tiePairKey,
			tiePairValue,
			tiePairSortFn,
			cursor,
			limit,
		)
		result := metaengine.PairsToScanResult(sorted, tiePairValue, limit)

		for _, item := range result.Items {
			keys = append(
				keys,
				item.(tiePair).Key,
			) //nolint:forcetypeassert // items are tiePair by construction
		}

		if !result.HasMore {
			return keys
		}

		cursor = mkCursor(sorted[len(result.Items)-1])
	}

	t.Fatal("runaway pagination: more than 100 pages")

	return nil
}

func assertExactlyOnceInOrder(t *testing.T, got, want []string) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("collected %d items, want %d", len(got), len(want))
	}

	seen := make(map[string]int, len(want))
	for i, key := range got {
		seen[key]++
		if seen[key] > 1 {
			t.Fatalf("item %q served more than once", key)
		}

		if key != want[i] {
			t.Fatalf("position %d = %q, want %q", i, key, want[i])
		}
	}
}

// TestSortPaginate_CompoundCursor_TieHeavy_PaginatesAllExactlyOnce pins the
// M06 fix: with a SortKeyCursor the cursor filter applies the same
// (sortValue, key) ordering as the sort, so tie-heavy datasets paginate with
// neither dropped nor duplicated rows — at every limit, including limits that
// straddle tie blocks.
func TestSortPaginate_CompoundCursor_TieHeavy_PaginatesAllExactlyOnce(t *testing.T) {
	t.Parallel()

	want := expectedTieOrder()

	for _, limit := range []int{1, 3, 4, 5, 7, 29, 30} {
		t.Run(fmt.Sprintf("limit=%d", limit), func(t *testing.T) {
			t.Parallel()

			got := paginateTiePairs(t, limit, func(last tiePair) any {
				return metaengine.SortKeyCursor{Sort: last, Key: []byte(last.Key)}
			})

			assertExactlyOnceInOrder(t, got, want)
		})
	}
}

// TestSortPaginate_LegacyCursor_TieHeavy_DropsStraddledTies documents the
// legacy value-only cursor semantics: ties at the cursor's sort value are all
// skipped, so any tie block straddling a page boundary loses its unseen tail.
// 30 items / 5-item tie blocks / limit 4 → every block loses its 5th item:
// 24 of 30 collected. This test PINS that known-lossy behavior; changing it
// means the legacy contract moved and every engine's pagination must be
// re-audited.
func TestSortPaginate_LegacyCursor_TieHeavy_DropsStraddledTies(t *testing.T) {
	t.Parallel()

	got := paginateTiePairs(t, 4, func(last tiePair) any { return last })

	if len(got) != 24 {
		t.Fatalf(
			"legacy cursor collected %d items, want the pinned 24 (6 straddled ties dropped)",
			len(got),
		)
	}

	for n := 0; n < 6; n++ {
		dropped := fmt.Sprintf("k%d-04", n)
		if slices.Contains(got, dropped) {
			t.Fatalf("legacy cursor unexpectedly served straddled tie %q", dropped)
		}
	}
}

// TestMapScan_MemoryEngine_CompoundCursor_TieHeavy proves the fix end-to-end
// through a real engine backend: the memory engine delegates to the shared
// SortPaginate core, so compound cursors paginate tie-heavy collections with
// no drops or dupes. The cursor key is the engine's deterministic key form
// (the map key rendered as a string).
func TestMapScan_MemoryEngine_CompoundCursor_TieHeavy(t *testing.T) {
	t.Parallel()

	eng := metaengine.NewMemoryEngine()
	t.Cleanup(func() { _ = eng.Close() })

	mb := eng.(metaengine.MapBackend)  //nolint:forcetypeassert // memory engine implements all backends
	sb := eng.(metaengine.ScanBackend) //nolint:forcetypeassert // memory engine implements all backends

	ctx := context.Background()

	for sortVal := 0; sortVal < 6; sortVal++ {
		for n := 0; n < 5; n++ {
			key := fmt.Sprintf("k%d-%02d", sortVal, n)
			err := mb.MapSet(ctx, "ties", key, map[string]any{"sort": sortVal, "key": key})
			if err != nil {
				t.Fatalf("MapSet %s: %v", key, err)
			}
		}
	}

	sortOf := func(v any) int {
		switch x := v.(type) {
		case map[string]any:
			return x["sort"].(int) //nolint:forcetypeassert // rows store int under "sort" by construction
		case int:
			return x
		default:
			t.Fatalf("unexpected sort operand %T", v)
			return 0
		}
	}

	sortFn := func(a, b any) int { return cmpInts(sortOf(a), sortOf(b)) }

	const limit = 4

	var got []string

	var cursor any

	for page := 0; page < 100; page++ {
		result, err := sb.MapScan(ctx, "ties", nil, sortFn, cursor, limit)
		if err != nil {
			t.Fatalf("MapScan page %d: %v", page, err)
		}

		for _, item := range result.Items {
			got = append(
				got,
				item.(map[string]any)["key"].(string),
			) //nolint:forcetypeassert // by construction
		}

		if !result.HasMore {
			break
		}

		last := result.Items[len(result.Items)-1].(map[string]any) //nolint:forcetypeassert // by construction
		cursor = metaengine.SortKeyCursor{
			Sort: last["sort"],
			Key:  []byte(fmt.Sprintf("%v", last["key"])),
		}
	}

	assertExactlyOnceInOrder(t, got, expectedTieOrder())
}
