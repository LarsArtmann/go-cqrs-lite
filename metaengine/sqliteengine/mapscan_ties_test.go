package sqliteengine_test

import (
	"context"
	"encoding/json/v2"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// sqliteMapScanTieRows seeds a tie-heavy collection (6 sort values × 5 rows)
// into the plain meta_map store (no layout plan, so MapScan takes the
// fallback path that sorts in Go).
func sqliteMapScanTieRows(ctx context.Context, mb metaengine.MapBackend) {
	for sortVal := 0; sortVal < 6; sortVal++ {
		for n := 0; n < 5; n++ {
			key := fmt.Sprintf("k%d-%02d", sortVal, n)
			Expect(mb.MapSet(ctx, "ties", key, map[string]any{"sort": sortVal, "key": key})).
				To(Succeed())
		}
	}
}

func sqliteTieSortFn(a, b any) int {
	av := a.(map[string]any)["sort"] //nolint:forcetypeassert // rows are decoded JSON maps by construction
	bv := b.(map[string]any)["sort"] //nolint:forcetypeassert // rows are decoded JSON maps by construction

	switch {
	case toF(av) < toF(bv):
		return -1
	case toF(av) > toF(bv):
		return 1
	default:
		return 0
	}
}

func toF(v any) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case int:
		return float64(n)
	default:
		Fail(fmt.Sprintf("unexpected sort operand %T", v))
		return 0
	}
}

var _ = Describe("SQLiteEngine MapScan compound cursor (regression)", func() {
	It("paginates a tie-heavy collection without drops or dupes via SortKeyCursor", func() {
		eng, db := newSQLiteEngine()
		DeferCleanup(func() { _ = eng.Close() })
		DeferCleanup(func() { _ = db.Close() })

		mb := eng.(metaengine.MapBackend)  //nolint:forcetypeassert // engine under test implements it
		sb := eng.(metaengine.ScanBackend) //nolint:forcetypeassert // engine under test implements it
		ctx := context.Background()

		sqliteMapScanTieRows(ctx, mb)

		const limit = 4

		var got []string

		var cursor any

		for page := 0; page < 100; page++ {
			result, err := sb.MapScan(ctx, "ties", nil, sqliteTieSortFn, cursor, limit)
			Expect(err).NotTo(HaveOccurred())

			for _, item := range result.Items {
				got = append(
					got,
					item.(map[string]any)["key"].(string),
				) //nolint:forcetypeassert // by construction
			}

			if !result.HasMore {
				break
			}

			// The sqlite fallback path keys pairs by the stored JSON value, so
			// the compound cursor key is the canonical encoding of the last
			// returned row.
			last := result.Items[len(result.Items)-1].(map[string]any) //nolint:forcetypeassert // by construction
			encoded, err := json.Marshal(last)
			Expect(err).NotTo(HaveOccurred())

			cursor = metaengine.SortKeyCursor{Sort: last["sort"], Key: encoded}
		}

		Expect(got).To(HaveLen(30))

		seen := map[string]bool{}
		for _, key := range got {
			Expect(seen[key]).To(BeFalse(), "duplicate row %q", key)
			seen[key] = true
		}

		// Global order: (sort, stored JSON bytes). The walk must serve every
		// row exactly once in that order across pages.
		type row struct {
			sort int
			key  string
		}

		rows := make([]row, 0, 30)
		for sortVal := 0; sortVal < 6; sortVal++ {
			for n := 0; n < 5; n++ {
				rows = append(rows, row{sort: sortVal, key: fmt.Sprintf("k%d-%02d", sortVal, n)})
			}
		}

		for i, r := range rows {
			Expect(got[i]).To(Equal(r.key), "position %d", i)
		}
	})
})
