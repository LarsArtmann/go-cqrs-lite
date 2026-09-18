package sqliteengine

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// CellVersioningOption tunes cell versioning on a SQLite engine.
type CellVersioningOption func(*sqliteEngine)

// WithRetention sets the version-retention policy trimmed on every versioned
// write: MaxVersions keeps the newest N versions per cell, MaxAge drops
// versions older than the cutoff relative to each write. The zero policy
// keeps everything.
func WithRetention(policy metaengine.RetentionPolicy) CellVersioningOption {
	return func(e *sqliteEngine) { e.versionRetention = &policy }
}

// WithCellVersioning enables BigTable-style versioned cells (ADR-0141): every
// Map write on a non-planned collection also records a timestamped version
// row in meta_cell_versions, enabling MapGetAsOf / MapHistory / MapSetAt.
// Planned-table collections (layout plans) are not versioned — temporal reads
// on them fail with ErrUnsupportedADT rather than degrading silently.
func WithCellVersioning(opts ...CellVersioningOption) EngineOption {
	return func(e *sqliteEngine) {
		e.versioning = true

		for _, opt := range opts {
			opt(e)
		}
	}
}

// CellVersioningEnabled implements [metaengine.CellVersioningToggle].
func (e *sqliteEngine) CellVersioningEnabled() bool { return e.versioning }

const cellVersionsDDL = `
CREATE TABLE IF NOT EXISTS meta_cell_versions (
	collection TEXT NOT NULL,
	key TEXT NOT NULL,
	ts INTEGER NOT NULL,
	value TEXT,
	PRIMARY KEY (collection, key, ts)
);`

func errPlannedNotVersioned(col string) error {
	return fmt.Errorf(
		"%w: planned collection %q does not support versioned cells",
		metaengine.ErrUnsupportedADT,
		col,
	)
}

// recordVersionRow inserts one version row (NULL value = tombstone) and trims
// retention. Caller has already written the latest view.
func (e *sqliteEngine) recordVersionRow(
	ctx context.Context,
	col, key string,
	value any,
	ts time.Time,
) error {
	var valStr any
	if value != nil {
		s := encodeValue(value)
		valStr = s
	}

	if _, err := e.xc(ctx).exec(
		ctx, `INSERT OR REPLACE INTO meta_cell_versions (collection, key, ts, value) VALUES (?, ?, ?, ?)`,
		col, key, ts.UnixNano(), valStr,
	); err != nil {
		return fmt.Errorf("insert cell version: %w", err)
	}

	return e.trimVersionRetention(ctx, col, key, ts)
}

// trimVersionRetention enforces the configured RetentionPolicy for one cell,
// never deleting the newest version.
func (e *sqliteEngine) trimVersionRetention(
	ctx context.Context,
	col, key string,
	newest time.Time,
) error {
	if e.versionRetention == nil {
		return nil
	}

	if e.versionRetention.MaxVersions > 0 {
		if _, err := e.xc(ctx).exec(ctx,
			`DELETE FROM meta_cell_versions WHERE collection = ? AND key = ? AND ts NOT IN (
				SELECT ts FROM meta_cell_versions WHERE collection = ? AND key = ? ORDER BY ts DESC LIMIT ?)`,
			col, key, col, key, e.versionRetention.MaxVersions,
		); err != nil {
			return fmt.Errorf("trim cell versions (max): %w", err)
		}
	}

	if e.versionRetention.MaxAge > 0 {
		cutoff := newest.Add(-e.versionRetention.MaxAge).UnixNano()
		if _, err := e.xc(ctx).exec(ctx,
			`DELETE FROM meta_cell_versions
			 WHERE collection = ? AND key = ? AND ts < ?
			   AND ts < (SELECT MAX(ts) FROM meta_cell_versions WHERE collection = ? AND key = ?)`,
			col, key, cutoff, col, key,
		); err != nil {
			return fmt.Errorf("trim cell versions (age): %w", err)
		}
	}

	return nil
}

// --- metaengine.VersionedWriter ---

func (e *sqliteEngine) MapSetAt(
	ctx context.Context,
	col string,
	key any,
	value any,
	ts time.Time,
) error {
	if _, planned := e.plans[col]; planned {
		return errPlannedNotVersioned(col)
	}

	keyStr := encodeKey(key)

	if _, err := e.xc(ctx).exec(ctx, e.queries.mapSet, col, keyStr, encodeValue(value)); err != nil {
		return fmt.Errorf("map set-at latest: %w", err)
	}

	return e.recordVersionRow(ctx, col, keyStr, value, ts)
}

