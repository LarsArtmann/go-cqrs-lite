package eventstore

import (
	"context"
	"database/sql"
	"fmt"
	"io"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel/v4"
	sqlpkg "github.com/larsartmann/go-cqrs-lite/storage/v4/sql"
)

// sqlEventIterator streams events one row at a time from *sql.Rows,
// avoiding materializing the full result set into memory.
// NOT goroutine-safe — each iterator is single-threaded (per event.EventIterator contract).
type sqlEventIterator struct {
	rows    *sql.Rows
	scanOne func(*sql.Rows) (event.Event, error)
	closed  bool
}

func newSQLEventIterator(
	rows *sql.Rows,
	scanOne func(*sql.Rows) (event.Event, error),
) *sqlEventIterator {
	return &sqlEventIterator{rows: rows, scanOne: scanOne}
}

// Next returns the next event, or io.EOF when exhausted or closed.
func (it *sqlEventIterator) Next() (event.Event, error) {
	if it.closed {
		return nil, io.EOF
	}

	if !it.rows.Next() {
		if err := it.rows.Err(); err != nil {
			return nil, errorfamily.WrapInfrastructure(err, "storage.stream_iterate",
				"iterate event stream")
		}

		return nil, io.EOF
	}

	return it.scanOne(it.rows)
}

// Close releases the underlying *sql.Rows. Safe to call multiple times.
func (it *sqlEventIterator) Close() error {
	if it.closed {
		return nil
	}

	it.closed = true
	if it.rows != nil {
		return it.rows.Close()
	}

	return nil
}

// LoadStream is the streaming equivalent of Load.
// It returns an EventIterator that yields events one row at a time without
// materializing the full slice. The caller must call Close on the iterator.
// The context must remain valid for the duration of iteration.
func (s *SQLEventStore) LoadStream(
	ctx context.Context,
	ref id.AggregateRef,
) (event.EventIterator, error) {
	return s.streamByAggregate(ctx, ref, "ORDER BY version ASC", nil, "event.store.load_stream")
}

// LoadStreamFromVersion is the streaming equivalent of LoadFromVersion.
func (s *SQLEventStore) LoadStreamFromVersion(
	ctx context.Context,
	ref id.AggregateRef,
	version event.Version,
) (event.EventIterator, error) {
	where := fmt.Sprintf("AND version > %s ORDER BY version ASC", s.Dialect.Placeholder(3))

	return s.streamByAggregate(
		ctx, ref, where, []any{version.Int()}, "event.store.load_stream_from_version",
	)
}

func (s *SQLEventStore) streamByAggregate(
	ctx context.Context,
	ref id.AggregateRef,
	where string,
	extraArgs []any,
	_ string,
) (event.EventIterator, error) {
	if err := s.checkClosed(); err != nil {
		return nil, errorfamily.WrapInfrastructure(err, "storage.stream_by_aggregate",
			"stream events for aggregate")
	}

	p1, p2 := s.Dialect.Placeholder(1), s.Dialect.Placeholder(2)
	query := fmt.Sprintf(
		"SELECT %s FROM %s WHERE aggregate_type = %s AND aggregate_id = %s %s",
		sqlpkg.EventColumns, sqlpkg.TableEvents, p1, p2, where,
	)

	args := make([]any, 0, 2+len(extraArgs))
	args = append(args, string(ref.Type), ref.ID)
	args = append(args, extraArgs...)

	rows, err := s.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errorfamily.WrapInfrastructure(err, "storage.stream_query",
			fmt.Sprintf("stream events for %s %s", ref.Type, ref.ID))
	}

	return newSQLEventIterator(rows, s.scanEvent), nil
}

// ReadStream is the streaming equivalent of ReadAll.
// It yields every event in the store ordered by occurred_at, one at a time.
func (s *SQLEventStore) ReadStream(ctx context.Context) (event.EventIterator, error) {
	if err := s.checkClosed(); err != nil {
		return nil, errorfamily.WrapInfrastructure(err, "storage.stream_read_all",
			"stream all events")
	}

	_, span := cqrsotel.StartSpan(ctx, sqlpkg.Tracer(), "event.store.read_stream",
		cqrsotel.SpanKindClient)
	defer span.End()

	rows, err := s.DB.QueryContext(ctx, sqlpkg.AllEventsQuery)
	if err != nil {
		cqrsotel.RecordError(span, err)

		return nil, errorfamily.WrapInfrastructure(
			err,
			"storage.stream_read_all",
			"stream all events",
		)
	}

	return newSQLEventIterator(rows, s.scanEvent), nil
}

// ReadStreamFrom is the streaming equivalent of ReadFrom.
func (s *SQLEventStore) ReadStreamFrom(
	ctx context.Context,
	afterEventID id.EventID,
	limit int,
) (event.EventIterator, error) {
	if err := s.checkClosed(); err != nil {
		return nil, errorfamily.Wrapf(err, errorfamily.Infrastructure, "storage.stream_read_from",
			"stream from store (limit=%d, after=%s)", limit, afterEventID)
	}

	if afterEventID.IsZero() {
		return s.streamFromStart(ctx, limit)
	}

	p1, p2, p3 := s.Dialect.Placeholder(1), s.Dialect.Placeholder(2), s.Dialect.Placeholder(3)
	query := fmt.Sprintf(
		`SELECT e.id, e.event_type, e.aggregate_type, e.aggregate_id, e.version, e.schema_version, e.payload, e.payload_encoding, e.metadata, e.occurred_at
		FROM `+sqlpkg.TableEvents+` e
		JOIN `+sqlpkg.TableEvents+` c ON c.id = %s
		WHERE (e.occurred_at > c.occurred_at) OR (e.occurred_at = c.occurred_at AND e.id > %s)
		ORDER BY e.occurred_at ASC, e.id ASC`,
		p1,
		p2,
	)
	args := []any{afterEventID.String(), afterEventID.String()}
	if limit > 0 {
		query += " LIMIT " + p3
		args = append(args, limit)
	}

	rows, err := s.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errorfamily.WrapInfrastructure(err, "storage.stream_query_from",
			fmt.Sprintf("stream events from position (limit=%d)", limit))
	}

	return newSQLEventIterator(rows, s.scanEvent), nil
}

func (s *SQLEventStore) streamFromStart(
	ctx context.Context,
	limit int,
) (event.EventIterator, error) {
	var (
		query string
		args  []any
	)
	if limit <= 0 {
		query = `SELECT ` + sqlpkg.EventColumns + `
			FROM ` + sqlpkg.TableEvents + ` ORDER BY occurred_at ASC`
	} else {
		p1 := s.Dialect.Placeholder(1)
		query = `SELECT ` + sqlpkg.EventColumns + `
			FROM ` + sqlpkg.TableEvents + ` ORDER BY occurred_at ASC LIMIT ` + p1
		args = []any{limit}
	}

	rows, err := s.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errorfamily.WrapInfrastructure(err, "storage.stream_from_start",
			fmt.Sprintf("stream events from start (limit=%d)", limit))
	}

	return newSQLEventIterator(rows, s.scanEvent), nil
}

var (
	_ event.StreamingSource  = (*SQLEventStore)(nil)
	_ event.StreamingJournal = (*SQLEventStore)(nil)
)
