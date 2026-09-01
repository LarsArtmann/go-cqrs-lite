package performance_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/rules/performance"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/ruletest"
)

func TestP012(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		sources   map[string]string
		wantCount int
	}{
		{
			"LiteralDSNWithoutWAL_Flagged",
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
			"LiteralDSNWithWAL_NotFlagged",
			map[string]string{
				"db.go": `package main

import "database/sql"

func setup() {
	db, _ := sql.Open("sqlite", "file:test.db?_pragma=journal_mode(WAL)")
	_ = db
}
`,
			},
			0,
		},
		{
			"ConstConcatDSNWithWAL_NotFlagged",
			map[string]string{
				"db.go": `package main

import "database/sql"

const pragmas = "?_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)"

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
			"PostOpenPragma_NotFlagged",
			map[string]string{
				"db.go": `package main

import "database/sql"

func setup() {
	db, _ := sql.Open("sqlite", "test.db")
	db.Exec("PRAGMA journal_mode = WAL")
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
	return "file:test.db?_pragma=journal_mode(WAL)"
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := analyzer.BuildContextFromSource(t, tt.sources)
			findings := ruletest.RunDetector(t, performance.NewP012Detector(ctx))
			ruletest.AssertRule(t, findings, "P012", tt.wantCount)
		})
	}
}
