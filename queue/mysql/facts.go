package mysql

import (
	"context"
	"database/sql"
	"encoding/json/v2"
	"errors"
	"time"

	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"github.com/larsartmann/go-cqrs-lite/queue/v4"
	"github.com/larsartmann/go-cqrs-lite/queue/v4/facts"
	"github.com/larsartmann/go-cqrs-lite/queue/v4/task"
)

// factColumns is the canonical facts-table read list.
const factColumns = `seq, time, task_id, type, owner, attempt, error, detail`

// appendFact records one fact inside the caller's transaction — the
// in-tx pairing that keeps the journal a consistent history of the
// queue. Time is assigned when zero; a nil detail binds as empty bytes
// (the column is NOT NULL — absent evidence is "", never NULL).
func (s *Store[T]) appendFact(ctx context.Context, tx *sql.Tx, f facts.Fact) error {
	if f.Time.IsZero() {
		f.Time = time.Now()
	}

	detail := f.Detail
	if detail == nil {
		detail = []byte{}
	}

	_, err := tx.ExecContext(ctx,
		`INSERT INTO facts (time, task_id, type, owner, attempt, error, detail)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		f.Time.UnixMilli(), f.TaskID, string(f.Type), f.Owner, f.Attempt, f.Error, detail)

	//art-dupl:accept dialect twin of queue/sqlite+postgres journal prologue; conformance pins journal semantics
	return err
}

// Facts returns journal facts with Seq > after, ascending, bounded to
// limit when > 0. The seq primary key makes the cursor scan O(limit)
// regardless of journal size.
// art-dupl:accept dialect twin — queue engines are dep-isolated mirrors; conformance pins semantics
func (s *Store[T]) Facts(ctx context.Context, after int64, limit int) ([]facts.Fact, error) {
	query := `SELECT ` + factColumns + ` FROM facts WHERE seq > ? ORDER BY seq ASC`
	args := []any{after}

	if limit > 0 {
		query += ` LIMIT ?`

		args = append(args, limit)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}

	defer metaengine.DeferClose(rows)

	return scanFacts(rows)
}

// FactsForTask returns one task's facts in Seq order, bounded to the
// most recent limit when > 0. The tail is read DESC LIMIT then flipped —
// the plain ASC+LIMIT shape silently returns the FIRST n (the donor's
// cross-store conformance catch).
// art-dupl:accept dialect twin — queue engines are dep-isolated mirrors; conformance pins semantics
func (s *Store[T]) FactsForTask(ctx context.Context, id task.ID, limit int) ([]facts.Fact, error) {
	query := `SELECT ` + factColumns + ` FROM facts WHERE task_id = ?`
	args := []any{id.String()}

	if limit > 0 {
		query += ` ORDER BY seq DESC LIMIT ?`

		args = append(args, limit)
	} else {
		query += ` ORDER BY seq ASC`
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}

	defer metaengine.DeferClose(rows)

	all, err := scanFacts(rows)
	if err != nil {
		return nil, err
	}

	if limit > 0 {
		for i, j := 0, len(all)-1; i < j; i, j = i+1, j-1 {
			all[i], all[j] = all[j], all[i]
		}
	}

	return all, nil
}

// HeadSeq returns the current highest fact Seq (0 when empty).
func (s *Store[T]) HeadSeq(ctx context.Context) (int64, error) {
	var seq int64

	err := s.db.QueryRowContext(ctx, `SELECT COALESCE(MAX(seq), 0) FROM facts`).Scan(&seq)

	return seq, err
}

// Watermark returns the persisted read cursor for a journal consumer
// and whether it ever checkpointed — the resume point for consumers
// after a restart.
func (s *Store[T]) Watermark(ctx context.Context, consumer string) (int64, bool, error) {
	var seq int64

	err := s.db.QueryRowContext(ctx,
		`SELECT seq FROM watermarks WHERE consumer = ?`, consumer).Scan(&seq)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, false, nil
	}

	if err != nil {
		return 0, false, err
	}

	return seq, true, nil
}

// Watermarks lists every consumer's cursor — the operator lag surface.
func (s *Store[T]) Watermarks(ctx context.Context) (map[string]int64, error) {
	rows, err := s.db.QueryContext(
		ctx,
		`SELECT consumer, seq FROM watermarks ORDER BY consumer`,
	)
	if err != nil {
		return nil, err
	}
	defer metaengine.DeferClose(rows)

	out := make(map[string]int64)

	for rows.Next() {
		var consumer string

		var seq int64

		if err := rows.Scan(&consumer, &seq); err != nil {
			return nil, err
		}

		out[consumer] = seq
	}

	return out, rows.Err()
}

// SaveWatermark checkpoints a consumer cursor as a monotonic upsert:
// the stored seq never regresses, so a lagging or misconfigured second
// process cannot drag a consumer backwards.
func (s *Store[T]) SaveWatermark(ctx context.Context, consumer string, seq int64) error {
	// MySQL/MariaDB have no upsert-WHERE (the sqlite/postgres guard), so the
	// monotonic contract is expressed as GREATEST + a conditional timestamp:
	// a regressing save changes nothing observable.
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO watermarks (consumer, seq, updated_at) VALUES (?, ?, ?)
		ON DUPLICATE KEY UPDATE
			seq = GREATEST(watermarks.seq, VALUES(seq)),
			updated_at = IF(VALUES(seq) > watermarks.seq, VALUES(updated_at), watermarks.updated_at)`,
		consumer, seq, time.Now().UnixMilli())

	return err
}

// scanFacts decodes a facts row set.
func scanFacts(rows *sql.Rows) ([]facts.Fact, error) {
	var out []facts.Fact

	for rows.Next() {
		var (
			f      facts.Fact
			msTime int64
			detail []byte
		)

		if err := rows.Scan(
			&f.Seq,
			&msTime,
			&f.TaskID,
			&f.Type,
			&f.Owner,
			&f.Attempt,
			&f.Error,
			&detail,
		); err != nil {
			return nil, err
		}

		f.Time = time.UnixMilli(msTime)
		f.Detail = detail

		out = append(out, f)
	}

	//art-dupl:accept dialect twin of queue/sqlite+postgres scanFacts tail; conformance pins fact decoding
	return out, rows.Err()
}

// WithFacts runs fn inside one transaction; every sink append commits
// with it or rolls back with fn's error — the same same-tx guarantee
// the store's own mutations enjoy (the ADR-0001 lineage).
func (s *Store[T]) WithFacts(ctx context.Context, fn func(sink queue.FactSink) error) error {
	//art-dupl:accept dialect twin of queue/sqlite+postgres WithFacts; conformance pins the atomicity split
	return s.withTx(ctx, func(tx *sql.Tx) error {
		return fn(factSink[T]{store: s, tx: tx})
	})
}

// factSink appends facts through the store's in-tx append path.
type factSink[T any] struct {
	store *Store[T]
	tx    *sql.Tx
}

// Append records one fact in the sink's transaction.
func (fs factSink[T]) Append(ctx context.Context, f facts.Fact) error {
	return fs.store.appendFact(ctx, fs.tx, f)
}

// ms converts to unix millis with the zero time as 0.
func ms(t time.Time) int64 {
	if t.IsZero() {
		return 0
	}

	return t.UnixMilli()
}

// mustJSON marshals or degrades to an empty object — fact details are
// evidence, never a failure path.
func mustJSON(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		return []byte("{}")
	}

	return b
}

// maybeBytes normalizes nil evidence to empty bytes (NOT NULL columns).
func maybeBytes(b []byte) []byte {
	if b == nil {
		return []byte{}
	}

	return b
}
