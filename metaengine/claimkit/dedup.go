package claimkit

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/larsartmann/go-cqrs-lite/claiming/v4"
)

// Dedup is the SQL runtime for the dedup window: one shared table
// (meta_dedup) scoped by collection, implementing [metaengine.DedupStore].
// The check-and-set is a single atomic upsert on SQLite/Postgres (the
// conflict-update fires only when the previous window lapsed) and a
// SELECT..FOR UPDATE two-step inside one transaction on MySQL.
type Dedup struct {
	db      *sql.DB
	dialect claiming.Dialect

	// mu serializes writes on single-writer engines (DuckDB): its ON
	// CONFLICT upserts raise PK violations under concurrency instead of
	// serializing — the in-process mutex is the row-lock equivalent for an
	// embedded single-process database.
	mu sync.Mutex
}

// lockWriter returns the write guard for the engine's concurrency model
	// (see Claims.lockWriter).
func (d *Dedup) lockWriter() func() {
	if d.dialect != claiming.DialectDuckDB {
		return func() {}
	}

	d.mu.Lock()

	return d.mu.Unlock
}

// NewDedup creates the runtime, ensuring the dedup table exists. The caller
// retains ownership of db.
func NewDedup(ctx context.Context, db *sql.DB, d claiming.Dialect) (*Dedup, error) {
	switch d {
	case claiming.DialectSQLite, claiming.DialectPostgres, claiming.DialectMySQL, claiming.DialectDuckDB:
	default:
		return nil, fmt.Errorf("claimkit.NewDedup: %w", claiming.ErrUnsupported)
	}

	if err := ensureClaimsTables(ctx, db, d); err != nil {
		return nil, fmt.Errorf("claimkit.NewDedup: %w", err)
	}

	return &Dedup{db: db, dialect: d}, nil
}

func (d *Dedup) encodeTime(t time.Time) any {
	if d.dialect == claiming.DialectSQLite {
		return t.Format(sqliteTimeFormat)
	}

	return t
}

// DedupCheckAndRecord implements [metaengine.DedupStore.DedupCheckAndRecord]:
// true exactly when a live, unexpired record existed before the call.
func (d *Dedup) DedupCheckAndRecord(
	ctx context.Context,
	collection, key string,
	ttl time.Duration,
	now time.Time,
) (bool, error) {
	defer d.lockWriter()()

	if now.IsZero() {
		now = time.Now()
	}

	if d.dialect == claiming.DialectMySQL {
		return d.checkAndRecordMySQL(ctx, collection, key, ttl, now)
	}

	query := `INSERT INTO meta_dedup (collection, key, expires_at)
VALUES (` + ph(d.dialect, 1) + ", " + ph(d.dialect, 2) + ", " + ph(d.dialect, 3) + `)
ON CONFLICT (collection, key) DO UPDATE SET expires_at = ` + conflictExcluded(d.dialect) +
		"\nWHERE meta_dedup.expires_at <= " + ph(d.dialect, 4) +
		"\nRETURNING key"

	var returned string

	err := d.db.QueryRowContext(ctx, query,
		collection, key, d.encodeTime(now.Add(ttl)), d.encodeTime(now)).Scan(&returned)
	if errors.Is(err, sql.ErrNoRows) {
		return true, nil // conflict with a live window: already seen
	}

	if err != nil {
		return false, fmt.Errorf("claimkit.DedupCheckAndRecord: %w", err)
	}

	return false, nil // inserted or re-claimed an expired window
}

// conflictExcluded renders the conflicting row's proposed value per dialect
// (EXCLUDED on Postgres, excluded on SQLite — both spellings are accepted,
// this keeps the SQL explicit).
func conflictExcluded(_ claiming.Dialect) string { return "excluded.expires_at" }

func (d *Dedup) checkAndRecordMySQL(
	ctx context.Context,
	collection, key string,
	ttl time.Duration,
	now time.Time,
) (bool, error) {
	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return false, fmt.Errorf("claimkit.DedupCheckAndRecord: begin: %w", err)
	}

	defer func() { _ = tx.Rollback() }()

	var expiresRaw any

	err = tx.QueryRowContext(ctx,
		"SELECT expires_at FROM meta_dedup WHERE collection = ? AND key = ? FOR UPDATE",
		collection, key).Scan(&expiresRaw)

	switch {
	case errors.Is(err, sql.ErrNoRows):
		if _, err := tx.ExecContext(ctx,
			"INSERT INTO meta_dedup (collection, "+idColumn(d.dialect)+", expires_at) VALUES (?, ?, ?)",
			collection, key, now.Add(ttl)); err != nil {
			return false, fmt.Errorf("claimkit.DedupCheckAndRecord: insert: %w", err)
		}
	case err != nil:
		return false, fmt.Errorf("claimkit.DedupCheckAndRecord: select: %w", err)
	default:
		expires, err := decodeTime(expiresRaw)
		if err != nil {
			return false, fmt.Errorf("claimkit.DedupCheckAndRecord: %w", err)
		}

		if expires.After(now) {
			if err := tx.Commit(); err != nil {
				return false, fmt.Errorf("claimkit.DedupCheckAndRecord: commit: %w", err)
			}

			return true, nil
		}

		if _, err := tx.ExecContext(ctx,
			"UPDATE meta_dedup SET expires_at = ? WHERE collection = ? AND "+idColumn(d.dialect)+" = ?",
			now.Add(ttl), collection, key); err != nil {
			return false, fmt.Errorf("claimkit.DedupCheckAndRecord: update: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return false, fmt.Errorf("claimkit.DedupCheckAndRecord: commit: %w", err)
	}

	return false, nil
}

// DedupSeen implements [metaengine.DedupStore.DedupSeen].
func (d *Dedup) DedupSeen(
	ctx context.Context,
	collection, key string,
	now time.Time,
) (bool, error) {
	if now.IsZero() {
		now = time.Now()
	}

	var expiresRaw any

	err := d.db.QueryRowContext(ctx,
		"SELECT expires_at FROM meta_dedup WHERE collection = "+ph(d.dialect, 1)+
			" AND "+idColumn(d.dialect)+" = "+ph(d.dialect, 2),
		collection, key).Scan(&expiresRaw)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}

	if err != nil {
		return false, fmt.Errorf("claimkit.DedupSeen: %w", err)
	}

	expires, err := decodeTime(expiresRaw)
	if err != nil {
		return false, fmt.Errorf("claimkit.DedupSeen: %w", err)
	}

	return expires.After(now), nil
}

// DedupSweep implements [metaengine.DedupStore.DedupSweep].
func (d *Dedup) DedupSweep(ctx context.Context, collection string, now time.Time) (int, error) {
	defer d.lockWriter()()

	if now.IsZero() {
		now = time.Now()
	}

	res, err := d.db.ExecContext(ctx,
		"DELETE FROM meta_dedup WHERE collection = "+ph(d.dialect, 1)+
			" AND expires_at <= "+ph(d.dialect, 2),
		collection, d.encodeTime(now))
	if err != nil {
		return 0, fmt.Errorf("claimkit.DedupSweep: %w", err)
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("claimkit.DedupSweep: rows affected: %w", err)
	}

	return int(affected), nil
}
