package architecture_test

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/ruletest"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/rules/architecture"
)

// --- E023: satellite-owned SQL beside go-cqrs-lite usage ---

func TestE023_FiresOnLibraryPlusDatabaseSQL(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"store.go": `package app

import (
	"database/sql"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
)

func save(db *sql.DB, e event.Event) error { return nil }
`,
	})
	findings := ruletest.RunDetector(t, architecture.NewE023Detector(ctx))
	ruletest.AssertRule(t, findings, "E023", 1)
}

// The mixing signal is PACKAGE-level: the SQL file and the library file may
// be different files of one package. BuildContextFromSource fabricates a
// unique PkgPath PER FILE, so the test pins both files to one package path
// to express the real (shared-PkgPath) shape.
func TestE023_FiresWhenSplitAcrossFilesOfOnePackage(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"sql.go": `package app

import "database/sql"

func open() (*sql.DB, error) { return nil, nil }
`,
		"events.go": `package app

import "github.com/larsartmann/go-cqrs-lite/event/v4"

func emit() {}
`,
	})
	for _, gf := range ctx.GoFiles {
		gf.Pkg.PkgPath = "example.com/app"
	}
	findings := ruletest.RunDetector(t, architecture.NewE023Detector(ctx))
	ruletest.AssertRule(t, findings, "E023", 1)
}

func TestE023_SilentOnLibraryOnly(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"app.go": `package app

import "github.com/larsartmann/go-cqrs-lite/system/v4"

func boot() {}
`,
	})
	findings := ruletest.RunDetector(t, architecture.NewE023Detector(ctx))
	ruletest.AssertRule(t, findings, "E023", 0)
}

func TestE023_SilentOnSQLOnly(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"db.go": `package app

import "database/sql"

func open() (*sql.DB, error) { return nil, nil }
`,
	})
	findings := ruletest.RunDetector(t, architecture.NewE023Detector(ctx))
	ruletest.AssertRule(t, findings, "E023", 0)
}

// Test files never carry the finding — test infrastructure may open engines
// and containers freely.
func TestE023_SilentOnTestFiles(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"store_test.go": `package app

import (
	"database/sql"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
)

func TestX(t *testing.T) { _ = sql.ErrNoRows }
`,
	})
	findings := ruletest.RunDetector(t, architecture.NewE023Detector(ctx))
	ruletest.AssertRule(t, findings, "E023", 0)
}
