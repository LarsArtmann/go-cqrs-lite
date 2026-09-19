package claiming

import (
	"context"
	"database/sql"
	"fmt"
)

// EnsureLeaseColumn adds the lease column to tables created before
// claiming existed. Idempotent per dialect: Postgres and DuckDB use ADD
// COLUMN IF NOT EXISTS; SQLite and MySQL have no such form, so the column
// is probed first (pragma_table_info / information_schema) and added only
// when missing. The column type matches each dialect's time representation
// (SQLite TEXT, Postgres/DuckDB TIMESTAMP WITH TIME ZONE, MySQL DATETIME(3)).
func EnsureLeaseColumn(ctx context.Context, db *sql.DB, d Dialect, s Spec) error {
	switch d {
	case DialectPostgres, DialectDuckDB:
		return ensureIfNotExistsLeaseColumn(ctx, db, s)
	case DialectSQLite:
		return ensureSQLiteLeaseColumn(ctx, db, s)
	case DialectMySQL:
		return ensureMySQLLeaseColumn(ctx, db, s)
	default:
		return ErrUnsupported
	}
}

func ensureIfNotExistsLeaseColumn(ctx context.Context, db *sql.DB, s Spec) error {
	stmt := "ALTER TABLE " + s.Table + " ADD COLUMN IF NOT EXISTS " + //nolint:gosec // identifiers are store-author constants
		s.LeaseColumn + " TIMESTAMP WITH TIME ZONE"

	if _, err := db.ExecContext(ctx, stmt); err != nil {
		return fmt.Errorf("claiming: add lease column: %w", err)
	}

	return nil
}

func ensureSQLiteLeaseColumn(ctx context.Context, db *sql.DB, s Spec) error {
	const probe = "SELECT COUNT(*) FROM pragma_table_info(?) WHERE name = ?"

	alter := "ALTER TABLE " + s.Table + " ADD COLUMN " + s.LeaseColumn + " TEXT"

	return addLeaseColumnIfMissing(ctx, db, probe, []any{s.Table, s.LeaseColumn}, alter)
}

// ensureMySQLLeaseColumn adds the lease column when missing. MySQL servers
// have no ADD COLUMN IF NOT EXISTS, so the column is probed via
// information_schema first (works on both MySQL and MariaDB).
func ensureMySQLLeaseColumn(ctx context.Context, db *sql.DB, s Spec) error {
	const probe = `
SELECT COUNT(*) FROM information_schema.COLUMNS
WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ? AND COLUMN_NAME = ?`

	alter := "ALTER TABLE " + s.Table + " ADD COLUMN " + s.LeaseColumn + " DATETIME(3) NULL"

	return addLeaseColumnIfMissing(ctx, db, probe, []any{s.Table, s.LeaseColumn}, alter)
}

// addLeaseColumnIfMissing is the shared probe-then-ALTER shape behind the
// SQLite and MySQL migrations (neither has ADD COLUMN IF NOT EXISTS): the
// probe counts an existing column, and the ALTER runs only when it is
// missing.
func addLeaseColumnIfMissing(
	ctx context.Context,
	db *sql.DB,
	probe string,
	probeArgs []any,
	alter string,
) error {
	var count int

	err := db.QueryRowContext(ctx, probe, probeArgs...).Scan(&count)
	if err != nil {
		return fmt.Errorf("claiming: probe lease column: %w", err)
	}

	if count > 0 {
		return nil
	}

	if _, err := db.ExecContext(ctx, alter); err != nil {
		return fmt.Errorf("claiming: add lease column: %w", err)
	}

	return nil
}
