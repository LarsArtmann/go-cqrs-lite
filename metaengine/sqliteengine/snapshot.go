package sqliteengine

import (
	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// metaengine.SnapshotBackend implementation for the SQLite engine.
// Snapshots are stored in the meta_snapshot table with a composite primary
// key (collection, stream_id). One snapshot per stream — saves overwrite.

func (e *sqliteEngine) SnapshotSave(
	ctx context.Context,
	collection, streamID string,
	version int64,
	data []byte,
) error {
	_, err := e.db.ExecContext(
		ctx,
		`INSERT OR REPLACE INTO meta_snapshot (collection, stream_id, version, data) VALUES (?, ?, ?, ?)`,
		collection,
		streamID,
		version,
		data,
	)
	if err != nil {
		return fmt.Errorf("metaengine: snapshot save %s/%s: %w", collection, streamID, err)
	}

	return nil
}

func (e *sqliteEngine) SnapshotLoad(
	ctx context.Context,
	collection, streamID string,
) ([]byte, int64, error) {
	var data []byte
	var version int64

	err := e.db.QueryRowContext(ctx,
		`SELECT data, version FROM meta_snapshot WHERE collection = ? AND stream_id = ?`,
		collection, streamID,
	).Scan(&data, &version)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, 0, metaengine.ErrNotFound
	}

	if err != nil {
		return nil, 0, fmt.Errorf("metaengine: snapshot load %s/%s: %w", collection, streamID, err)
	}

	return data, version, nil
}

func (e *sqliteEngine) SnapshotLoadAtVersion(
	ctx context.Context,
	collection, streamID string,
	maxVersion int64,
) ([]byte, int64, error) {
	var data []byte
	var version int64

	err := e.db.QueryRowContext(ctx,
		`SELECT data, version FROM meta_snapshot
		 WHERE collection = ? AND stream_id = ? AND version <= ?
		 ORDER BY version DESC LIMIT 1`,
		collection, streamID, maxVersion,
	).Scan(&data, &version)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, 0, metaengine.ErrNotFound
	}

	if err != nil {
		return nil, 0, fmt.Errorf(
			"metaengine: snapshot load at version %s/%s: %w",
			collection,
			streamID,
			err,
		)
	}

	return data, version, nil
}

func (e *sqliteEngine) SnapshotDelete(
	ctx context.Context,
	collection, streamID string,
) error {
	_, err := e.db.ExecContext(ctx,
		`DELETE FROM meta_snapshot WHERE collection = ? AND stream_id = ?`,
		collection, streamID,
	)
	if err != nil {
		return fmt.Errorf("metaengine: snapshot delete %s/%s: %w", collection, streamID, err)
	}

	return nil
}

// Compile-time assertion.
var _ metaengine.SnapshotBackend = (*sqliteEngine)(nil)
