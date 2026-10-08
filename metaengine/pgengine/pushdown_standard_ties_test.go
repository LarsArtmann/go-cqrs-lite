package pgengine_test

import (
	"context"
	"fmt"
	"slices"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// TestPg_PushdownStandardTies pins compound-cursor pagination on the STANDARD
// pushdown path: collections without a layout plan route through the
// meta_map + JSONB query, the variant the enginetest harness never reaches
// because metaengine.Plan auto-applies a layout for declared sorts.
func TestPg_PushdownStandardTies(t *testing.T) {
	t.Parallel()

	eng := mustNewPgEngine(t)

	mb := eng.(metaengine.MapBackend)   //nolint:forcetypeassert // engine under test implements it
	pd := eng.(metaengine.PushdownScan) //nolint:forcetypeassert // engine under test implements it
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
				"pd_ties_pg",
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
				res, err := pd.PushdownMapScan(ctx, "pd_ties_pg", nil, &sortSpec, cursor, limit)
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
