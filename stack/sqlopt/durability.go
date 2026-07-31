package sqlopt

import (
	"context"
	"database/sql"

	"github.com/larsartmann/go-cqrs-lite/stack/v4"
	"github.com/larsartmann/go-cqrs-lite/storage/v4"
)

// SQLiteSynchronousLevel translates a [stack.DurabilityTier] to the
// corresponding SQLite PRAGMA synchronous value.
//
//   - DurabilityStrict  → "FULL"  (fsync per commit)
//   - DurabilityNormal  → "NORMAL" (WAL default — fsync at checkpoint only)
//   - DurabilityRelaxed → "OFF"   (no fsync — data loss possible on crash)
func SQLiteSynchronousLevel(tier stack.DurabilityTier) string {
	switch tier {
	case stack.DurabilityStrict:
		return "FULL"
	case stack.DurabilityRelaxed:
		return "OFF"
	default:
		return "NORMAL"
	}
}

// ApplySQLiteDurability runs the synchronous PRAGMA for the given tier.
// Shared by the SQLite and Turso presets (both use the SQLite synchronous
// PRAGMA — Turso is libSQL, a SQLite fork). Call after [storage.SQLiteEnableWAL]
// so the tier override takes precedence.
func ApplySQLiteDurability(ctx context.Context, db *sql.DB, tier stack.DurabilityTier) error {
	if tier == "" || tier == stack.DurabilityNormal {
		return nil // NORMAL is already set by SQLiteEnableWAL
	}

	return storage.SQLiteSetSynchronous(ctx, db, SQLiteSynchronousLevel(tier))
}
