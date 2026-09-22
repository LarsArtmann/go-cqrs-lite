package bigtableengine

import (
	"context"
	"errors"
	"time"

	"cloud.google.com/go/bigtable"
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// ms truncation: BigTable cell timestamps are millisecond-granularity —
// anything finer collapses to the same version (ADR-0141 §1).
func cellTimestamp(ts time.Time) bigtable.Timestamp {
	return bigtable.Time(ts.Truncate(time.Millisecond))
}

// asOfEnd converts an INCLUSIVE as-of bound into the filter's exclusive
// end. The SDK truncates range bounds to milliseconds, so the end must step
// a full millisecond past the floor — with ms-aligned cell timestamps this
// captures exactly the cells with ts <= t.
func asOfEnd(t time.Time) bigtable.Timestamp {
	return cellTimestamp(t) + 1000
}

// --- metaengine.MapBackend ---

func (e *bigtableEngine) MapSet(ctx context.Context, col string, key any, value any) error {
	return e.MapSetAt(ctx, col, key, value, time.Now())
}

func (e *bigtableEngine) MapDelete(ctx context.Context, col string, key any) error {
	return e.MapDeleteAt(ctx, col, key, time.Now())
}

func (e *bigtableEngine) MapGet(ctx context.Context, col string, key any) (any, bool, error) {
	row, err := e.tbl.ReadRow(ctx, rowKey(col, key),
		bigtable.RowFilter(bigtable.ChainFilters(
			bigtable.FamilyFilter(family),
			bigtable.LatestNFilter(1),
		)))
	if err != nil {
		return nil, false, wrapOp("map get", err)
	}

	cell, ok := latestCell(row)
	if !ok || len(cell.Value) == 0 {
		return nil, false, nil // absent or tombstone
	}

	return metaengine.DecodeStreamValue(string(cell.Value)), true, nil
}

func latestCell(row bigtable.Row) (bigtable.ReadItem, bool) {
	cells, ok := row[family]
	if !ok || len(cells) == 0 {
		return bigtable.ReadItem{}, false
	}

	return cells[0], true // ReadRows returns newest-first
}

func wrapOp(op string, err error) error {
	return errors.Join(
		errors.New("bigtableengine: "+op),
		err,
	) //nolint:err113,goerr113 // dynamic join
}

// --- metaengine.VersionedWriter (native) ---

func (e *bigtableEngine) MapSetAt(
	ctx context.Context,
	col string,
	key any,
	value any,
	ts time.Time,
) error {
	mut := bigtable.NewMutation()
	mut.Set(family, column, cellTimestamp(ts), encodeJSON(value))

	if err := e.tbl.Apply(ctx, rowKey(col, key), mut); err != nil {
		return wrapOp("map set-at", err)
	}

	return nil
}

// MapDeleteAt writes an EMPTY-value cell at ts — BigTable's timestamped
// tombstone. As-of reads at t >= ts resolve to the empty value and report
// the key absent; earlier versions survive untouched.
func (e *bigtableEngine) MapDeleteAt(
	ctx context.Context,
	col string,
	key any,
	ts time.Time,
) error {
	mut := bigtable.NewMutation()
	mut.Set(family, column, cellTimestamp(ts), nil)

	if err := e.tbl.Apply(ctx, rowKey(col, key), mut); err != nil {
		return wrapOp("map delete-at", err)
	}

	return nil
}

// --- metaengine.VersionedStorage (native) ---

func (e *bigtableEngine) readAsOf(
	ctx context.Context,
	row string,
	at time.Time,
) (bigtable.ReadItem, bool, error) {
	rowData, err := e.tbl.ReadRow(ctx, row,
		bigtable.RowFilter(bigtable.ChainFilters(
			bigtable.FamilyFilter(family),
			// Order matters: narrow to versions <= at FIRST, then take the
			// newest — LatestN before the range filter would select the
			// newest overall and drop it against the bound.
			bigtable.TimestampRangeFilterMicros(0, asOfEnd(at)),
			bigtable.LatestNFilter(1),
		)))
	if err != nil {
		return bigtable.ReadItem{}, false, wrapOp("map get-as-of", err)
	}

	cell, ok := latestCell(rowData)

	return cell, ok, nil
}

func (e *bigtableEngine) MapGetAsOf(
	ctx context.Context,
	col, key string,
	t time.Time,
) (any, error) {
	cell, ok, err := e.readAsOf(ctx, rowKey(col, key), t)
	if err != nil {
		return nil, err
	}

	if !ok || len(cell.Value) == 0 {
		return nil, metaengine.ErrNotFound
	}

	return metaengine.DecodeStreamValue(string(cell.Value)), nil
}

func (e *bigtableEngine) MapExistsAsOf(
	ctx context.Context,
	col, key string,
	t time.Time,
) (bool, error) {
	cell, ok, err := e.readAsOf(ctx, rowKey(col, key), t)
	if err != nil || !ok {
		return false, err
	}

	return len(cell.Value) > 0, nil
}

// --- metaengine.CellHistoryReader (native) ---

// MapHistory returns the cell's versions within [from, to], newest-first.
func (e *bigtableEngine) MapHistory(
	ctx context.Context,
	col, key string,
	from, to time.Time,
) ([]metaengine.CellVersion, error) {
	rowData, err := e.tbl.ReadRow(ctx, rowKey(col, key),
		bigtable.RowFilter(bigtable.ChainFilters(
			bigtable.FamilyFilter(family),
			bigtable.TimestampRangeFilterMicros(cellTimestamp(from), asOfEnd(to)),
		)))
	if err != nil {
		return nil, wrapOp("map history", err)
	}

	cells := rowData[family] // newest-first

	out := make([]metaengine.CellVersion, 0, len(cells))

	for _, cell := range cells {
		version := metaengine.CellVersion{Timestamp: cell.Timestamp.Time()}
		if len(cell.Value) > 0 {
			version.Value = metaengine.DecodeStreamValue(string(cell.Value))
		}

		out = append(out, version)
	}

	return out, nil
}

// Compile-time assertions: the full temporal capability set, natively.
var (
	_ metaengine.MapBackend        = (*bigtableEngine)(nil)
	_ metaengine.VersionedStorage  = (*bigtableEngine)(nil)
	_ metaengine.VersionedWriter   = (*bigtableEngine)(nil)
	_ metaengine.CellHistoryReader = (*bigtableEngine)(nil)
)
