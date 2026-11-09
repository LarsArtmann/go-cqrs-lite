package view

import (
	"context"
	"fmt"
	"strings"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/kv/v4"
	sqlpkg "github.com/larsartmann/go-cqrs-lite/storage/v4/sql"
)

// bytesPerViewRowOverhead bounds the fixed per-row statement cost beyond the
// column values: key, placeholder syntax, and driver protocol framing.
const bytesPerViewRowOverhead = 64

// BatchSet upserts multiple records atomically (within each chunk). This
// implements [kv.ViewBatchSetter] and is designed for projection replay
// throughput — replaying thousands of events one Set at a time is O(n) round
// trips; BatchSet reduces that to O(n / batchSize).
//
// The batch is chunked automatically to respect BOTH the dialect's
// bound-parameter limit (999 for SQLite, 32767 for PostgreSQL/MySQL/DuckDB)
// and an estimated serialized-statement byte cap
// (sql.MaxStatementBytes), so large view values shrink chunks instead of
// exceeding MariaDB/MySQL max_allowed_packet. Each chunk runs in its own
// INSERT ... ON CONFLICT statement; the entire operation is NOT wrapped in a
// single transaction (callers that need all-or-nothing semantics should wrap
// in their own transaction).
func (s *SQLViewStore[V, K]) BatchSet(ctx context.Context, items []kv.ViewItem[V, K]) error {
	if len(items) == 0 {
		return nil
	}

	paramsPerRow := s.colCount + 1 // key + data columns
	maxRows := max(sqlpkg.MaxParametersForDialect(s.Dialect)/paramsPerRow, 1)

	for offset := 0; offset < len(items); {
		rows := sqlpkg.RowsWithinByteCap(offset, len(items), maxRows, func(i int) int {
			return s.estimateItemBytes(items[i])
		})

		if err := s.batchChunk(ctx, items[offset:offset+rows]); err != nil {
			return err
		}
		offset += rows
	}

	return nil
}

// estimateItemBytes bounds the serialized statement bytes one item's row
// contributes. Extract is a pure field projection, so calling it for the
// estimate in addition to the real bind is safe and cheap.
func (s *SQLViewStore[V, K]) estimateItemBytes(item kv.ViewItem[V, K]) int {
	total := len(s.keyString(item.Key)) + bytesPerViewRowOverhead
	for _, c := range s.mapper.Columns {
		total += estimateArgBytes(c.Extract(item.Value))
	}

	return total
}

// estimateArgBytes bounds the wire bytes a bound argument contributes to a
// statement. String and byte-slice values dominate; other types get fixed
// slack since they cannot carry large payloads.
func estimateArgBytes(v any) int {
	switch t := v.(type) {
	case string:
		return len(t) + 16
	case []byte:
		return len(t) + 16
	case time.Time:
		return 64
	default:
		return 32
	}
}

func (s *SQLViewStore[V, K]) batchChunk(ctx context.Context, items []kv.ViewItem[V, K]) error {
	cols := append([]string{keyColumnName}, columnNames(s.mapper.Columns)...)
	rowCount := len(items)
	paramsPerRow := len(cols)

	placeholders := make([]string, 0, rowCount)
	args := make([]any, 0, rowCount*paramsPerRow)

	for rowIdx, item := range items {
		if item.Value == nil {
			return errorfamily.WrapRejection(errNilViewValue, "storage.view.batch_nil",
				fmt.Sprintf("nil view value: key %q", item.Key.String()))
		}

		rowPlaceholders := make([]string, 0, paramsPerRow)
		base := rowIdx * paramsPerRow

		rowPlaceholders = append(rowPlaceholders, s.Dialect.Placeholder(base+1))
		args = append(args, s.keyString(item.Key))

		for _, c := range s.mapper.Columns {
			rowPlaceholders = append(
				rowPlaceholders,
				s.Dialect.Placeholder(base+len(rowPlaceholders)+1),
			)
			args = append(args, c.Extract(item.Value))
		}

		placeholders = append(placeholders, "("+strings.Join(rowPlaceholders, ", ")+")")
	}

	q := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES %s %s",
		s.mapper.Table,
		strings.Join(cols, ", "),
		strings.Join(placeholders, ", "),
		s.Dialect.OnConflictDoUpdate([]string{"key"}, []string{s.buildConflictSet(cols[1:])}),
	)

	if _, err := s.executor().ExecContext(ctx, q, args...); err != nil {
		return errorfamily.WrapTransient(err, "storage.view.batch_chunk", "batch insert chunk")
	}

	return nil
}

func columnNames[V any](cols []ViewColumn[V]) []string {
	names := make([]string, 0, len(cols))
	for _, c := range cols {
		names = append(names, c.Name)
	}

	return names
}

// DeleteAll removes all records from the view table. This implements
// [kv.ViewResetter] and is used for projection resets — wiping a read model
// before rebuilding it from the event journal.
func (s *SQLViewStore[V, K]) DeleteAll(ctx context.Context) error {
	q := "DELETE FROM " + s.mapper.Table

	if _, err := s.executor().ExecContext(ctx, q); err != nil {
		return errorfamily.WrapTransient(err, "storage.view.delete_all", "delete all records")
	}

	return nil
}
