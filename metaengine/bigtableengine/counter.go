package bigtableengine

import (
	"context"
	"encoding/binary"
	"strings"

	"cloud.google.com/go/bigtable"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// --- metaengine.CounterBackend (native ReadModifyWrite increments) ---

// counterRow keys counter cells separately from map cells: family
// counterFamily, one row per counter, column "v" holding the running total.
func counterRow(col string, ckey string) string {
	return "c\x00" + col + "\x00" + ckey
}

func (e *bigtableEngine) CounterIncrement(
	ctx context.Context,
	col string,
	deltas metaengine.Delta,
) error {
	for ckey, delta := range deltas {
		rmw := bigtable.NewReadModifyWrite()
		rmw.Increment(counterFamily, column, delta)

		if _, err := e.tbl.ApplyReadModifyWrite(ctx, counterRow(col, ckey), rmw); err != nil {
			return wrapOp("counter increment", err)
		}
	}

	return nil
}

// CounterGet streams every counter row of the collection (prefix scan, one
// RPC) and decodes the running totals. ReadModifyWrite increments store the
// total as an 8-byte big-endian integer.
func (e *bigtableEngine) CounterGet(ctx context.Context, col string) (map[string]int64, error) {
	prefix := "c\x00" + col + "\x00"

	result := make(map[string]int64)

	err := e.tbl.ReadRows(ctx, bigtable.PrefixRange(prefix), func(row bigtable.Row) bool {
		cells, ok := row[counterFamily]
		if !ok || len(cells) == 0 || len(cells[0].Value) != 8 {
			return true
		}

		result[strings.TrimPrefix(row.Key(), prefix)] =
			int64(binary.BigEndian.Uint64(cells[0].Value))

		return true
	}, bigtable.RowFilter(bigtable.ChainFilters(
		bigtable.FamilyFilter(counterFamily),
		bigtable.LatestNFilter(1),
	)))
	if err != nil {
		return nil, wrapOp("counter get", err)
	}

	return result, nil
}

var _ metaengine.CounterBackend = (*bigtableEngine)(nil)