func (e *sqliteEngine) MapDeleteAt(
	ctx context.Context,
	col string,
	key any,
	ts time.Time,
) error {
	if _, planned := e.plans[col]; planned {
		return errPlannedNotVersioned(col)
	}

	keyStr := encodeKey(key)

	if _, err := e.xc(ctx).exec(ctx, e.queries.mapDelete, col, keyStr); err != nil {
		return fmt.Errorf("map delete-at latest: %w", err)
	}

	return e.recordVersionRow(ctx, col, keyStr, nil, ts)
}

// --- metaengine.VersionedUpdater ---

// MapUpdateAt applies update to the current latest value and records the
// result as the version at ts, in one transaction.
func (e *sqliteEngine) MapUpdateAt(
	ctx context.Context,
	col string,
	key any,
	update func(prev any) any,
	ts time.Time,
) error {
	if _, planned := e.plans[col]; planned {
		return errPlannedNotVersioned(col)
	}

	err := e.RunInTx(ctx, func(ctx context.Context) error {
		prev, found, err := e.MapGet(ctx, col, key)
		if err != nil {
			return err
		}

		var prevVal any
		if found {
			prevVal = prev
		}

		return e.MapSetAt(ctx, col, key, update(prevVal), ts)
	})
	if err != nil {
		return fmt.Errorf("map update-at: %w", err) //nolint:wrapcheck // already wrapped upstream
	}

	return nil
}

// --- metaengine.VersionedStorage ---

func (e *sqliteEngine) scanVersionRow(
	ctx context.Context,
	col, key string,
	at time.Time,
) (any, bool, error) {
	var valStr sql.NullString

	err := e.xc(ctx).queryRow(ctx,
		`SELECT value FROM meta_cell_versions
		 WHERE collection = ? AND key = ? AND ts <= ? ORDER BY ts DESC LIMIT 1`,
		col, key, at.UnixNano(),
	).Scan(&valStr)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, false, nil
		}

		return nil, false, fmt.Errorf("select cell version: %w", err)
	}

	if !valStr.Valid {
		return nil, false, nil // tombstone
	}

	return metaengine.DecodeStreamValue(valStr.String), true, nil
}

func (e *sqliteEngine) MapGetAsOf(
	ctx context.Context,
	col, key string,
	t time.Time,
) (any, error) {
	val, found, err := e.scanVersionRow(ctx, col, key, t)
	if err != nil {
		return nil, err
	}

	if !found {
		return nil, metaengine.ErrNotFound
	}

	return val, nil
}

func (e *sqliteEngine) MapExistsAsOf(
	ctx context.Context,
	col, key string,
	t time.Time,
) (bool, error) {
	_, found, err := e.scanVersionRow(ctx, col, key, t)

	return found, err
}

// --- metaengine.CellHistoryReader ---

func (e *sqliteEngine) MapHistory(
	ctx context.Context,
	col, key string,
	from, to time.Time,
) ([]metaengine.CellVersion, error) {
	rows, err := e.xd(ctx).QueryContext(ctx,
		`SELECT ts, value FROM meta_cell_versions
		 WHERE collection = ? AND key = ? AND ts >= ? AND ts <= ? ORDER BY ts DESC`,
		col, key, from.UnixNano(), to.UnixNano(),
	) //nolint:sqlclosecheck
	if err != nil {
		return nil, fmt.Errorf("select cell history: %w", err)
	}

	defer metaengine.DeferClose(rows)

	var out []metaengine.CellVersion

	for rows.Next() {
		var tsNano int64

		var valStr sql.NullString

		if err := rows.Scan(&tsNano, &valStr); err != nil {
			return nil, fmt.Errorf("scan cell history: %w", err)
		}

		version := metaengine.CellVersion{Timestamp: time.Unix(0, tsNano)}
		if valStr.Valid {
			version.Value = metaengine.DecodeStreamValue(valStr.String)
		}

		out = append(out, version)
	}

	return out, rows.Err() //nolint:wrapcheck // passthrough
}

var (
	_ metaengine.VersionedStorage     = (*sqliteEngine)(nil)
	_ metaengine.VersionedWriter      = (*sqliteEngine)(nil)
	_ metaengine.VersionedUpdater     = (*sqliteEngine)(nil)
	_ metaengine.CellHistoryReader    = (*sqliteEngine)(nil)
	_ metaengine.CellVersioningToggle = (*sqliteEngine)(nil)
)
