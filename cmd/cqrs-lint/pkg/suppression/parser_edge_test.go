package suppression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/larsartmann/go-finding"
)

// --- F024: every directive on one line is honored ---

func TestParseSuppressions_MultipleDirectivesOnOneLine(t *testing.T) {
	t.Parallel()

	got := ParseSuppressions("//cqrs-lint:ignore(C007) reason //cqrs-lint:ignore(A008)")
	if _, ok := got["C007"]; !ok {
		t.Errorf("C007 missing from %v", got)
	}

	if _, ok := got["A008"]; !ok {
		t.Errorf("A008 missing — second directive on the line was swallowed: %v", got)
	}
}

func TestParseSuppressions_DisableAndIgnoreCombined(t *testing.T) {
	t.Parallel()

	got := ParseSuppressions("//cqrs-lint:disable(E001) //cqrs-lint:ignore(V007)")
	if len(got) != 2 {
		t.Fatalf("want 2 rules, got %v", got)
	}
}

func TestParseSuppressions_ProseMentionIsInert(t *testing.T) {
	t.Parallel()

	got := ParseSuppressions("// see the cqrs-lint:ignore(C007) syntax docs")
	if len(got) != 0 {
		t.Errorf("prose mention must not suppress: %v", got)
	}
}

// --- F025: spacing between // and the keyword is irrelevant ---

func TestParseSuppressions_MultiSpacePrefix(t *testing.T) {
	t.Parallel()

	if got := ParseSuppressions("//   cqrs-lint:ignore(B002)"); len(got) != 1 {
		t.Errorf("multi-space prefix not normalized: %v", got)
	}

	if got := ParseSuppressions("code() //\tcqrs-lint:ignore(B002)"); len(got) != 1 {
		t.Errorf("tab after // not normalized: %v", got)
	}
}

func TestNormalizeCommentPrefix_LeavesNonDirectivesAlone(t *testing.T) {
	t.Parallel()

	// A comment that does not carry the cqrs-lint keyword must not gain one.
	if got := normalizeCommentPrefix("// plain note"); got != "// plain note" {
		t.Errorf("non-directive comment rewritten: %q", got)
	}
}

// --- F026: /* */ block comments are inert ---

func TestCommentTextStart_IgnoresCommentInsideBlockComment(t *testing.T) {
	t.Parallel()

	if got := commentTextStart(`x := f() /* trailing //cqrs-lint:ignore(C001) */`); got != -1 {
		t.Errorf("// inside /* */ treated as a line comment at %d", got)
	}
}

func TestFilter_SuppressionInsideBlockCommentIsInert(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "sample.go")

	src := `package sample

/*
cqrs-lint:ignore-start(C001)
everything here is prose, not a directive
*/
func f() {}
`
	if err := os.WriteFile(path, []byte(src), 0o600); err != nil {
		t.Fatal(err)
	}

	f := makeFinding(path, 6, "C001")
	if checkBlockSuppressionInFile(newLineCache(), f) {
		t.Error(
			"block suppression should not apply — the directive lines are inside a /* */ comment",
		)
	}
}

func TestFilter_SuppressionAfterClosingBlockCommentWorks(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "sample.go")

	src := `package sample

/* prose */
//cqrs-lint:ignore-start(C001)
func f() {}
//cqrs-lint:ignore-end(C001)
`
	if err := os.WriteFile(path, []byte(src), 0o600); err != nil {
		t.Fatal(err)
	}

	f := makeFinding(path, 5, "C001")
	if !checkBlockSuppressionInFile(newLineCache(), f) {
		t.Error("a directive following a closed /* */ comment must still work")
	}
}

// --- F027: structural block-directive issues are reported ---

func TestDetectStaleSuppressions_UnmatchedIgnoreEnd(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "sample.go")

	src := "package sample\n\n//cqrs-lint:ignore-end(C001)\nfunc f() {}\n"
	if err := os.WriteFile(path, []byte(src), 0o600); err != nil {
		t.Fatal(err)
	}

	stale := DetectStaleSuppressions([]string{path}, nil)
	if len(stale) != 1 || stale[0].Reason != unmatchedEndReason {
		t.Fatalf("want exactly the unmatched-end issue, got %+v", stale)
	}

	msg := FormatStaleWarning(stale[0])
	if !strings.Contains(msg, "delete the stray ignore-end") {
		t.Errorf("unmatched-end warning lost its remediation hint: %q", msg)
	}
}

func TestDetectStaleSuppressions_UnterminatedIgnoreStart(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "sample.go")

	src := "package sample\n\n//cqrs-lint:ignore-start(C001)\nfunc f() {}\n"
	if err := os.WriteFile(path, []byte(src), 0o600); err != nil {
		t.Fatal(err)
	}

	stale := DetectStaleSuppressions([]string{path}, nil)
	if len(stale) != 1 || stale[0].Reason != unterminatedStartReason {
		t.Fatalf("want exactly the unterminated-start issue, got %+v", stale)
	}
}

func makeFinding(path string, line int, rule string) finding.Finding {
	f, _ := finding.NewBuilder(
		finding.RuleName(rule), "cqrs-lint", "test finding", finding.SeverityWarning,
		finding.Pos(finding.FilePath(path), line, 1),
	).Build()

	return f
}
