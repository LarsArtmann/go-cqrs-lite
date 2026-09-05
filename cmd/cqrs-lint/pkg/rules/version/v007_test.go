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
