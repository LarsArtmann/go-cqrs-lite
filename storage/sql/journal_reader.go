package sql

import (
	"context"
	"database/sql"
	"fmt"

	errorfamily "github.com/larsartmann/go-error-family"

	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel/v4"
)

// JournalReader encapsulates the entity-specific bits needed to read a SQL
// journal (ReadAll + ReadFrom) for one CQRS message type (events, commands,
// queries). Each instance replaces ~80 lines of duplicated boilerplate that
// previously lived in eventstore, command_store, and query_store.
//
// Construct one per call site via a small helper on the store (e.g.
// s.journalReader()) and call ReadAll / ReadFrom. The receiver types still
// own their method-level wrappers so they satisfy command.CommandJournal,
// query.QueryJournal, event.SeekableJournal etc.
type JournalReader[T any] struct {
	DB          *sql.DB
	Dialect     Dialect
	CheckClosed func() error

	// SpanNameAll is the OTel span name for ReadAll (e.g. "command.store.read_all").
	SpanNameAll string
	// SpanNameFrom is the OTel span name for ReadFrom (e.g. "command.store.read_from").
	SpanNameFrom string
	// CountAttr is the OTel attribute name stamped with the returned count
	// (e.g. "command.count" or cqrsotel.AttrEventCount).
	CountAttr string

	// ErrCodeAll is the errorfamily code for ReadAll query failures.
	ErrCodeAll string
	// ErrCodeReadFrom is the errorfamily code for ReadFrom checkClosed failures.
	ErrCodeReadFrom string
	// ErrCodeFromStart is the errorfamily code for ReadFrom's "load from start" failures.
	ErrCodeFromStart string
	// ErrCodeQueryStart is the errorfamily code for loadFromStart's raw query failure.
	ErrCodeQueryStart string
	// ErrCodeScan is the errorfamily code for ReadFrom's scan failure.
	ErrCodeScan string

	// EntityNoun is the singular noun used in error messages ("command", "event", "query").
	EntityNoun string
	// EntityNounPlural is the plural form used in error messages.
	EntityNounPlural string

	// Table is the SQL table name (e.g. TableCommands).
	Table string
	// AllColumns is the column list for SELECT (e.g. CommandColumns).
	AllColumns string
	// PositionColumns overrides the column list used in the position query.
	// Leave empty to reuse AllColumns.
	PositionColumns string
	// TimestampColumn is the column used for cursor tie-breaking
	// ("received_at" for commands/queries, "occurred_at" for events).
	TimestampColumn string

	// Scan converts SQL rows into a slice of T. Caller closes the rows.
	Scan func(*sql.Rows) ([]T, error)
}

// ReadAll returns every row in the journal ordered by the entity's timestamp column.
// Used by event.Journal, command.CommandJournal, and query.QueryJournal.
func (r *JournalReader[T]) ReadAll(ctx context.Context) ([]T, error) {
	if err := r.CheckClosed(); err != nil {
		return nil, err
	}

	ctx, span := cqrsotel.StartSpan(ctx, Tracer(), r.SpanNameAll, cqrsotel.SpanKindClient)
	defer span.End()

	query := `SELECT ` + r.AllColumns + `
		FROM ` + r.Table + ` ORDER BY ` + r.TimestampColumn + ` ASC`

	rows, err := r.DB.QueryContext(ctx, query)
	if err != nil {
		cqrsotel.RecordError(span, err)

		return nil, errorfamily.WrapInfrastructure(
			err,
			r.ErrCodeAll,
			"query all "+r.EntityNounPlural,
		)
	}
	defer CloseRows(rows)

	items, scanErr := r.Scan(rows)
	if scanErr != nil {
		cqrsotel.RecordError(span, scanErr)

		return nil, errorfamily.WrapInfrastructure(
			scanErr,
			r.ErrCodeScan,
			"scan all "+r.EntityNounPlural,
		)
	}

	if err := rows.Err(); err != nil {
		cqrsotel.RecordError(span, err)

		return nil, errorfamily.WrapInfrastructure(
			err,
			r.ErrCodeScan,
			"iterate all "+r.EntityNounPlural,
		)
	}

	span.SetAttributes(cqrsotel.AttrInt(r.CountAttr, len(items)))

	return items, nil
}

