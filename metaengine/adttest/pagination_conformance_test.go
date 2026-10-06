package adttest

import (
	"context"
	"strings"
	"testing"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// TestPaginationWalk_MemoryExactOnce proves the harness passes on a conformant
// engine/key-form pair: the memory engine with its raw map-key tiebreak form.
func TestPaginationWalk_MemoryExactOnce(t *testing.T) {
	t.Parallel()

	eng := metaengine.NewMemoryEngine()
	defer eng.Close()

	sb := eng.(metaengine.ScanBackend) //nolint:forcetypeassert // memory implements ScanBackend
	ctx := context.Background()
	col := t.Name()
	seedPaginationCollection(ctx, t, eng.(metaengine.MapBackend), col) //nolint:forcetypeassert // memory implements MapBackend

	probe := PaginationProbe{
		Factory:   Factory{Name: "memory"},
		CursorKey: CursorKeyRaw,
	}

	for _, limit := range []int{1, 3, 4, 5, 7, 23, 24, 25} {
		if problems := paginationWalk(ctx, t, sb, probe, col, paginationSortFn(t), limit); len(problems) > 0 {
			t.Errorf("limit=%d: expected conformance, got violations: %s", limit, strings.Join(problems, "; "))
		}
	}
}

// TestPaginationWalk_WrongKeyFormDetected proves the harness FAILS when the
// probe rebuilds cursors with the wrong tiebreak key form: the value-JSON form
// on the memory engine (whose ties order by the raw map key) must produce
// drops or duplicates, not a false pass.
func TestPaginationWalk_WrongKeyFormDetected(t *testing.T) {
	t.Parallel()

	eng := metaengine.NewMemoryEngine()
	defer eng.Close()

	sb := eng.(metaengine.ScanBackend) //nolint:forcetypeassert // memory implements ScanBackend
	ctx := context.Background()
	col := t.Name()
	seedPaginationCollection(ctx, t, eng.(metaengine.MapBackend), col) //nolint:forcetypeassert // memory implements MapBackend

	probe := PaginationProbe{
		Factory:   Factory{Name: "memory-wrong-form"},
		CursorKey: CursorKeyValueJSON,
	}

	detected := false

	for _, limit := range []int{1, 3, 4, 5, 7, 23, 24, 25} {
		if problems := paginationWalk(ctx, t, sb, probe, col, paginationSortFn(t), limit); len(problems) > 0 {
			detected = true
			break
		}
	}

	if !detected {
		t.Error("wrong CursorKey form produced zero violations — the conformance walk cannot detect tie-lossy pagination")
	}
}
