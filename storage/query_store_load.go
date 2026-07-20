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

// ReadAllQueries returns all queries across the system, ordered by received_at.
// Implements query.QueryJournal.
func (s *SQLQueryStore) ReadAllQueries(ctx context.Context) ([]*query.PersistedQuery, error) {
	if err := s.checkClosed(); err != nil {
		return nil, err
	}

	ctx, span := cqrsotel.StartSpan(
		ctx,
		sqlpkg.Tracer(),
		"query.store.read_all",
		cqrsotel.SpanKindClient,
	)
	defer span.End()

	sqlText := `SELECT ` + sqlpkg.QueryColumns + `
		FROM ` + sqlpkg.TableQueries + ` ORDER BY received_at ASC`

	rows, err := s.DB.QueryContext(ctx, sqlText)
	if err != nil {
		cqrsotel.RecordError(span, err)
		return nil, errorfamily.WrapInfrastructure(
			err,
			"storage.query_all_queries",
			"query all queries",
		)
	}
	defer sqlpkg.CloseRows(rows)

	queries, scanErr := s.scanQueries(rows)
	if scanErr != nil {
		cqrsotel.RecordError(span, scanErr)
	}
	span.SetAttributes(cqrsotel.AttrInt("query.count", len(queries)))

	return queries, scanErr
}

// ReadQueriesFrom returns queries after the given RequestID, ordered by received_at.
// Implements query.SeekableQueryJournal for position-based query replay.
func (s *SQLQueryStore) ReadQueriesFrom(
	ctx context.Context,
	afterRequestID id.RequestID,
	limit int,
) ([]*query.PersistedQuery, error) {
	if err := s.checkClosed(); err != nil {
		return nil, errorfamily.Wrapf(err, errorfamily.Infrastructure, "storage.query_read_from",
			"read from query store (limit=%d, after=%s)", limit, afterRequestID)
	}

	ctx, span := cqrsotel.StartSpan(
		ctx,
		sqlpkg.Tracer(),
		"query.store.read_from",
		cqrsotel.SpanKindClient,
		cqrsotel.WithAttributes(cqrsotel.AttrInt("cqrs.journal.limit", limit)),
	)
	defer span.End()

	if afterRequestID.IsZero() {
		queries, err := s.loadQueriesFromStart(ctx, limit)
		if err != nil {
			cqrsotel.RecordError(span, err)
			return queries, errorfamily.WrapInfrastructure(err, "storage.read_queries_from_start",
				fmt.Sprintf("read queries from start (limit=%d)", limit))
		}
		span.SetAttributes(cqrsotel.AttrInt("query.count", len(queries)))
		return queries, nil
	}

	p1 := s.Dialect.Placeholder(1)
	p2 := s.Dialect.Placeholder(2)
	p3 := s.Dialect.Placeholder(3)
	sqlText := fmt.Sprintf(
		`SELECT e.id, e.query_type, e.payload, e.metadata, e.received_at
		FROM `+sqlpkg.TableQueries+` e
		JOIN `+sqlpkg.TableQueries+` c ON c.id = %s
		WHERE (e.received_at > c.received_at) OR (e.received_at = c.received_at AND e.id > %s)
		ORDER BY e.received_at ASC, e.id ASC`,
		p1, p2,
	)
	args := sqlpkg.CursorArgs(afterRequestID.String())
	if limit > 0 {
		sqlText += " LIMIT " + p3
		args = append(args, limit)
	}

	rows, err := s.DB.QueryContext(ctx, sqlText, args...)
	if err != nil {
		cqrsotel.RecordError(span, err)
		return nil, errorfamily.WrapInfrastructure(err, "storage.query_from_position",
			fmt.Sprintf("query queries from position (limit=%d)", limit))
	}
	defer sqlpkg.CloseRows(rows)

	queries, scanErr := s.scanQueries(rows)
	if scanErr != nil {
		cqrsotel.RecordError(span, scanErr)
		return queries, errorfamily.WrapInfrastructure(scanErr, "storage.scan_from_position",
			fmt.Sprintf("scan queries from position (limit=%d)", limit))
	}
	span.SetAttributes(cqrsotel.AttrInt("query.count", len(queries)))

	return queries, nil
}

func (s *SQLQueryStore) loadQueriesFromStart(
	ctx context.Context,
	limit int,
) ([]*query.PersistedQuery, error) {
	if limit <= 0 {
		return s.ReadAllQueries(ctx)
	}

	p1 := s.Dialect.Placeholder(1)
	sqlText := `SELECT ` + sqlpkg.QueryColumns + `
		FROM ` + sqlpkg.TableQueries + ` ORDER BY received_at ASC LIMIT ` + p1

	rows, err := s.DB.QueryContext(ctx, sqlText, limit)
	if err != nil {
		return nil, errorfamily.WrapInfrastructure(err, "storage.query_from_start",
			fmt.Sprintf("query queries from start (limit=%d)", limit))
	}
	defer sqlpkg.CloseRows(rows)

	return s.scanQueries(rows)
}
