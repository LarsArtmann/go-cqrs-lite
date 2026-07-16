package suppression_test

import (
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

func TestNewSuppressionFilter_MarksSuppressedFindings(t *testing.T) {
	filter := suppression.NewSuppressionFilter()

	f, _ := finding.NewBuilder(
		"C007", "cqrs-lint", "time.Now in decider",
		finding.SeverityWarning, finding.Pos("test.go", 5, 1),
	).
		WithSnippet("//cqrs-lint:ignore(C007) wall-clock is domain logic\nnow := time.Now()").
		Build()

	out, err := filter.Transform(nil, []finding.Finding{f})
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

	out, err := filter.Transform(nil, []finding.Finding{f})
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

	out, err := filter.Transform(nil, []finding.Finding{f})
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
