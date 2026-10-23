package sqliteengine

import (
	"database/sql"
	"encoding/base64"
	"encoding/json/v2"
	"fmt"
	"strings"
	"time"
)

// The typed WAL columns (disk-truth plan A11): meta_stream_log historically
// stores one opaque JSON envelope per row, so the journal of record has NO
// database-level identity or ordering constraints — a duplicate append is a
// silent second row. The typed columns extract the envelope's identity into
// real columns guarded by partial UNIQUE indexes:
//
//	uq_stream_log_event          (collection, event_id)
//	uq_stream_log_stream_version (collection, stream_id, version)
//
// The journal is COPIED into the typed columns, never rewritten (ADR-0143):
// the value envelope stays the read path; legacy rows are backfilled from it
// via json_extract. Payload/metadata/occurred_at are filled by the WRITER
// (base64-decoded raw bytes, unix nanos) but stay NULL on backfilled legacy
// rows — SQLite has no base64 or sub-second-epoch functions — and readers
// fall back to the envelope for those rows.
type typedStreamRow struct {
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
// event envelope. ok=false for anything else (command envelopes, foreign
// rows, plain strings) — those rows keep NULL typed columns and never trip
// the partial unique indexes.
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
	}

	if env.SchemaVersion != 0 {
		sv := int64(env.SchemaVersion)
		row.schemaVersion = &sv
	}

	if decoded, err := base64.StdEncoding.DecodeString(string(env.Payload)); err == nil {
		row.payload = decoded
	}

	if decoded, err := base64.StdEncoding.DecodeString(string(env.Metadata)); err == nil {
		row.metadata = decoded
	}

	if t, err := time.Parse(time.RFC3339Nano, env.OccurredAt); err == nil {
		nanos := t.UnixNano()
		row.occurredAt = &nanos
	}

	return row, true
}

// typedStreamColumns is the ordered list of typed columns added to
// meta_stream_log ("name TYPE" pairs).
var typedStreamColumns = []string{
	"event_id TEXT",
	"stream_type TEXT",
	"version INTEGER",
	"schema_version INTEGER",
	"payload BLOB",
	"encoding TEXT",
	"metadata TEXT",
	"occurred_at INTEGER",
}

// ensureTypedStreamLog migrates an existing meta_stream_log to the typed
// columns: guarded ALTER ADD (no table recreate — the journal is never
// rewritten), json_extract backfill for the identity columns, then the two
// partial UNIQUE indexes that make duplicate appends conflict. Idempotent;
// fresh databases already carry the columns via the base DDL. A unique-index
// creation failure means the journal already contains duplicates — a data
// integrity alarm, not a migration bug; the error names the indexes so the
// operator can clean up and re-open.
func ensureTypedStreamLog(db *sql.DB) error {
	existing := map[string]bool{}

	rows, err := db.Query(`PRAGMA table_info(meta_stream_log)`)
	if err != nil {
		return fmt.Errorf("sqliteengine typed WAL: table_info: %w", err)
	}

	defer rows.Close()

	for rows.Next() {
		var cid int

		var name, colType string

		var notNull, pk int

		var dflt sql.NullString

		if err := rows.Scan(&cid, &name, &colType, &notNull, &dflt, &pk); err != nil {
			return fmt.Errorf("sqliteengine typed WAL: scan table_info: %w", err)
		}

		existing[name] = true
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("sqliteengine typed WAL: table_info rows: %w", err)
	}

	for _, col := range typedStreamColumns {
		name := col[:strings.IndexByte(col, ' ')]
		if existing[name] {
			continue
		}

		if _, err := db.Exec(`ALTER TABLE meta_stream_log ADD COLUMN ` + col); err != nil {
			return fmt.Errorf("sqliteengine typed WAL: add column %s: %w", name, err)
		}
	}

	// Identity backfill for rows written before the typed columns existed.
	// payload/metadata/occurred_at stay NULL on legacy rows (no SQLite
	// base64/nano functions); readers fall back to the envelope.
	if _, err := db.Exec(`
		UPDATE meta_stream_log SET
			event_id       = json_extract(value, '$.id'),
			stream_type    = json_extract(value, '$.stream_type'),
			version        = json_extract(value, '$.version'),
			schema_version = json_extract(value, '$.schema_version'),
			encoding       = json_extract(value, '$.encoding')
		WHERE event_id IS NULL
		  AND json_valid(value)
		  AND json_extract(value, '$.id') IS NOT NULL
		  AND json_extract(value, '$.stream_type') IS NOT NULL`); err != nil {
		return fmt.Errorf("sqliteengine typed WAL: backfill: %w", err)
	}

	indexes := []string{
		`CREATE UNIQUE INDEX IF NOT EXISTS uq_stream_log_event
		 ON meta_stream_log(collection, event_id) WHERE event_id IS NOT NULL`,
		`CREATE UNIQUE INDEX IF NOT EXISTS uq_stream_log_stream_version
		 ON meta_stream_log(collection, stream_id, version) WHERE version IS NOT NULL`,
	}

	for _, idx := range indexes {
		if _, err := db.Exec(idx); err != nil {
			return fmt.Errorf(
				"sqliteengine typed WAL: unique index failed — the journal contains "+
					"duplicate event ids or (stream, version) pairs (clean up the duplicates "+
					"and re-open): %w",
				err,
			)
		}
	}

	return nil
}
