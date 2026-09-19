package sqlite

// schema is the full DDL. Fresh databases take it whole; legacy
// databases converge via migrate()'s pragma-probe ALTER path (new-column
// indexes are created only after the column exists — never put a
// new-column index into this const).
const schema = `
CREATE TABLE IF NOT EXISTS tasks (
	id             TEXT PRIMARY KEY,
	project        TEXT NOT NULL DEFAULT '',
	type           TEXT NOT NULL,
	payload        BLOB NOT NULL DEFAULT '',
	deps           TEXT NOT NULL DEFAULT '[]', -- JSON array of task IDs
	priority       INTEGER NOT NULL DEFAULT 0,
	attempts       INTEGER NOT NULL DEFAULT 0,
	max_attempts   INTEGER NOT NULL DEFAULT 3,
	not_before     INTEGER NOT NULL DEFAULT 0, -- unix millis
	status         TEXT NOT NULL DEFAULT 'pending',
	lease_owner    TEXT NOT NULL DEFAULT '',
	lease_expires  INTEGER,                    -- unix millis, NULL when unleased
	lease_token    TEXT,                       -- ADR-0134 claim fencing token, NULL when unclaimed
	last_error     TEXT NOT NULL DEFAULT '',
	created_at     INTEGER NOT NULL,
	updated_at     INTEGER NOT NULL,
	completed_at   INTEGER,
	dedup_key      TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_tasks_status_due ON tasks(status, not_before);
CREATE INDEX IF NOT EXISTS idx_tasks_project ON tasks(project);

CREATE TABLE IF NOT EXISTS deps (
	task_id TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
	dep_id  TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
	PRIMARY KEY (task_id, dep_id)
);
CREATE INDEX IF NOT EXISTS idx_deps_dep ON deps(dep_id);

CREATE TABLE IF NOT EXISTS facts (
	seq      INTEGER PRIMARY KEY AUTOINCREMENT,
	time     INTEGER NOT NULL, -- unix millis
	task_id  TEXT NOT NULL,
	type     TEXT NOT NULL,
	owner    TEXT NOT NULL DEFAULT '',
	attempt  INTEGER NOT NULL DEFAULT 0,
	error    TEXT NOT NULL DEFAULT '',
	detail   BLOB NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_facts_task ON facts(task_id, seq);

CREATE TABLE IF NOT EXISTS watermarks (
	consumer   TEXT PRIMARY KEY, -- journal consumer identity
	seq        INTEGER NOT NULL, -- last checkpointed fact seq
	updated_at INTEGER NOT NULL  -- unix millis
);
`
