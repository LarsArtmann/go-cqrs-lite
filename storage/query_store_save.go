package storage

import (
	"context"
	"database/sql"
	"fmt"

	errorfamily "github.com/larsartmann/go-error-family"

	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
	sqlpkg "github.com/larsartmann/go-cqrs-lite/storage/v4/sql"
)

// SaveQuery persists a single query for audit purposes.
// Returns ErrDuplicateQuery if the query ID already exists (PRIMARY KEY violation).
func (s *SQLQueryStore) SaveQuery(
	ctx context.Context,
	q *query.PersistedQuery,
) error {
	if err := s.checkClosed(); err != nil {
		return err
	}

	ctx, span := cqrsotel.StartSpan(
		ctx,
		sqlpkg.Tracer(),
		"query.store.save",
		cqrsotel.SpanKindClient,
		cqrsotel.WithAttributes(cqrsotel.AttrString("query.type", string(q.Type()))),
	)
	defer span.End()

	return sqlpkg.RunInTx(ctx, s.DB, span, func(tx *sql.Tx) error {
		if err := s.queryInserter().Insert(ctx, tx, q); err != nil {
			cqrsotel.RecordError(span, err)

			return err
		}

		return nil
	})
}

// queryInserter builds the shared write path for the queries table.
func (s *SQLQueryStore) queryInserter() *sqlpkg.Inserter[*query.PersistedQuery] {
	return &sqlpkg.Inserter[*query.PersistedQuery]{
		Dialect:        s.Dialect,
		Table:          sqlpkg.TableQueries,
		Columns:        []string{"id", "query_type", "payload", colMetadata, colReceivedAt},
		EntityNoun:     "query",
		MarshalErrCode: "storage.marshal_query_metadata",
		InsertErrCode:  "storage.insert_query",
		Describe: func(q *query.PersistedQuery) string {
			return string(q.Type())
		},
		RowArgs: func(q *query.PersistedQuery) ([]any, error) {
			metadata, err := sqlpkg.MarshalMetadata(q.Metadata())
			if err != nil {
				return nil, err
			}

			return []any{
				q.ID(),
				string(q.Type()),
				q.Payload(),
				metadata,
				s.Dialect.FormatTime(q.ReceivedAt()),
			}, nil
		},
		Duplicate: func(_ error, q *query.PersistedQuery) error {
			return errorfamily.WrapConflict(
				query.ErrDuplicateQuery,
				"storage.duplicate_query",
				fmt.Sprintf("query with ID %s already exists", q.ID()),
			)
		},
	}
}
