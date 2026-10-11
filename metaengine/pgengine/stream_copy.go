package pgengine

import (
	"context"
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

		rows := make([][]any, len(values))
		for i, v := range values {
			rows[i] = []any{col, sid, metaengine.EncodeStreamValue(v)}
		}

		copied, err := native.Conn().CopyFrom(
			ctx,
			pgx.Identifier{"meta_stream_log"},
			[]string{"collection", "stream_id", "value"},
			pgx.CopyFromRows(rows),
		)
		if err != nil {
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
// 3 bound params per row x 10000 rows = 30000 params, comfortably below
// Postgres' 65535-parameter limit.
const streamAppendRowsPerStmt = 10000

// streamInsertBatch inserts values as chunked multi-VALUES statements
// (one round trip per 10k rows instead of one per row) on conn.
func streamInsertBatch(
	ctx context.Context,
	conn metaengine.SQLExec,
	col, sid string,
	values []any,
) error {
	for start := 0; start < len(values); start += streamAppendRowsPerStmt {
		end := min(start+streamAppendRowsPerStmt, len(values))
		chunk := values[start:end]

		var sb strings.Builder
		args := make([]any, 0, len(chunk)*3)

		sb.WriteString(`INSERT INTO meta_stream_log (collection, stream_id, value) VALUES `)

		for i, v := range chunk {
			if i > 0 {
				sb.WriteByte(',')
			}

			base := i * 3
			fmt.Fprintf(&sb, "($%d,$%d,$%d)", base+1, base+2, base+3)
			args = append(args, col, sid, metaengine.EncodeStreamValue(v))
		}

		if _, err := conn.ExecContext(ctx, sb.String(), args...); err != nil {
			return fmt.Errorf("pgengine.streamInsertBatch: %w", err)
		}
	}

	return nil
}
