package toolspec

// Repro test for the inline-suppression drop at the toolsdk boundary (M12,
// DiscordSync 2026-10-05). NOT part of the upstream suite: this file exists
// to prove, on identical fixture input, that the CLI honors
// //cqrs-lint:ignore(C003) while Spec().Detect does not.
//
// Control: the same fixture WITHOUT the directive must yield C003 (the
// trigger fires). Repro: with the directive, C003 must NOT appear — the
// CLI's suppression.NewSuppressionFilter transformer is absent from this
// package's detect() composition.

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/larsartmann/go-finding"
)

const reproSrcTpl = `package main

import (
	"github.com/larsartmann/go-cqrs-lite/event/v4"
)

type state struct{ N int }

{{.APPLY}}(state state, evt event.Event) (state, error) {
	switch evt.Type() {
	case "x.created":
		state.N++
	default:
		return state, nil
	}

	return state, nil
}
`

func writeReproFixture(t *testing.T, withDirective bool) string {
	t.Helper()

	repoRoot, err := filepath.Abs(filepath.Join("..", "..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}

	gomod := `module fixture

go 1.26

require github.com/larsartmann/go-cqrs-lite/event/v4 v4.0.0

replace github.com/larsartmann/go-cqrs-lite/event/v4 => ` + repoRoot + `/event
`
	apply := "func apply"
	if withDirective {
		apply = "//cqrs-lint:ignore(C003) repro: suppressed via inline directive\nfunc apply"
	}
	src := strings.ReplaceAll(reproSrcTpl, "{{.APPLY}}", apply)

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(gomod), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte(src), 0o600); err != nil {
		t.Fatal(err)
	}

	tidy := exec.Command("go", "mod", "tidy")
	tidy.Dir = dir
	tidy.Env = append(os.Environ(), "GOFLAGS=-mod=mod")
	if out, err := tidy.CombinedOutput(); err != nil {
		t.Fatalf("fixture go mod tidy: %v\n%s", err, out)
	}

	return dir
}

func hasC003(findings []finding.Finding) bool {
	for _, f := range findings {
		if string(f.Rule) == "C003" {
			return true
		}
	}
	return false
}

func TestRepro_Control_NoDirectiveFiresC003(t *testing.T) {
	dir := writeReproFixture(t, false)
	findings, err := Spec().Detect.Detect(finding.WithWorkingDir(context.Background(), dir))
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if !hasC003(findings) {
		t.Fatalf("control failed: C003 did not fire without the directive; findings: %+v", findings)
	}
}

func TestRepro_DirectiveSuppressedThroughToolspec(t *testing.T) {
	dir := writeReproFixture(t, true)
	findings, err := Spec().Detect.Detect(finding.WithWorkingDir(context.Background(), dir))
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if hasC003(findings) {
		t.Fatalf("REPRO CONFIRMED: //cqrs-lint:ignore(C003) is DROPPED at the toolsdk boundary; C003 still reported. findings: %+v", findings)
	}
}
