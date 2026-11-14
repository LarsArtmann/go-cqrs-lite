package sqliteengine

// sqliteQuerySet holds pre-built SQL strings per operation; two variants
// handle placeholder differences (? vs $N).
type sqliteQuerySet struct {
	// Map
	mapSet    string
	mapGet    string
	mapDelete string
	// Set
	setAdd      string
	setContains string
	// Counter
	counterIncrement string
	counterGet       string
	// Multimap
	multiAdd string
	multiGet string
	// Log
	logAppend string
	logTail   string
	// Stream Log
	streamAppend        string
	streamRead          string
	streamVersion       string
	streamAppendExp     string
	journalReadAll      string
	journalReadFrom     string
	streamLoadByEventID string
	// Seq-resumable journal reads (SeqSeekableStreamLog)
	journalReadAllWithSeq string
	journalReadFromSeq    string
	journalReadFromSeqAll string
	// Graph (iterative BFS on meta_graph_edges)
	graphAddEdge string
	// DDL
	ddl string
}

func defaultSQLiteQueries() sqliteQuerySet {
	return sqliteQuerySet{
		ddl: `CREATE TABLE IF NOT EXISTS meta_map (
		collection TEXT NOT NULL, key TEXT NOT NULL, value TEXT NOT NULL,
		PRIMARY KEY (collection, key)
	);
	CREATE TABLE IF NOT EXISTS meta_set (
		collection TEXT NOT NULL, key TEXT NOT NULL,
		PRIMARY KEY (collection, key)
	);
	CREATE TABLE IF NOT EXISTS meta_counter (
		collection TEXT NOT NULL, key TEXT NOT NULL, value INTEGER NOT NULL DEFAULT 0,
		PRIMARY KEY (collection, key)
	);
	CREATE TABLE IF NOT EXISTS meta_multimap (
		collection TEXT NOT NULL, key TEXT NOT NULL, seq INTEGER NOT NULL, value TEXT NOT NULL,
		PRIMARY KEY (collection, key, seq)
	);
	CREATE TABLE IF NOT EXISTS meta_log (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		collection TEXT NOT NULL, value TEXT NOT NULL
	);
	CREATE TABLE IF NOT EXISTS meta_stream_log (
		seq INTEGER PRIMARY KEY AUTOINCREMENT,
		collection TEXT NOT NULL,
		stream_id TEXT NOT NULL,
		value TEXT NOT NULL,
		event_id TEXT,
		stream_type TEXT,
		version INTEGER,
		schema_version INTEGER,
		payload BLOB,
		encoding TEXT,
		metadata TEXT,
		occurred_at INTEGER
	);
	CREATE INDEX IF NOT EXISTS idx_stream_log_stream ON meta_stream_log(collection, stream_id, seq);
	CREATE INDEX IF NOT EXISTS idx_stream_log_journal ON meta_stream_log(collection, seq);
	CREATE INDEX IF NOT EXISTS idx_stream_log_event_id ON meta_stream_log(collection, json_extract(value, '$.id')) WHERE json_valid(value);
	CREATE TABLE IF NOT EXISTS meta_graph_edges (
		collection TEXT NOT NULL, from_node TEXT NOT NULL, to_node TEXT NOT NULL,
		PRIMARY KEY (collection, from_node, to_node)
	);
	CREATE INDEX IF NOT EXISTS idx_graph_edges_from ON meta_graph_edges(collection, from_node);
	CREATE INDEX IF NOT EXISTS idx_graph_edges_to ON meta_graph_edges(collection, to_node);
	CREATE TABLE IF NOT EXISTS meta_snapshot (
		collection TEXT NOT NULL, stream_id TEXT NOT NULL, version INTEGER NOT NULL, data BLOB NOT NULL,
		PRIMARY KEY (collection, stream_id)
	);` + vectorTableDDL + cellVersionsDDL,
		mapSet:           `INSERT OR REPLACE INTO meta_map (collection, key, value) VALUES (?, ?, ?)`,
		mapGet:           `SELECT value FROM meta_map WHERE collection = ? AND key = ?`,
		mapDelete:        `DELETE FROM meta_map WHERE collection = ? AND key = ?`,
		setAdd:           `INSERT OR IGNORE INTO meta_set (collection, key) VALUES (?, ?)`,
		setContains:      `SELECT 1 FROM meta_set WHERE collection = ? AND key = ?`,
		counterIncrement: `INSERT INTO meta_counter (collection, key, value) VALUES (?, ?, ?) ON CONFLICT(collection, key) DO UPDATE SET value = value + excluded.value`,
		counterGet:       `SELECT key, value FROM meta_counter WHERE collection = ?`,
		multiAdd:         `INSERT INTO meta_multimap (collection, key, seq, value) VALUES (?, ?, ?, ?)`,
		multiGet:         `SELECT value FROM meta_multimap WHERE collection = ? AND key = ? ORDER BY seq`,
		logAppend:        `INSERT INTO meta_log (collection, value) VALUES (?, ?)`,
		logTail:          `SELECT value FROM meta_log WHERE collection = ? ORDER BY id DESC LIMIT ?`,
		streamAppend: `INSERT INTO meta_stream_log
			(collection, stream_id, value, event_id, stream_type, version, schema_version, payload, encoding, metadata, occurred_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		streamRead:    `SELECT value FROM meta_stream_log WHERE collection = ? AND stream_id = ? ORDER BY seq`,
		streamVersion: `SELECT COUNT(*) FROM meta_stream_log WHERE collection = ? AND stream_id = ?`,
		streamAppendExp: `INSERT INTO meta_stream_log
			(collection, stream_id, value, event_id, stream_type, version, schema_version, payload, encoding, metadata, occurred_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		journalReadAll: `SELECT value FROM meta_stream_log WHERE collection = ? ORDER BY seq`,
		// afterSeq is a journal position within the collection (seq is a
		// global AUTOINCREMENT shared across collections — see JournalReadFrom).
		journalReadFrom: `SELECT value FROM meta_stream_log WHERE collection = ? ORDER BY seq LIMIT ? OFFSET ?`,
		streamLoadByEventID: `SELECT value FROM (
			SELECT seq, value FROM meta_stream_log WHERE collection = ? AND event_id = ?
			UNION ALL
			SELECT seq, value FROM meta_stream_log WHERE collection = ? AND event_id IS NULL AND json_valid(value) AND json_extract(value, '$.id') = ?
		) ORDER BY seq LIMIT 1`,
		journalReadAllWithSeq: `SELECT seq, value FROM meta_stream_log WHERE collection = ? ORDER BY seq`,
		// Token resumption: a pure index range seek on
		// idx_stream_log_journal(collection, seq) — O(log n) per page, and
		// gaps in the shared counter cannot shift the cursor.
		journalReadFromSeq:    `SELECT seq, value FROM meta_stream_log WHERE collection = ? AND seq > ? ORDER BY seq LIMIT ?`,
		journalReadFromSeqAll: `SELECT seq, value FROM meta_stream_log WHERE collection = ? AND seq > ? ORDER BY seq LIMIT -1`,
		// Graph (recursive CTE)
		graphAddEdge: `INSERT OR IGNORE INTO meta_graph_edges (collection, from_node, to_node) VALUES (?, ?, ?)`,
	}
}
