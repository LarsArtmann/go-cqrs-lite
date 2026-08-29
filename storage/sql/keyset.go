package sql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	errorfamily "github.com/larsartmann/go-error-family"

	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel/v4"
)

// ResolveCursorTimestamp loads the timestamp-column value of the journal row
// identified by id via a primary-key point lookup.
//
// The value is returned in its driver-native representation (string for SQLite
// TEXT columns, time.Time for Postgres TIMESTAMP columns) so callers can bind
// it back into KeysetPositionQuery verbatim — no reformatting round-trip that
// could break ordering semantics. found is false when no row with that ID
// exists (dangling cursor, e.g. after journal pruning).
func ResolveCursorTimestamp(
	ctx context.Context,
	db *sql.DB,
	dialect Dialect,
	table, timestampColumn, id string,
) (any, bool, error) {
	ctx, span := cqrsotel.StartSpan(
		ctx,
		Tracer(),
		"sql.resolve_cursor_timestamp",
		cqrsotel.SpanKindClient,
	)
	defer span.End()

	if err := ValidateJournalIdentifiers(table, timestampColumn); err != nil {
		return nil, false, err
	}

	query := fmt.Sprintf(
		`SELECT %s FROM %s WHERE id = %s`,
		timestampColumn, table, dialect.Placeholder(1),
	)

	var ts any

	err := db.QueryRowContext(ctx, query, id).Scan(&ts)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, false, nil
	}

	if err != nil {
		cqrsotel.RecordError(span, err)

		return nil, false, errorfamily.WrapInfrastructure(err, "storage.resolve_cursor_timestamp",
			fmt.Sprintf("resolve cursor timestamp in %s (id=%s)", table, id))
	}

	return ts, true, nil
}

// KeysetPositionQuery builds an index-usable keyset-pagination query that
// returns journal rows strictly after the (timestampColumn, id) cursor,
// ordered by timestamp then id. The caller binds [timestamp, timestamp, id]
// (plus limit via AppendLimit) with placeholder indices 1-3 (4 for the limit).
//
// The redundant timestamp range prefix (`e.<ts> >= ?`) lets SQLite and MySQL
// drive the scan from the timestamp index; the tie-break disjunction only
// filters inside the scanned range and excludes the cursor row itself.
//
// The previous formulation — a self-JOIN on the cursor row with
// `e.ts > c.ts OR (e.ts = c.ts AND e.id > c.id)` — defeated the index in
// SQLite (MULTI-INDEX OR plus a temp B-tree sort of the remaining tail per
// batch), making batched journal drains O(N²): draining a 200k-event journal
// in batches of 100 took ~63s with the self-JOIN vs 0.22s with this query.
func KeysetPositionQuery(dialect Dialect, columns, table, timestampColumn string) string {
	if err := ValidateJournalIdentifiers(table, timestampColumn); err != nil {
		return ""
	}

	p1, p2, p3 := dialect.Placeholder(1), dialect.Placeholder(2), dialect.Placeholder(3)

	return fmt.Sprintf(
		`SELECT %s
		FROM %s e
		WHERE e.%s >= %s AND (e.%s > %s OR e.id > %s)
		ORDER BY e.%s ASC, e.id ASC`,
		columns, table,
		timestampColumn, p1,
		timestampColumn, p2, p3,
		timestampColumn,
	)
}
