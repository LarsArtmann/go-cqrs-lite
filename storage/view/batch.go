package view

import (
	"context"
	"fmt"
	"strings"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/kv/v4"
)

const sqliteMaxParams = 999

// BatchSet upserts multiple records atomically (within each chunk). This
// implements [kv.ViewBatchSetter] and is designed for projection replay
// throughput — replaying thousands of events one Set at a time is O(n) round
// trips; BatchSet reduces that to O(n / batchSize).
//
// The batch is chunked automatically to respect the SQLite 999-parameter
// limit. Each chunk runs in its own INSERT ... ON CONFLICT statement; the
// entire operation is NOT wrapped in a single transaction (callers that need
// all-or-nothing semantics should wrap in their own transaction).
func (s *SQLViewStore[V, K]) BatchSet(ctx context.Context, items []kv.ViewItem[V, K]) error {
	if len(items) == 0 {
		return nil
	}

	paramsPerRow := s.colCount + 1 // key + data columns
	maxRows := max(sqliteMaxParams/paramsPerRow, 1)

	for offset := 0; offset < len(items); offset += maxRows {
		end := min(offset+maxRows, len(items))

		if err := s.batchChunk(ctx, items[offset:end]); err != nil {
			return err
		}
	}

	return nil
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
		"INSERT INTO %s (%s) VALUES %s ON CONFLICT(key) DO UPDATE SET %s",
		s.mapper.Table,
		strings.Join(cols, ", "),
		strings.Join(placeholders, ", "),
		s.buildConflictSet(cols[1:]),
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
