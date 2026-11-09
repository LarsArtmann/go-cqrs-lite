package postgres

import (
	"context"
	"encoding/json/v2"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/larsartmann/go-cqrs-lite/queue/v4/facts"
	"github.com/larsartmann/go-cqrs-lite/queue/v4/task"
)

// factColumns is the canonical facts-table read list.
const factColumns = `seq, time, task_id, type, owner, attempt, error, detail`

// appendFact records one fact inside the caller's transaction. Time is
// assigned when zero; detail binds as text (the column is NOT NULL).
func (s *Store[T]) appendFact(ctx context.Context, tx pgx.Tx, f facts.Fact) error {
	if f.Time.IsZero() {
		f.Time = time.Now()
	}

	detail := string(f.Detail)

	_, err := tx.Exec(ctx,
		`INSERT INTO facts (time, task_id, type, owner, attempt, error, detail)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		f.Time.UnixMilli(), f.TaskID, string(f.Type), f.Owner, f.Attempt, f.Error, detail)

	//art-dupl:accept dialect twin of queue/sqlite journal prologue; conformance pins journal semantics
	return err
}

// Facts returns journal facts with Seq > after, ascending, bounded to
// limit when > 0.
// art-dupl:accept dialect twin — queue postgres/sqlite stores are dep-isolated mirrors; conformance pins semantics
func (s *Store[T]) Facts(ctx context.Context, after int64, limit int) ([]facts.Fact, error) {
	query := `SELECT ` + factColumns + ` FROM facts WHERE seq > $1 ORDER BY seq ASC`
	args := []any{after}

	if limit > 0 {
		query += ` LIMIT $2`

		args = append(args, limit)
	}

	//art-dupl:accept rows-query prologue idiom (Facts/FactsForTask); not domain logic
	rows, err := s.pool.Query(ctx, query, args...)
	//art-dupl:accept dialect twin of queue/sqlite Facts flow
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	return scanFacts(rows)
}

// FactsForTask returns one task's facts in Seq order, bounded to the
// most recent limit when > 0 (tail read DESC LIMIT, then flipped).
// art-dupl:accept dialect twin — queue postgres/sqlite stores are dep-isolated mirrors; conformance pins semantics
func (s *Store[T]) FactsForTask(ctx context.Context, id task.ID, limit int) ([]facts.Fact, error) {
	query := `SELECT ` + factColumns + ` FROM facts WHERE task_id = $1`
	args := []any{id.String()}

	if limit > 0 {
		query += ` ORDER BY seq DESC LIMIT $2`

		args = append(args, limit)
	} else {
		query += ` ORDER BY seq ASC`
	}

	//art-dupl:accept rows-query prologue idiom (Facts/FactsForTask); not domain logic
	rows, err := s.pool.Query(ctx, query, args...)
	//art-dupl:accept dialect twin of queue/sqlite FactsForTask tail-read flow
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	//art-dupl:accept dialect twin — queue postgres/sqlite stores are dep-isolated mirrors; conformance pins semantics
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

	err := s.pool.QueryRow(ctx, `SELECT COALESCE(MAX(seq), 0) FROM facts`).Scan(&seq)

	return seq, err
}

// Watermark returns the persisted read cursor for a journal consumer
// and whether it ever checkpointed.
func (s *Store[T]) Watermark(ctx context.Context, consumer string) (int64, bool, error) {
	var seq int64

	err := s.pool.QueryRow(ctx,
		`SELECT seq FROM watermarks WHERE consumer = $1`, consumer).Scan(&seq)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, false, nil
	}

	if err != nil {
		return 0, false, err
	}

	return seq, true, nil
}

// Watermarks lists every consumer's cursor — the operator lag surface.
func (s *Store[T]) Watermarks(ctx context.Context) (map[string]int64, error) {
	rows, err := s.pool.Query(ctx, `SELECT consumer, seq FROM watermarks ORDER BY consumer`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

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

// SaveWatermark checkpoints a consumer cursor as a monotonic upsert.
func (s *Store[T]) SaveWatermark(ctx context.Context, consumer string, seq int64) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO watermarks (consumer, seq, updated_at) VALUES ($1, $2, $3)
		ON CONFLICT(consumer) DO UPDATE SET
			seq = excluded.seq,
			updated_at = excluded.updated_at
		WHERE watermarks.seq < excluded.seq`,
		consumer, seq, time.Now().UnixMilli())

	return err
}

// scanFacts decodes a facts row set.
func scanFacts(rows pgx.Rows) ([]facts.Fact, error) {
	var out []facts.Fact

	for rows.Next() {
		var (
			f      facts.Fact
			msTime int64
			detail string
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
		f.Detail = []byte(detail)

		out = append(out, f)
	}

	//art-dupl:accept dialect twin of queue/sqlite scanFacts tail; conformance pins fact decoding
	return out, rows.Err()
}

// ms converts to unix millis with the zero time as 0.
func ms(t time.Time) int64 {
	if t.IsZero() {
		return 0
	}

	return t.UnixMilli()
}

// mustJSON marshals or degrades to an empty object.
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
