package performance_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/performance"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/ruletest"
)

func TestP012_LiteralDSNWithoutWAL_Flagged(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"db.go": `package main

import "database/sql"

func setup() {
	db, _ := sql.Open("sqlite", "test.db")
	_ = db
}
`,
	})
	findings := ruletest.RunDetector(t, performance.NewP012Detector(ctx))
	ruletest.AssertRule(t, findings, "P012", 1)
}

func TestP012_LiteralDSNWithWAL_NotFlagged(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"db.go": `package main

import "database/sql"

func setup() {
	db, _ := sql.Open("sqlite", "file:test.db?_pragma=journal_mode(WAL)")
	_ = db
}
`,
	})
	findings := ruletest.RunDetector(t, performance.NewP012Detector(ctx))
	ruletest.AssertRule(t, findings, "P012", 0)
}

func TestP012_ConstConcatDSNWithWAL_NotFlagged(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"db.go": `package main

import "database/sql"

const pragmas = "?_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)"

func openDB(path string) {
	dsn := path + pragmas
	db, _ := sql.Open("sqlite", dsn)
	_ = db
}
`,
	})
	findings := ruletest.RunDetector(t, performance.NewP012Detector(ctx))
	ruletest.AssertRule(t, findings, "P012", 0)
}

func TestP012_PostOpenPragma_NotFlagged(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"db.go": `package main

import "database/sql"

func setup() {
	db, _ := sql.Open("sqlite", "test.db")
	db.Exec("PRAGMA journal_mode = WAL")
	_ = db
}
`,
	})
	findings := ruletest.RunDetector(t, performance.NewP012Detector(ctx))
	ruletest.AssertRule(t, findings, "P012", 0)
}

func TestP012_LibraryWrapperCall_NotFlagged(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"db.go": `package main

import "database/sql"

func setup() {
	db, _ := sql.Open("sqlite", "test.db")
	_ = storage.SQLiteEnableWAL(ctx, db)
	_ = db
}
`,
	})
	findings := ruletest.RunDetector(t, performance.NewP012Detector(ctx))
	ruletest.AssertRule(t, findings, "P012", 0)
}

func TestP012_OpaqueDSN_NotFlagged(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
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
	})
	findings := ruletest.RunDetector(t, performance.NewP012Detector(ctx))
	ruletest.AssertRule(t, findings, "P012", 0)
}

func TestP012_NonSQLite_NotFlagged(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

func main() {
	println("hello")
}
`,
	})
	findings := ruletest.RunDetector(t, performance.NewP012Detector(ctx))
	ruletest.AssertRule(t, findings, "P012", 0)
}
