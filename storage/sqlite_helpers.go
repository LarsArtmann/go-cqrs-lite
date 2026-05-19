package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

func parseSQLiteTimestamp(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, nil
	}

	for _, layout := range []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05Z07:00",
		"2006-01-02 15:04:05",
	} {
		t, err := time.Parse(layout, s)
		if err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("%w: %q", ErrUnsupportedTimestamp, s)
}

// SQLiteInitSchema creates all required tables in the SQLite database.
func SQLiteInitSchema(ctx context.Context, db *sql.DB) error {
	for _, ddl := range []string{SQLiteSchema(), SQLiteSnapshotSchema(), SQLiteCheckpointSchema(), SQLiteOutboxSchema()} {
		_, err := db.ExecContext(ctx, ddl)
		if err != nil {
			return fmt.Errorf("exec DDL: %w\nDDL: %s", err, ddl)
		}
	}

	return nil
}
