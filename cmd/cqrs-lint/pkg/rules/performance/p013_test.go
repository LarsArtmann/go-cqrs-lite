package performance_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/rules/performance"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/ruletest"
)

func TestP013(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		sources   map[string]string
		wantCount int
	}{
		{
			"LiteralDSNWithoutBusyTimeout_Flagged",
			map[string]string{
				"db.go": `package main

import "database/sql"

func setup() {
	db, _ := sql.Open("sqlite", "test.db")
	_ = db
}
`,
			},
			1,
		},
		{
			"LiteralDSNWithBusyTimeoutModernc_NotFlagged",
			map[string]string{
				"db.go": `package main

import "database/sql"

func setup() {
	db, _ := sql.Open("sqlite", "file:test.db?_pragma=busy_timeout(5000)")
	_ = db
}
`,
			},
			0,
		},
		{
			"LiteralDSNWithBusyTimeoutMattn_NotFlagged",
			map[string]string{
				"db.go": `package main

import "database/sql"

func setup() {
	db, _ := sql.Open("sqlite3", "file:test.db?_busy_timeout=5000")
	_ = db
}
`,
			},
			0,
		},
		{
			"ConstConcatDSNWithBusyTimeout_NotFlagged",
			map[string]string{
				"db.go": `package main

import "database/sql"

const pragmas = "?_pragma=busy_timeout(15000)&_pragma=synchronous(NORMAL)"

func openDB(path string) {
	dsn := path + pragmas
	db, _ := sql.Open("sqlite", dsn)
	_ = db
}
`,
			},
			0,
		},
		{
			"ConstConcatDSNWithoutBusyTimeout_Flagged",
			map[string]string{
				"db.go": `package main

import "database/sql"

const pragmas = "?_pragma=synchronous(NORMAL)&_pragma=journal_mode(WAL)"

func openDB(path string) {
	dsn := path + pragmas
	db, _ := sql.Open("sqlite", dsn)
	_ = db
}
`,
			},
			1,
		},
		{
			"MultiLineConstConcatenation_NotFlagged",
			map[string]string{
				"db.go": `package main

import "database/sql"

const sqlitePragmaDSNParams = "?_pragma=busy_timeout(15000)" +
	"&_pragma=synchronous(NORMAL)" +
	"&_pragma=foreign_keys(ON)"

func openSQLite(path string) {
	dsn := path + sqlitePragmaDSNParams
	db, _ := sql.Open("sqlite", dsn)
	_ = db
}
`,
			},
			0,
		},
		{
			"PostOpenPragma_NotFlagged",
			map[string]string{
				"db.go": `package main

import (
	"context"
	"database/sql"
)

func setup() {
	db, _ := sql.Open("sqlite", "test.db")
	db.Exec("PRAGMA busy_timeout = 5000")
	_ = db
}
`,
			},
			0,
		},
		{
			"PostOpenPragmaExecContext_NotFlagged",
			map[string]string{
				"db.go": `package main

import (
	"context"
	"database/sql"
)

func setup(ctx context.Context) {
	db, _ := sql.Open("sqlite", "test.db")
	db.ExecContext(ctx, "PRAGMA busy_timeout = 15000")
	_ = db
}
`,
			},
			0,
		},
		{
			"LibraryWrapperCall_NotFlagged",
			map[string]string{
				"db.go": `package main

import "database/sql"

func setup() {
	db, _ := sql.Open("sqlite", "test.db")
	_ = storage.SQLiteEnableWAL(ctx, db)
	_ = db
}
`,
			},
			0,
		},
		{
			"OpaqueDSN_NotFlagged",
			map[string]string{
				"db.go": `package main

import "database/sql"

func buildDSN() string {
	return "file:test.db?_pragma=busy_timeout(5000)"
}

func setup() {
	db, _ := sql.Open("sqlite", buildDSN())
	_ = db
}
`,
			},
			0,
		},
		{
			"NonSQLite_NotFlagged",
			map[string]string{
				"main.go": `package main

func main() {
	println("hello")
}
`,
			},
			0,
		},
		{
			"PostgresOpen_NotFlagged",
			map[string]string{
				"db.go": `package main

import "database/sql"

func setup() {
	db, _ := sql.Open("postgres", "host=localhost")
	_ = db
}
`,
			},
			0,
		},
		{
			"InlineConcatWithBusyTimeout_NotFlagged",
			map[string]string{
				"db.go": `package main

import "database/sql"

func setup(path string) {
	db, _ := sql.Open("sqlite", path + "?_pragma=busy_timeout(5000)")
	_ = db
}
`,
			},
			0,
		},
		{
			"InlineConcatWithoutBusyTimeout_Flagged",
			map[string]string{
				"db.go": `package main

import "database/sql"

func setup(path string) {
	db, _ := sql.Open("sqlite", path + "?cache=shared")
	_ = db
}
`,
			},
			1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := analyzer.BuildContextFromSource(t, tt.sources)
			findings := ruletest.RunDetector(t, performance.NewP013Detector(ctx))
			ruletest.AssertRule(t, findings, "P013", tt.wantCount)
		})
	}
}
