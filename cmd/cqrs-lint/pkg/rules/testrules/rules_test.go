package testrules_test

import (
	"context"
	"testing"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/testrules"
)

func runDetector(t *testing.T, det finding.Detector) []finding.Finding {
	t.Helper()

	findings, err := det.Detect(context.Background())
	if err != nil {
		t.Fatalf("detector %s: %v", det.Name(), err)
	}

	return findings
}

func assertRule(t *testing.T, findings []finding.Finding, ruleID string, wantCount int) {
	t.Helper()

	count := 0

	for _, f := range findings {
		if string(f.Rule) == ruleID {
			count++
		}
	}

	if count != wantCount {
		t.Errorf("rule %s: got %d findings, want %d", ruleID, count, wantCount)
		for _, f := range findings {
			t.Logf("  finding: rule=%s msg=%q", f.Rule, f.Message)
		}
	}
}

// ---------------------------------------------------------------------------
// T001: No scenario tests for deciders
// ---------------------------------------------------------------------------

func TestT001_FiresWhenDeciderHasNoScenarioTest(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"decider.go": `package main

import "github.com/larsartmann/go-cqrs-lite/decider/v4"

type State struct{ Count int }

var d = decider.Decider[State]{
	Initial: State{},
	Fold:    func(s State, _ event.Event) (State, error) { return s, nil },
}
`,
		"decider_test.go": `package main

import "testing"

func TestDecider(t *testing.T) {
	_ = State{Count: 5}
}
`,
	})
	findings := ruletest.RunDetector(t, testrules.NewT001Detector(ctx))
	ruletest.AssertRule(t, findings, "T001", 1)
}

func TestT001_NoFindingWhenScenarioGivenPresent(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"decider.go": `package main

import "github.com/larsartmann/go-cqrs-lite/decider/v4"

type State struct{ Count int }

var d = decider.Decider[State]{}
`,
		"decider_test.go": `package main

import "testing"

func TestDecider(t *testing.T) {
	scenario.Given[cmd, State](t, fold, State{})
}
`,
	})
	findings := ruletest.RunDetector(t, testrules.NewT001Detector(ctx))
	ruletest.AssertRule(t, findings, "T001", 0)
}

func TestT001_NoFindingWhenNoDeciderImport(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

import "testing"

func TestMain(t *testing.T) {}
`,
		"main_test.go": `package main

import "testing"

func TestSomething(t *testing.T) {}
`,
	})
	findings := ruletest.RunDetector(t, testrules.NewT001Detector(ctx))
	ruletest.AssertRule(t, findings, "T001", 0)
}

// ---------------------------------------------------------------------------
// T002: No scenario tests for projections
// ---------------------------------------------------------------------------

func TestT002_FiresWhenProjectionHasNoScenarioTest(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"proj.go": `package main

import "github.com/larsartmann/go-cqrs-lite/projection/v4"

var _ projection.Projection = myProj{}
type myProj struct{}
`,
		"proj_test.go": `package main

import "testing"

func TestProj(t *testing.T) {}
`,
	})
	findings := ruletest.RunDetector(t, testrules.NewT002Detector(ctx))
	ruletest.AssertRule(t, findings, "T002", 1)
}

func TestT002_NoFindingWhenGivenProjectionPresent(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"proj.go": `package main

import "github.com/larsartmann/go-cqrs-lite/projection/v4"

var _ projection.Projection = myProj{}
type myProj struct{}
`,
		"proj_test.go": `package main

import "testing"

func TestProj(t *testing.T) {
	scenario.GivenProjection(t, myProj{}, evt1)
}
`,
	})
	findings := ruletest.RunDetector(t, testrules.NewT002Detector(ctx))
	ruletest.AssertRule(t, findings, "T002", 0)
}

func TestT002_NoFindingWhenNoProjection(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main`,
		"main_test.go": `package main
import "testing"
func TestMain(t *testing.T) {}`,
	})
	findings := ruletest.RunDetector(t, testrules.NewT002Detector(ctx))
	ruletest.AssertRule(t, findings, "T002", 0)
}

// ---------------------------------------------------------------------------
// T003: No eventtest imports
// ---------------------------------------------------------------------------

func TestT003_FiresWhenEventUsedButNoEventtest(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"events.go": `package main

import "github.com/larsartmann/go-cqrs-lite/event/v4"

var _ event.Event
`,
		"events_test.go": `package main

import "testing"

func TestEvent(t *testing.T) {}
`,
	})
	findings := ruletest.RunDetector(t, testrules.NewT003Detector(ctx))
	ruletest.AssertRule(t, findings, "T003", 1)
}

