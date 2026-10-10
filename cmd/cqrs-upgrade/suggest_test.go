package main

import (
	"strings"
	"testing"
)

// TestSuggestions_EventuallyLoopFlagged pins the advisory matcher end to
// end: the eventually fixture (deadline condition + sleep polling) yields
// exactly one suggest:then-query hint on the await loop, while the
// backoff loop in the same module (sleep without a deadline condition)
// stays silent. Suggestions are report-only — the scan result proves the
// pipeline surfaces them without touching --strict semantics.
func TestSuggestions_EventuallyLoopFlagged(t *testing.T) {
	dir := setupFixture(t, "eventually")

	deprecations, suggestions, err := scanFindings(dir)
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}

	if len(deprecations) != 0 {
		t.Fatalf("fixture has no v5-removed API usage, got %d deprecations", len(deprecations))
	}

	if len(suggestions) != 1 {
		t.Fatalf("expected exactly 1 suggestion (the await loop), got %d: %+v",
			len(suggestions), suggestions)
	}

	got := suggestions[0]
	if got.Rule != ruleSuggestThenQuery {
		t.Errorf("rule mismatch: want %s, got %s", ruleSuggestThenQuery, got.Rule)
	}

	if !strings.Contains(got.Position, "await_test.go") {
		t.Errorf("position should anchor at the await loop's file, got %s", got.Position)
	}

	if !strings.Contains(got.Message, "ThenQuery") {
		t.Errorf("message should name the harness assertion, got %s", got.Message)
	}
}

// TestSuggestions_NeverGatesStrict pins the advisory contract end to end:
// a module whose only finding is a suggestion exits --strict clean (it is
// not a v5 violation) while the suggestion still appears in the JSON wire.
func TestSuggestions_NeverGatesStrict(t *testing.T) {
	dir := setupFixture(t, "eventually")

	stdout := captureStdout(t)

	if err := run(t.Context(), []string{"--json", "--strict", "--no-build", dir}); err != nil {
		t.Fatalf("--strict must pass on suggestion-only modules, got: %v", err)
	}

	out := stdout()

	modules := decodeWire(t, out)
	if len(modules) != 1 {
		t.Fatalf("expected 1 module in wire, got %d", len(modules))
	}

	if len(modules[0].Suggestions) != 1 {
		t.Errorf("wire must carry the suggestion, got %d", len(modules[0].Suggestions))
	}
}
