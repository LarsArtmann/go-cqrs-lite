package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/larsartmann/go-finding"
)

// baselineFinding builds a finding at root/rel:line:col with the given rule and
// severity. The generated ID embeds the absolute path, which is why the
// baseline normalizes paths before keying.
func baselineFinding(root, rel, rule string, line, col int, sev finding.Severity) finding.Finding {
	return finding.NewFinding(
		finding.RuleName(rule),
		"cqrs-lint",
		"fixture finding",
		sev,
		finding.Pos(finding.FilePath(filepath.Join(root, rel)), line, col),
		finding.ConfidenceHigh,
	)
}

func writeFixtureBaseline(t *testing.T, root string, findings []finding.Finding) string {
	t.Helper()

	path := filepath.Join(root, ".cqrs-lint-baseline.json")

	if err := writeBaseline(path, entriesFromFindings(findings, root, nil)); err != nil {
		t.Fatalf("writeBaseline: %v", err)
	}

	return path
}

func TestBaselineSplit_ClassifiesKnownNewAndStale(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	knownA := baselineFinding(root, "a.go", "S001", 3, 1, finding.SeverityWarning)
	knownB := baselineFinding(root, "b.go", "E003", 7, 2, finding.SeverityWarning)
	removed := baselineFinding(root, "c.go", "C010", 9, 1, finding.SeverityWarning)
	fresh := baselineFinding(root, "d.go", "F010", 11, 1, finding.SeverityInfo)

	doc := &baselineDocument{
		Version: baselineVersion,
		Tool:    "cqrs-lint",
		Entries: entriesFromFindings([]finding.Finding{knownA, knownB, removed}, root, nil),
	}

	outcome := splitBaseline(doc, []finding.Finding{knownA, knownB, fresh}, root)

	if len(outcome.Known) != 2 {
		t.Fatalf("Known = %d, want 2", len(outcome.Known))
	}

	if len(outcome.New) != 1 || string(outcome.New[0].Rule) != "F010" {
		t.Fatalf("New = %+v, want the single F010 finding", outcome.New)
	}

	if len(outcome.Stale) != 1 || outcome.Stale[0].Rule != "C010" {
		t.Fatalf("Stale = %+v, want the removed C010 entry", outcome.Stale)
	}
}

// A baseline written from one checkout must match findings from another: keys
// are relative to the baseline's directory, never absolute.
func TestBaselineKeys_ArePortableAcrossRoots(t *testing.T) {
	t.Parallel()

	rootA := t.TempDir()
	rootB := t.TempDir()

	findingA := baselineFinding(rootA, "internal/x.go", "E004", 5, 3, finding.SeverityInfo)
	findingB := baselineFinding(rootB, "internal/x.go", "E004", 5, 3, finding.SeverityInfo)

	entries := entriesFromFindings([]finding.Finding{findingA}, rootA, nil)
	doc := &baselineDocument{Version: baselineVersion, Tool: "cqrs-lint", Entries: entries}

	outcome := splitBaseline(doc, []finding.Finding{findingB}, rootB)

	if len(outcome.Known) != 1 {
		t.Fatalf("cross-root known = %d, want 1 (portable relative key)", len(outcome.Known))
	}

	if len(outcome.New) != 0 {
		t.Fatalf("cross-root new = %d, want 0", len(outcome.New))
	}
}

func TestEntriesFromFindings_PreservesReasonsAndSorts(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	first := baselineFinding(root, "b.go", "E003", 20, 1, finding.SeverityWarning)
	second := baselineFinding(root, "a.go", "S001", 10, 2, finding.SeverityWarning)

	prior := &baselineDocument{
		Version: baselineVersion,
		Tool:    "cqrs-lint",
		Entries: []baselineEntry{
			{Rule: "E003", File: "b.go", Line: 20, Column: 1, Reason: "deliberate colocation"},
		},
	}

	entries := entriesFromFindings([]finding.Finding{first, second}, root, prior)

	if len(entries) != 2 {
		t.Fatalf("entries = %d, want 2", len(entries))
	}

	// Sorted by rule first: E003 before S001.
	if entries[0].Rule != "E003" || entries[1].Rule != "S001" {
		t.Fatalf("entries are not rule-sorted: %+v", entries)
	}

	if entries[0].Reason != "deliberate colocation" {
		t.Errorf("reason not preserved: %q", entries[0].Reason)
	}

	if entries[1].Reason != "" {
		t.Errorf("new entry gained a bogus reason: %q", entries[1].Reason)
	}
}

