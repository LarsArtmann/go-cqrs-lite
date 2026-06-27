package storage

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/larsartmann/go-cqrs-lite/kv/v3"
	sqlpkg "github.com/larsartmann/go-cqrs-lite/storage/v3/sql"
)

// RelationalStore provides dialect-agnostic read access to the tables of a
// [RelationalSchema]. It is the read-side companion to [RelationalProjection]:
// the projection writes events into the schema (via a transactional sink), and
// RelationalStore serves typed queries against the materialised tables — counts,
// cursor pagination, filtered lists — without the caller writing SQL.
//
// Like the projection, it is backend-portable: the same query code runs on
// SQLite or PostgreSQL depending on the dialect chosen at construction.
type RelationalStore struct {
	schema  RelationalSchema
	db      *sql.DB
	dialect sqlpkg.Dialect
}

// NewRelationalStore creates a read store for schema backed by db and dialect.
// The schema must already be migrated (NewRelationalProjection migrates
// automatically); this constructor performs no DDL.
func NewRelationalStore(
	schema RelationalSchema,
	db *sql.DB,
	dialect sqlpkg.Dialect,
) (*RelationalStore, error) {
	if err := schema.Validate(); err != nil {
		return nil, err
	}

	if db == nil {
		return nil, errRelationalNilDB
	}

	if dialect == nil {
		return nil, errRelationalNilDialect
	}

	return &RelationalStore{schema: schema, db: db, dialect: dialect}, nil
}

// Count returns the number of rows in table matching conditions (AND-joined).
// Empty conditions count every row in the table.
func (s *RelationalStore) Count(
	ctx context.Context,
	table string,
	conditions []kv.Condition,
) (int64, error) {
	if err := s.requireTable(table); err != nil {
		return 0, err
	}

	var b strings.Builder

	fmt.Fprintf(&b, "SELECT COUNT(*) FROM %s", table)

	whereClause, args := buildWhereClause(
		formatConditions(conditions, s.dialect),
		s.dialect.Placeholder,
	)

	if whereClause != "" {
		fmt.Fprintf(&b, " WHERE %s", whereClause)
	}

	var count int64

	if err := s.db.QueryRowContext(ctx, b.String(), args...).Scan(&count); err != nil {
		return 0, fmt.Errorf("relational store: count %s: %w", table, err)
	}

	return count, nil
}

// CountMany returns the row count for each named table in a single call. It is
// the read-side primitive for "stats" endpoints that report counts across many
// entity types (e.g. messages, channels, users, attachments). The returned map
// is keyed by table name.
func (s *RelationalStore) CountMany(
	ctx context.Context,
	tables []string,
) (map[string]int64, error) {
	out := make(map[string]int64, len(tables))

	for _, t := range tables {
		c, err := s.Count(ctx, t, nil)
		if err != nil {
			return nil, err
		}

		out[t] = c
	}

	return out, nil
}

// RelationalQuery describes a filtered, ordered, paginated read against a table.
// Conditions, OrderBy, Limit, and Offset mirror [kv.ViewQuery] semantics.
type RelationalQuery struct {
	Conditions []kv.Condition
	OrderBy    string
	Desc       bool
	Limit      int
	Offset     int
}

// Query runs a filtered, ordered, paginated query against table and scans each
// row with scanFn. scanFn receives a func(dest ...any) error (the same callback
// shape [ViewMapper].ScanRow uses), so callers control how columns map to a
// struct. The columns read are NOT inferred; pass them explicitly via columns.
//
// Conditions are AND-joined into a parameterised WHERE clause. OrderBy defaults
// to the first primary-key column (or "rowid" when the table has none). Limit
// caps the result set; Offset skips leading rows.
func (s *RelationalStore) Query(
	ctx context.Context,
	table string,
	columns []string,
	q RelationalQuery,
	scanFn func(scan func(dest ...any) error) error,
) error {
	if err := s.requireTable(table); err != nil {
		return err
	}

	if len(columns) == 0 {
		return fmt.Errorf("relational store: query %s: %w", table, errQueryNoColumns)
	}

	colList := strings.Join(columns, ", ")

	var b strings.Builder

	fmt.Fprintf(&b, "SELECT %s FROM %s", colList, table)

	whereClause, args := buildWhereClause(
		formatConditions(q.Conditions, s.dialect),
		s.dialect.Placeholder,
	)

	if whereClause != "" {
		fmt.Fprintf(&b, " WHERE %s", whereClause)
	}

	orderCol := q.OrderBy
	if orderCol == "" {
		orderCol = s.defaultOrder(table)
	}

	dir := "ASC"
	if q.Desc {
		dir = "DESC"
	}

	fmt.Fprintf(&b, " ORDER BY %s %s", orderCol, dir)

	paramIdx := len(args) + 1

	if q.Limit > 0 {
		fmt.Fprintf(&b, " LIMIT %s", s.dialect.Placeholder(paramIdx))
		args = append(args, q.Limit)
		paramIdx++

		if q.Offset > 0 {
			fmt.Fprintf(&b, " OFFSET %s", s.dialect.Placeholder(paramIdx))
			args = append(args, q.Offset)
		}
	} else if q.Offset > 0 {
		fmt.Fprintf(&b, " LIMIT %s OFFSET %s",
			s.dialect.Placeholder(paramIdx), s.dialect.Placeholder(paramIdx+1))
		args = append(args, -1, q.Offset)
	}

	rows, err := s.db.QueryContext(ctx, b.String(), args...)
	if err != nil {
		return fmt.Errorf("relational store: query %s: %w", table, err)
	}

	defer func() { _ = rows.Close() }()

	for rows.Next() {
		if err := scanFn(rows.Scan); err != nil {
			return fmt.Errorf("relational store: scan %s row: %w", table, err)
		}
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("relational store: query %s: %w", table, err)
	}

	return nil
}

// defaultOrder returns the table's first primary-key column, or "rowid" when
// the table declares no primary key (SQLite/Postgres both expose rowid).
func (s *RelationalStore) defaultOrder(table string) string {
	t := s.schema.Table(table)
	if t != nil && len(t.PrimaryKey) > 0 {
		return t.PrimaryKey[0]
	}

	return "rowid"
}

func (s *RelationalStore) requireTable(table string) error {
	if s.schema.Table(table) == nil {
		return fmt.Errorf("relational store: %w: %q", errQueryUnknownTable, table)
	}

	return nil
}

var (
	errQueryNoColumns    = fmt.Errorf("%w: query: at least one column is required", errRelational)
	errQueryUnknownTable = fmt.Errorf("%w: query: table not declared in schema", errRelational)
)
