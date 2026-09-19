package mysql

// schemaStmts is the full DDL, idempotent, one statement per element
// (database/sql executes single statements; MySQL connections do not
// accept multi-statement scripts without the multiStatements DSN flag,
// which this engine deliberately does not require). It mirrors the
// SQLite/Postgres engines' shapes with MySQL types: BIGINT unix-milli
// timestamps, AUTO_INCREMENT fact seqs. dedup_key is NULLABLE with a
// plain UNIQUE KEY — MySQL treats NULLs as distinct, so keyless tasks
// never collide (the partial-index semantics the other engines express
// with WHERE).
//
//nolint:gochecknoglobals // DDL is immutable compile-time package data
var schemaStmts = []string{
	`CREATE TABLE IF NOT EXISTS tasks (
	id            VARCHAR(64) PRIMARY KEY,
	project       VARCHAR(255) NOT NULL DEFAULT '',
	type          VARCHAR(255) NOT NULL,
	payload       LONGBLOB NOT NULL,
	deps          LONGTEXT NOT NULL,
	priority      BIGINT NOT NULL DEFAULT 0,
	attempts      INT NOT NULL DEFAULT 0,
	max_attempts  INT NOT NULL DEFAULT 3,
	not_before    BIGINT NOT NULL DEFAULT 0,
	status        VARCHAR(32) NOT NULL DEFAULT 'pending',
	lease_owner   VARCHAR(255) NOT NULL DEFAULT '',
	lease_expires BIGINT NULL,
	lease_token   VARCHAR(64) NULL,
	last_error    LONGTEXT NOT NULL,
	created_at    BIGINT NOT NULL,
	updated_at    BIGINT NOT NULL,
	completed_at  BIGINT NULL,
	dedup_key     VARCHAR(255) NULL DEFAULT NULL,
	UNIQUE KEY idx_tasks_dedup (dedup_key),
	KEY idx_tasks_status_due (status, not_before),
	KEY idx_tasks_project (project)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
	`CREATE TABLE IF NOT EXISTS deps (
	task_id VARCHAR(64) NOT NULL,
	dep_id  VARCHAR(64) NOT NULL,
	PRIMARY KEY (task_id, dep_id),
	KEY idx_deps_dep (dep_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
	`CREATE TABLE IF NOT EXISTS facts (
	seq      BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
	time     BIGINT NOT NULL,
	task_id  VARCHAR(64) NOT NULL,
	type     VARCHAR(64) NOT NULL,
	owner    VARCHAR(255) NOT NULL DEFAULT '',
	attempt  INT NOT NULL DEFAULT 0,
	error    LONGTEXT NOT NULL,
	detail   LONGBLOB NOT NULL,
	KEY idx_facts_task (task_id, seq)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
	`CREATE TABLE IF NOT EXISTS watermarks (
	consumer   VARCHAR(255) PRIMARY KEY,
	seq        BIGINT NOT NULL,
	updated_at BIGINT NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
}
