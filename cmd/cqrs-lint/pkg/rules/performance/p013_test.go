package performance_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/performance"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/ruletest"
)

func TestP013_LiteralDSNWithoutBusyTimeout_Flagged(t *testing.T) {
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
	findings := ruletest.RunDetector(t, performance.NewP013Detector(ctx))
	ruletest.AssertRule(t, findings, "P013", 1)
}

func TestP013_LiteralDSNWithBusyTimeoutModernc_NotFlagged(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"db.go": `package main

import "database/sql"

func setup() {
	db, _ := sql.Open("sqlite", "file:test.db?_pragma=busy_timeout(5000)")
	_ = db
}
`,
	})
	findings := ruletest.RunDetector(t, performance.NewP013Detector(ctx))
	ruletest.AssertRule(t, findings, "P013", 0)
}

func TestP013_LiteralDSNWithBusyTimeoutMattn_NotFlagged(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"db.go": `package main

import "database/sql"

func setup() {
	db, _ := sql.Open("sqlite3", "file:test.db?_busy_timeout=5000")
	_ = db
}
`,
	})
	findings := ruletest.RunDetector(t, performance.NewP013Detector(ctx))
	ruletest.AssertRule(t, findings, "P013", 0)
}

func TestP013_ConstConcatDSNWithBusyTimeout_NotFlagged(t *testing.T) {
	t.Parallel()

	// This is the DiscordSync pattern: DSN is built by concatenating a path
	// variable with a package-level const that contains busy_timeout.
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"db.go": `package main

import "database/sql"

const pragmas = "?_pragma=busy_timeout(15000)&_pragma=synchronous(NORMAL)"

func openDB(path string) {
	dsn := path + pragmas
	db, _ := sql.Open("sqlite", dsn)
	_ = db
}
`,
	})
	findings := ruletest.RunDetector(t, performance.NewP013Detector(ctx))
	ruletest.AssertRule(t, findings, "P013", 0)
}

func TestP013_ConstConcatDSNWithoutBusyTimeout_Flagged(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"db.go": `package main

import "database/sql"

const pragmas = "?_pragma=synchronous(NORMAL)&_pragma=journal_mode(WAL)"

func openDB(path string) {
	dsn := path + pragmas
	db, _ := sql.Open("sqlite", dsn)
	_ = db
}
`,
	})
	findings := ruletest.RunDetector(t, performance.NewP013Detector(ctx))
	ruletest.AssertRule(t, findings, "P013", 1)
}

func TestP013_MultiLineConstConcatenation_NotFlagged(t *testing.T) {
	t.Parallel()

	// Tests the exact DiscordSync pattern: multi-line const concatenation.
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
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
	})
	findings := ruletest.RunDetector(t, performance.NewP013Detector(ctx))
	ruletest.AssertRule(t, findings, "P013", 0)
}

func TestP013_PostOpenPragma_NotFlagged(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
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
	})
	findings := ruletest.RunDetector(t, performance.NewP013Detector(ctx))
	ruletest.AssertRule(t, findings, "P013", 0)
}

func TestP013_PostOpenPragmaExecContext_NotFlagged(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
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
	})
	findings := ruletest.RunDetector(t, performance.NewP013Detector(ctx))
	ruletest.AssertRule(t, findings, "P013", 0)
}

func TestP013_LibraryWrapperCall_NotFlagged(t *testing.T) {
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
	findings := ruletest.RunDetector(t, performance.NewP013Detector(ctx))
	ruletest.AssertRule(t, findings, "P013", 0)
}

func TestP013_OpaqueDSN_NotFlagged(t *testing.T) {
	t.Parallel()

	// DSN returned from a function call — we can't statically inspect it.
	// Suppress to avoid false positives.
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
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
	})
	findings := ruletest.RunDetector(t, performance.NewP013Detector(ctx))
	ruletest.AssertRule(t, findings, "P013", 0)
}

func TestP013_NonSQLite_NotFlagged(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

func main() {
	println("hello")
}
`,
	})
	findings := ruletest.RunDetector(t, performance.NewP013Detector(ctx))
	ruletest.AssertRule(t, findings, "P013", 0)
}

func TestP013_PostgresOpen_NotFlagged(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"db.go": `package main

import "database/sql"

func setup() {
	db, _ := sql.Open("postgres", "host=localhost")
	_ = db
}
`,
	})
	findings := ruletest.RunDetector(t, performance.NewP013Detector(ctx))
	ruletest.AssertRule(t, findings, "P013", 0)
}

func TestP013_InlineConcatWithBusyTimeout_NotFlagged(t *testing.T) {
	t.Parallel()

	// DSN built inline via concatenation at the call site.
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"db.go": `package main

import "database/sql"

func setup(path string) {
	db, _ := sql.Open("sqlite", path + "?_pragma=busy_timeout(5000)")
	_ = db
}
`,
	})
	findings := ruletest.RunDetector(t, performance.NewP013Detector(ctx))
	ruletest.AssertRule(t, findings, "P013", 0)
}

func TestP013_InlineConcatWithoutBusyTimeout_Flagged(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"db.go": `package main

import "database/sql"

func setup(path string) {
	db, _ := sql.Open("sqlite", path + "?cache=shared")
	_ = db
}
`,
	})
	findings := ruletest.RunDetector(t, performance.NewP013Detector(ctx))
	ruletest.AssertRule(t, findings, "P013", 1)
}
