package sqlstore

import (
	"strings"
	"testing"
)

// TestMySQLQueries_SQLSyntax verifies the MySQL-specific SQL strings use
// correct MySQL syntax (ON DUPLICATE KEY UPDATE, IF() conditional, backtick
// quoting) without needing a live MySQL connection.
func TestMySQLQueries_SQLSyntax(t *testing.T) {
	t.Parallel()

	q := mysqlQueries()

	t.Run("ddl uses backtick quoting for reserved word key", func(t *testing.T) {
		t.Parallel()

		if !strings.Contains(q.ddl, "`key`") {
			t.Errorf("MySQL DDL should backtick-quote the reserved word 'key':\n%s", q.ddl)
		}
	})

	t.Run("ddl creates idempotency_keys table", func(t *testing.T) {
		t.Parallel()

		if !strings.Contains(q.ddl, "CREATE TABLE IF NOT EXISTS idempotency_keys") {
			t.Errorf("MySQL DDL should create idempotency_keys table:\n%s", q.ddl)
		}
	})

	t.Run("record uses ON DUPLICATE KEY UPDATE no-op", func(t *testing.T) {
		t.Parallel()

		if !strings.Contains(q.record, "ON DUPLICATE KEY UPDATE") {
			t.Errorf("MySQL record should use ON DUPLICATE KEY UPDATE:\n%s", q.record)
		}

		if !strings.Contains(q.record, "`key` = `key`") {
			t.Errorf(
				"MySQL record should use self-assignment no-op (key = key) for ON CONFLICT DO NOTHING equivalent:\n%s",
				q.record,
			)
		}
	})

	t.Run("checkAndRecord uses IF conditional update", func(t *testing.T) {
		t.Parallel()

		if !strings.Contains(q.checkAndRecord, "ON DUPLICATE KEY UPDATE") {
			t.Errorf(
				"MySQL checkAndRecord should use ON DUPLICATE KEY UPDATE:\n%s",
				q.checkAndRecord,
			)
		}

		if !strings.Contains(q.checkAndRecord, "IF(") {
			t.Errorf(
				"MySQL checkAndRecord should use IF() for conditional update (no WHERE in ON DUPLICATE KEY):\n%s",
				q.checkAndRecord,
			)
		}

		if !strings.Contains(q.checkAndRecord, "VALUES(expires_at)") {
			t.Errorf(
				"MySQL checkAndRecord should reference VALUES(expires_at) for the new value:\n%s",
				q.checkAndRecord,
			)
		}
	})

	t.Run("checkAndRecord has three placeholders", func(t *testing.T) {
		t.Parallel()

		count := strings.Count(q.checkAndRecord, "?")
		if count != 3 {
			t.Errorf(
				"MySQL checkAndRecord should have 3 placeholders (key, expiry, now), got %d:\n%s",
				count,
				q.checkAndRecord,
			)
		}
	})

	t.Run("seen uses backtick quoting", func(t *testing.T) {
		t.Parallel()

		if !strings.Contains(q.seen, "`key` = ?") {
			t.Errorf("MySQL seen should backtick-quote 'key':\n%s", q.seen)
		}
	})

	t.Run("all queries use question mark placeholders", func(t *testing.T) {
		t.Parallel()

		for name, sql := range map[string]string{
			"seen":           q.seen,
			"deleteKey":      q.deleteKey,
			"record":         q.record,
			"checkAndRecord": q.checkAndRecord,
			"sweep":          q.sweep,
		} {
			if strings.Contains(sql, "$1") {
				t.Errorf("%s should use ? placeholders (MySQL), not $1:\n%s", name, sql)
			}
		}
	})
}
