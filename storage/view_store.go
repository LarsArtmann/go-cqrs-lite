package storage

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	sqlpkg "github.com/larsartmann/go-cqrs-lite/storage/v3/sql"
)

// ViewColumn describes one SQL column that maps a field of view type V.
//
// Name is the SQL column name. Type is the SQL type declaration ("TEXT",
// "INTEGER", "REAL", etc.). Extract returns the column value from a *V for
// INSERT/UPDATE. Extract must not be nil.
type ViewColumn[V any] struct {
	Name    string
	Type    string
	Extract func(v *V) any
}

// ViewMapper defines how view type V maps to a dedicated SQL table.
//
// Table is the SQL table name (e.g. "users_view"). Columns lists the data
// columns; the key column (always TEXT PRIMARY KEY) is added automatically and
// must NOT appear in Columns. ScanRow reconstructs a *V from a sql.Rows.Scan
// callback — the dest slice matches the order of Columns exactly.
//
// Example:
//
//	mapper := storage.ViewMapper[TodoView]{
//	    Table: "todos_view",
//	    Columns: []storage.ViewColumn[TodoView]{
//	        {Name: "title", Type: "TEXT", Extract: func(v *TodoView) any { return v.Title }},
//	        {Name: "completed", Type: "INTEGER", Extract: func(v *TodoView) any { return v.Completed }},
//	        {Name: "tombstoned", Type: "INTEGER", Extract: func(v *TodoView) any { return v.Tombstoned }},
//	    },
//	    ScanRow: func(scan func(dest ...any) error) (*TodoView, error) {
//	        var v TodoView
//	        if err := scan(&v.Title, &v.Completed, &v.Tombstoned); err != nil {
//	            return nil, err
//	        }
//	        return &v, nil
//	    },
//	    TombstoneColumn: "tombstoned", // enables server-side tombstone filtering
//	}
type ViewMapper[V any] struct {
	Table   string
	Columns []ViewColumn[V]
	ScanRow func(scan func(dest ...any) error) (*V, error)

	// TombstoneColumn optionally names a boolean/integer column that marks
	// tombstoned records (0 = active, non-zero = tombstoned). When set, the
	// store implements [kv.TombstoneQuerier] and [Materialize.List] pushes
	// tombstone filtering to SQL instead of loading every record.
	//
	// The column must also appear in Columns with its Extract function.
	TombstoneColumn string
}

// SQLViewStore is a [kv.ViewStore] backed by a dedicated SQL table with real
// columns. Unlike [SQLKVStore] (which stores opaque blobs in a shared cqrs_kv
// table), SQLViewStore gives each view type its own table where every field is
// a queryable, indexable SQL column.
//
// This enables server-side WHERE, ORDER BY, and LIMIT/OFFSET pagination — the
// key advantage over KV-backed read models, which must load every record into
// memory to filter. See [SQLViewStore.Query].
//
// The store does NOT own the *sql.DB; the caller manages the connection
// lifecycle (same convention as [SQLKVStore]). The table is auto-created on
// construction unless [WithoutViewAutoMigrate] is passed.
type SQLViewStore[V any, K fmt.Stringer] struct {
	sqlpkg.DBHandle

	mapper ViewMapper[V]

	selectCols string // comma-separated column names for SELECT
	colCount   int    // number of data columns (excluding key)
}

// NewSQLiteViewStore creates a SQLViewStore for SQLite.
func NewSQLiteViewStore[V any, K fmt.Stringer](
	db *sql.DB,
	mapper ViewMapper[V],
	opts ...ViewStoreOption,
) (*SQLViewStore[V, K], error) {
	return newViewStore[V, K](db, sqlpkg.SQLiteDialect{}, mapper, opts)
}

// NewSQLViewStore creates a SQLViewStore for PostgreSQL.
func NewSQLViewStore[V any, K fmt.Stringer](
	db *sql.DB,
	mapper ViewMapper[V],
	opts ...ViewStoreOption,
) (*SQLViewStore[V, K], error) {
	return newViewStore[V, K](db, sqlpkg.PostgresDialect{}, mapper, opts)
}

// NewViewStoreWithDialect creates a SQLViewStore with an explicit dialect.
func NewViewStoreWithDialect[V any, K fmt.Stringer](
	db *sql.DB,
	dialect sqlpkg.Dialect,
	mapper ViewMapper[V],
	opts ...ViewStoreOption,
) (*SQLViewStore[V, K], error) {
	return newViewStore[V, K](db, dialect, mapper, opts)
}

func newViewStore[V any, K fmt.Stringer](
	db *sql.DB,
	dialect sqlpkg.Dialect,
	mapper ViewMapper[V],
	opts []ViewStoreOption,
) (*SQLViewStore[V, K], error) {
	if err := validateMapper(mapper); err != nil {
		return nil, fmt.Errorf("storage: view store: %w", err)
	}

	handle, err := sqlpkg.NewDBHandle(db, dialect)
	if err != nil {
		return nil, fmt.Errorf("storage: view store: %w", err)
	}

	cfg := viewStoreConfig{autoMigrate: true}

	for _, opt := range opts {
		opt(&cfg)
	}

	s := &SQLViewStore[V, K]{
		DBHandle:   handle,
		mapper:     mapper,
		selectCols: buildSelectCols(mapper),
		colCount:   len(mapper.Columns),
	}

	if cfg.autoMigrate {
		if err := s.createTable(context.Background()); err != nil {
			return nil, fmt.Errorf("storage: view store migrate: %w", err)
		}
	}

	return s, nil
}

func validateMapper[V any](m ViewMapper[V]) error {
	if m.Table == "" {
		return errMapperTableRequired
	}

	if m.ScanRow == nil {
		return errMapperScanRowRequired
	}

	if len(m.Columns) == 0 {
		return errMapperColumnsRequired
	}

	seen := make(map[string]struct{}, len(m.Columns))

	for i, col := range m.Columns {
		if col.Name == "" {
			return fmt.Errorf("mapper: column %d: %w", i, errMapperColumnNameEmpty)
		}

		if col.Extract == nil {
			return fmt.Errorf("mapper: column %d (%s): %w", i, col.Name, errMapperExtractRequired)
		}

		if strings.EqualFold(col.Name, "key") {
			return fmt.Errorf("mapper: column %d (%s): %w", i, col.Name, errMapperKeyReserved)
		}

		if _, dup := seen[col.Name]; dup {
			return fmt.Errorf("mapper: column %d (%s): %w", i, col.Name, errMapperDuplicateColumn)
		}

		seen[col.Name] = struct{}{}
	}

	return nil
}

func buildSelectCols[V any](mapper ViewMapper[V]) string {
	names := make([]string, 0, len(mapper.Columns))

	for _, c := range mapper.Columns {
		names = append(names, c.Name)
	}

	return strings.Join(names, ", ")
}

func (s *SQLViewStore[V, K]) createTable(ctx context.Context) error {
	var b strings.Builder

	fmt.Fprintf(&b, "CREATE TABLE IF NOT EXISTS %s (key TEXT PRIMARY KEY", s.mapper.Table)

	for _, col := range s.mapper.Columns {
		fmt.Fprintf(&b, ", %s %s", col.Name, col.Type)
	}

	b.WriteString(")")

	_, err := s.DB.ExecContext(ctx, b.String())
	if err != nil {
		return fmt.Errorf("create table %s: %w", s.mapper.Table, err)
	}

	return nil
}

func (s *SQLViewStore[V, K]) keyString(key K) string { return key.String() }
