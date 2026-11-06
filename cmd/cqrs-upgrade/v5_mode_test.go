package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Fixture bodies for the v5 pass/mode tests. event/v4 is folded (has a
// core/v5 twin in v5ImportTable); metaengine/v4 is not, so it must surface
// in RemainingV4 instead of being rewritten.

const v5MixedGo = `package fixture

import (
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/metaengine/v4"
)

var _ = event.Type("x")
var _ = metaengine.Version
`

const v5PlainGo = `package plain

import "fmt"

var _ = fmt.Sprintf
`

func writeV5Tree(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, body := range files {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}
	return dir
}

func TestRunV5PassDryRunPlansWithoutWriting(t *testing.T) {
	dir := writeV5Tree(t, map[string]string{"fixture.go": v5MixedGo})

	rep := runV5Pass(dir, true)

	if rep.Error != "" {
		t.Fatalf("unexpected error: %s", rep.Error)
	}
	if rep.FilesScanned != 1 {
		t.Fatalf("FilesScanned = %d, want 1", rep.FilesScanned)
	}
	if len(rep.Rewrites) != 1 || len(rep.Rewrites[0].Changes) != 1 {
		t.Fatalf("Rewrites = %+v, want one file with one change", rep.Rewrites)
	}
	if got := rep.Rewrites[0].Changes[0].From; got != "github.com/larsartmann/go-cqrs-lite/event/v4" {
		t.Fatalf("change from %q, want the event/v4 import", got)
	}
	if len(rep.RemainingV4) != 1 || !strings.Contains(rep.RemainingV4[0], "metaengine/v4") {
		t.Fatalf("RemainingV4 = %v, want [metaengine/v4]", rep.RemainingV4)
	}

	after, err := os.ReadFile(filepath.Join(dir, "fixture.go"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(after), "core/v5") {
		t.Fatal("dry run wrote to disk")
	}
}

func TestRunV5PassAppliesRewrite(t *testing.T) {
	dir := writeV5Tree(t, map[string]string{"fixture.go": v5MixedGo})

	rep := runV5Pass(dir, false)

	if rep.Error != "" {
		t.Fatalf("unexpected error: %s", rep.Error)
	}
	after, err := os.ReadFile(filepath.Join(dir, "fixture.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(after), "github.com/larsartmann/go-cqrs-lite/core/v5/event") {
		t.Fatalf("event/v4 import not rewritten; file now:\n%s", after)
	}
	if !strings.Contains(string(after), "github.com/larsartmann/go-cqrs-lite/metaengine/v4") {
		t.Fatalf("not-yet-folded import must stay untouched; file now:\n%s", after)
	}
}

func TestRunV5PassSkipsScannedDirs(t *testing.T) {
	dir := writeV5Tree(t, map[string]string{
		"main.go":           v5PlainGo,
		"vendor/v.go":       v5MixedGo,
		"testdata/td.go":    v5MixedGo,
		"node_modules/n.go": v5MixedGo,
	})

	rep := runV5Pass(dir, true)

	if rep.Error != "" {
		t.Fatalf("unexpected error: %s", rep.Error)
	}
	if rep.FilesScanned != 1 {
		t.Fatalf("FilesScanned = %d, want 1 (vendor/testdata/node_modules excluded)", rep.FilesScanned)
	}
	if len(rep.Rewrites) != 0 {
		t.Fatalf("Rewrites = %+v, want none", rep.Rewrites)
	}
}

func TestRunV5PassReportsParseErrors(t *testing.T) {
	dir := writeV5Tree(t, map[string]string{"broken.go": "this is not go ☃"})

	rep := runV5Pass(dir, true)

	if rep.Error == "" || !strings.Contains(rep.Error, "parse:") {
		t.Fatalf("Error = %q, want a parse failure", rep.Error)
	}
	if rep.FilesScanned != 0 {
		t.Fatalf("FilesScanned = %d, want 0", rep.FilesScanned)
	}
}

func TestRunV5ModeStrictVerdicts(t *testing.T) {
	clean := writeV5Tree(t, map[string]string{"plain.go": v5PlainGo})
	mixed := writeV5Tree(t, map[string]string{"fixture.go": v5MixedGo})
	missing := filepath.Join(t.TempDir(), "does-not-exist")

	if err := runV5Mode(config{strict: true}, []string{clean}); err != nil {
		t.Fatalf("clean module must pass strict: %v", err)
	}
	if err := runV5Mode(config{strict: true}, []string{mixed}); !errors.Is(err, errStrictV5Remaining) {
		t.Fatalf("mixed module strict verdict = %v, want errStrictV5Remaining", err)
	}
	if err := runV5Mode(config{strict: true}, []string{missing}); !errors.Is(err, errStrictV5ModuleFailed) {
		t.Fatalf("missing dir strict verdict = %v, want errStrictV5ModuleFailed", err)
	}
	if err := runV5Mode(config{}, []string{missing}); err != nil {
		t.Fatalf("non-strict must not fail on module errors: %v", err)
	}
}

func TestRunV5ModeReportOutput(t *testing.T) {
	mixed := writeV5Tree(t, map[string]string{"fixture.go": v5MixedGo})
	missing := filepath.Join(t.TempDir(), "nope")

	done := captureStdout(t)
	if err := runV5Mode(config{workspace: true, dryRun: true}, []string{mixed, missing}); err != nil {
		t.Fatalf("runV5Mode: %v", err)
	}
	out := string(done())

	for _, want := range []string{
		"== " + mixed,
		"scanned 1 Go files: 1 import rewrite(s) in 1 file(s)",
		"github.com/larsartmann/go-cqrs-lite/core/v5/event",
		"not-yet-folded v4 imports (1):",
		"go.mod: report-only",
		"v5 pass errored:",
		"dry run: no changes written",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("report output missing %q; got:\n%s", want, out)
		}
	}
}

func TestPrintReportBranches(t *testing.T) {
	done := captureStdout(t)
	printReport(moduleReport{Error: "boom"})
	if out := string(done()); !strings.Contains(out, "error: boom") {
		t.Errorf("error branch: got %q", out)
	}

	done = captureStdout(t)
	printReport(moduleReport{
		Dir:     "/somewhere",
		NoPins:  true,
		ScanErr: errors.New("load exploded"),
	})
	out := string(done())
	if !strings.Contains(out, "no direct go-cqrs-lite pins found") ||
		!strings.Contains(out, "deprecation report: scan failed — load exploded") {
		t.Errorf("NoPins+ScanErr branch: got %q", out)
	}

	done = captureStdout(t)
	printReport(moduleReport{
		Dir: "/mod",
		Bumps: []bump{
			{Module: "m1", From: "v1", To: "v2"},
			{Module: "m2", From: "v1", To: "", resolveErr: errors.New("nope")},
			{Module: "m3", From: "v1", To: "v1", held: true},
			{Module: "m4", From: "v1", To: "v1", upToDate: true},
		},
		Deprecations: []findingJSON{{Position: "a.go:1:2", Rule: "V007", Message: "old API"}},
		Suggestions:  []findingJSON{{Position: "b.go:3:4", Rule: "V012", Message: "try systemscenario"}},
	})
	out = string(done())
	for _, want := range []string{
		"deprecation report: 1 finding(s)",
		"suggestions: 1 hint(s)",
		"a.go:1:2 old API [V007]",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("default branch missing %q; got:\n%s", want, out)
		}
	}
	for _, status := range []string{"bump", "resolve-error", "held", "up-to-date"} {
		if !strings.Contains(out, status) {
			t.Errorf("bump table missing status %q; got:\n%s", status, out)
		}
	}
}

func TestVerifyFailsWithoutGoMod(t *testing.T) {
	err := verify(t.TempDir())
	if err == nil || !strings.Contains(err.Error(), "go mod tidy") {
		t.Fatalf("verify on a dir without go.mod = %v, want a tidy failure", err)
	}
}
