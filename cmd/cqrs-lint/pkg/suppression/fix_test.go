package suppression_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/suppression"
)

func writeFixture(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "example.go")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func readFixture(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// TestRemoveStaleInlineSuppressions_RemovesWholeLine verifies that a stale
// directive occupying an entire line is deleted, with the file's trailing
// newline preserved.
func TestRemoveStaleInlineSuppressions_RemovesWholeLine(t *testing.T) {
	t.Parallel()

	path := writeFixture(t,
		"package main\n\n//cqrs-lint:ignore(D002) legacy workaround\ntype Foo struct{}\n")

	res := suppression.RemoveStaleInlineSuppressions([]suppression.SuppressionAuditEntry{
		{File: path, Line: 3, Rule: "D002", Status: suppression.AuditStale},
	})

	if len(res.Removed) != 1 {
		t.Fatalf("Removed = %v, want 1 entry", res.Removed)
	}
	if len(res.Skipped) != 0 {
		t.Errorf("Skipped = %v, want none", res.Skipped)
	}
	if len(res.Files) != 1 || res.Files[0] != path {
		t.Errorf("Files = %v, want [%s]", res.Files, path)
	}

	want := "package main\n\ntype Foo struct{}\n"
	if got := readFixture(t, path); got != want {
		t.Errorf("file after fix = %q, want %q", got, want)
	}
}

// TestRemoveStaleInlineSuppressions_KeepsTrailingOnCode verifies a directive
// sharing its line with code is never removed — deleting the line would
// delete the code.
func TestRemoveStaleInlineSuppressions_KeepsTrailingOnCode(t *testing.T) {
	t.Parallel()

	content := "package main\n\nvar x = 1 //cqrs-lint:ignore(D002)\n"
	path := writeFixture(t, content)

	res := suppression.RemoveStaleInlineSuppressions([]suppression.SuppressionAuditEntry{
		{File: path, Line: 3, Rule: "D002", Status: suppression.AuditStale},
	})

	if len(res.Removed) != 0 || len(res.Files) != 0 {
		t.Errorf("trailing-on-code must not be removed, got %+v", res)
	}
	if len(res.Skipped) != 1 {
		t.Fatalf("Skipped = %v, want 1 entry", res.Skipped)
	}
	if got := readFixture(t, path); got != content {
		t.Errorf("file must be unchanged, got %q", got)
	}
}

// TestRemoveStaleInlineSuppressions_KeepsBlockMarkers verifies block marker
// lines are immune even if a stale entry (defensively) points at one.
func TestRemoveStaleInlineSuppressions_KeepsBlockMarkers(t *testing.T) {
	t.Parallel()

	content := "package main\n\n//cqrs-lint:ignore-start\ntype Foo struct{}\n//cqrs-lint:ignore-end\n"
	path := writeFixture(t, content)

	res := suppression.RemoveStaleInlineSuppressions([]suppression.SuppressionAuditEntry{
		{File: path, Line: 3, Rule: "D002", Status: suppression.AuditStale},
		{File: path, Line: 5, Rule: "D002", Status: suppression.AuditStale},
	})

	if len(res.Removed) != 0 || len(res.Files) != 0 {
		t.Errorf("block markers must not be removed, got %+v", res)
	}
	if len(res.Skipped) != 2 {
		t.Errorf("Skipped = %v, want 2 entries", res.Skipped)
	}
	if got := readFixture(t, path); got != content {
		t.Errorf("file must be unchanged, got %q", got)
	}
}

// TestRemoveStaleInlineSuppressions_CombinedDirectiveOneLine verifies a
// combined ignore(A,B) directive that went fully stale removes its single
// line once, reporting one Removed entry.
func TestRemoveStaleInlineSuppressions_CombinedDirectiveOneLine(t *testing.T) {
	t.Parallel()

	path := writeFixture(t, "package main\n\n//cqrs-lint:ignore(A001,D002)\ntype Foo struct{}\n")

	res := suppression.RemoveStaleInlineSuppressions([]suppression.SuppressionAuditEntry{
		{File: path, Line: 3, Rule: "A001", Status: suppression.AuditStale},
		{File: path, Line: 3, Rule: "D002", Status: suppression.AuditStale},
	})

	if len(res.Removed) != 1 {
		t.Fatalf("Removed = %v, want 1 deduplicated entry", res.Removed)
	}
	if len(res.Skipped) != 0 {
		t.Errorf("Skipped = %v, want none", res.Skipped)
	}

	want := "package main\n\ntype Foo struct{}\n"
	if got := readFixture(t, path); got != want {
		t.Errorf("file after fix = %q, want %q", got, want)
	}
}

// TestRemoveStaleInlineSuppressions_ToleratesMissingFile verifies an
// unreadable file degrades to Skipped instead of erroring.
func TestRemoveStaleInlineSuppressions_ToleratesMissingFile(t *testing.T) {
	t.Parallel()

	missing := filepath.Join(t.TempDir(), "gone.go")

	res := suppression.RemoveStaleInlineSuppressions([]suppression.SuppressionAuditEntry{
		{File: missing, Line: 3, Rule: "D002", Status: suppression.AuditStale},
	})

	if len(res.Removed) != 0 || len(res.Files) != 0 {
		t.Errorf("missing file must not be removed, got %+v", res)
	}
	if len(res.Skipped) != 1 {
		t.Errorf("Skipped = %v, want 1 entry", res.Skipped)
	}
}

// TestRemoveStaleInlineSuppressions_IgnoresNonStaleEntries verifies active
// and unknown-rule suppressions are never rewritten — only stale ones.
func TestRemoveStaleInlineSuppressions_IgnoresNonStaleEntries(t *testing.T) {
	t.Parallel()

	content := "package main\n\n//cqrs-lint:ignore(D002)\n//cqrs-lint:ignore(ZZZZ)\ntype Foo struct{}\n"
	path := writeFixture(t, content)

	res := suppression.RemoveStaleInlineSuppressions([]suppression.SuppressionAuditEntry{
		{File: path, Line: 3, Rule: "D002", Status: suppression.AuditActive},
		{File: path, Line: 4, Rule: "ZZZZ", Status: suppression.AuditUnknownRule},
	})

	if len(res.Removed) != 0 || len(res.Skipped) != 0 || len(res.Files) != 0 {
		t.Errorf("non-stale entries must be ignored entirely, got %+v", res)
	}
	if got := readFixture(t, path); got != content {
		t.Errorf("file must be unchanged, got %q", got)
	}
}

// TestRemoveStaleInlineSuppressions_IndentedWholeLine verifies an indented
// directive-only line (typical inside functions) is removed.
func TestRemoveStaleInlineSuppressions_IndentedWholeLine(t *testing.T) {
	t.Parallel()

	path := writeFixture(t, "package main\n\nfunc f() {\n\t//cqrs-lint:ignore(D002)\n\t_ = 1\n}\n")

	res := suppression.RemoveStaleInlineSuppressions([]suppression.SuppressionAuditEntry{
		{File: path, Line: 4, Rule: "D002", Status: suppression.AuditStale},
	})

	if len(res.Removed) != 1 {
		t.Fatalf("Removed = %v, want 1 entry", res.Removed)
	}

	want := "package main\n\nfunc f() {\n\t_ = 1\n}\n"
	if got := readFixture(t, path); got != want {
		t.Errorf("file after fix = %q, want %q", got, want)
	}
}

// TestRemoveStaleInlineSuppressions_MultipleLinesShift verifies multiple
// stale lines in one file are all removed despite line-index shifts.
func TestRemoveStaleInlineSuppressions_MultipleLinesShift(t *testing.T) {
	t.Parallel()

	path := writeFixture(t, strings.Join([]string{
		"package main",
		"",
		"//cqrs-lint:ignore(A001)",
		"type Foo struct{}",
		"",
		"//cqrs-lint:ignore(D002)",
		"type Bar struct{}",
		"",
	}, "\n"))

	res := suppression.RemoveStaleInlineSuppressions([]suppression.SuppressionAuditEntry{
		{File: path, Line: 3, Rule: "A001", Status: suppression.AuditStale},
		{File: path, Line: 6, Rule: "D002", Status: suppression.AuditStale},
	})

	if len(res.Removed) != 2 {
		t.Fatalf("Removed = %v, want 2 entries", res.Removed)
	}

	want := "package main\n\ntype Foo struct{}\n\ntype Bar struct{}\n"
	if got := readFixture(t, path); got != want {
		t.Errorf("file after fix = %q, want %q", got, want)
	}
}
