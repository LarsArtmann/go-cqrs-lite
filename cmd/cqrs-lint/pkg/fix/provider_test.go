package fix_test

import (
	"bytes"
	"testing"

	"github.com/larsartmann/go-finding"
	"github.com/larsartmann/go-finding/pipeline"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/fix"
)

func TestCQRSFixProvider_C006(t *testing.T) {
	provider := fix.NewCQRSFixProvider()

	f, err := finding.NewBuilder(
		"C006", "cqrs-lint", "manual version arithmetic",
		finding.SeverityWarning, finding.Pos("test.go", 5, 1),
	).
		WithFixStrategy(finding.FixStrategyDirect).
		WithBeforeCode("event.Version(version.Int()+1)").
		WithAfterCode("version.Increment()").
		WithMetadata(map[string]string{
			"oldExpr": "event.Version(version.Int()+1)",
			"newExpr": "version.Increment()",
		}).
		Build()
	if err != nil {
		t.Fatal(err)
	}

	if !provider.CanHandle(f) {
		t.Fatal("CanHandle() should return true for C006 finding")
	}

	content := []byte(`package main

func decide(version event.Version) {
	_ = event.Version(version.Int()+1)
}
`)

	edits, err := provider.Edits(content, f)
	if err != nil {
		t.Fatalf("Edits() error: %v", err)
	}
	if len(edits) != 1 {
		t.Fatalf("Edits() returned %d edits, want 1", len(edits))
	}

	result := applyEdit(content, edits[0])
	expected := "version.Increment()"
	if !bytes.Contains(result, []byte(expected)) {
		t.Errorf("Edit result does not contain %q\ngot: %s", expected, result)
	}
}

func TestCQRSFixProvider_C003(t *testing.T) {
	provider := fix.NewCQRSFixProvider()

	f, err := finding.NewBuilder(
		"C003", "cqrs-lint", "silent fold",
		finding.SeverityError, finding.Pos("test.go", 10, 2),
	).
		WithFixStrategy(finding.FixStrategyDirect).
		WithBeforeCode("return state, nil").
		WithAfterCode(`return state, fmt.Errorf("fold: unknown event type: %s", evt.Type())`).
		Build()
	if err != nil {
		t.Fatal(err)
	}

	if !provider.CanHandle(f) {
		t.Fatal("CanHandle() should return true for C003 finding")
	}

	content := []byte("func fold() error {\n    return state, nil\n}\n")
	edits, err := provider.Edits(content, f)
	if err != nil {
		t.Fatalf("Edits() error: %v", err)
	}
	if len(edits) != 1 {
		t.Fatalf("Edits() returned %d edits, want 1", len(edits))
	}

	result := applyEdit(content, edits[0])
	if !bytes.Contains(result, []byte("fmt.Errorf")) {
		t.Errorf("Edit result should contain fmt.Errorf\ngot: %s", result)
	}
}

func TestCQRSFixProvider_CannotHandleNonCQRS(t *testing.T) {
	provider := fix.NewCQRSFixProvider()

	f, _ := finding.NewBuilder(
		"OTHER", "other-tool", "some issue",
		finding.SeverityWarning, finding.Pos("test.go", 1, 1),
	).Build()

	if provider.CanHandle(f) {
		t.Fatal("CanHandle() should return false for non-cqrs-lint finding")
	}
}

func applyEdit(content []byte, edit pipeline.FixEdit) []byte {
	result := make([]byte, 0, len(content)-edit.Length+len(edit.Replacement))
	result = append(result, content[:edit.Offset]...)
	result = append(result, edit.Replacement...)
	result = append(result, content[edit.Offset+edit.Length:]...)

	return result
}
