package sqlstore

import (
	"strings"
	"testing"
)

// TestMySQLQueries_SQLSyntax verifies the MySQL-specific SQL strings use
// correct MySQL syntax (ON DUPLICATE KEY UPDATE, DATETIME(3), VARCHAR(255))
// without needing a live MySQL connection.
func TestMySQLQueries_SQLSyntax(t *testing.T) {
	t.Parallel()

	q := mysqlQueries()

	t.Run("ddl creates timers table", func(t *testing.T) {
		t.Parallel()

		if !strings.Contains(q.ddl, "CREATE TABLE IF NOT EXISTS timers") {
			t.Errorf("MySQL DDL should create timers table:\n%s", q.ddl)
		}
	})

	t.Run("ddl uses VARCHAR(255) primary key", func(t *testing.T) {
		t.Parallel()

		if !strings.Contains(q.ddl, "VARCHAR(255) PRIMARY KEY") {
			t.Errorf("MySQL DDL should use VARCHAR(255) PRIMARY KEY for id:\n%s", q.ddl)
		}
	})

	t.Run("ddl uses DATETIME(3) for fire_at", func(t *testing.T) {
		t.Parallel()

		if !strings.Contains(q.ddl, "DATETIME(3)") {
			t.Errorf("MySQL DDL should use DATETIME(3) for millisecond precision:\n%s", q.ddl)
		}
	})

	t.Run("ddl uses CURRENT_TIMESTAMP(3) default", func(t *testing.T) {
		t.Parallel()

		if !strings.Contains(q.ddl, "CURRENT_TIMESTAMP(3)") {
			t.Errorf("MySQL DDL should use CURRENT_TIMESTAMP(3) default:\n%s", q.ddl)
		}
	})

	t.Run("schedule uses ON DUPLICATE KEY UPDATE no-op", func(t *testing.T) {
		t.Parallel()

		if !strings.Contains(q.schedule, "ON DUPLICATE KEY UPDATE") {
			t.Errorf("MySQL schedule should use ON DUPLICATE KEY UPDATE:\n%s", q.schedule)
		}

		if !strings.Contains(q.schedule, "id = id") {
			t.Errorf(
				"MySQL schedule should use self-assignment no-op (id = id) for idempotent insert:\n%s",
				q.schedule,
			)
		}
	})

	t.Run("all queries use question mark placeholders", func(t *testing.T) {
		t.Parallel()

		for name, sql := range map[string]string{
			"schedule":   q.schedule,
			"due":        q.due,
			"deleteByID": q.deleteByID,
		} {
			if strings.Contains(sql, "$1") {
				t.Errorf("%s should use ? placeholders (MySQL), not $1:\n%s", name, sql)
			}
		}
	})
}
