package sql

// MaxStatementBytes bounds the estimated serialized byte size of a single
// multi-VALUES statement. It is half of MariaDB's default max_allowed_packet
// (16 MiB): a statement above the server limit fails the entire batch write
// with a packet error, so 50% leaves headroom for driver framing while keeping
// batch writes working on default MariaDB and MySQL deployments. Dialects
// without a packet limit (SQLite, DuckDB, PostgreSQL's 1 GiB protocol ceiling)
// still benefit from bounded statement memory.
const MaxStatementBytes = 8 << 20

// bytesPerEventOverhead bounds the fixed per-row statement cost beyond payload
// and metadata: event ID, type and stream names, version, encoding, timestamp,
// placeholder syntax, and driver protocol framing.
const bytesPerEventOverhead = 256

// RowsWithinByteCap returns how many rows from [start, n), capped at maxRows,
// fit into a single statement without the cumulative estimated row sizes
// exceeding MaxStatementBytes. rowBytes receives the absolute row index.
//
// At least one row is always returned: a single row whose estimate already
// exceeds the cap still inserts as a one-row statement, and callers must size
// their server's packet limit (e.g. MariaDB max_allowed_packet) to accept
// their largest single event.
func RowsWithinByteCap(start, n, maxRows int, rowBytes func(i int) int) int {
	maxRows = max(maxRows, 1)

	end := min(start+maxRows, n)
	total := 0

	for i := start; i < end; i++ {
		total += rowBytes(i)
		if i > start && total > MaxStatementBytes {
			return i - start
		}
	}

	return max(end-start, 0)
}
