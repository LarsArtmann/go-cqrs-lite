package toolspec

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/larsartmann/go-finding"
	"github.com/larsartmann/go-finding/toolsdk"
)

// TestRegisteredInDefaultRegistry: importing this package registers the
// cqrs-lint Spec into the default toolsdk registry (database/sql pattern) —
// BuildFlow hosts discover it via toolsdk.All() with no extra wiring.
func TestRegisteredInDefaultRegistry(t *testing.T) {
	t.Parallel()

	for _, s := range toolsdk.All() {
		if s.Name == "cqrs-lint" {
			return
		}
	}

	t.Fatal("cqrs-lint spec not found in toolsdk.All() after import")
}

// TestDetectRunsFullPipelineOnCQRSProject drives the Spec's Detect over a
// module importing go-cqrs-lite and asserts real rule findings come back
// through the toolsdk boundary (A018-style import analysis proves the
// analyzer, registry, and rule set all ran). Fold-level fixables need the
// full decider wiring real consumer projects have; that path is covered by
// fix_e2e_test and the runPipeline integration tests, which share the same
// pipeline and fix provider this Spec calls.
func TestDetectRunsFullPipelineOnCQRSProject(t *testing.T) {
	t.Parallel()

	dir := writeFixableFixture(t)
	ctx := finding.WithWorkingDir(context.Background(), dir)

	findings, err := Spec().Detect.Detect(ctx)
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}

	found := false
	for _, f := range findings {
		if string(f.Rule) == "A018" {
			found = true
		}
	}

	if !found {
		t.Fatalf("Detect missed the A018 dead-import finding: %+v", findings)
	}
}

// TestRepairRunsEndToEnd drives Repair over the fixture and asserts the
// cqrs-lint fix pipeline executes through the toolsdk boundary and reports
// a (possibly zero) measured result — BuildFlow re-detects the delta itself.
func TestRepairRunsEndToEnd(t *testing.T) {
	t.Parallel()

	dir := writeFixableFixture(t)
	ctx := finding.WithWorkingDir(context.Background(), dir)

	res, err := Spec().Repair.Repair(ctx)
	if err != nil {
		t.Fatalf("Repair: %v", err)
	}

	if !strings.Contains(res.Description, "cqrs-lint fix pipeline") {
		t.Fatalf("unexpected RepairResult description: %q", res.Description)
	}
}

func writeFixableFixture(t *testing.T) string {
	t.Helper()

	// C003 needs a compilable module whose `event` import resolves to the
	// real go-cqrs-lite event package; a replace to this checkout keeps the
	// fixture hermetic (no network, no go.sum entries).
	repoRoot, err := filepath.Abs(filepath.Join("..", "..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}

	gomod := `module fixture

go 1.26

require github.com/larsartmann/go-cqrs-lite/event/v4 v4.0.0

replace github.com/larsartmann/go-cqrs-lite/event/v4 => ` + repoRoot + `/event
`
	src := `package main

import (
	"github.com/larsartmann/go-cqrs-lite/event/v4"
)

type state struct{ N int }

func apply(state state, evt event.Event) (state, error) {
	switch evt.Type() {
	case "x.created":
		state.N++
	default:
		return state, nil
	}

	return state, nil
}
`
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(gomod), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte(src), 0o600); err != nil {
		t.Fatal(err)
	}

	return dir
}
