package pgengine

import (
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// Option configures a pgEngine at construction.
type Option func(*pgEngine)

// WithCopyAppend routes bulk StreamAppend calls (len(values) >= minValues)
// through Postgres COPY FROM instead of chunked multi-VALUES INSERTs. COPY
// streams rows in the native binary-ish text protocol with no bound-parameter
// overhead — typically 2-5x faster for large backfills (see
// BenchmarkStreamAppend_* in copy_bench_test.go).
//
// Off by default. When a StreamAppend participates in an active RunInTx (the
// COPY protocol cannot join a database/sql transaction on another pooled
// connection), or the driver is not pgx, it silently falls back to the INSERT
// path — enabling the option never changes correctness, only speed.
func WithCopyAppend(minValues int) Option {
	return func(e *pgEngine) {
		if minValues < 1 {
			minValues = 1
		}
		e.copyMin = minValues
	}
}

// errCopyUnavailable signals the COPY fast path cannot serve this call and the
// INSERT path should take over.
var errCopyUnavailable = errors.New("pgengine: copy fast path unavailable")

// copyAppend bulk-inserts values via pgx COPY FROM on a dedicated pooled
// connection. A single COPY statement is atomic — equivalent to the
// single-transaction INSERT path it replaces.
func (e *pgEngine) copyAppend(ctx context.Context, col, sid string, values []any) error {
	conn, err := e.db.Conn(ctx)
	if err != nil {
		return fmt.Errorf("pgengine.copyAppend: acquire conn: %w", err)
	}
	defer metaengine.DeferClose(conn)

	err = conn.Raw(func(driverConn any) error {
		// *stdlib.Conn exposes the underlying *pgx.Conn; anything else
		// (registered under a different driver) falls back to INSERTs.
		native, ok := driverConn.(interface{ Conn() *pgx.Conn })
		if !ok {
			return errCopyUnavailable
		}

		encodedValues, encodeErr := metaengine.EncodeStreamValuesErr(values)
		if encodeErr != nil {
			return fmt.Errorf("pgengine.copyAppend: %w", encodeErr)
		}

		rows := make([][]any, len(encodedValues))
		for i, encoded := range encodedValues {
			rows[i] = typedCopyRow(col, sid, encoded)
		}

		copied, err := native.Conn().CopyFrom(
			ctx,
			pgx.Identifier{"meta_stream_log"},
			[]string{
				"collection", "stream_id", "value",
				"event_id", "stream_type", "version", "schema_version",
				"payload", "encoding", "metadata", "occurred_at",
			},
			pgx.CopyFromRows(rows),
		)
		if err != nil {
			if isUniqueViolation(err) {
				return fmt.Errorf("pgengine.copyAppend: %w", metaengine.ErrVersionConflict)
			}

			return fmt.Errorf("pgengine.copyAppend: %w", err)
		}

		if copied != int64(len(values)) {
			return fmt.Errorf("pgengine.copyAppend: copied %d of %d rows", copied, len(values))
		}

		return nil
	})
	if errors.Is(err, errCopyUnavailable) {
		return err
	}

	return err
}

// streamAppendRowsPerStmt caps rows per multi-VALUES INSERT statement.
// 11 bound params per row x 5000 rows = 55000 params, below Postgres'
// 65535-parameter limit.
const streamAppendRowsPerStmt = 5000

// typedCopyRow builds the 11-column tuple for one stream-log row: the three
// historical columns plus the typed WAL columns extracted from the event
// envelope (A12). metadata lands as a JSON string for the JSONB column and
// stays NULL when the decoded bytes are not valid JSON (foreign rows).
func typedCopyRow(col, sid, encoded string) []any {
	typed, _ := extractTypedStreamRow(encoded)

	var metadata any
	if len(typed.metadata) > 0 && json.Unmarshal(typed.metadata, new(any)) == nil {
		metadata = string(typed.metadata)
	}

	return []any{
		col, sid, encoded,
		typed.eventID, typed.streamType, typed.version, typed.schemaVersion,
		typed.payload, typed.encoding, metadata, typed.occurredAt,
	}
}

// streamInsertBatch inserts values as chunked multi-VALUES statements
// (one round trip per 5k rows instead of one per row) on conn. A unique
// violation on the typed WAL identity indexes is a duplicate append —
// surfaced as metaengine.ErrVersionConflict so the adapter layer maps it
// onto the optimistic-concurrency contract.
func streamInsertBatch(
	ctx context.Context,
	conn metaengine.SQLExec,
	col, sid string,
	values []any,
) error {
	encodedValues, encodeErr := metaengine.EncodeStreamValuesErr(values)
	if encodeErr != nil {
		return fmt.Errorf("pgengine.streamInsertBatch: %w", encodeErr)
	}

	for start := 0; start < len(values); start += streamAppendRowsPerStmt {
		end := min(start+streamAppendRowsPerStmt, len(values))
		chunk := values[start:end]

		var sb strings.Builder
		args := make([]any, 0, len(chunk)*11)

		sb.WriteString(`INSERT INTO meta_stream_log
			(collection, stream_id, value, event_id, stream_type, version, schema_version,
			 payload, encoding, metadata, occurred_at) VALUES `)

		for i := range chunk {
			if i > 0 {
				sb.WriteByte(',')
			}

			base := i * 11
			fmt.Fprintf(&sb, "($%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d)",
				base+1, base+2, base+3, base+4, base+5, base+6,
				base+7, base+8, base+9, base+10, base+11)

			args = append(args, typedCopyRow(col, sid, encodedValues[start+i])...)
		}

		if _, err := conn.ExecContext(ctx, sb.String(), args...); err != nil {
			if isUniqueViolation(err) {
				return fmt.Errorf("pgengine.streamInsertBatch: %w", metaengine.ErrVersionConflict)
			}

			return fmt.Errorf("pgengine.streamInsertBatch: %w", err)
		}
	}

	return nil
}
