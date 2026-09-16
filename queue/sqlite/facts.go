package sqlite

import (
	"context"
	"database/sql"
	"encoding/json/v2"
	"errors"
	"time"

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

	//art-dupl:accept dialect twin of queue/postgres journal prologue; conformance pins journal semantics
	return err
}

// Facts returns journal facts with Seq > after, ascending, bounded to
// limit when > 0. The seq primary key makes the cursor scan O(limit)
// regardless of journal size.
//art-dupl:accept dialect twin — queue postgres/sqlite stores are dep-isolated mirrors; conformance pins semantics
func (s *Store[T]) Facts(ctx context.Context, after int64, limit int) ([]facts.Fact, error) {
	query := `SELECT ` + factColumns + ` FROM facts WHERE seq > ? ORDER BY seq ASC`
	args := []any{after}

	if limit > 0 {
		query += ` LIMIT ?`

		args = append(args, limit)
	}

	//art-dupl:accept rows-query prologue idiom (Facts/FactsForTask); not domain logic
	rows, err := s.db.QueryContext(ctx, query, args...)
	//art-dupl:accept dialect twin of queue/postgres Facts flow
	if err != nil {
		return nil, err
	}

	defer func() { _ = rows.Close() }()

	return scanFacts(rows)
}

// FactsForTask returns one task's facts in Seq order, bounded to the
// most recent limit when > 0. The tail is read DESC LIMIT then flipped —
// the plain ASC+LIMIT shape silently returns the FIRST n (the donor's
// cross-store conformance catch).
//art-dupl:accept dialect twin — queue postgres/sqlite stores are dep-isolated mirrors; conformance pins semantics
func (s *Store[T]) FactsForTask(ctx context.Context, id task.ID, limit int) ([]facts.Fact, error) {
	query := `SELECT ` + factColumns + ` FROM facts WHERE task_id = ?`
	args := []any{id.String()}

	if limit > 0 {
		query += ` ORDER BY seq DESC LIMIT ?`

		args = append(args, limit)
	} else {
		query += ` ORDER BY seq ASC`
	}

	//art-dupl:accept rows-query prologue idiom (Facts/FactsForTask); not domain logic
	rows, err := s.db.QueryContext(ctx, query, args...)
	//art-dupl:accept dialect twin of queue/postgres FactsForTask tail-read flow
	if err != nil {
		return nil, err
	}

	defer func() { _ = rows.Close() }()

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

// SaveWatermark checkpoints a consumer cursor as a monotonic upsert:
// the stored seq never regresses, so a lagging or misconfigured second
// process cannot drag a consumer backwards.
func (s *Store[T]) SaveWatermark(ctx context.Context, consumer string, seq int64) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO watermarks (consumer, seq, updated_at) VALUES (?, ?, ?)
		ON CONFLICT(consumer) DO UPDATE SET
			seq = excluded.seq,
			updated_at = excluded.updated_at
		WHERE watermarks.seq < excluded.seq`,
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

	//art-dupl:accept dialect twin of queue/postgres scanFacts tail; conformance pins fact decoding
	return out, rows.Err()
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

// maybeBytes maps an absent payload to a nil detail.
func maybeBytes(b []byte) []byte {
	if len(b) == 0 {
		return nil
	}

	return b
}
