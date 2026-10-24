package duckdbengine

import (
	"database/sql"
	"encoding/json/v2"
	"fmt"
	"strings"
	"time"
)

// The typed WAL columns (disk-truth plan A13, mirroring the sqliteengine A11
// reference and the pg/mysql A12 twins): meta_stream_log historically stores
// one opaque JSON envelope per row, so the journal of record has NO
// database-level identity or ordering constraints — a duplicate append is a
// silent second row. The typed columns extract the envelope's identity into
// real columns guarded by UNIQUE indexes:
//
//	uq_stream_log_event          (collection, event_id)
//	uq_stream_log_stream_version (collection, stream_id, version)
//
// DuckDB's ART unique indexes treat NULLs as distinct (verified live against
// the pinned driver on both file and :memory: databases), which reproduces
// the other engines' partial-index semantics exactly — non-envelope rows
// (NULL identity) never trip the constraint — so the planned
// dedup-table fallback is NOT needed: this is the same constraint story,
// DuckDB-dialect. The journal is COPIED into the typed columns, never
// rewritten (ADR-0143): the value envelope stays the read path; legacy rows
// are identity-backfilled from it via json_extract_string. payload/metadata/
// occurred_at are writer-filled and stay NULL on backfilled rows (the uniform
// cross-engine contract; readers fall back to the envelope). metadata is
// VARCHAR holding the JSON text (DuckDB's JSON type is a validated VARCHAR —
// same bytes, no extension dependency).
type typedStreamRow struct { // art-dupl:accept cross-module SQL engine twin (sqliteengine A11 / pgengine A12a / mysqlengine A12b) — dep-isolated go.mod modules
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
// meta_stream_log ("name TYPE" pairs, DuckDB dialect).
var typedStreamColumns = []string{
	"event_id VARCHAR",
	"stream_type VARCHAR",
	"version BIGINT",
	"schema_version BIGINT",
	"payload BLOB",
	"encoding VARCHAR",
	"metadata VARCHAR",
	"occurred_at BIGINT",
}

// ensureTypedStreamLog migrates an existing meta_stream_log to the typed
// columns: ADD COLUMN IF NOT EXISTS (no table recreate — the journal is never
// rewritten), a json_extract_string backfill guarded by json_valid and
// try_cast, then the two UNIQUE indexes. Idempotent; fresh databases already
// carry the columns via the base DDL. A unique-index creation failure means
// the journal already contains duplicates — a data integrity alarm, not a
// migration bug; the error names the indexes so the operator can clean up
// and re-open.
func ensureTypedStreamLog(db *sql.DB) error {
	for _, col := range typedStreamColumns {
		name := col[:strings.IndexByte(col, ' ')]
		if _, err := db.Exec(
			`ALTER TABLE meta_stream_log ADD COLUMN IF NOT EXISTS ` + col,
		); err != nil {
			return fmt.Errorf("duckdbengine typed WAL: add column %s: %w", name, err)
		}
	}

	if err := backfillTypedStreamColumns(db); err != nil {
		return err
	}

	indexes := []string{
		`CREATE UNIQUE INDEX IF NOT EXISTS uq_stream_log_event
		 ON meta_stream_log (collection, event_id)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS uq_stream_log_stream_version
		 ON meta_stream_log (collection, stream_id, version)`,
	}

	for _, idx := range indexes {
		if _, err := db.Exec(idx); err != nil {
			return fmtErr(
				"unique index failed — the journal contains duplicate event ids or "+
					"(stream, version) pairs (clean up the duplicates and re-open)",
				err,
			)
		}
	}

	return nil
}

// backfillTypedStreamColumns fills the identity columns for rows written
// before the typed columns existed — the same contract as the other engines.
// json_valid keeps non-JSON values outside the extraction; try_cast keeps
// non-numeric envelope fields outside the BIGINT casts (NULL on cast failure,
// no error, no regexp).
func backfillTypedStreamColumns(db *sql.DB) error {
	const stmt = `
		UPDATE meta_stream_log SET
			event_id       = json_extract_string(value, '$.id'),
			stream_type    = json_extract_string(value, '$.stream_type'),
			version        = try_cast(json_extract_string(value, '$.version') AS BIGINT),
			schema_version = try_cast(json_extract_string(value, '$.schema_version') AS BIGINT),
			encoding       = json_extract_string(value, '$.encoding')
		WHERE event_id IS NULL
		  AND json_valid(value)
		  AND json_extract_string(value, '$.id') IS NOT NULL
		  AND json_extract_string(value, '$.stream_type') IS NOT NULL`

	if _, err := db.Exec(stmt); err != nil {
		return fmtErr(
			"backfill — if this is a constraint violation, the journal already "+
				"contains duplicate event ids or (stream, version) pairs (clean up "+
				"the duplicates and re-open)",
			err,
		)
	}

	return nil
}

// isUniqueViolation reports whether err is DuckDB's unique-constraint
// rejection ("Constraint Error: Duplicate key ... violates unique
// constraint.").
func isUniqueViolation(err error) bool {
	return err != nil && strings.Contains(err.Error(), "violates unique constraint")
}

func fmtErr(what string, err error) error {
	return fmt.Errorf("duckdbengine typed WAL: %s: %w", what, err)
}