// ReadFrom returns rows after the given cursor ID, ordered by timestamp then ID.
// If afterID is empty, falls back to LoadFromStart. Used by event.SeekableJournal,
// command.SeekableCommandJournal, and query.SeekableQueryJournal.
func (r *JournalReader[T]) ReadFrom(ctx context.Context, afterID string, limit int) ([]T, error) {
	if err := r.CheckClosed(); err != nil {
		return nil, errorfamily.Wrapf(err, errorfamily.Infrastructure, r.ErrCodeReadFrom,
			"read from %s store (limit=%d, after=%s)", r.EntityNoun, limit, afterID)
	}

	ctx, span := cqrsotel.StartSpan(
		ctx,
		Tracer(),
		r.SpanNameFrom,
		cqrsotel.SpanKindClient,
		cqrsotel.WithAttributes(cqrsotel.AttrInt("cqrs.journal.limit", limit)),
	)
	defer span.End()

	if afterID == "" {
		items, err := r.LoadFromStart(ctx, limit)
		if err != nil {
			cqrsotel.RecordError(span, err)

			return items, errorfamily.WrapInfrastructure(err, r.ErrCodeFromStart,
				fmt.Sprintf("read %s from start (limit=%d)", r.EntityNounPlural, limit))
		}

		span.SetAttributes(cqrsotel.AttrInt(r.CountAttr, len(items)))

		return items, nil
	}

	positionColumns := r.PositionColumns

	p1 := r.Dialect.Placeholder(1)
	p2 := r.Dialect.Placeholder(2)
	p3 := r.Dialect.Placeholder(3)
	ts := r.TimestampColumn
	query := fmt.Sprintf(
		`SELECT %s
		FROM %s e
		JOIN %s c ON c.id = %s
		WHERE (e.%s > c.%s) OR (e.%s = c.%s AND e.id > %s)
		ORDER BY e.%s ASC, e.id ASC`,
		positionColumns,
		r.Table, r.Table, p1,
		ts, ts, ts, ts, p2,
		ts,
	)

	rows, err := QueryFromPosition(
		ctx,
		r.DB,
		span,
		query,
		afterID,
		limit,
		p3,
		r.EntityNounPlural,
	)
	if err != nil {
		return nil, err
	}
	defer CloseRows(rows)

	items, scanErr := r.Scan(rows)
	if scanErr != nil {
		cqrsotel.RecordError(span, scanErr)

		return nil, errorfamily.WrapInfrastructure(scanErr, r.ErrCodeScan,
			fmt.Sprintf("scan %s from position (limit=%d)", r.EntityNounPlural, limit))
	}

	if err := rows.Err(); err != nil {
		cqrsotel.RecordError(span, err)

		return nil, errorfamily.WrapInfrastructure(err, r.ErrCodeScan,
			fmt.Sprintf("iterate %s from position (limit=%d)", r.EntityNounPlural, limit))
	}

	span.SetAttributes(cqrsotel.AttrInt(r.CountAttr, len(items)))

	return items, nil

	return items, nil
}

// LoadFromStart returns the first N rows ordered by timestamp. If limit <= 0,
// returns every row via ReadAll. Exposed so callers can use it directly
// (loadAllFromStart / loadCommandsFromStart / loadQueriesFromStart historically).
func (r *JournalReader[T]) LoadFromStart(ctx context.Context, limit int) ([]T, error) {
	if limit <= 0 {
		return r.ReadAll(ctx)
	}

	p1 := r.Dialect.Placeholder(1)
	query := `SELECT ` + r.AllColumns + `
		FROM ` + r.Table + ` ORDER BY ` + r.TimestampColumn + ` ASC LIMIT ` + p1

	rows, err := r.DB.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, errorfamily.WrapInfrastructure(err, r.ErrCodeQueryStart,
			fmt.Sprintf("query %s from start (limit=%d)", r.EntityNounPlural, limit))
	}
	defer CloseRows(rows)

	items, scanErr := r.Scan(rows)
	if scanErr != nil {
		return nil, errorfamily.WrapInfrastructure(scanErr, r.ErrCodeScan,
			fmt.Sprintf("scan %s from start (limit=%d)", r.EntityNounPlural, limit))
	}

	if err := rows.Err(); err != nil {
		return nil, errorfamily.WrapInfrastructure(err, r.ErrCodeScan,
			fmt.Sprintf("iterate %s from start (limit=%d)", r.EntityNounPlural, limit))
	}

	return items, nil
}
