package claimkit

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/larsartmann/go-cqrs-lite/claiming/v4"
)

// Dedup is the SQL runtime for the dedup window: one shared table
// (meta_dedup) scoped by collection, implementing [metaengine.DedupStore].
// The check-and-set is a single atomic upsert on SQLite/Postgres (the
// conflict-update fires only when the previous window lapsed) and a
// lock-free read plus INSERT IGNORE / conditional UPDATE pair on MySQL
// (SELECT..FOR UPDATE on an absent row takes gap locks that deadlock
// concurrent inserts — Error 1213).
type Dedup struct {
	db      *sql.DB
	dialect claiming.Dialect

	duckWriteLock
}

// lockWriter returns the write guard for the engine's concurrency model
// (see Claims.lockWriter).
func (d *Dedup) lockWriter() func() {
	return d.lockWriterFor(d.dialect)
}

// NewDedup creates the runtime, ensuring the dedup table exists. The caller
// retains ownership of db.
func NewDedup(ctx context.Context, db *sql.DB, d claiming.Dialect) (*Dedup, error) {
	switch d {
	case claiming.DialectSQLite,
		claiming.DialectPostgres,
		claiming.DialectMySQL,
		claiming.DialectDuckDB:
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
	var expiresRaw any

	err := d.db.QueryRowContext(ctx,
		"SELECT expires_at FROM meta_dedup WHERE collection = ? AND "+idColumn(d.dialect)+" = ?",
		collection, key).Scan(&expiresRaw)

	switch {
	case errors.Is(err, sql.ErrNoRows):
		// INSERT IGNORE keeps the absent-row race single-statement: exactly
		// one concurrent racer's insert lands (RowsAffected 1, new window,
		// not seen); everyone else's is ignored (RowsAffected 0, a window
		// was created concurrently — live by construction — so seen).
		res, err := d.db.ExecContext(
			ctx,
			"INSERT IGNORE INTO meta_dedup (collection, "+idColumn(
				d.dialect,
			)+", expires_at) VALUES (?, ?, ?)",
			collection,
			key,
			now.Add(ttl),
		)
		if err != nil {
			return false, fmt.Errorf("claimkit.DedupCheckAndRecord: insert: %w", err)
		}

		n, err := res.RowsAffected()
		if err != nil {
			return false, fmt.Errorf("claimkit.DedupCheckAndRecord: rows affected: %w", err)
		}

		return n == 0, nil
	case err != nil:
		return false, fmt.Errorf("claimkit.DedupCheckAndRecord: select: %w", err)
	}

	expires, err := decodeTime(expiresRaw)
	if err != nil {
		return false, fmt.Errorf("claimkit.DedupCheckAndRecord: %w", err)
	}

	if expires.After(now) {
		return true, nil // live window: seen, no write, no locks
	}

	// Lapsed window: the conditional UPDATE is the CAS — it lands only while
	// the row is still expired, so a concurrent takeover (RowsAffected 0)
	// means a live window exists: seen.
	res, err := d.db.ExecContext(ctx,
		"UPDATE meta_dedup SET expires_at = ? WHERE collection = ? AND "+idColumn(d.dialect)+
			" = ? AND expires_at <= ?",
		now.Add(ttl), collection, key, now)
	//art-dupl:accept Exec+RowsAffected error-handling idiom shared with ClaimDeleteFacts; not domain logic
	if err != nil {
		return false, fmt.Errorf("claimkit.DedupCheckAndRecord: update: %w", err)
	}

	n, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("claimkit.DedupCheckAndRecord: rows affected: %w", err)
	}

	return n == 0, nil
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