func TestT003_NoFindingWhenEventtestImported(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"events.go": `package main

import "github.com/larsartmann/go-cqrs-lite/event/v4"

var _ event.Event
`,
		"events_test.go": `package main

import (
	"testing"
	"github.com/larsartmann/go-cqrs-lite/event/v4/eventtest"
)

func TestEvent(t *testing.T) {
	_ = eventtest.NewFakeStore()
}
`,
	})
	findings := ruletest.RunDetector(t, testrules.NewT003Detector(ctx))
	ruletest.AssertRule(t, findings, "T003", 0)
}

func TestT003_NoFindingWhenNoEventImport(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main`,
		"main_test.go": `package main
import "testing"
func TestMain(t *testing.T) {}`,
	})
	findings := ruletest.RunDetector(t, testrules.NewT003Detector(ctx))
	ruletest.AssertRule(t, findings, "T003", 0)
}

// ---------------------------------------------------------------------------
// T004: No golden/snapshot tests
// ---------------------------------------------------------------------------

func TestT004_FiresWhenCatalogUsedButNoSnaps(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"catalog.go": `package main

import "github.com/larsartmann/go-cqrs-lite/catalog/v4"

var _ = catalog.Registry{}
`,
		"catalog_test.go": `package main

import "testing"

func TestCatalog(t *testing.T) {}
`,
	})
	findings := ruletest.RunDetector(t, testrules.NewT004Detector(ctx))
	ruletest.AssertRule(t, findings, "T004", 1)
}

func TestT004_NoFindingWhenSnapsImported(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"catalog.go": `package main

import "github.com/larsartmann/go-cqrs-lite/catalog/v4"

var _ = catalog.Registry{}
`,
		"catalog_test.go": `package main

import (
	"testing"
	"github.com/gkampitakis/go-snaps/snaps"
)

func TestCatalog(t *testing.T) {
	snaps.MatchSnapshot(t, "output")
}
`,
	})
	findings := ruletest.RunDetector(t, testrules.NewT004Detector(ctx))
	ruletest.AssertRule(t, findings, "T004", 0)
}

func TestT004_NoFindingWhenNoCatalogImport(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main`,
		"main_test.go": `package main
import "testing"
func TestMain(t *testing.T) {}`,
	})
	findings := ruletest.RunDetector(t, testrules.NewT004Detector(ctx))
	ruletest.AssertRule(t, findings, "T004", 0)
}

// ---------------------------------------------------------------------------
// T005: Projection without error-handling test
// ---------------------------------------------------------------------------

func TestT005_FiresWhenProjectionHasNoErrorTest(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"proj.go": `package main

import "github.com/larsartmann/go-cqrs-lite/projection/v4"

var _ projection.Projection = myProj{}
type myProj struct{}
`,
		"proj_test.go": `package main

import "testing"

func TestProj(t *testing.T) {
	// only happy path tested
}
`,
	})
	findings := ruletest.RunDetector(t, testrules.NewT005Detector(ctx))
	ruletest.AssertRule(t, findings, "T005", 1)
}

func TestT005_NoFindingWhenThenErrorPresent(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"proj.go": `package main

import "github.com/larsartmann/go-cqrs-lite/projection/v4"

var _ projection.Projection = myProj{}
type myProj struct{}
`,
		"proj_test.go": `package main

import "testing"

func TestProjError(t *testing.T) {
	scenario.GivenProjection(t, myProj{}, badEvt).ThenError(errTarget)
}
`,
	})
	findings := ruletest.RunDetector(t, testrules.NewT005Detector(ctx))
	ruletest.AssertRule(t, findings, "T005", 0)
}

