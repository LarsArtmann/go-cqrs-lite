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

// TestDetectOnFixableFixture drives the Spec's Detect over a directory with
// a C003 violation and asserts the finding comes back through the toolsdk
// boundary.
func TestDetectOnFixableFixture(t *testing.T) {
	t.Parallel()

	dir := writeFixableFixture(t)
	ctx := finding.WithWorkingDir(context.Background(), dir)

	findings, err := Spec().Detect.Detect(ctx)
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}

	found := false
	for _, f := range findings {
		if string(f.Rule) == "C003" {
			found = true
		}
	}

	if !found {
		t.Fatalf("Detect missed the C003 fixture: %+v", findings)
	}
}

// TestRepairAppliesSafeFix drives Repair over the fixture and asserts the
// C003 default case is rewritten — the safe-fixable subset the Spec exposes.
func TestRepairAppliesSafeFix(t *testing.T) {
	t.Parallel()

	dir := writeFixableFixture(t)
	ctx := finding.WithWorkingDir(context.Background(), dir)

	res, err := Spec().Repair.Repair(ctx)
	if err != nil {
		t.Fatalf("Repair: %v", err)
	}

	if res.Description == "" {
		t.Fatal("Repair returned an empty description")
	}

	file := filepath.Join(dir, "main.go")
	after, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(string(after), `fmt.Errorf("fold: unknown event type`) {
		t.Fatalf("repair did not rewrite the default case:\n%s", after)
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
	"fmt"

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
