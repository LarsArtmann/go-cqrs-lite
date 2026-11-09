package enginetest

import (
	"context"
	"fmt"
	"slices"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// RunPushdownStandardTiesTest pins compound-cursor pagination on the STANDARD
// pushdown path: collections WITHOUT a layout plan route through the engine's
// generic meta-map query, the variant RunKeysetPaginationTest never reaches
// because metaengine.Plan auto-applies a layout for declared sorts. The test
// seeds an unplanned tie-heavy collection (6 sort values × 5 keys each) via
// MapBackend, then walks it ASC and DESC through PushdownMapScan directly,
// asserting every row is visited exactly once at limit 4 (pages [4×7, 2]).
// Engines without MapBackend or PushdownScan skip.
func RunPushdownStandardTiesTest(t *testing.T, eng metaengine.Engine, collection string) {
	t.Helper()

	mb, ok := eng.(metaengine.MapBackend)
	if !ok {
		t.Skipf("engine %T does not implement MapBackend", eng)
	}

	pd, ok := eng.(metaengine.PushdownScan)
	if !ok {
		t.Skipf("engine %T does not implement PushdownScan", eng)
	}

	ctx := context.Background()

	const (
		sortValues = 6
		perSort    = 5
		limit      = 4
	)

	for sortVal := range sortValues {
		for n := range perSort {
			key := fmt.Sprintf("k%d-%02d", sortVal, n)

			if err := mb.MapSet(
				ctx,
				collection,
				key,
				map[string]any{"sort": sortVal, "key": key},
			); err != nil {
				t.Fatalf("MapSet[%s]: %v", key, err)
			}
		}
	}

	wantAsc := make([]string, 0, sortValues*perSort)
	for sortVal := range sortValues {
		for n := range perSort {
			wantAsc = append(wantAsc, fmt.Sprintf("k%d-%02d", sortVal, n))
		}
	}

	wantDesc := make([]string, 0, sortValues*perSort)
	for sortVal := sortValues - 1; sortVal >= 0; sortVal-- {
		for n := range perSort {
			wantDesc = append(wantDesc, fmt.Sprintf("k%d-%02d", sortVal, n))
		}
	}

	for _, tc := range []struct {
		name string
		desc bool
		want []string
	}{
		{name: "asc", desc: false, want: wantAsc},
		{name: "desc", desc: true, want: wantDesc},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sortSpec := metaengine.SortSpec{Column: "sort", Desc: tc.desc}

			var (
				visited []string
				pages   []int
				cursor  any
			)

			for range 50 {
				res, err := pd.PushdownMapScan(ctx, collection, nil, &sortSpec, cursor, limit)
				if err != nil {
					t.Fatalf("PushdownMapScan: %v", err)
				}

				for _, item := range res.Items {
					visited = append(
						visited,
						fmt.Sprintf("%v", metaengine.ItemFieldByName(item, "key")),
					)
				}

				pages = append(pages, len(res.Items))

				if !res.HasMore {
					break
				}

				if res.NextCursor == nil {
					t.Fatal("HasMore=true without NextCursor: keyset paging cannot continue")
				}

				cursor = res.NextCursor
			}

			if !slices.Equal(visited, tc.want) {
				t.Fatalf("visited %v\nwant    %v", visited, tc.want)
			}

			wantPages := []int{4, 4, 4, 4, 4, 4, 4, 2}
			if !slices.Equal(pages, wantPages) {
				t.Fatalf("page sizes %v, want %v", pages, wantPages)
			}
		})
	}
}
