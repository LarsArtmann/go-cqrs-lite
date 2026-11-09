package sql_test

import (
	"testing"

	sqlpkg "github.com/larsartmann/go-cqrs-lite/storage/v4/sql"
)

func TestUpsertMethods(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		dialect     sqlpkg.Dialect
		excludedRef string
		doNothing   string
		doUpdate    string
		quoteKey    string
	}{
		{
			name:        "postgres",
			dialect:     sqlpkg.PostgresDialect{},
			excludedRef: "excluded.col",
			doNothing:   "ON CONFLICT DO NOTHING",
			doUpdate:    "ON CONFLICT(a, b) DO UPDATE SET x = excluded.x",
			quoteKey:    "key",
		},
		{
			name:        "sqlite",
			dialect:     sqlpkg.SQLiteDialect{},
			excludedRef: "excluded.col",
			doNothing:   "ON CONFLICT DO NOTHING",
			doUpdate:    "ON CONFLICT(a, b) DO UPDATE SET x = excluded.x",
			quoteKey:    "key",
		},
		{
			name:        "duckdb",
			dialect:     sqlpkg.DuckDBDialect{},
			excludedRef: "excluded.col",
			doNothing:   "ON CONFLICT DO NOTHING",
			doUpdate:    "ON CONFLICT(a, b) DO UPDATE SET x = excluded.x",
			quoteKey:    "key",
		},
		{
			name:        "mysql",
			dialect:     sqlpkg.MySQLDialect{},
			excludedRef: "VALUES(col)",
			doNothing:   "ON DUPLICATE KEY UPDATE id = id",
			doUpdate:    "ON DUPLICATE KEY UPDATE x = VALUES(x)",
			quoteKey:    "`key`",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := tt.dialect.ExcludedRef("col"); got != tt.excludedRef {
				t.Errorf("ExcludedRef(col) = %q, want %q", got, tt.excludedRef)
			}

			if got := tt.dialect.OnConflictDoNothing("id"); got != tt.doNothing {
				t.Errorf("OnConflictDoNothing(id) = %q, want %q", got, tt.doNothing)
			}

			setExprs := []string{"x = " + tt.dialect.ExcludedRef("x")}
			if got := tt.dialect.OnConflictDoUpdate(
				[]string{"a", "b"},
				setExprs,
			); got != tt.doUpdate {
				t.Errorf("OnConflictDoUpdate([a,b], [x=...]) = %q, want %q", got, tt.doUpdate)
			}

			if got := tt.dialect.QuoteIdentifier("key"); got != tt.quoteKey {
				t.Errorf("QuoteIdentifier(key) = %q, want %q", got, tt.quoteKey)
			}
		})
	}
}

func TestIsDuplicateKeyError_MySQL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"nil", nil, false},
		{"mysql dup string", strErr("Error 1062: Duplicate entry 'x' for key 'PRIMARY'"), true},
		{"mysql deadlock string", strErr("Error 1213: Deadlock found"), false},
		{"sqlite dup string", strErr("UNIQUE constraint failed: events.id"), true},
		{"postgres dup string", strErr("duplicate key value violates unique constraint"), true},
		{"unrelated error", strErr("connection refused"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := sqlpkg.IsDuplicateKeyError(tt.err); got != tt.expected {
				t.Errorf("IsDuplicateKeyError(%v) = %v, want %v", tt.err, got, tt.expected)
			}
		})
	}
}

type strErr string

func (e strErr) Error() string { return string(e) }
