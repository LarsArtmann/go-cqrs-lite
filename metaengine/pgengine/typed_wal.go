package pgengine

import (
	"database/sql"
	"encoding/json/v2"
	"errors"
	"fmt"
	"strings"
	"time"
)

// The typed WAL columns (disk-truth plan A12, mirroring the sqliteengine A11
// reference): meta_stream_log historically stores one opaque JSON envelope
// per row, so the journal of record has NO database-level identity or
// ordering constraints — a duplicate append is a silent second row. The typed
// columns extract the envelope's identity into real columns guarded by
// partial UNIQUE indexes:
//
//	uq_stream_log_event          (collection, event_id)
//	uq_stream_log_stream_version (collection, stream_id, version)
//
// The journal is COPIED into the typed columns, never rewritten (ADR-0143):
// the value envelope stays the read path; legacy rows are backfilled from it
// via jsonb accessors. Identity columns are backfilled server-side;
// payload/metadata/occurred_at are writer-filled (NULL on legacy rows,
// readers fall back to the envelope — one uniform contract across the SQL
// engines, and a malformed foreign row can never abort the migration).
// metadata lands as a real JSONB column — the plan's "JSONB metadata
// variant"; occurred_at is unix-nanos BIGINT (RFC3339Nano precision would
// truncate through timestamptz microseconds).
type typedStreamRow struct { // art-dupl:accept cross-module SQL engine twin (sqliteengine A11) — dep-isolated go.mod modules
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
// those rows keep NULL typed columns and never trip the partial unique
// indexes.
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
// meta_stream_log ("name TYPE" pairs, Postgres dialect).
var typedStreamColumns = []string{
	"event_id TEXT",
	"stream_type TEXT",
	"version BIGINT",
	"schema_version BIGINT",
	"payload BYTEA",
	"encoding TEXT",
	"metadata JSONB",
	"occurred_at BIGINT",
}

// ensureTypedStreamLog migrates an existing meta_stream_log to the typed
// columns: ADD COLUMN IF NOT EXISTS (no table recreate — the journal is never
// rewritten), jsonb backfill, then the two partial UNIQUE indexes that make
// duplicate appends conflict. Idempotent; fresh databases already carry the
// columns via the base DDL. A unique-index creation failure means the journal
// already contains duplicates — a data integrity alarm, not a migration bug;
// the error names the indexes so the operator can clean up and re-open.
func ensureTypedStreamLog(db *sql.DB) error {
	for _, col := range typedStreamColumns {
		name := col[:strings.IndexByte(col, ' ')]
		if _, err := db.Exec(
			`ALTER TABLE meta_stream_log ADD COLUMN IF NOT EXISTS ` + col,
		); err != nil {
			return fmt.Errorf("pgengine typed WAL: add column %s: %w", name, err)
		}
	}

	if err := backfillTypedStreamColumns(db); err != nil {
		return err
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
				"pgengine typed WAL: unique index failed — the journal contains "+
					"duplicate event ids or (stream, version) pairs (clean up the duplicates "+
					"and re-open): %w",
				err,
			)
		}
	}

	return nil
}

// backfillTypedStreamColumns fills the IDENTITY columns for rows written
// before the typed columns existed — the same contract as the sqliteengine
// reference. pg_input_is_valid (Postgres 16+, the version floor of both test
// surfaces) keeps invalid-JSON values outside the cast; numeric-text regexp
// guards keep non-numeric envelope fields outside the bigint casts.
// payload/metadata/occurred_at deliberately stay NULL on backfilled rows: a
// malformed foreign value must never abort the migration (the decode/jsonb
// casts cannot be guarded in one statement), and readers already fall back
// to the envelope; the writer fills them for every new row.
func backfillTypedStreamColumns(db *sql.DB) error {
	const stmt = `
		UPDATE meta_stream_log SET
			event_id       = value::jsonb->>'id',
			stream_type    = value::jsonb->>'stream_type',
			version        = CASE WHEN value::jsonb->>'version' ~ '^[0-9]+$'
			                      THEN (value::jsonb->>'version')::bigint END,
			schema_version = CASE WHEN value::jsonb->>'schema_version' ~ '^[0-9]+$'
			                      THEN (value::jsonb->>'schema_version')::bigint END,
			encoding       = value::jsonb->>'encoding'
		WHERE event_id IS NULL
		  AND pg_input_is_valid(value, 'jsonb')
		  AND jsonb_typeof(value::jsonb) = 'object'
		  AND value::jsonb->>'id' IS NOT NULL
		  AND value::jsonb->>'stream_type' IS NOT NULL`

	if _, err := db.Exec(stmt); err != nil {
		return fmt.Errorf(
			"pgengine typed WAL: backfill: %w — if this is a unique violation, the "+
				"journal already contains duplicate event ids or (stream, version) pairs "+
				"(clean up the duplicates and re-open)",
			err,
		)
	}

	return nil
}

// isUniqueViolation reports whether err is a Postgres unique-constraint
// violation (SQLSTATE 23505), via the driver's structured error when
// available (pgx, lib/pq) and the message text otherwise.
func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}

	var st interface{ SQLState() string }
	if errors.As(err, &st) {
		return st.SQLState() == "23505"
	}

	msg := err.Error()

	return strings.Contains(msg, "duplicate key value violates unique constraint") ||
		strings.Contains(msg, "SQLSTATE 23505")
}
