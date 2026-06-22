package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/id/v3"
	"github.com/larsartmann/go-cqrs-lite/query/v3"
	sqlpkg "github.com/larsartmann/go-cqrs-lite/storage/v3/sql"
)

func (s *SQLQueryStore) scanQueries(rows *sql.Rows) ([]*query.PersistedQuery, error) {
	return sqlpkg.ScanSlice(rows, s.scanQuery)
}

func (s *SQLQueryStore) scanQuery(rows *sql.Rows) (*query.PersistedQuery, error) {
	var (
		requestIDStr string
		queryType    string
		payload      []byte
		metadataJSON []byte
	)
	timeDest := s.Dialect.ScanTimeDest()
	err := rows.Scan(
		&requestIDStr,
		&queryType,
		&payload,
		&metadataJSON,
		timeDest,
	)
	if err != nil {
		return nil, query.WrapInfrastructure(err, "storage.scan_query",
			fmt.Sprintf("scan query row for %s (id %s)", queryType, requestIDStr))
	}

	receivedAt, err := s.Dialect.ParseTime(timeDest)
	if err != nil {
		return nil, query.WrapCorruption(err, "storage.parse_received_at",
			fmt.Sprintf("parse received_at for %s query %s (id %s)",
				queryType, requestIDStr, requestIDStr))
	}

	parsedRequestID, err := id.ParseRequestID(requestIDStr)
	if err != nil {
		return nil, query.WrapCorruption(
			err,
			"storage.parse_request_id",
			fmt.Sprintf("parse request ID %q for %s query", requestIDStr, queryType),
		)
	}

	opts := []query.QueryPersistOption{
		query.WithQueryID(parsedRequestID),
		query.WithQueryReceivedAt(receivedAt),
	}

	if len(metadataJSON) > 0 {
		var meta query.Metadata
		if jsonErr := json.Unmarshal(metadataJSON, &meta); jsonErr != nil {
			return nil, query.WrapCorruption(jsonErr, "storage.parse_query_metadata",
				fmt.Sprintf("unmarshal metadata for %s query (id %s)", queryType, requestIDStr))
		}
		opts = append(opts, query.WithQueryMetadata(meta))
	}

	q, err := query.NewPersistedQuery(
		query.Type(queryType),
		payload,
		opts...,
	)
	if err != nil {
		return nil, query.WrapCorruption(err, "storage.reconstruct_query",
			"reconstruct query "+queryType)
	}

	return q, nil
}
