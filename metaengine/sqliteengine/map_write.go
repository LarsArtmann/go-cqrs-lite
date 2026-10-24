package sqliteengine

import (
	"context"
	"time"
)

// mapSetEncoded serializes value, writes the (collection, key) row, and records
// the version row when cell versioning is enabled. Encoding happens first so a
// value JSON cannot represent (channel, function, cyclic structure) is rejected
// before any row is written, and the version row reuses the same bytes as the
// latest view.
func (e *sqliteEngine) mapSetEncoded(
	ctx context.Context,
	col string,
	key any,
	value any,
	ts time.Time,
) error {
	encoded, err := encodeValue(value)
	if err != nil {
		return err
	}

	keyStr := encodeKey(key)

	if _, err := e.xc(ctx).exec(ctx, e.queries.mapSet, col, keyStr, encoded); err != nil {
		return err //nolint:wrapcheck // passthrough
	}

	if e.versioning {
		return e.recordVersionRow(ctx, col, keyStr, encoded, ts)
	}

	return nil
}
