package suppression_test

import (
	"context"
	"os"
	"testing"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/suppression"
)

func TestParseSuppressions(t *testing.T) {
	comment := `// some comment
//cqrs-lint:ignore(C007) wall-clock is domain logic
//cqrs-lint:ignore(C003) forward-compatible fold
`
	suppressions := suppression.ParseSuppressions(comment)
	if len(suppressions) != 2 {
		t.Fatalf("got %d suppressions, want 2", len(suppressions))
	}
	if reason, ok := suppressions["C007"]; !ok || reason != "wall-clock is domain logic" {
		t.Errorf("C007 reason = %q, want %q", reason, "wall-clock is domain logic")
	}
	if reason, ok := suppressions["C003"]; !ok || reason != "forward-compatible fold" {
		t.Errorf("C003 reason = %q, want %q", reason, "forward-compatible fold")
	}
}

func TestParseSuppressions_CommaSeparated(t *testing.T) {
	suppressions := suppression.ParseSuppressions(
		"//cqrs-lint:ignore(A001,E005) multiple rules",
	)
	if len(suppressions) != 2 {
		t.Fatalf("got %d suppressions, want 2", len(suppressions))
	}
	if _, ok := suppressions["A001"]; !ok {
		t.Error("A001 not found")
	}
	if _, ok := suppressions["E005"]; !ok {
		t.Error("E005 not found")
	}
}

func TestParseSuppressions_SpaceAfterSlashes(t *testing.T) {
	// The Go-idiomatic comment style has a space after //: "// cqrs-lint:ignore(C007)".
	// gofmt does not normalize this, so both variants must work.
	suppressions := suppression.ParseSuppressions(
		"// cqrs-lint:ignore(C007) wall-clock is domain logic",
	)
	if len(suppressions) != 1 {
		t.Fatalf("got %d suppressions, want 1", len(suppressions))
	}
	if _, ok := suppressions["C007"]; !ok {
		t.Error("C007 not found in space-variant suppression")
	}
}

