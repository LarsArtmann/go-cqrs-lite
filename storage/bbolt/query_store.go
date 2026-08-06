package bbolt

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"time"

	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
	bolt "go.etcd.io/bbolt"
)

// QueryStore persists queries in a bbolt database. It implements
// query.QueryStore, query.QueryJournal, and query.SeekableQueryJournal.
type QueryStore struct {
	storeBase
}

// NewQueryStore creates a QueryStore sharing the given *bbolt.DB.
func NewQueryStore(db *bolt.DB, logger *slog.Logger) (*QueryStore, error) {
	return &QueryStore{storeBase{db: db, logger: logger}}, nil
}

func queryKey(requestID id.RequestID) []byte {
	return []byte(requestID.String())
}

// SaveQuery persists a single query. Returns query.ErrDuplicateQuery if a
// query with the same request ID already exists.
func (s *QueryStore) SaveQuery(_ context.Context, q *query.PersistedQuery) error {
	key := queryKey(q.ID())

	return wrapBucketErr(s.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketQueries))
		if bucket.Get(key) != nil {
			return query.ErrDuplicateQuery
		}

		data, err := marshalQuery(q)
		if err != nil {
			return err
		}

		return bucket.Put(key, data)
	}), "bbolt.query_save", "save query")
}

// LoadQueries returns all queries received after the given time.
func (s *QueryStore) LoadQueries(
	_ context.Context,
	after time.Time,
) ([]*query.PersistedQuery, error) {
	var queries []*query.PersistedQuery

	err := s.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketQueries))
		c := bucket.Cursor()

		for k, v := c.First(); k != nil; k, v = c.Next() {
			q, err := unmarshalQuery(v)
			if err != nil {
				return err
			}

			if q.ReceivedAt().After(after) {
				queries = append(queries, q)
			}
		}

		return nil
	})

	return queries, wrapBucketErr(err, "bbolt.query_load", "load queries after timestamp")
}

// ReadAllQueries returns all queries ordered by request ID.
func (s *QueryStore) ReadAllQueries(_ context.Context) ([]*query.PersistedQuery, error) {
	var queries []*query.PersistedQuery

	err := s.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketQueries))

		return bucket.ForEach(func(_ []byte, v []byte) error {
			q, err := unmarshalQuery(v)
			if err != nil {
				return err
			}

			queries = append(queries, q)

			return nil
		})
	})

	return queries, wrapBucketErr(err, "bbolt.query_read_all", "read all queries")
}

// ReadQueriesFrom returns queries starting after the given request ID,
// up to limit entries. A limit of 0 means no limit.
func (s *QueryStore) ReadQueriesFrom(
	_ context.Context,
	afterReqID id.RequestID,
	limit int,
) ([]*query.PersistedQuery, error) {
	seekKey := queryKey(afterReqID)
	var queries []*query.PersistedQuery

	err := s.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketQueries))
		c := bucket.Cursor()

		k, v := c.Seek(seekKey)
		if k != nil && bytes.Equal(k, seekKey) {
			k, v = c.Next()
		}

		for ; k != nil; k, v = c.Next() {
			q, err := unmarshalQuery(v)
			if err != nil {
				return err
			}

			queries = append(queries, q)

			if limit > 0 && len(queries) >= limit {
				break
			}
		}

		return nil
	})

	return queries, wrapBucketErr(err, "bbolt.query_read_from", "read queries from position")
}

// Close is a no-op — the *bbolt.DB is owned by the Backend.
func (s *QueryStore) Close() error { return nil }

var _ io.Closer = (*QueryStore)(nil)