func TestWriteBaseline_IsDeterministic(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	path := filepath.Join(root, ".cqrs-lint-baseline.json")

	findings := []finding.Finding{
		baselineFinding(root, "z.go", "S001", 30, 1, finding.SeverityWarning),
		baselineFinding(root, "a.go", "E003", 10, 1, finding.SeverityWarning),
		baselineFinding(root, "m.go", "C010", 20, 1, finding.SeverityWarning),
	}

	entries := entriesFromFindings(findings, root, nil)

	if err := writeBaseline(path, entries); err != nil {
		t.Fatalf("first write: %v", err)
	}

	first, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	if err := writeBaseline(path, entries); err != nil {
		t.Fatalf("second write: %v", err)
	}

	second, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	if string(first) != string(second) {
		t.Fatal("baseline rewrite is not byte-deterministic")
	}

	doc, err := loadBaseline(path)
	if err != nil {
		t.Fatalf("loadBaseline: %v", err)
	}

	if len(doc.Entries) != 3 {
		t.Fatalf("entries = %d, want 3", len(doc.Entries))
	}

	for i := 1; i < len(doc.Entries); i++ {
		if compareBaselineEntries(doc.Entries[i-1], doc.Entries[i]) > 0 {
			t.Fatalf("entries are not sorted at %d: %+v", i, doc.Entries)
		}
	}
}

func TestLoadBaseline_ErrorCases(t *testing.T) {
	t.Parallel()

	root := t.TempDir()

	_, err := loadBaseline(filepath.Join(root, "missing.json"))
	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("missing baseline err = %v, want os.ErrNotExist", err)
	}

	malformed := filepath.Join(root, "malformed.json")

	if err := os.WriteFile(malformed, []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err = loadBaseline(malformed)
	if !errors.Is(err, errBaselineInvalid) {
		t.Errorf("malformed baseline err = %v, want errBaselineInvalid", err)
	}

	wrongVersion := filepath.Join(root, "v99.json")

	if err := os.WriteFile(wrongVersion,
		[]byte(`{"version":99,"tool":"cqrs-lint","entries":[]}`), 0o644,
	); err != nil {
		t.Fatal(err)
	}

	_, err = loadBaseline(wrongVersion)
	if !errors.Is(err, errBaselineInvalid) {
		t.Errorf("wrong-version baseline err = %v, want errBaselineInvalid", err)
	}

	if !strings.Contains(err.Error(), "version 99") {
		t.Errorf("wrong-version error does not name the version: %v", err)
	}
}

func TestLoadBaselineIfPresent_MissingIsNil(t *testing.T) {
	t.Parallel()

	doc, err := loadBaselineIfPresent(filepath.Join(t.TempDir(), "absent.json"))
	if err != nil {
		t.Fatalf("missing file err = %v, want nil", err)
	}

	if doc != nil {
		t.Fatalf("missing file doc = %+v, want nil", doc)
	}
}

func TestApplyBaseline_UpdateRequiresPath(t *testing.T) {
	t.Parallel()

	_, err := applyBaseline(&AppConfig{UpdateBaseline: true}, nil)
	if err == nil || !strings.Contains(err.Error(), "--baseline") {
		t.Fatalf("err = %v, want an error naming --baseline", err)
	}
}

