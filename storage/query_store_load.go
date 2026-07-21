package storage

import (
	"context"
	"fmt"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/id/v4"
	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
	sqlpkg "github.com/larsartmann/go-cqrs-lite/storage/v4/sql"
)

// LoadQueries retrieves queries where ReceivedAt > after, ordered by received_at.
func (s *SQLQueryStore) LoadQueries(
	ctx context.Context,
	after time.Time,
) ([]*query.PersistedQuery, error) {
	if err := s.checkClosed(); err != nil {
		return nil, err
	}

	ctx, span := cqrsotel.StartSpan(
		ctx,
		sqlpkg.Tracer(),
		"query.store.load_queries",
		cqrsotel.SpanKindClient,
	)
	defer span.End()

	p1 := s.Dialect.Placeholder(1)
	sqlText := fmt.Sprintf(
		`SELECT %s FROM %s WHERE received_at > %s ORDER BY received_at ASC`,
		sqlpkg.QueryColumns, sqlpkg.TableQueries, p1,
	)

	rows, err := s.DB.QueryContext(ctx, sqlText, s.Dialect.FormatTime(after))
	if err != nil {
		cqrsotel.RecordError(span, err)
		return nil, errorfamily.WrapInfrastructure(err, "storage.query_queries",
			"query queries after timestamp")
	}
	defer sqlpkg.CloseRows(rows)

	queries, scanErr := s.scanQueries(rows)
	if scanErr != nil {
		cqrsotel.RecordError(span, scanErr)
	}
	span.SetAttributes(cqrsotel.AttrInt("query.count", len(queries)))

	return queries, scanErr
}

// queryJournalReader builds a fresh JournalReader over the queries table.
func (s *SQLQueryStore) queryJournalReader() *sqlpkg.JournalReader[*query.PersistedQuery] {
	return &sqlpkg.JournalReader[*query.PersistedQuery]{
		DB:          s.DB,
		Dialect:     s.Dialect,
		CheckClosed: s.checkClosed,

		SpanNameAll:  "query.store.read_all",
		SpanNameFrom: "query.store.read_from",
		CountAttr:    "query.count",

		ErrCodeAll:        "storage.query_all_queries",
		ErrCodeReadFrom:   "storage.query_read_from",
		ErrCodeFromStart:  "storage.read_queries_from_start",
		ErrCodeQueryStart: "storage.query_from_start",
		ErrCodeScan:       "storage.scan_from_position",

		EntityNoun:       "query",
		EntityNounPlural: "queries",

		Table:           sqlpkg.TableQueries,
		AllColumns:      sqlpkg.QueryColumns,
		PositionColumns: "e.id, e.query_type, e.payload, e.metadata, e.received_at",
		TimestampColumn: "received_at",

		Scan: s.scanQueries,
	}
}

// ReadAllQueries returns all queries across the system, ordered by received_at.
// Implements query.QueryJournal.
func (s *SQLQueryStore) ReadAllQueries(ctx context.Context) ([]*query.PersistedQuery, error) {
	return s.queryJournalReader().ReadAll(ctx)
}

// ReadQueriesFrom returns queries after the given RequestID, ordered by received_at.
// Implements query.SeekableQueryJournal for position-based query replay.
func (s *SQLQueryStore) ReadQueriesFrom(
	ctx context.Context,
	afterRequestID id.RequestID,
	limit int,
) ([]*query.PersistedQuery, error) {
	afterID := ""
	if !afterRequestID.IsZero() {
		afterID = afterRequestID.String()
	}

	return s.queryJournalReader().ReadFrom(ctx, afterID, limit)
}