func TestNewSuppressionFilter_MarksSuppressedFindings(t *testing.T) {
	filter := suppression.NewSuppressionFilter()

	f, _ := finding.NewBuilder(
		"C007", "cqrs-lint", "time.Now in decider",
		finding.SeverityWarning, finding.Pos("test.go", 5, 1),
	).
		WithSnippet("//cqrs-lint:ignore(C007) wall-clock is domain logic\nnow := time.Now()").
		Build()

	out, err := filter.Transform(context.TODO(), []finding.Finding{f})
	if err != nil {
		t.Fatalf("Transform() error: %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("got %d findings, want 1", len(out))
	}
	if out[0].Suppression == nil {
		t.Fatal("finding should be marked as suppressed")
	}
}

func TestNewSuppressionFilter_DoesNotMarkUnsuppressed(t *testing.T) {
	filter := suppression.NewSuppressionFilter()

	f, _ := finding.NewBuilder(
		"C007", "cqrs-lint", "time.Now in decider",
		finding.SeverityWarning, finding.Pos("test.go", 5, 1),
	).Build()

	out, err := filter.Transform(context.TODO(), []finding.Finding{f})
	if err != nil {
		t.Fatalf("Transform() error: %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("got %d findings, want 1", len(out))
	}
	if out[0].Suppression != nil {
		t.Fatal("finding should NOT be marked as suppressed")
	}
}

func TestNewSuppressionFilter_CommaSeparatedSnippetSuppressesSecondID(t *testing.T) {
	t.Parallel()

	filter := suppression.NewSuppressionFilter()

	// Finding for E005 — the SECOND ID in a comma-separated suppression.
	// Before the fix, extractRuleID returned only "A001" (the first ID),
	// so E005 was NOT suppressed in the snippet fallback path.
	f, _ := finding.NewBuilder(
		"E005", "cqrs-lint", "orphaned command",
		finding.SeverityWarning, finding.Pos("test.go", 5, 1),
	).
		WithSnippet("//cqrs-lint:ignore(A001,E005) both rules suppressed\nhandle()").
		Build()

	out, err := filter.Transform(context.TODO(), []finding.Finding{f})
	if err != nil {
		t.Fatalf("Transform() error: %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("got %d findings, want 1", len(out))
	}
	if out[0].Suppression == nil {
		t.Fatal("E005 should be suppressed by comma-separated snippet (A001,E005)")
	}
}

func TestNewSuppressionFilter_ReadsActualFile(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := tmpDir + "/example.go"

	content := `package main

import "time"

func fold() {
	//cqrs-lint:ignore(C007) domain clock
	now := time.Now()
	_ = now
}
`
	if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	filter := suppression.NewSuppressionFilter()

	f, _ := finding.NewBuilder(
		"C007", "cqrs-lint", "time.Now in decider",
		finding.SeverityWarning, finding.Pos(finding.FilePath(filePath), 7, 2),
	).Build()

	out, err := filter.Transform(context.TODO(), []finding.Finding{f})
	if err != nil {
		t.Fatalf("Transform() error: %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("got %d findings, want 1", len(out))
	}
	if out[0].Suppression == nil {
		t.Fatal("finding should be suppressed by file-based comment")
	}
}

func TestSuppression_SkipsBlankLinesWhenScanningUpward(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	filePath := tmpDir + "/example.go"

	// The suppression is on line 5, then a blank line, then the finding on line 7.
	// Before the fix, the blank line broke suppression scanning.
	content := `package main

import "time"

//cqrs-lint:ignore(C007) domain clock

func fold() {
	now := time.Now()
	_ = now
}
`
	if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	filter := suppression.NewSuppressionFilter()

	f, _ := finding.NewBuilder(
		"C007", "cqrs-lint", "time.Now in decider",
		finding.SeverityWarning, finding.Pos(finding.FilePath(filePath), 7, 1),
	).Build()

	out, err := filter.Transform(context.TODO(), []finding.Finding{f})
	if err != nil {
		t.Fatalf("Transform() error: %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("got %d findings, want 1", len(out))
	}
	if out[0].Suppression == nil {
		t.Fatal("finding should be suppressed despite blank line between comment and finding")
	}
}

func TestSuppression_DoesNotSkipNonBlankLines(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	filePath := tmpDir + "/example.go"

	// The suppression is on line 5 for C007, but line 6 is a non-blank line
	// with a different comment. The finding on line 7 should NOT be suppressed
	// because the scan stops at the first non-blank line above.
	content := `package main

import "time"

//cqrs-lint:ignore(C007) domain clock
// some other comment
func fold() {
	now := time.Now()
	_ = now
}
`
	if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	filter := suppression.NewSuppressionFilter()

	f, _ := finding.NewBuilder(
		"C007", "cqrs-lint", "time.Now in decider",
		finding.SeverityWarning, finding.Pos(finding.FilePath(filePath), 7, 1),
	).Build()

	out, err := filter.Transform(context.TODO(), []finding.Finding{f})
	if err != nil {
		t.Fatalf("Transform() error: %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("got %d findings, want 1", len(out))
	}
	if out[0].Suppression != nil {
		t.Fatal(
			"finding should NOT be suppressed — non-blank line between comment and finding breaks the scan",
		)
	}
}

func TestParseSuppressions_EndOfLine(t *testing.T) {
	t.Parallel()

	// An end-of-line comment: code precedes the suppression directive. Before
	// the fix, ParseSuppressions required the line to START with the comment
	// prefix, so trailing suppressions were silently ignored.
	suppressions := suppression.ParseSuppressions(
		"EventType = sdk.EventType //cqrs-lint:ignore(A008) re-export of SDK type",
	)
	if len(suppressions) != 1 {
		t.Fatalf("got %d suppressions, want 1", len(suppressions))
	}
	if _, ok := suppressions["A008"]; !ok {
		t.Error("A008 not recognized in end-of-line suppression")
	}
}

func TestParseSuppressions_EndOfLine_SpaceAfterSlashes(t *testing.T) {
	t.Parallel()

	// End-of-line variant WITH a space after //: "// cqrs-lint:ignore".
	// normalizeCommentPrefix must rewrite it before the Index search.
	suppressions := suppression.ParseSuppressions(
		"var Foo = bar.Foo // cqrs-lint:ignore(E003) domain-model package",
	)
	if len(suppressions) != 1 {
		t.Fatalf("got %d suppressions, want 1", len(suppressions))
	}
	if reason, ok := suppressions["E003"]; !ok || reason != "domain-model package" {
		t.Errorf("E003 reason = %q, want %q", reason, "domain-model package")
	}
}

func TestParseSuppressions_EndOfLine_CommaSeparated(t *testing.T) {
	t.Parallel()

	suppressions := suppression.ParseSuppressions(
		"repo := decider.NewRepository(s, b, d) //cqrs-lint:ignore(B025,A017,E008)",
	)
	if len(suppressions) != 3 {
		t.Fatalf("got %d suppressions, want 3", len(suppressions))
	}
	for _, id := range []string{"B025", "A017", "E008"} {
		if _, ok := suppressions[id]; !ok {
			t.Errorf("%s not recognized in end-of-line comma-separated suppression", id)
		}
	}
}

// TestParseSuppressions_IgnoresDocStringMentions ensures documentation/example
// strings that merely mention the suppression syntax are NOT mistaken for real
// suppressions. Without the out-of-string check, a line like the help text
// fmt.Println("//cqrs-lint:ignore(RULE)") would be flagged as a (stale)
// suppression referencing unknown rule "RULE".
func TestParseSuppressions_IgnoresDocStringMentions(t *testing.T) {
	t.Parallel()

	cases := []string{
		// double-quoted string mentions the syntax as documentation
		`fmt.Println("Suppress confirmed FPs with //cqrs-lint:ignore(RULE)")`,
		// double-quoted string with the space variant
		`s := "// cqrs-lint:ignore(C007) example"`,
		// backtick raw string
		"msg := `use //cqrs-lint:ignore(A001) to suppress`",
	}
	for _, line := range cases {
		suppressions := suppression.ParseSuppressions(line)
		if len(suppressions) != 0 {
			t.Errorf("doc string should not be parsed as suppression: %q -> %v", line, suppressions)
		}
	}
}

// TestParseSuppressions_RealCommentAfterStringLiteral ensures a genuine
// end-of-line suppression following a string literal on the same line still
// works — the string-aware scan must not over-suppress.
func TestParseSuppressions_RealCommentAfterStringLiteral(t *testing.T) {
	t.Parallel()

	suppressions := suppression.ParseSuppressions(
		`_ = "//cqrs-lint:ignore(NOPE)" //cqrs-lint:ignore(C007) real`,
	)
	if len(suppressions) != 1 {
		t.Fatalf("got %d suppressions, want 1 (only the real comment)", len(suppressions))
	}
	if _, ok := suppressions["C007"]; !ok {
		t.Errorf("C007 (real end-of-line comment) should be recognized; got %v", suppressions)
	}
}

func TestNewSuppressionFilter_EndOfLineCommentInFile(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	filePath := tmpDir + "/example.go"

	// The suppression is an end-of-line comment on the SAME line as the finding.
	content := `package main

import "time"

func fold() {
	now := time.Now() //cqrs-lint:ignore(C007) domain clock
	_ = now
}
`
	if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	filter := suppression.NewSuppressionFilter()

	f, _ := finding.NewBuilder(
		"C007", "cqrs-lint", "time.Now in decider",
		finding.SeverityWarning, finding.Pos(finding.FilePath(filePath), 6, 2),
	).Build()

	out, err := filter.Transform(context.TODO(), []finding.Finding{f})
	if err != nil {
		t.Fatalf("Transform() error: %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("got %d findings, want 1", len(out))
	}
	if out[0].Suppression == nil {
		t.Fatal("end-of-line suppression on the finding's own line should be recognized")
	}
}

func TestSuppression_WorksForAllNewRuleIDs(t *testing.T) {
	t.Parallel()

	// New rules that need suppression verification.
	newRuleIDs := []string{
		"C031", "C032", "C033", "C034", "C035", "C036", "C037", "C038", "C039",
		"P011", "P012", "P013",
		"D014", "D015", "D016", "D017",
		"A032", "A033",
		"E016", "E017",
		"S010", "S011",
		"F018", "F019", "F020", "F021",
	}

	for _, ruleID := range newRuleIDs {
		t.Run(ruleID, func(t *testing.T) {
			t.Parallel()

			filter := suppression.NewSuppressionFilter()

			f, _ := finding.NewBuilder(
				finding.RuleName(ruleID), "cqrs-lint", "test finding",
				finding.SeverityWarning, finding.Pos("test.go", 5, 1),
			).
				WithSnippet("//cqrs-lint:ignore(" + ruleID + ") intentional\noffending_line()").
				Build()

			out, err := filter.Transform(context.TODO(), []finding.Finding{f})
			if err != nil {
				t.Fatalf("Transform() error: %v", err)
			}
			if len(out) != 1 {
				t.Fatalf("got %d findings, want 1", len(out))
			}
			if out[0].Suppression == nil {
				t.Errorf("%s should be suppressible via //cqrs-lint:ignore(%s)", ruleID, ruleID)
			}
		})
	}
}
