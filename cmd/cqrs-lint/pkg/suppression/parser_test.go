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
