package adttest

import (
	"bytes"
	"context"
	"encoding/json/v2"
	"fmt"
	"slices"
	"testing"
	"time"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/keycodec"
)

// PaginationSeed identifies the boundary row of a page for compound-cursor
// derivation: the collection being walked, the row's seeded map key, and the
// row's item as returned by MapScan.
type PaginationSeed struct {
	Collection string
	Key        string
	Item       any
}

// PaginationProbe pairs an engine Factory with the externally observable
// derivation of the engine's MapScan tiebreak key. The conformance walk
// rebuilds a metaengine.SortKeyCursor from each page's last row; CursorKey
// tells it which byte form the engine sorts ties by.
//
// The byte forms differ per engine family (2026-10-06 M06/F22 conformance
// finding, pinned here instead of silently accommodated):
//
//   - memory, sqlite, postgres, mysql, duckdb, dgraph: the rendered map key —
//     those engines SELECT the key column (memory renders
//     fmt.Sprintf("%v", key)) (CursorKeyRaw). sqlite joined this family in
//     the F22 fix: its MapScan previously selected only the value column and
//     tiebreaked on the stored value JSON, which json v2's non-canonical map
//     key order made impossible to rebuild from a returned item — the
//     conformance matrix failed sqlite with drops and duplicates until the
//     engine read the key column (mirroring pgengine).
//   - badger, pebble, bbolt: the full prefixed stored key,
//     keycodec.MapKey(col, EncodeKeyStr(key)) — their MapScan iterates raw
//     KV keys (CursorKeyKVMapKey).
//
// Every form paginates tie-heavy collections exactly once with a compound
// cursor; the forms differ only in the within-tie ORDER, which is why
// RunPaginationConformance asserts per-engine exactness (no drops, no
// duplicates, monotone order, honest HasMore) instead of cross-engine
// page-by-page parity.
type PaginationProbe struct {
	Factory

	// CursorKey derives the byte tiebreak key for a boundary row. Use one of
	// CursorKeyRaw, CursorKeyKVMapKey, CursorKeyValueJSON.
	CursorKey func(PaginationSeed) []byte
}

// CursorKeyRaw derives the tiebreak key for engines whose MapScan sorts ties
// by the rendered map key (memory, postgres, mysql, duckdb, dgraph).
func CursorKeyRaw(seed PaginationSeed) []byte {
	return []byte(seed.Key)
}

// CursorKeyKVMapKey derives the tiebreak key for the KV engines (badger,
// pebble, bbolt), whose MapScan tiebreaks on the full prefixed stored key.
func CursorKeyKVMapKey(seed PaginationSeed) []byte {
	return keycodec.MapKey(seed.Collection, keycodec.EncodeKeyStr(seed.Key))
}

// CursorKeyValueJSON derives the tiebreak key for engines whose MapScan
// tiebreaks on the stored value bytes. No current engine uses this form —
// sqlite, its last consumer, moved to the key column in the F22 conformance
// fix — but the derivation stays because the harness self-test uses it to
// prove that a wrong CursorKey form produces detected violations instead of
// a false pass.
func CursorKeyValueJSON(seed PaginationSeed) []byte {
	encoded, err := json.Marshal(seed.Item)
	if err != nil {
		panic(fmt.Sprintf("adttest.CursorKeyValueJSON: item not re-marshallable: %v", err))
	}

	return encoded
}

const (
	paginationSortValues = 6 // distinct sort values in the dataset
	paginationKeysPerVal = 4 // rows tying on each sort value
)

// paginationSeededKeys returns every seeded key in (sortValue, key) order.
// Keys are zero-padded so lexical order equals seeding order.
func paginationSeededKeys() []string {
	keys := make([]string, 0, paginationSortValues*paginationKeysPerVal)
	for v := range paginationSortValues {
		for n := range paginationKeysPerVal {
			keys = append(keys, fmt.Sprintf("k%02d-%02d", v, n))
		}
	}

	return keys
}

