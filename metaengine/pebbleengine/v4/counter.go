package pebbleengine

import (
	"context"
	"errors"

	"github.com/cockroachdb/pebble"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

func (e *pebbleEngine) CounterIncrement(
	_ context.Context,
	col string,
	deltas metaengine.Delta,
) error {
	// Get-then-Set across separate writes loses increments when two callers
	// read the same value, and a mid-loop failure would leave a multi-key
	// Delta half applied. e.mu serialises the read-modify-write against other
	// counter writers; the single batch commits every key or none.
	e.mu.Lock()
	defer e.mu.Unlock()

	batch := e.db.NewIndexedBatch()
	defer metaengine.DeferClose(batch)

	for k, d := range deltas {
		ck := counterKey(col, k)

		var current int64

		val, closer, err := batch.Get(ck)
		if err == nil {
			current = decodeCounterValue(val)
			//cqrs-lint:ignore(C023) library code or intentional pattern
			_ = closer.Close() //cqrs-lint:ignore(C015) pebble closer, error is always nil
		} else if !errors.Is(err, pebble.ErrNotFound) {
			return err
		}

		if err := batch.Set(ck, encodeCounterValue(current+d), nil); err != nil {
			return err
		}
	}

	return batch.Commit(e.writeOptions())
}
