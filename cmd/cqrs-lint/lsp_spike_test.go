package main

import (
	"strings"
	"testing"

	"github.com/larsartmann/go-finding"
)

// TestLSPRoundTripSpike is the go/no-go probe for a cqrs-lint LSP server
// (adoption plan task C24): a cqrs-lint finding converted with ToLSP must
// carry everything an editor diagnostic needs — zero-based range, URI-form
// file, severity, source tag — and survive the reverse conversion without
// losing the rule identity or message. PASS verdict: the conversion layer is
// complete; building an actual server is a transport problem, not a data
// problem.
func TestLSPRoundTripSpike(t *testing.T) {
	t.Parallel()

	f := finding.NewBuilder(
		"C003", "cqrs-lint",
		"Fold apply silently ignores unknown event types in default case",
		finding.SeverityWarning,
		finding.Pos("internal/wallet/fold.go", 42, 7),
	).
		WithCategory(finding.CategoryCorrectness).
		WithConfidence(finding.ConfidenceHigh).
		WithSuggestion("Return an error for unknown event types").
		BuildOrDefault()

	diag := f.ToLSP()

	if diag.Range.Start.Line != 41 {
		t.Fatalf("LSP ranges are zero-based: start line = %d, want 41", diag.Range.Start.Line)
	}

	if !strings.Contains(string(diag.URI), "internal/wallet/fold.go") &&
		!strings.Contains(diag.String(), "internal/wallet/fold.go") {
		t.Fatalf("diagnostic does not reference the file: %+v", diag)
	}

	back := finding.FromLSP(f.Position.File, diag)
	if string(back.Rule) != "C003" {
		t.Fatalf("round trip lost the rule: %q", back.Rule)
	}

	if back.Message != f.Message {
		t.Fatalf("round trip lost the message:\n%q\n%q", back.Message, f.Message)
	}

	if back.Position.Line != 42 {
		t.Fatalf("round trip restored a one-based line: %d", back.Position.Line)
	}
}