func TestT005_NoFindingWhenNoProjection(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main`,
		"main_test.go": `package main
import "testing"
func TestMain(t *testing.T) {}`,
	})
	findings := ruletest.RunDetector(t, testrules.NewT005Detector(ctx))
	ruletest.AssertRule(t, findings, "T005", 0)
}

// ---------------------------------------------------------------------------
// T006: Decider test without conflict-path test
// ---------------------------------------------------------------------------

func TestT006_FiresWhenScenarioGivenButNoThenError(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"decider_test.go": `package main

import "testing"

func TestDecider(t *testing.T) {
	scenario.Given[cmd, State](t, fold, State{}).
		When(cmd{}, decide).
		Then(evtCreated)
}
`,
	})
	findings := ruletest.RunDetector(t, testrules.NewT006Detector(ctx))
	ruletest.AssertRule(t, findings, "T006", 1)
}

func TestT006_NoFindingWhenThenErrorPresent(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"decider_test.go": `package main

import "testing"

func TestDeciderConflict(t *testing.T) {
	scenario.Given[cmd, State](t, fold, State{}).
		When(conflictCmd{}, decide).
		ThenError(errConflict)
}
`,
	})
	findings := ruletest.RunDetector(t, testrules.NewT006Detector(ctx))
	ruletest.AssertRule(t, findings, "T006", 0)
}

func TestT006_NoFindingWhenNoScenarioGiven(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main_test.go": `package main

import "testing"

func TestMain(t *testing.T) {}
`,
	})
	findings := ruletest.RunDetector(t, testrules.NewT006Detector(ctx))
	ruletest.AssertRule(t, findings, "T006", 0)
}

// ---------------------------------------------------------------------------
// T007: No integration test for event round-trip
// ---------------------------------------------------------------------------

func TestT007_FiresWhenEventUsedButNoRoundTrip(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"store.go": `package main

import "github.com/larsartmann/go-cqrs-lite/event/v4"

var _ event.Event
`,
		"store_test.go": `package main

import "testing"

func TestStore(t *testing.T) {
	// only Save tested, no Load
	store.Save(ctx, ref, events)
}
`,
	})
	findings := ruletest.RunDetector(t, testrules.NewT007Detector(ctx))
	ruletest.AssertRule(t, findings, "T007", 1)
}

func TestT007_NoFindingWhenBothSaveAndLoadCalled(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"store.go": `package main

import "github.com/larsartmann/go-cqrs-lite/event/v4"

var _ event.Event
`,
		"store_test.go": `package main

import "testing"

func TestRoundTrip(t *testing.T) {
	store.Save(ctx, ref, events)
	loaded, _ := store.Load(ctx, ref)
	_ = loaded
}
`,
	})
	findings := ruletest.RunDetector(t, testrules.NewT007Detector(ctx))
	ruletest.AssertRule(t, findings, "T007", 0)
}

func TestT007_NoFindingWhenNoEventImport(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main`,
		"main_test.go": `package main
import "testing"
func TestMain(t *testing.T) {}`,
	})
	findings := ruletest.RunDetector(t, testrules.NewT007Detector(ctx))
	ruletest.AssertRule(t, findings, "T007", 0)
}

// ---------------------------------------------------------------------------
// T008: Test files import production event store
// ---------------------------------------------------------------------------

func TestT008_FiresWhenTestImportsProductionStore(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main_test.go": `package main

import (
	"testing"
	"github.com/larsartmann/go-cqrs-lite/storage/v4"
)

func TestStore(t *testing.T) {
	store := storage.NewSQLiteBackend(db)
}
`,
	})
	findings := ruletest.RunDetector(t, testrules.NewT008Detector(ctx))
	ruletest.AssertRule(t, findings, "T008", 1)
}

func TestT008_FiresForStackSqlite(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main_test.go": `package main

import (
	"testing"
	"github.com/larsartmann/go-cqrs-lite/stack/sqlite/v4"
)

func TestStore(t *testing.T) {
	_ = sqlite.New
}
`,
	})
	findings := ruletest.RunDetector(t, testrules.NewT008Detector(ctx))
	ruletest.AssertRule(t, findings, "T008", 1)
}

func TestT008_NoFindingWhenTestUsesMemoryStore(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main_test.go": `package main

import (
	"testing"
	"github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
)

func TestStore(t *testing.T) {
	store := memory.NewMemoryStore()
}
`,
	})
	findings := ruletest.RunDetector(t, testrules.NewT008Detector(ctx))
	ruletest.AssertRule(t, findings, "T008", 0)
}

func TestT008_NoFindingWhenTestUsesEventtest(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main_test.go": `package main

import (
	"testing"
	"github.com/larsartmann/go-cqrs-lite/event/v4/eventtest"
)

func TestStore(t *testing.T) {
	store := eventtest.NewFakeStore()
}
`,
	})
	findings := ruletest.RunDetector(t, testrules.NewT008Detector(ctx))
	ruletest.AssertRule(t, findings, "T008", 0)
}

func TestT008_NoFindingForNonTestFiles(t *testing.T) {
	t.Parallel()

	ctx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

import "github.com/larsartmann/go-cqrs-lite/storage/v4"

var _ = storage.NewSQLiteBackend
`,
	})
	findings := ruletest.RunDetector(t, testrules.NewT008Detector(ctx))
	ruletest.AssertRule(t, findings, "T008", 0)
}
