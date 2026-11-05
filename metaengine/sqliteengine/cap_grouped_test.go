package sqliteengine_test

import (
	"errors"
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4/adttest"
)

// TestSQLiteGroupedCap pins the per-group cap contract (ADR-0164, #114) on
// every sqliteengine layout variant:
//
//   - meta_map: the json_extract window-function pushdown (one DELETE with
//     ROW_NUMBER OVER PARTITION BY, RETURNING the deleted keys)
//   - planned table: the same statement against the REAL extracted columns
//     (WithColumnarLayout extracts every exported field)
//   - cell-versioned: the pushdown refuses (a bulk DELETE would bypass
//     meta_cell_versions) and the store's Go path still satisfies the
//     observable contract through MapDeleteAt
func TestSQLiteGroupedCap(t *testing.T) {
	t.Parallel()

	t.Run("metamap_pushdown", func(t *testing.T) {
		t.Parallel()
		adttest.AssertGroupedCapContract(t, newEngine(t))
	})

	t.Run("planned_columns", func(t *testing.T) {
		t.Parallel()
		adttest.AssertGroupedCapContract(t, newEngine(t), metaengine.WithColumnarLayout())
	})

	t.Run("versioned_go_fallback", func(t *testing.T) {
		t.Parallel()
		adttest.AssertGroupedCapContract(t, newVersionedSQLite(t))
	})

	// Plain WithCap on a SQL engine exercises the Go trim path over
	// map[string]any scan items (SQL engines decode rows via
	// DecodeStreamValue, not typed structs) — the path that panicked before
	// FieldValueAt taught the extractors both renderings.
	t.Run("plain_cap_map_items", func(t *testing.T) {
		t.Parallel()

		adttest.AssertPlainCapContract(t, newEngine(t))
	})
}

// TestSQLiteTrimGroupedCap_Refusals pins the two pushdown refusal branches:
// cell-versioned engines must not bulk-delete (ADR-0141 history would be
// bypassed), and neither identifier reaching the statement may be anything
// but a plain identifier. Both return ErrCapGroupPushdownUnsupported so the
// store falls back to its portable Go path (or the declaration-time
// validation already rejected the identifier).
func TestSQLiteTrimGroupedCap_Refusals(t *testing.T) {
	t.Parallel()

	trimmer, ok := newVersionedSQLite(t).(metaengine.CapGroupTrimmer)
	if !ok {
		t.Fatal("versioned sqlite engine does not satisfy metaengine.CapGroupTrimmer")
	}

	_, err := trimmer.TrimGroupedCap(t.Context(), "some_collection", "Owner", "Seq", 2)
	if !errors.Is(err, metaengine.ErrCapGroupPushdownUnsupported) {
		t.Fatalf("versioned engine: error %v, want ErrCapGroupPushdownUnsupported", err)
	}

	plain, ok := newEngine(t).(metaengine.CapGroupTrimmer)
	if !ok {
		t.Fatal("sqlite engine does not satisfy metaengine.CapGroupTrimmer")
	}

	_, err = plain.TrimGroupedCap(t.Context(), "some_collection", "Owner; DROP TABLE x", "Seq", 2)
	if !errors.Is(err, metaengine.ErrCapGroupPushdownUnsupported) {
		t.Fatalf("non-identifier group column: error %v, want ErrCapGroupPushdownUnsupported", err)
	}

	_, err = plain.TrimGroupedCap(t.Context(), "some_collection", "Owner", "Seq--", 2)
	if !errors.Is(err, metaengine.ErrCapGroupPushdownUnsupported) {
		t.Fatalf("non-identifier order column: error %v, want ErrCapGroupPushdownUnsupported", err)
	}
}
