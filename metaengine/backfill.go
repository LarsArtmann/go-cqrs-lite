package metaengine

import (
	"context"
	"errors"
	"fmt"

	errorfamily "github.com/larsartmann/go-error-family"
)

// KeyScanBackend is an optional capability: a paged key+value read over a
// collection's BASE meta_map rows in deterministic key order. Unlike MapScan,
// which routes to the planned table for planned collections, this capability
// always reads meta_map — that is exactly what makes it the right primitive
// for backfill (the rows a planned table is missing). Values arrive decoded
// (any), matching the MapGet contract.
type KeyScanBackend interface {
	MapScanKeyValues(
		ctx context.Context,
		collection string,
		cursor any,
		limit int,
	) (keys []any, values []any, hasMore bool, err error)
}

// ErrBackfillUnsupported is returned by BackfillPlannedCollection when the
// engine does not implement KeyScanBackend (the capability only makes sense
// for engines with a physical meta_map table).
var ErrBackfillUnsupported = errorfamily.NewRejection(
	"metaengine.backfill_unsupported",
	"engine does not implement KeyScanBackend; planned-table backfill requires a SQL meta_map store",
)

// BackfillPlannedCollection is the opt-in helper that softens the planned
// tables' no-backfill contract where operators need it: it copies the
// collection's existing meta_map rows into its registered planned table by
// re-issuing them through MapSet, so the engine's normal planned-write path
// recomputes every extracted column. Idempotent — MapSet upserts, so
// re-running converges to the same state. Rows are copied in deterministic
// key order with keyset paging (batchSize rows per round trip; <=0 selects
// the 500 default). Returns the number of rows copied.
//
// Values round-trip through a JSON decode/encode, so integers beyond
// float64 precision (> 2^53) lose exactness — backfilling such payloads
// requires a domain-specific copy. The planned collection must already be
// registered (ApplyLayoutPlan); rows written after registration are already
// visible and are harmlessly re-upserted.
func BackfillPlannedCollection(
	ctx context.Context,
	eng Engine,
	collection string,
	batchSize int,
) (int, error) {
	if batchSize <= 0 {
		batchSize = 500
	}

	kb, ok := eng.(KeyScanBackend)
	if !ok {
		return 0, fmt.Errorf("metaengine.BackfillPlannedCollection: %w", ErrBackfillUnsupported)
	}

	mb, ok := eng.(MapBackend)
	if !ok {
		return 0, errors.New("metaengine.BackfillPlannedCollection: engine does not implement MapBackend")
	}

	total := 0

	var cursor any

	for {
		keys, values, hasMore, err := kb.MapScanKeyValues(ctx, collection, cursor, batchSize)
		if err != nil {
			return total, fmt.Errorf("metaengine.BackfillPlannedCollection: scan: %w", err)
		}

		for i := range keys {
			if err := mb.MapSet(ctx, collection, keys[i], values[i]); err != nil {
				return total, fmt.Errorf(
					"metaengine.BackfillPlannedCollection: set %v: %w", keys[i], err)
			}

			cursor = keys[i]
			total++
		}

		if !hasMore {
			break
		}
	}

	return total, nil
}
