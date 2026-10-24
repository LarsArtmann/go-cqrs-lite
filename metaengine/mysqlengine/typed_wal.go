package mysqlengine

import (
	"context"
	"database/sql"
	"encoding/json/v2"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
)

// The typed WAL columns (disk-truth plan A12b, mirroring the sqliteengine A11
// reference and the pgengine A12a twin): meta_stream_log historically stores
// one opaque JSON envelope per row, so the journal of record has NO
// database-level identity or ordering constraints — a duplicate append is a
// silent second row. The typed columns extract the envelope's identity into
// real columns guarded by UNIQUE indexes:
//
//	uq_stream_log_event          (collection, event_id)
//	uq_stream_log_stream_version (collection, stream_id, version)
//
// (MySQL/MariaDB unique indexes already allow multiple NULLs, matching the
// partial-index semantics of the other engines: non-envelope rows never
// trip the constraint.) The journal is COPIED into the typed columns, never
// rewritten (ADR-0143): the value envelope stays the read path; legacy rows
// are identity-backfilled from it via JSON_EXTRACT. payload/metadata/
// occurred_at are writer-filled and stay NULL on backfilled rows (the same
// uniform contract across engines: a malformed foreign value must never
// abort the migration; readers fall back to the envelope).
type typedStreamRow struct { // art-dupl:accept cross-module SQL engine twin (sqliteengine A11 / pgengine A12a) — dep-isolated go.mod modules
	eventID       *string
	streamType    *string
	version       *int64
	schemaVersion *int64
	payload       []byte
	encoding      *string
	metadata      []byte
	occurredAt    *int64
}

// extractTypedStreamRow best-effort parses a stream-log value as the system
// event envelope. ok=false for anything else (foreign rows, plain scalars) —
// those rows keep NULL typed columns and never trip the unique indexes.
func extractTypedStreamRow(encoded string) (typedStreamRow, bool) {
	var env struct {
		ID            string `json:"id"`
		StreamType    string `json:"stream_type"`
		Version       int    `json:"version"`
		SchemaVersion int    `json:"schema_version"`
		Payload       []byte `json:"payload"`
		Encoding      string `json:"encoding"`
		Metadata      []byte `json:"metadata"`
		OccurredAt    string `json:"occurred_at"`
	}

	if err := json.Unmarshal([]byte(encoded), &env); err != nil {
		return typedStreamRow{}, false
	}

	if env.ID == "" || env.StreamType == "" || env.Version == 0 {
		return typedStreamRow{}, false
	}

	id, st, enc := env.ID, env.StreamType, env.Encoding
	version := int64(env.Version)

	row := typedStreamRow{
		eventID:    &id,
		streamType: &st,
		version:    &version,
		encoding:   &enc,
		payload:    env.Payload,
		metadata:   env.Metadata,
	}

	if env.SchemaVersion != 0 {
		sv := int64(env.SchemaVersion)
		row.schemaVersion = &sv
	}

	// env.Payload / env.Metadata arrive ALREADY base64-decoded: encoding/json
	// decodes []byte fields from their base64 wire form. Re-decoding here would
	// corrupt the column whenever the raw bytes are coincidentally valid
	// base64 (the A11 pin lesson, 2026-10-23).

	if t, err := time.Parse(time.RFC3339Nano, env.OccurredAt); err == nil {
		nanos := t.UnixNano()
		row.occurredAt = &nanos
	}

	return row, true
}

// typedStreamColumns is the ordered list of typed columns added to
// meta_stream_log ("name TYPE" pairs, MySQL dialect). event_id is
// VARCHAR(512): under utf8mb4 the 3072-byte InnoDB index-key limit leaves
// 512 chars next to the existing VARCHAR(255) collection column
// ((255+512)*4 = 3068 <= 3072); ULID event ids stay far inside it.
var typedStreamColumns = []string{
	"event_id VARCHAR(512)",
	"stream_type VARCHAR(255)",
	"version BIGINT",
	"schema_version BIGINT",
	"payload BLOB",
	"encoding VARCHAR(64)",
	"metadata JSON",
	"occurred_at BIGINT",
}