// seedPaginationCollection inserts the tie-heavy dataset via MapBackend:
// paginationSortValues sort values with paginationKeysPerVal tying keys each.
func seedPaginationCollection(
	ctx context.Context,
	t *testing.T,
	mb metaengine.MapBackend,
	col string,
) {
	t.Helper()

	for v := range paginationSortValues {
		for n := range paginationKeysPerVal {
			key := fmt.Sprintf("k%02d-%02d", v, n)
			if err := mb.MapSet(ctx, col, key, map[string]any{"key": key, "sort": v}); err != nil {
				t.Fatalf("MapSet %s: %v", key, err)
			}
		}
	}
}

// RunPaginationConformance walks MapScan pages with compound SortKeyCursor
// pagination over every probe and asserts exact-once coverage: every seeded
// row appears in exactly one page, sort order is monotone across page
// boundaries, ties stay in the engine's tiebreak-key order, and HasMore is
// set exactly while rows remain. Limits sweep around page boundaries and the
// dataset size (1, 3, 4, 5, 7, 23, 24, 25 over 24 rows).
func RunPaginationConformance(t *testing.T, probes []PaginationProbe) {
	t.Helper()

	for _, probe := range probes {
		t.Run(probe.Name, func(t *testing.T) {
			t.Parallel()

			eng := probe.Create(t)
			defer metaengine.DeferClose(eng)

			sb, ok := eng.(metaengine.ScanBackend)
			if !ok {
				t.Skipf("%s does not implement ScanBackend", probe.Name)
				return
			}

			mb, ok := eng.(metaengine.MapBackend)
			if !ok {
				t.Skipf("%s does not implement MapBackend", probe.Name)
				return
			}

			ctx := context.Background()
			col := fmt.Sprintf("pagination_conformance_%d", time.Now().UnixNano())
			seedPaginationCollection(ctx, t, mb, col)

			sortFn := paginationSortFn(t)

			for _, limit := range []int{1, 3, 4, 5, 7, 23, 24, 25} {
				t.Run(fmt.Sprintf("limit=%d", limit), func(t *testing.T) {
					for _, problem := range paginationWalk(ctx, t, sb, probe, col, sortFn, limit) {
						t.Errorf("limit=%d: %s", limit, problem)
					}
				})
			}
		})
	}
}