func TestApplyBaseline_UpdateWritesAndIsQuiet(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	path := filepath.Join(root, "baseline.json")
	findings := []finding.Finding{
		baselineFinding(root, "a.go", "S001", 1, 1, finding.SeverityCritical),
		baselineFinding(root, "b.go", "E003", 2, 1, finding.SeverityWarning),
	}

	res, err := applyBaseline(
		&AppConfig{Baseline: path, UpdateBaseline: true, Quiet: true},
		findings,
	)
	if err != nil {
		t.Fatalf("applyBaseline: %v", err)
	}

	if res == nil || !res.Updated {
		t.Fatalf("result = %+v, want Updated", res)
	}

	if res.Reported != nil {
		t.Fatalf("update-baseline reported findings: %d", len(res.Reported))
	}

	doc, err := loadBaseline(path)
	if err != nil {
		t.Fatalf("loadBaseline: %v", err)
	}

	if len(doc.Entries) != 2 {
		t.Fatalf("entries = %d, want 2", len(doc.Entries))
	}
}

// The core contract: baseline-known findings are hidden and cannot fail the
// build; a finding absent from the baseline is reported and does.
func TestApplyBaseline_FiltersKnownAndGatesOnlyNew(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	known := baselineFinding(root, "a.go", "S001", 1, 1, finding.SeverityCritical)
	path := writeFixtureBaseline(t, root, []finding.Finding{known, {
		// second known finding, generated separately to keep the helper simple
		Rule:     "E003",
		Severity: finding.SeverityWarning,
		Message:  "known",
		Position: finding.Pos(finding.FilePath(filepath.Join(root, "b.go")), 2, 1),
	}})

	fresh := baselineFinding(root, "c.go", "C010", 3, 1, finding.SeverityError)
	active := []finding.Finding{known, fresh}

	res, err := applyBaseline(&AppConfig{Baseline: path, Quiet: true}, active)
	if err != nil {
		t.Fatalf("applyBaseline: %v", err)
	}

	if len(res.Reported) != 1 || string(res.Reported[0].Rule) != "C010" {
		t.Fatalf("Reported = %+v, want the single C010 finding", res.Reported)
	}

	cfg := &AppConfig{}

	if err := shouldExitWithError(cfg, res.Reported); err == nil {
		t.Fatal("new error finding did not fail the build")
	}
}

// A fixed finding must surface as a shrunk baseline so the file can be pruned.
func TestApplyBaseline_ReportsShrunkBaseline(t *testing.T) {
	root := t.TempDir()
	kept := baselineFinding(root, "a.go", "S001", 1, 1, finding.SeverityWarning)
	removed := baselineFinding(root, "b.go", "E003", 2, 1, finding.SeverityWarning)
	path := writeFixtureBaseline(t, root, []finding.Finding{kept, removed})

	var stderr string = captureStderr(t, func() {
		res, err := applyBaseline(&AppConfig{Baseline: path}, []finding.Finding{kept})
		if err != nil {
			t.Fatalf("applyBaseline: %v", err)
		}

		if res.Stale != 1 {
			t.Errorf("Stale = %d, want 1", res.Stale)
		}

		if len(res.Reported) != 0 {
			t.Errorf("Reported = %d, want 0", len(res.Reported))
		}
	})

	if !strings.Contains(stderr, "Baseline shrunk") {
		t.Errorf("stderr = %q, want a shrunk-baseline notice", stderr)
	}

	if !strings.Contains(stderr, "E003") {
		t.Errorf("stderr = %q, want the stale rule named", stderr)
	}
}

// Quiet silences the baseline drift notices (stderr) without changing the
// classification.
func TestApplyBaseline_QuietSilencesDriftNotices(t *testing.T) {
	root := t.TempDir()
	known := baselineFinding(root, "a.go", "S001", 1, 1, finding.SeverityWarning)
	path := writeFixtureBaseline(t, root, []finding.Finding{known})

	stderr := captureStderr(t, func() {
		res, err := applyBaseline(&AppConfig{Baseline: path, Quiet: true}, []finding.Finding{known})
		if err != nil {
			t.Fatalf("applyBaseline: %v", err)
		}

		if res.Known != 1 {
			t.Errorf("Known = %d, want 1", res.Known)
		}
	})

	if stderr != "" {
		t.Errorf("quiet baseline run wrote to stderr: %q", stderr)
	}
}
