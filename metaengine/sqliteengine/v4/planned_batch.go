package sqliteengine

// Buffered multi-row planned-table writes inside an ambient transaction.
//
// Contract (dnsblockd metaflood MF1.3, design memo 2026-11-14): Sets on a
// planned collection that arrive while ctx descends from this engine's
// RunInTx buffer per table instead of executing immediately. Buffers flush
//
//   - at RunInTx commit, before COMMIT reaches SQLite,
//   - when a table's buffer reaches the chunk cap, and
//   - BEFORE any engine operation that reads or deletes from that table,
//
// preserving sequential semantics (read-your-writes, delete-after-set
// ordering). Rollback discards buffers without executing. Rows within one
// flushed statement preserve batch order; duplicate keys in one batch
// resolve last-write-wins (SQLite processes multi-row VALUES left to right),
// identical to sequential single-row INSERT OR REPLACE execution.

import (
	"context"
	"fmt"
	"strings"

	metaengine "github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

// plannedBatchChunk caps rows per multi-row INSERT statement. It matches the
// dnsblockd BatchWriter flush window (500 commands / 1s) and keeps parameter
// counts at chunk x column-count, far below SQLite's parameter ceiling for
// any plausible planned column count (<= 60 columns stays under 32766).
const plannedBatchChunk = 500

// plannedTableBuffer accumulates ordered rows for one planned table.
type plannedTableBuffer struct {
	table   string
	columns []string // quoted column names including the key/value prefix
	rows    [][]any  // one arg slice per row, len == len(columns)
}

// plannedBuffers holds per-table row buffers for one ambient transaction. It
// is confined to the txExecutor and therefore to the RunInTx call tree — the
// tx protocol already serializes writers through txMu, so no extra lock.
type plannedBuffers struct {
	byTable map[string]*plannedTableBuffer
}

func newPlannedBuffers() *plannedBuffers {
	return &plannedBuffers{byTable: make(map[string]*plannedTableBuffer)}
}

// add appends one row. columns must be identical across rows of a table;
// the slice is retained per table, the row slice is retained as-is.
func (p *plannedBuffers) add(table string, quotedColumns []string, row []any) {
	b, ok := p.byTable[table]
	if !ok {
		b = &plannedTableBuffer{table: table, columns: quotedColumns}
		p.byTable[table] = b
	}

	b.rows = append(b.rows, row)
}

// len returns the buffered row count for one table (0 when unbuffered).
func (p *plannedBuffers) len(table string) int {
	if b, ok := p.byTable[table]; ok {
		return len(b.rows)
	}

	return 0
}

// flushTable emits buffered rows for one table as chunked multi-row INSERT
// OR REPLACE statements and clears the buffer.
func (p *plannedBuffers) flushTable(
	ctx context.Context,
	exec metaengine.SQLExec,
	table string,
) error {
	b, ok := p.byTable[table]
	if !ok {
		return nil
	}

	delete(p.byTable, table)

	return emitPlannedBuffer(ctx, exec, b)
}

// flushAll emits every buffered table. All buffers are detached first so a
// mid-flush error cannot leave a half-flushed table registered (the ambient
// transaction rolls everything back anyway).
func (p *plannedBuffers) flushAll(ctx context.Context, exec metaengine.SQLExec) error {
	tables := make([]string, 0, len(p.byTable))
	buffers := make([]*plannedTableBuffer, 0, len(p.byTable))

	for table, b := range p.byTable {
		tables = append(tables, table)
		buffers = append(buffers, b)
	}

	p.byTable = make(map[string]*plannedTableBuffer)

	var firstErr error

	for i, b := range buffers {
		if err := emitPlannedBuffer(ctx, exec, b); err != nil && firstErr == nil {
			firstErr = fmt.Errorf("table %s: %w", tables[i], err)
		}
	}

	return firstErr
}

// discard drops every buffered row without executing (rollback path).
func (p *plannedBuffers) discard() {
	p.byTable = make(map[string]*plannedTableBuffer)
}

// emitPlannedBuffer writes one table's buffered rows as multi-row INSERT OR
// REPLACE statements, chunked at plannedBatchChunk. The statement is executed
// directly on the transaction executor (never the stmt cache): the row count
// varies per chunk, and caching ragged statement texts would churn the cache.
func emitPlannedBuffer(ctx context.Context, exec metaengine.SQLExec, b *plannedTableBuffer) error {
	if len(b.rows) == 0 {
		return nil
	}

	cols := len(b.columns)
	oneRow := "(" + strings.TrimSuffix(strings.Repeat("?,", cols), ",") + ")"

	for start := 0; start < len(b.rows); start += plannedBatchChunk {
		end := min(start+plannedBatchChunk, len(b.rows))
		chunk := b.rows[start:end]

		var sb strings.Builder

		sb.WriteString("INSERT OR REPLACE INTO ")
		sb.WriteString(metaengine.QuoteIdent(b.table))
		sb.WriteString(" (")
		sb.WriteString(strings.Join(b.columns, ", "))
		sb.WriteString(") VALUES ")
		sb.WriteString(oneRow)

		args := make([]any, 0, cols*len(chunk))
		args = append(args, chunk[0]...)

		for i := 1; i < len(chunk); i++ {
			sb.WriteString(", ")
			sb.WriteString(oneRow)
			args = append(args, chunk[i]...)
		}

		if _, err := exec.ExecContext(ctx, sb.String(), args...); err != nil {
			return fmt.Errorf(
				"sqliteengine: planned batch insert %s (%d rows): %w",
				b.table,
				len(chunk),
				err,
			)
		}
	}

	return nil
}