// ensureTypedStreamLog migrates an existing meta_stream_log to the typed
// columns: information_schema column probe (MySQL 8 has no ADD COLUMN IF NOT
// EXISTS), one ADD COLUMN per statement pinned to ALGORITHM=INSTANT — the
// server REFUSES the ALTER if it would need to rebuild the table, so the
// journal of record is never recreated and never locked (the plan's risk
// register item, enforced by construction rather than convention). Then the
// JSON_EXTRACT identity backfill and the two UNIQUE indexes. Idempotent;
// fresh databases already carry the columns via the base DDL. A unique-index
// creation failure means the journal already contains duplicates — a data
// integrity alarm, not a migration bug; the error names the indexes so the
// operator can clean up and re-open.
func ensureTypedStreamLog(db *sql.DB) error {
	existing, err := probeStreamLogColumns(db)
	if err != nil {
		return err
	}

	for _, col := range typedStreamColumns {
		name := col[:strings.IndexByte(col, ' ')]
		if existing[name] {
			continue
		}

		if _, err := db.ExecContext(context.Background(),
			`ALTER TABLE meta_stream_log ADD COLUMN `+col+`, ALGORITHM=INSTANT`,
		); err != nil {
			return fmt.Errorf(
				"mysqlengine typed WAL: add column %s: %w — the server cannot add it "+
					"INSTANTly (a rebuild would lock the journal); resolve manually",
				name, err,
			)
		}
	}

	if err := backfillTypedStreamColumns(db); err != nil {
		return err
	}

	indexes := []string{
		`CREATE UNIQUE INDEX uq_stream_log_event ON meta_stream_log (collection, event_id)`,
		`CREATE UNIQUE INDEX uq_stream_log_stream_version ON meta_stream_log (collection, stream_id, version)`,
	}

	for _, idx := range indexes {
		if _, err := db.ExecContext(context.Background(), idx); err != nil && !isMySQLDuplicateIndex(err) {
			return fmt.Errorf(
				"mysqlengine typed WAL: unique index failed — the journal contains "+
					"duplicate event ids or (stream, version) pairs (clean up the duplicates "+
					"and re-open): %w",
				err,
			)
		}
	}

	return nil
}

// probeStreamLogColumns returns the set of column names meta_stream_log
// currently has.
func probeStreamLogColumns(db *sql.DB) (map[string]bool, error) {
	rows, err := db.QueryContext(context.Background(), `
		SELECT column_name FROM information_schema.columns
		WHERE table_schema = DATABASE() AND table_name = 'meta_stream_log'`)
	if err != nil {
		return nil, fmt.Errorf("mysqlengine typed WAL: probe columns: %w", err)
	}

	defer func() { _ = rows.Close() }()

	existing := map[string]bool{}

	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("mysqlengine typed WAL: scan probe: %w", err)
		}

		existing[name] = true
	}

	return existing, rows.Err()
}

// backfillTypedStreamColumns fills the identity columns for rows written
// before the typed columns existed — the same contract as the sqliteengine
// and pgengine references. JSON_VALID keeps non-JSON values outside the
// extraction; the regexp guards keep non-numeric envelope fields outside the
// UNSIGNED casts.
func backfillTypedStreamColumns(db *sql.DB) error {
	const stmt = `
		UPDATE meta_stream_log SET
			event_id       = JSON_UNQUOTE(JSON_EXTRACT(value, '$.id')),
			stream_type    = JSON_UNQUOTE(JSON_EXTRACT(value, '$.stream_type')),
			version        = CASE WHEN JSON_UNQUOTE(JSON_EXTRACT(value, '$.version')) REGEXP '^[0-9]+$'
			                      THEN CAST(JSON_UNQUOTE(JSON_EXTRACT(value, '$.version')) AS UNSIGNED) END,
			schema_version = CASE WHEN JSON_UNQUOTE(JSON_EXTRACT(value, '$.schema_version')) REGEXP '^[0-9]+$'
			                      THEN CAST(JSON_UNQUOTE(JSON_EXTRACT(value, '$.schema_version')) AS UNSIGNED) END,
			encoding       = JSON_UNQUOTE(JSON_EXTRACT(value, '$.encoding'))
		WHERE event_id IS NULL
		  AND JSON_VALID(value)
		  AND JSON_UNQUOTE(JSON_EXTRACT(value, '$.id')) IS NOT NULL
		  AND JSON_UNQUOTE(JSON_EXTRACT(value, '$.stream_type')) IS NOT NULL`

	if _, err := db.ExecContext(context.Background(), stmt); err != nil {
		return fmt.Errorf(
			"mysqlengine typed WAL: backfill: %w — if this is a duplicate-entry error, the "+
				"journal already contains duplicate event ids or (stream, version) pairs "+
				"(clean up the duplicates and re-open)",
			err,
		)
	}

	return nil
}

// isUniqueViolation reports whether err is the MySQL duplicate-entry error
// (1062) via the driver's structured error, falling back to message text for
// non-go-sql-driver connections.
func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}

	var my *mysql.MySQLError
	if errors.As(err, &my) {
		return my.Number == 1062
	}

	return strings.Contains(err.Error(), "Duplicate entry")
}
