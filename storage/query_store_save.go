package storage

import (
	"context"
	"database/sql"
	"fmt"

	errorfamily "github.com/larsartmann/go-error-family"

	cqrsotel "github.com/larsartmann/go-cqrs-lite/otel/v3"
	"github.com/larsartmann/go-cqrs-lite/query/v3"
	sqlpkg "github.com/larsartmann/go-cqrs-lite/storage/v3/sql"
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
		err := s.insertQuery(ctx, tx, q)
		if err != nil {
			cqrsotel.RecordError(span, err)
			return errorfamily.WrapInfrastructure(err, "storage.insert_query",
				fmt.Sprintf("insert query %s", q.Type()))
		}
		return nil
	})
}

func (s *SQLQueryStore) insertQuery(
	ctx context.Context,
	tx *sql.Tx,
	q *query.PersistedQuery,
) error {
	ph := make([]string, 5)
	for i := range 5 {
		ph[i] = s.Dialect.Placeholder(i + 1)
	}

	insertSQL := fmt.Sprintf(
		`INSERT INTO `+sqlpkg.TableQueries+` (id, query_type, payload, metadata, received_at)
		VALUES (%s, %s, %s, %s, %s)`,
		ph[0], ph[1], ph[2], ph[3], ph[4],
	)

	metadata, err := sqlpkg.MarshalMetadata(q.Metadata())
	if err != nil {
		return errorfamily.WrapCorruption(err, "storage.marshal_query_metadata",
			"marshal metadata for query "+string(q.Type()))
	}

	_, err = tx.ExecContext(
		ctx,
		insertSQL,
		q.ID(),
		string(q.Type()),
		q.Payload(),
		metadata,
		s.Dialect.FormatTime(q.ReceivedAt()),
	)
	if err != nil {
		if sqlpkg.IsDuplicateKeyError(err) {
			return errorfamily.WrapConflict(
				query.ErrDuplicateQuery,
				"storage.duplicate_query",
				fmt.Sprintf("query with ID %s already exists", q.ID()),
			)
		}
		return errorfamily.WrapInfrastructure(err, "storage.insert_query",
			"insert query "+string(q.Type()))
	}

	return nil
}