// paginationWalk drives one full pagination walk at a fixed limit and returns
// every rule break it finds (empty result = conformant). Returning
// problems instead of calling t.Errorf directly keeps the harness itself
// testable: the self-test proves a wrong CursorKey form produces violations.
func paginationWalk(
	ctx context.Context,
	t *testing.T,
	sb metaengine.ScanBackend,
	probe PaginationProbe,
	col string,
	sortFn func(a, b any) int,
	limit int,
) []string {
	var (
		problems []string
		keys     []string
		items    []any
		cursor   any
		pages    int
	)

	for {
		res, err := sb.MapScan(ctx, col, nil, sortFn, cursor, limit)
		if err != nil {
			return append(problems, fmt.Sprintf("MapScan page %d: %v", pages+1, err))
		}

		pages++

		if pages > paginationSortValues*paginationKeysPerVal+2 {
			return append(problems, fmt.Sprintf("walk did not terminate after %d pages", pages))
		}

		if len(res.Items) > limit {
			problems = append(problems, fmt.Sprintf(
				"page %d returned %d items, limit is %d", pages, len(res.Items), limit))
		}

		if len(res.Items) == 0 && res.HasMore {
			problems = append(problems, fmt.Sprintf("page %d is empty but reports HasMore", pages))
		}

		for _, item := range res.Items {
			key, _, ok := paginationRowFields(item)
			if !ok {
				return append(problems, fmt.Sprintf(
					"page %d item lacks the (string key, numeric sort) shape: %#v", pages, item))
			}

			keys = append(keys, key)
			items = append(items, item)
		}

		if !res.HasMore {
			break
		}

		if len(res.Items) == 0 {
			return append(problems, fmt.Sprintf(
				"page %d is empty with HasMore set — cursor deadlock", pages))
		}

		last := res.Items[len(res.Items)-1]
		key, sortVal, _ := paginationRowFields(last)
		cursor = metaengine.SortKeyCursor{
			Sort: sortVal,
			Key:  probe.CursorKey(PaginationSeed{Collection: col, Key: key, Item: last}),
		}
	}

	seeded := paginationSeededKeys()

	if got := len(keys); got != len(seeded) {
		problems = append(problems, fmt.Sprintf(
			"walk returned %d rows, want %d (drops and/or duplicates)", got, len(seeded)))
	}

	sorted := slices.Clone(keys)
	slices.Sort(sorted)
	if !slices.Equal(sorted, seeded) {
		problems = append(problems, fmt.Sprintf(
			"walk deviates from the seeded key set (drops/duplicates): got %v", sorted))
	}

	// Monotone sort order across page boundaries, and strictly increasing
	// tiebreak keys within equal sort values (the per-engine order the sort
	// itself uses, rebuilt through the probe's CursorKey derivation).
	for i := 1; i < len(items); i++ {
		prevKey, prevSort, prevOK := paginationRowFields(items[i-1])
		curKey, curSort, curOK := paginationRowFields(items[i])
		if !prevOK || !curOK {
			problems = append(problems, fmt.Sprintf("row %d: unparseable item pair", i))
			continue
		}

		if curSort < prevSort {
			problems = append(problems, fmt.Sprintf(
				"row %d: sort value %v after %v — order not monotone", i, curSort, prevSort))
		}

		if curSort == prevSort {
			prevCK := probe.CursorKey(
				PaginationSeed{Collection: col, Key: prevKey, Item: items[i-1]},
			)
			curCK := probe.CursorKey(PaginationSeed{Collection: col, Key: curKey, Item: items[i]})
			if bytes.Compare(prevCK, curCK) >= 0 {
				problems = append(problems, fmt.Sprintf(
					"rows %d-%d tie on sort value %v but tiebreak keys are not strictly increasing",
					i-1, i, curSort))
			}
		}
	}

	return problems
}

// paginationRowFields extracts the embedded map key and sort value from a
// returned row. Memory engines return the stored Go values (int sort fields);
// SQL and KV engines return decoded JSON (float64 sort fields).
func paginationRowFields(item any) (key string, sortVal float64, ok bool) {
	m, isMap := item.(map[string]any)
	if !isMap {
		return "", 0, false
	}

	k, isStr := m["key"].(string)
	if !isStr {
		return "", 0, false
	}

	switch sv := m["sort"].(type) {
	case int:
		return k, float64(sv), true
	case int64:
		return k, float64(sv), true
	case float64:
		return k, sv, true
	default:
		return "", 0, false
	}
}

// paginationSortFn builds the numeric sort comparator handed to MapScan. The
// comparator accepts both rows (maps with a numeric "sort" field) and bare
// cursor sort values (engines call sortFn(item, cursor.Sort) with the raw
// field value), and normalizes int vs float64 across engine families.
func paginationSortFn(t *testing.T) func(a, b any) int {
	return func(a, b any) int {
		av, aok := paginationSortableValue(a)
		bv, bok := paginationSortableValue(b)
		if !aok || !bok {
			t.Fatalf("pagination conformance: unsortable values %#v / %#v", a, b)
			return 0
		}

		switch {
		case av < bv:
			return -1
		case av > bv:
			return 1
		default:
			return 0
		}
	}
}

func paginationSortableValue(v any) (float64, bool) {
	switch x := v.(type) {
	case int:
		return float64(x), true
	case int64:
		return float64(x), true
	case float64:
		return x, true
	case map[string]any:
		if _, sv, ok := paginationRowFields(x); ok {
			return sv, true
		}
	}

	return 0, false
}
