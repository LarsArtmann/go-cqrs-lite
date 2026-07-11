package projectionhost

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"
)

// Count returns the total number of dead-letter entries across all projections.
func (s *SQLiteDeadLetterStore) Count(ctx context.Context) (int64, error) {
	var count int64

	err := s.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM projection_dead_letters").Scan(&count)
	if err != nil {
		return 0, errorfamily.WrapInfrastructure(
			err, "projectionhost.dlq_count",
			"count dead-letter entries",
		)
	}

	return count, nil
}

// ListPaged returns dead-letter entries with pagination.
// An empty projectionName returns entries across all projections.
// offset is zero-based; limit is the maximum number of entries to return.
func (s *SQLiteDeadLetterStore) ListPaged(
	ctx context.Context,
	projectionName string,
	offset, limit int,
) ([]DeadLetterEntry, error) {
	if limit <= 0 {
		limit = 50
	}

	query := `SELECT projection_name, event_id, event_type, aggregate_type, aggregate_id,
        version, schema_version, payload, payload_encoding, metadata, occurred_at,
        error_text, error_code, error_family, failed_at
        FROM projection_dead_letters`

	var (
		rows *sql.Rows

		err error
	)

	if projectionName == "" {
		query += " ORDER BY failed_at LIMIT ? OFFSET ?"
		rows, err = s.db.QueryContext(ctx, query, limit, offset)
	} else {
		query += " WHERE projection_name = ? ORDER BY failed_at LIMIT ? OFFSET ?"
		rows, err = s.db.QueryContext(ctx, query, projectionName, limit, offset)
	}

	if err != nil {
		return nil, errorfamily.WrapInfrastructure(
			err, "projectionhost.dlq_list_paged",
			"query paginated dead-letter entries",
		)
	}

	defer func() { _ = rows.Close() }()

	var result []DeadLetterEntry

	for rows.Next() {
		entry, err := scanDLQRow(rows)
		if err != nil {
			return nil, err
		}

		result = append(result, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate dead-letter rows: %w", err)
	}

	return result, nil
}

// PurgeBefore removes all dead-letter entries that failed before the given timestamp.
// Returns the number of entries removed.
func (s *SQLiteDeadLetterStore) PurgeBefore(
	ctx context.Context,
	before time.Time,
) (int64, error) {
	res, err := s.db.ExecContext(ctx,
		"DELETE FROM projection_dead_letters WHERE failed_at < ?",
		before.Format(time.RFC3339Nano))
	if err != nil {
		return 0, errorfamily.WrapInfrastructure(
			err, "projectionhost.dlq_purge_before",
			"purge dead-letter entries before "+before.Format(time.RFC3339Nano),
		)
	}

	count, _ := res.RowsAffected()

	return count, nil
}
