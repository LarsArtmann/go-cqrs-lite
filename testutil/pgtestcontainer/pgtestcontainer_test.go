package pgtestcontainer

import (
	"strings"
	"testing"
)

// TestReplaceDBInDSN_URLFormat verifies that replaceDBInDSN correctly swaps the
// database name in URL-format Postgres DSNs (the primary format produced by
// testcontainers and the ephemeral-pg.sh script).
func TestReplaceDBInDSN_URLFormat(t *testing.T) {
	tests := []struct {
		name   string
		dsn    string
		newDB  string
		expect string
	}{
		{
			name:   "standard with query params",
			dsn:    "postgres://cqrs:cqrs@localhost:5432/cqrs_test?sslmode=disable",
			newDB:  "test_1",
			expect: "postgres://cqrs:cqrs@localhost:5432/test_1?sslmode=disable",
		},
		{
			name:   "postgresql scheme",
			dsn:    "postgresql://user:pass@host:5432/mydb",
			newDB:  "test_2",
			expect: "postgresql://user:pass@host:5432/test_2",
		},
		{
			name:   "no query params",
			dsn:    "postgres://user@host/db",
			newDB:  "test_3",
			expect: "postgres://user@host/test_3",
		},
		{
			name:   "multiple query params preserved",
			dsn:    "postgres://u:p@h:5432/db?sslmode=disable&connect_timeout=10",
			newDB:  "test_4",
			expect: "postgres://u:p@h:5432/test_4?sslmode=disable&connect_timeout=10",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := replaceDBInDSN(tc.dsn, tc.newDB)
			if got != tc.expect {
				t.Errorf(
					"replaceDBInDSN(%q, %q)\n  got:  %q\n  want: %q",
					tc.dsn,
					tc.newDB,
					got,
					tc.expect,
				)
			}
		})
	}
}

// TestReplaceDBInDSN_KeywordValueFormat verifies that replaceDBInDSN correctly
// swaps the database name in keyword/value format DSNs (host=localhost
// port=5432 dbname=mydb sslmode=disable). This format is used by some CI
// service containers and libpq-compatible tools.
func TestReplaceDBInDSN_KeywordValueFormat(t *testing.T) {
	tests := []struct {
		name   string
		dsn    string
		newDB  string
		expect string
	}{
		{
			name:   "dbname in middle",
			dsn:    "host=localhost port=5432 user=cqrs dbname=mydb sslmode=disable",
			newDB:  "test_5",
			expect: "host=localhost port=5432 user=cqrs dbname=test_5 sslmode=disable",
		},
		{
			name:   "dbname at end",
			dsn:    "host=localhost dbname=mydb",
			newDB:  "test_6",
			expect: "host=localhost dbname=test_6",
		},
		{
			name:   "dbname at start",
			dsn:    "dbname=mydb host=localhost",
			newDB:  "test_7",
			expect: "dbname=test_7 host=localhost",
		},
		{
			name:   "no dbname keyword — appends",
			dsn:    "host=localhost user=cqrs",
			newDB:  "test_8",
			expect: "host=localhost user=cqrs dbname=test_8",
		},
		{
			name:   "empty dsn — appends",
			dsn:    "",
			newDB:  "test_9",
			expect: "dbname=test_9",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := replaceDBInDSN(tc.dsn, tc.newDB)
			if got != tc.expect {
				t.Errorf(
					"replaceDBInDSN(%q, %q)\n  got:  %q\n  want: %q",
					tc.dsn,
					tc.newDB,
					got,
					tc.expect,
				)
			}
		})
	}
}

// TestReplaceDBInDSN_ProducesIsolatedDSNs verifies that two calls with the same
// base DSN but different newDB values produce DIFFERENT DSNs — the core
// invariant of per-test isolation (M18).
func TestReplaceDBInDSN_ProducesIsolatedDSNs(t *testing.T) {
	base := "postgres://cqrs:cqrs@localhost:5432/cqrs_test?sslmode=disable"

	dsn1 := replaceDBInDSN(base, "test_a")
	dsn2 := replaceDBInDSN(base, "test_b")

	if dsn1 == dsn2 {
		t.Fatalf("two different newDB values produced the same DSN: %s", dsn1)
	}

	if !strings.Contains(dsn1, "test_a") {
		t.Errorf("dsn1 should contain test_a: %s", dsn1)
	}

	if !strings.Contains(dsn2, "test_b") {
		t.Errorf("dsn2 should contain test_b: %s", dsn2)
	}
}
