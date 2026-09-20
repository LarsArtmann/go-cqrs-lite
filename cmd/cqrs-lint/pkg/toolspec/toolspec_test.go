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

// TestDetectHonorsProjectConfig proves the Spec's Detect path reads
// .cqrs-lint.json from the working directory: a config disabling A018
// suppresses exactly that finding while the same fixture without a config
// reports it. Before the embedded path honored config, hosts (e.g. BuildFlow)
// silently ignored every setting the CLI honored — a no-op config lie.
func TestDetectHonorsProjectConfig(t *testing.T) {
	t.Parallel()

	dir := writeFixableFixture(t)
	ctx := finding.WithWorkingDir(context.Background(), dir)

	countA018 := func(findings []finding.Finding) int {
		n := 0
		for _, f := range findings {
			if string(f.Rule) == "A018" {
				n++
			}
		}

		return n
	}

	baseline, err := Spec().Detect.Detect(ctx)
	if err != nil {
		t.Fatalf("Detect without config: %v", err)
	}
	if countA018(baseline) == 0 {
		t.Fatal("fixture must produce A018 without config (test precondition)")
	}

	config := `{
		// A018 is intentional here: the import registers a tool, it is not dead.
		"rules": {"disable": ["A018"]},
	}`
	if err := os.WriteFile(filepath.Join(dir, ".cqrs-lint.json"), []byte(config), 0o600); err != nil {
		t.Fatal(err)
	}

	configured, err := Spec().Detect.Detect(ctx)
	if err != nil {
		t.Fatalf("Detect with config: %v", err)
	}
	if countA018(configured) != 0 {
		t.Fatalf("A018 must be suppressed by rules.disable, got %d", countA018(configured))
	}
}

// TestDetectHonorsPresetFromConfig proves preset expansion works through the
// embedded path: the read-only preset pins command-flow, so command-flow
// adoption rules cannot fire on a fixture they would otherwise flag.
func TestDetectHonorsPresetFromConfig(t *testing.T) {
	t.Parallel()

	dir := writeFixableFixture(t)
	ctx := finding.WithWorkingDir(context.Background(), dir)

	if err := os.WriteFile(filepath.Join(dir, ".cqrs-lint.json"), []byte(`{"preset": "read-only"}`), 0o600); err != nil {
		t.Fatal(err)
	}

	findings, err := Spec().Detect.Detect(ctx)
	if err != nil {
		t.Fatalf("Detect with preset config: %v", err)
	}

	for _, f := range findings {
		if string(f.Rule) == "A018" {
			t.Fatal("read-only preset must suppress A018 via its preset defaults")
		}
	}
}

// TestDetectRejectsMalformedConfig proves a broken config is a loud error,
// never a silently-unconfigured lint run.
func TestDetectRejectsMalformedConfig(t *testing.T) {
	t.Parallel()

	dir := writeFixableFixture(t)
	ctx := finding.WithWorkingDir(context.Background(), dir)

	if err := os.WriteFile(filepath.Join(dir, ".cqrs-lint.json"), []byte(`{"preset":`), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := Spec().Detect.Detect(ctx); err == nil {
		t.Fatal("malformed config must fail Detect, not lint unconfigured")
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
