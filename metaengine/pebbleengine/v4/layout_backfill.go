package pebbleengine

import (
	"github.com/cockroachdb/pebble"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// backfillBatchRows bounds the memory a backfill batch holds before it is
// committed; the plan is published only once every chunk has committed.
const backfillBatchRows = 5000

// backfillIndex writes the secondary index entries of every row already in
// the collection. MapSet only indexes writes made after the plan exists, so
// without this pass older rows would be invisible to indexed scans.
func (e *pebbleEngine) backfillIndex(col string, plan layoutPlan) error {
	prefix := collectionPrefix(col)

	iter, err := e.db.NewIter(&pebble.IterOptions{
		LowerBound: prefix,
		UpperBound: nextKey(prefix),
	})
	if err != nil {
		return err
	}

	defer metaengine.DeferClose(iter)

	batch := e.db.NewBatch()
	defer metaengine.DeferClose(batch)

	rows := 0

	for iter.First(); iter.Valid(); iter.Next() {
		// The map key is prefix + the encoded key string MapSet used, so the
		// suffix reproduces exactly the index entry keys MapSet would write.
		key := string(iter.Key()[len(prefix):])

		if err := e.writeIndexEntries(batch, col, key, iter.Value(), plan); err != nil {
			return err
		}

		if rows++; rows >= backfillBatchRows {
			if err := batch.Commit(e.writeOptions()); err != nil {
				return err
			}

			batch.Reset()

			rows = 0
		}
	}

	if err := iter.Error(); err != nil {
		return err
	}

	return batch.Commit(e.writeOptions())
}
