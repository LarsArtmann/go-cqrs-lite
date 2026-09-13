package claiming

import (
	"context"
	"database/sql"
	"fmt"
)

// EnsureLeaseColumn adds the lease column to tables created before
// claiming existed. Idempotent per dialect: Postgres uses ADD COLUMN IF
// NOT EXISTS; SQLite and MySQL have no such form, so the column is probed
// first (pragma_table_info / information_schema) and added only when
// missing. The column type matches each dialect's time representation
// (SQLite TEXT, Postgres TIMESTAMP WITH TIME ZONE, MySQL DATETIME(3)).
func EnsureLeaseColumn(ctx context.Context, db *sql.DB, d Dialect, s Spec) error {
	switch d {
	case DialectPostgres:
		return ensurePostgresLeaseColumn(ctx, db, s)
	case DialectSQLite:
		return ensureSQLiteLeaseColumn(ctx, db, s)
	case DialectMySQL:
		return ensureMySQLLeaseColumn(ctx, db, s)
	default:
		return ErrUnsupported
	}
}

func ensurePostgresLeaseColumn(ctx context.Context, db *sql.DB, s Spec) error {
	stmt := "ALTER TABLE " + s.Table + " ADD COLUMN IF NOT EXISTS " + //nolint:gosec // identifiers are store-author constants
		s.LeaseColumn + " TIMESTAMP WITH TIME ZONE"

	if _, err := db.ExecContext(ctx, stmt); err != nil {
		return fmt.Errorf("claiming: add lease column: %w", err)
	}

	return nil
}

func ensureSQLiteLeaseColumn(ctx context.Context, db *sql.DB, s Spec) error {
	var count int

	err := db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM pragma_table_info(?) WHERE name = ?",
		s.Table, s.LeaseColumn,
	).Scan(&count)
	if err != nil {
		return fmt.Errorf("claiming: probe lease column: %w", err)
	}

	if count > 0 {
		return nil
	}

	stmt := "ALTER TABLE " + s.Table + " ADD COLUMN " + s.LeaseColumn + " TEXT" //nolint:gosec // identifiers are store-author constants

	if _, err := db.ExecContext(ctx, stmt); err != nil {
		return fmt.Errorf("claiming: add lease column: %w", err)
	}

	return nil
}

// ensureMySQLLeaseColumn adds the lease column when missing. MySQL servers
// have no ADD COLUMN IF NOT EXISTS, so the column is probed via
// information_schema first (works on both MySQL and MariaDB).
func ensureMySQLLeaseColumn(ctx context.Context, db *sql.DB, s Spec) error {
	var count int

	err := db.QueryRowContext(ctx, `
SELECT COUNT(*) FROM information_schema.COLUMNS
WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ? AND COLUMN_NAME = ?`,
		s.Table, s.LeaseColumn,
	).Scan(&count)
	if err != nil {
		return fmt.Errorf("claiming: probe lease column: %w", err)
	}

	if count > 0 {
		return nil
	}

	stmt := "ALTER TABLE " + s.Table + " ADD COLUMN " + s.LeaseColumn + " DATETIME(3) NULL" //nolint:gosec // identifiers are store-author constants

	if _, err := db.ExecContext(ctx, stmt); err != nil {
		return fmt.Errorf("claiming: add lease column: %w", err)
	}

	return nil
}
