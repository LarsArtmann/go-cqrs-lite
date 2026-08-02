package performance_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/performance"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/ruletest"
)

func TestP012_DetectsMissingWAL(t *testing.T) {
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

func TestP012_NoFindingWhenWALEnabled(t *testing.T) {
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

func TestP012_NoFindingForNonSQLite(t *testing.T) {
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
