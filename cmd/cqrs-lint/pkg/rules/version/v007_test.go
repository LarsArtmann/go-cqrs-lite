package version

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/ruletest"
)

// --- V007: v5-removed API usage ---

func TestV007_DetectsWholeRemovedModule(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

import "github.com/larsartmann/go-cqrs-lite/stack/sqlite/v4"

func main() {
	_, _ = sqlite.New("file:db.sqlite")
}
`,
	})
	findings := ruletest.RunDetector(t, NewV007Detector(ctx))
	ruletest.AssertRule(t, findings, "V007", 1)
}

func TestV007_DetectsAliasedRemovedModule(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

import csql "github.com/larsartmann/go-cqrs-lite/stack/sqlite/v4"

func main() {
	_, _ = csql.New("file:db.sqlite")
}
`,
	})
	findings := ruletest.RunDetector(t, NewV007Detector(ctx))
	ruletest.AssertRule(t, findings, "V007", 1)
}

func TestV007_DetectsDeprecatedSymbol(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

import "github.com/larsartmann/go-cqrs-lite/schema/v4"

var _ = schema.Upcaster(nil)

func newStore(s eventStore) error {
	_, err := schema.NewVersionedStore(s)
	return err
}
`,
	})
	findings := ruletest.RunDetector(t, NewV007Detector(ctx))
	ruletest.AssertRule(t, findings, "V007", 1)
}

func TestV007_DetectsTypeReference(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

import "github.com/larsartmann/go-cqrs-lite/stack/v4"

var views = stack.Materialize[UserView, string]{}
`,
	})
	findings := ruletest.RunDetector(t, NewV007Detector(ctx))
	ruletest.AssertRule(t, findings, "V007", 1)
}

func TestV007_SilentOnSurvivingSymbols(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

import (
	"github.com/larsartmann/go-cqrs-lite/schema/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/v4"
)

var _ = schema.UpcastSourceTransform(nil)
var _ stack.Option
`,
	})
	findings := ruletest.RunDetector(t, NewV007Detector(ctx))
	ruletest.AssertRule(t, findings, "V007", 0)
}

func TestV007_SilentOnForeignSameNamePackage(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

import "myapp/internal/stack"

func main() {
	_ = stack.Materialize{}
}
`,
	})
	findings := ruletest.RunDetector(t, NewV007Detector(ctx))
	ruletest.AssertRule(t, findings, "V007", 0)
}

func TestV007_SilentOnUnrelatedImport(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

import "example.com/app/event"

func main() {
	_ = event.TombstoneStatus(0)
}
`,
	})
	findings := ruletest.RunDetector(t, NewV007Detector(ctx))
	ruletest.AssertRule(t, findings, "V007", 0)
}

func TestV007_SkipsTestFiles(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main_test.go": `package main

import "github.com/larsartmann/go-cqrs-lite/stack/sqlite/v4"

import "testing"

func TestX(t *testing.T) {
	_, _ = sqlite.New("file:db.sqlite")
}
`,
	})
	findings := ruletest.RunDetector(t, NewV007Detector(ctx))
	ruletest.AssertRule(t, findings, "V007", 0)
}

func TestV007_SilentInSelfLintMode(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

import "github.com/larsartmann/go-cqrs-lite/stack/v4"

var b stack.Bundle
`,
	})
	ctx.ModulePath = "github.com/larsartmann/go-cqrs-lite/stack/v4"
	findings := ruletest.RunDetector(t, NewV007Detector(ctx))
	ruletest.AssertRule(t, findings, "V007", 0)
}

func TestV007_DetectsTombstoneHelpers(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

import "github.com/larsartmann/go-cqrs-lite/event/v4"

func check(events []event.Event) bool {
	return event.DetectTombstone(events) == event.TombstoneTombstoned
}
`,
	})
	findings := ruletest.RunDetector(t, NewV007Detector(ctx))
	ruletest.AssertRule(t, findings, "V007", 2)
}

// storage/relational is a SUBPACKAGE of the storage module, so its import
// path carries the /v4 segment mid-path ("storage/v4/relational"). The
// fragment normalization must handle that, or two of the ten removed modules
// are silently undetectable.
func TestV007_DetectsStorageSubpackageImports(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

import "github.com/larsartmann/go-cqrs-lite/storage/v4/relational"

var schema relational.RelationalSchema
`,
	})
	findings := ruletest.RunDetector(t, NewV007Detector(ctx))
	ruletest.AssertRule(t, findings, "V007", 1)
}

func TestV007_DetectsStorageRootAliasSymbols(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

import "github.com/larsartmann/go-cqrs-lite/storage/v4"

var _ storage.Row
`,
	})
	findings := ruletest.RunDetector(t, NewV007Detector(ctx))
	ruletest.AssertRule(t, findings, "V007", 1)
}

func TestV007_DetectsStorageSQLKeysetQuery(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

import "github.com/larsartmann/go-cqrs-lite/storage/v4/sql"

var q = sql.KeysetPositionQuery(sql.DialectSQLite, nil, "events", "ts")
`,
	})
	findings := ruletest.RunDetector(t, NewV007Detector(ctx))
	ruletest.AssertRule(t, findings, "V007", 1)
}

// stack/bench is a surviving stack subpackage: having "stack" as a prefix
// must not make the wholly-removed preset modules swallow it.
func TestV007_SilentOnBenchmarkPreset(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

import "github.com/larsartmann/go-cqrs-lite/stack/bench/v4"

var _ = bench.Something
`,
	})
	findings := ruletest.RunDetector(t, NewV007Detector(ctx))
	ruletest.AssertRule(t, findings, "V007", 0)
}

func TestV007_DetectsDotImport(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

import . "github.com/larsartmann/go-cqrs-lite/stack/sqlite/v4"

func main() {
	_, _ = New("file:db.sqlite")
}
`,
	})
	findings := ruletest.RunDetector(t, NewV007Detector(ctx))
	ruletest.AssertRule(t, findings, "V007", 1)
}

func TestV007_DotImportOnNonCQRSPackageIsSilent(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

import . "strings"

func main() {
	_ = Title("x")
}
`,
	})
	findings := ruletest.RunDetector(t, NewV007Detector(ctx))
	ruletest.AssertRule(t, findings, "V007", 0)
}

func TestV007_DotImportOnSurvivingModuleStillFires(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

import . "github.com/larsartmann/go-cqrs-lite/event/v4"
`,
	})
	findings := ruletest.RunDetector(t, NewV007Detector(ctx))
	ruletest.AssertRule(t, findings, "V007", 1)
}

// TestV007_ShadowedQualifierFiresOnce is the F091 Tier 1 pin: a local value
// named like the package must not produce a second finding. The import-table
// string scan matched both selectors by name (2 findings); typed resolution
// attributes only the real package reference (1 finding). The fixture
// compiles — the local declaration legally shadows the package inside main.
func TestV007_ShadowedQualifierFiresOnce(t *testing.T) {
	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

import "github.com/larsartmann/go-cqrs-lite/stack/sqlite/v4"

type shadowedDB struct{}

func (shadowedDB) New(string) (*struct{}, error) { return nil, nil }

func main() {
	_, _ = sqlite.New("file:real.sqlite")

	sqlite := shadowedDB{}
	_, _ = sqlite.New("file:shadowed.sqlite")
}
`,
	})
	findings := ruletest.RunDetector(t, NewV007Detector(ctx))
	ruletest.AssertRule(t, findings, "V007", 1)
}
