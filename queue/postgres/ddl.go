package postgres

// schema is the full DDL, idempotent (CREATE TABLE/INDEX IF NOT EXISTS).
// It mirrors the SQLite engine's shapes with PG types: BIGINT
// unix-milli timestamps, BIGSERIAL fact seqs.
const schema = `
CREATE TABLE IF NOT EXISTS tasks (
	id            TEXT PRIMARY KEY,
	project       TEXT NOT NULL DEFAULT '',
	type          TEXT NOT NULL,
	payload       TEXT NOT NULL DEFAULT '',
	deps          TEXT NOT NULL DEFAULT '[]',
	priority      INTEGER NOT NULL DEFAULT 0,
	attempts      INTEGER NOT NULL DEFAULT 0,
	max_attempts  INTEGER NOT NULL DEFAULT 3,
	not_before    BIGINT NOT NULL DEFAULT 0,
	status        TEXT NOT NULL DEFAULT 'pending',
	lease_owner   TEXT NOT NULL DEFAULT '',
	lease_expires BIGINT,
	last_error    TEXT NOT NULL DEFAULT '',
	created_at    BIGINT NOT NULL,
	updated_at    BIGINT NOT NULL,
	completed_at  BIGINT,
	dedup_key     TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_tasks_status_due ON tasks(status, not_before);
CREATE INDEX IF NOT EXISTS idx_tasks_project ON tasks(project);
CREATE UNIQUE INDEX IF NOT EXISTS idx_tasks_dedup ON tasks(dedup_key) WHERE dedup_key != '';

CREATE TABLE IF NOT EXISTS deps (
	task_id TEXT NOT NULL,
	dep_id  TEXT NOT NULL,
	PRIMARY KEY (task_id, dep_id)
);
CREATE INDEX IF NOT EXISTS idx_deps_dep ON deps(dep_id);

CREATE TABLE IF NOT EXISTS facts (
	seq      BIGSERIAL PRIMARY KEY,
	time     BIGINT NOT NULL,
	task_id  TEXT NOT NULL,
	type     TEXT NOT NULL,
	owner    TEXT NOT NULL DEFAULT '',
	attempt  INTEGER NOT NULL DEFAULT 0,
	error    TEXT NOT NULL DEFAULT '',
	detail   TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_facts_task ON facts(task_id, seq);

CREATE TABLE IF NOT EXISTS watermarks (
	consumer   TEXT PRIMARY KEY,
	seq        BIGINT NOT NULL,
	updated_at BIGINT NOT NULL
);
`
