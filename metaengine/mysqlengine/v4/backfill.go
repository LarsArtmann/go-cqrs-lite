package mysqlengine

import (
	"context"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// MapScanKeyValues implements metaengine.KeyScanBackend: a paged key+value
// read over the BASE meta_map table (never the planned table), in
// deterministic key order — the read primitive for planned-table backfill.
// cursor is the last key of the previous page (nil/"" for the first page).
func (e *mysqlEngine) MapScanKeyValues(
	ctx context.Context,
	collection string,
	cursor any,
	limit int,
) ([]any, []any, bool, error) {
	limit = metaengine.ScanLimit(limit)

	return metaengine.ScanKeyValuesPage(ctx, e.db,
		"SELECT `key`, value FROM meta_map "+
			"WHERE collection = ? AND (? IS NULL OR `key` > ?) "+
			"ORDER BY `key` LIMIT ?",
		[]any{collection, metaengine.CursorArg(cursor), metaengine.CursorArg(cursor), limit},
		limit, "mysqlengine.MapScanKeyValues")
}
