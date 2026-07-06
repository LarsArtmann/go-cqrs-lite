package view

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/kv/v3"
)

// Get retrieves the record for key. Returns [kv.ErrNotFound] if no row exists.
func (s *SQLViewStore[V, K]) Get(ctx context.Context, key K) (*V, error) {
	q := fmt.Sprintf("SELECT %s FROM %s WHERE key = %s",
		s.selectCols, s.mapper.Table, s.Dialect.Placeholder(1))

	row := s.DB.QueryRowContext(ctx, q, s.keyString(key))

	val, err := s.mapper.ScanRow(row.Scan)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, kv.ErrNotFound
		}

		return nil, errorfamily.WrapCorruption(err, "storage.view.get",
			fmt.Sprintf("get key %q", key.String()))
	}

	return val, nil
}

// Set upserts val under key, replacing any existing record.
func (s *SQLViewStore[V, K]) Set(ctx context.Context, key K, val *V) error {
	if val == nil {
		return errorfamily.WrapRejection(errNilViewValue, "storage.view.set_nil",
			fmt.Sprintf("nil view value: key %q", key.String()))
	}

	cols := make([]string, 0, s.colCount+1)
	args := make([]any, 0, s.colCount+1)

	cols = append(cols, "key")
	args = append(args, s.keyString(key))

	for _, c := range s.mapper.Columns {
		cols = append(cols, c.Name)
		args = append(args, c.Extract(val))
	}

	placeholders := make([]string, 0, len(cols))

	for i := range cols {
		placeholders = append(placeholders, s.Dialect.Placeholder(i+1))
	}

	q := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s) ON CONFLICT(key) DO UPDATE SET %s",
		s.mapper.Table,
		strings.Join(cols, ", "),
		strings.Join(placeholders, ", "),
		s.buildConflictSet(cols[1:]),
	)

	_, err := s.DB.ExecContext(ctx, q, args...)
	if err != nil {
		return errorfamily.WrapTransient(err, "storage.view.set",
			fmt.Sprintf("set key %q", key.String()))
	}

	return nil
}

func (s *SQLViewStore[V, K]) buildConflictSet(dataCols []string) string {
	parts := make([]string, 0, len(dataCols))

	for _, col := range dataCols {
		parts = append(parts, col+" = excluded."+col)
	}

	return strings.Join(parts, ", ")
}

// Delete removes the record for key. Deleting a missing key is a no-op.
func (s *SQLViewStore[V, K]) Delete(ctx context.Context, key K) error {
	q := fmt.Sprintf("DELETE FROM %s WHERE key = %s", s.mapper.Table, s.Dialect.Placeholder(1))

	_, err := s.DB.ExecContext(ctx, q, s.keyString(key))
	if err != nil {
		return errorfamily.WrapTransient(err, "storage.view.delete",
			fmt.Sprintf("delete key %q", key.String()))
	}

	return nil
}

// Scan returns all records ordered by key. If prefix is non-empty, only records
// whose key starts with prefix are returned.
func (s *SQLViewStore[V, K]) Scan(ctx context.Context, prefix []byte) ([]*V, error) {
	var q string

	var args []any

	if len(prefix) > 0 {
		q = fmt.Sprintf("SELECT %s FROM %s WHERE key LIKE %s ORDER BY key",
			s.selectCols, s.mapper.Table, s.Dialect.Placeholder(1))
		args = []any{string(prefix) + "%"}
	} else {
		q = fmt.Sprintf("SELECT %s FROM %s ORDER BY key", s.selectCols, s.mapper.Table)
	}

	rows, err := s.DB.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, errorfamily.WrapTransient(err, "storage.view.scan", "scan records")
	}

	defer func() { _ = rows.Close() }()

	return s.scanRows(rows)
}

func (s *SQLViewStore[V, K]) scanRows(rows *sql.Rows) ([]*V, error) {
	results := make([]*V, 0)

	for rows.Next() {
		val, err := s.mapper.ScanRow(rows.Scan)
		if err != nil {
			return nil, errorfamily.WrapCorruption(
				err,
				"storage.view.scan_row",
				"scan row during scan",
			)
		}

		results = append(results, val)
	}

	if err := rows.Err(); err != nil {
		return nil, errorfamily.WrapTransient(err, "storage.view.scan_rows_err", "rows iteration")
	}

	return results, nil
}
