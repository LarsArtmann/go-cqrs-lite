package pebble

import (
	"context"
	"time"

	"github.com/cockroachdb/pebble"

	"github.com/larsartmann/go-cqrs-lite/event/v2"
	"github.com/larsartmann/go-cqrs-lite/id/v2"
	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel/v2"
	"github.com/larsartmann/go-cqrs-lite/query/v2"
)

// LoadQueries retrieves all queries where ReceivedAt > after.
// Implements query.QuerySource.
func (s *QueryStore) LoadQueries(
	ctx context.Context,
	after time.Time,
) ([]*query.PersistedQuery, error) {
	_, span := cqrsotel.StartSpan(ctx, tracer(), "pebble.query.load_queries",
		cqrsotel.SpanKindClient)
	defer span.End()

	queries, err := s.scanQueries(0, "", func(q *query.PersistedQuery) bool {
		return q.ReceivedAt().After(after)
	})
	if err != nil {
		cqrsotel.RecordError(span, err)

		return nil, err
	}

	span.SetAttributes(cqrsotel.AttrInt("query.count", len(queries)))

	return queries, nil
}

// ReadAllQueries returns all queries, ordered by request ID (time-ordered).
// Implements query.QueryJournal.
func (s *QueryStore) ReadAllQueries(ctx context.Context) ([]*query.PersistedQuery, error) {
	_, span := cqrsotel.StartSpan(ctx, tracer(), "pebble.query.read_all",
		cqrsotel.SpanKindClient)
	defer span.End()

	queries, err := s.scanQueries(0, "", nil)
	if err != nil {
		cqrsotel.RecordError(span, err)

		return nil, err
	}

	span.SetAttributes(cqrsotel.AttrInt("query.count", len(queries)))

	return queries, nil
}

// ReadQueriesFrom returns queries after the given RequestID, ordered by ID.
// Implements query.SeekableQueryJournal for position-based replay.
// Pass a zero RequestID to read from the beginning.
func (s *QueryStore) ReadQueriesFrom(
	ctx context.Context,
	afterRequestID id.RequestID,
	limit int,
) ([]*query.PersistedQuery, error) {
	_, span := cqrsotel.StartSpan(ctx, tracer(), "pebble.query.read_from",
		cqrsotel.SpanKindClient,
		cqrsotel.WithAttributes(cqrsotel.AttrInt("limit", limit)))
	defer span.End()

	skipID := ""
	if !afterRequestID.IsZero() {
		skipID = afterRequestID.String()
	}

	queries, err := s.scanQueries(limit, skipID, nil)
	if err != nil {
		cqrsotel.RecordError(span, err)

		return nil, err
	}

	span.SetAttributes(cqrsotel.AttrInt("query.count", len(queries)))

	return queries, nil
}

// scanQueries iterates over the query journal key space.
//
// Parameters:
//   - limit: 0 means no limit.
//   - skipUntilID: if non-empty, skip all entries until the one whose key ends
//     with this ID is found (inclusive — that entry is also skipped).
//   - filter: optional predicate; entries where filter returns false are skipped.
func (s *QueryStore) scanQueries(
	limit int,
	skipUntilID string,
	filter func(*query.PersistedQuery) bool,
) ([]*query.PersistedQuery, error) {
	iter, err := s.db.NewIter(
		&pebble.IterOptions{
			LowerBound: []byte(s.prefix),
			UpperBound: []byte(s.prefix + "\xff"),
		},
	)
	if err != nil {
		return nil, event.WrapInfrastructure(err, "pebble.query_iter",
			"create query iterator")
	}

	defer func() { _ = iter.Close() }()

	skipping := skipUntilID != ""

	var queries []*query.PersistedQuery

	for iter.First(); iter.Valid(); iter.Next() {
		if skipping {
			if journalKeyQueryID(iter.Key(), s.prefix) == skipUntilID {
				skipping = false
			}

			continue
		}

		q, err := s.deserializeQuery(iter.Value())
		if err != nil {
			return nil, event.WrapCorruption(err, "pebble.query_corrupt",
				"corrupt query at key "+string(iter.Key()))
		}

		if filter != nil && !filter(q) {
			continue
		}

		queries = append(queries, q)

		if limit > 0 && len(queries) >= limit {
			break
		}
	}

	err = checkIteratorError(iter)
	if err != nil {
		return nil, event.WrapInfrastructure(err, "pebble.query_iter_error",
			"query iterator error")
	}

	return queries, nil
}

// journalKeyQueryID extracts the request ID portion from a query journal key.
// Key format: {prefix}{requestID}.
func journalKeyQueryID(key []byte, prefix string) string {
	if len(key) > len(prefix) {
		return string(key[len(prefix):])
	}

	return string(key)
}
