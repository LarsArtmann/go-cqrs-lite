package suppression_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/suppression"
	"github.com/larsartmann/go-finding"
)

func TestDetectStaleSuppressions_FindsStaleComment(t *testing.T) {
	tmp := t.TempDir()
	src := filepath.Join(tmp, "example.go")
	_ = os.WriteFile(
		src,
		[]byte("package main\n\n//cqrs-lint:ignore(D002)\ntype Foo struct{}\n"),
		0o644,
	)

	findings := []finding.Finding{} // no findings → suppression is stale

	stale := suppression.DetectStaleSuppressions([]string{src}, findings)
	if len(stale) != 1 {
		t.Fatalf("got %d stale suppressions, want 1", len(stale))
	}

	if stale[0].Rule != "D002" {
		t.Errorf("rule = %s, want D002", stale[0].Rule)
	}

	if stale[0].Line != 3 {
		t.Errorf("line = %d, want 3", stale[0].Line)
	}
}

// TestDetectStaleSuppressions_IgnoresDocStringMentions ensures documentation
// strings and doc comments that merely mention the suppression syntax are NOT
// reported as stale suppressions. This guards against false positives where a
// help text string or a godoc comment contains "//cqrs-lint:ignore(RULE)" or
// "//cqrs-lint:ignore-start" as illustrative text rather than a real directive.
func TestDetectStaleSuppressions_IgnoresDocStringMentions(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	src := filepath.Join(tmp, "example.go")

	content := `package main

import "fmt"

// countSuppressions scans for //cqrs-lint:ignore(RULE) comments in the source.
func help() {
	fmt.Println("Suppress with //cqrs-lint:ignore(RULE)")
	fmt.Println("    //cqrs-lint:ignore-start ... //cqrs-lint:ignore-end")
}

type Foo struct{}
`
	if err := os.WriteFile(src, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	// With zero findings every REAL suppression would be stale. Asserting ZERO
	// stale reports proves none of the doc strings / godoc mentions were
	// mistaken for real directives.
	stale := suppression.DetectStaleSuppressions([]string{src}, nil)
	if len(stale) != 0 {
		t.Errorf(
			"doc strings/comments should not be flagged as stale; got %d: %v",
			len(stale),
			stale,
		)
	}
}

func TestDetectStaleSuppressions_NoStaleWhenFindingMatches(t *testing.T) {
	tmp := t.TempDir()
	src := filepath.Join(tmp, "example.go")
	_ = os.WriteFile(
		src,
		[]byte("package main\n\n//cqrs-lint:ignore(D002)\ntype Foo struct{}\n"),
		0o644,
	)

	findings := []finding.Finding{
		{
			Rule: "D002",
			Position: finding.Position{
				File: finding.FilePath(src),
				Line: 4,
			},
		},
	}

	stale := suppression.DetectStaleSuppressions([]string{src}, findings)
	if len(stale) != 0 {
		t.Fatalf("got %d stale suppressions, want 0 (finding matches)", len(stale))
	}
}

func TestDetectStaleSuppressions_MatchesOnSameLine(t *testing.T) {
	tmp := t.TempDir()
	src := filepath.Join(tmp, "example.go")
	_ = os.WriteFile(
		src,
		[]byte("package main\n\ntype Foo struct{} //cqrs-lint:ignore(D002)\n"),
		0o644,
	)

	findings := []finding.Finding{
		{Rule: "D002", Position: finding.Position{File: finding.FilePath(src), Line: 3}},
	}

	stale := suppression.DetectStaleSuppressions([]string{src}, findings)
	if len(stale) != 0 {
		t.Fatalf("got %d stale, want 0 (same-line match)", len(stale))
	}
}

func TestDetectStaleSuppressions_CombinedDirectivePartialMatch(t *testing.T) {
	tmp := t.TempDir()
	src := filepath.Join(tmp, "example.go")
	_ = os.WriteFile(
		src,
		[]byte("package main\n\n//cqrs-lint:ignore(A001,B002)\ntype Foo struct{}\n"),
		0o644,
	)

	findings := []finding.Finding{
		{Rule: "A001", Position: finding.Position{File: finding.FilePath(src), Line: 4}},
	}

	stale := suppression.DetectStaleSuppressions([]string{src}, findings)
	if len(stale) != 0 {
		t.Fatalf(
			"got %d stale, want 0 (combined directive has at least one matching rule)",
			len(stale),
		)
	}
}

func TestDetectStaleSuppressions_CombinedDirectiveNoMatch(t *testing.T) {
	tmp := t.TempDir()
	src := filepath.Join(tmp, "example.go")
	_ = os.WriteFile(
		src,
		[]byte("package main\n\n//cqrs-lint:ignore(A001,B002)\ntype Foo struct{}\n"),
		0o644,
	)

	findings := []finding.Finding{}

	stale := suppression.DetectStaleSuppressions([]string{src}, findings)
	if len(stale) != 2 {
		t.Fatalf("got %d stale, want 2 (no rule in the combined directive matches)", len(stale))
	}
}

func TestDetectStaleBlocks_StaleBlock(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	src := filepath.Join(tmp, "example.go")
	_ = os.WriteFile(src, []byte(`package main

//cqrs-lint:ignore-start
type Foo struct{}
type Bar struct{}
//cqrs-lint:ignore-end
`), 0o644)

	stale := suppression.DetectStaleSuppressions([]string{src}, nil)
	if len(stale) != 1 {
		t.Fatalf("got %d stale, want 1 (block has no findings)", len(stale))
	}

	if stale[0].Rule != "block:all" {
		t.Errorf("rule = %s, want block:all", stale[0].Rule)
	}
}

func TestDetectStaleBlocks_NotStaleWhenFindingInside(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	src := filepath.Join(tmp, "example.go")
	_ = os.WriteFile(src, []byte(`package main

//cqrs-lint:ignore-start
type Foo struct{}
type Bar struct{}
//cqrs-lint:ignore-end
`), 0o644)

	findings := []finding.Finding{
		{Rule: "A001", Position: finding.Position{File: finding.FilePath(src), Line: 5}},
	}

	stale := suppression.DetectStaleSuppressions([]string{src}, findings)
	if len(stale) != 0 {
		t.Fatalf("got %d stale, want 0 (finding inside block)", len(stale))
	}
}

func TestDetectStaleBlocks_PerRuleStaleBlock(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	src := filepath.Join(tmp, "example.go")
	_ = os.WriteFile(src, []byte(`package main

//cqrs-lint:ignore-start(A001)
type Foo struct{}
//cqrs-lint:ignore-end
`), 0o644)

	// A002 fires inside the block, but the block only suppresses A001
	findings := []finding.Finding{
		{Rule: "A002", Position: finding.Position{File: finding.FilePath(src), Line: 5}},
	}

	stale := suppression.DetectStaleSuppressions([]string{src}, findings)
	if len(stale) != 1 {
		t.Fatalf("got %d stale, want 1 (A001 suppressed but never fires)", len(stale))
	}

	if stale[0].Rule != "block:A001" {
		t.Errorf("rule = %s, want block:A001", stale[0].Rule)
	}
}

func TestAuditSuppressions_ClassifiesAllStatuses(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	src := filepath.Join(tmp, "example.go")

	content := `package main

//cqrs-lint:ignore(C007) this one is active
type Handler struct{}

//cqrs-lint:ignore(D002) this one is stale
type Config struct{}

//cqrs-lint:ignore(X999) this one has an unknown rule
type Legacy struct{}
`
	_ = os.WriteFile(src, []byte(content), 0o644)

	findings := []finding.Finding{
		{Rule: "C007", Position: finding.Position{File: finding.FilePath(src), Line: 4}},
	}

	knownRuleIDs := map[string]bool{
		"C007": true,
		"D002": true,
	}

	entries := suppression.AuditSuppressions([]string{src}, findings, knownRuleIDs)
	if len(entries) != 3 {
		t.Fatalf("got %d entries, want 3", len(entries))
	}

	statusByRule := make(map[string]suppression.AuditStatus, len(entries))
	for _, e := range entries {
		statusByRule[e.Rule] = e.Status
	}

	if statusByRule["C007"] != suppression.AuditActive {
		t.Errorf("C007 status = %s, want active", statusByRule["C007"])
	}
	if statusByRule["D002"] != suppression.AuditStale {
		t.Errorf("D002 status = %s, want stale", statusByRule["D002"])
	}
	if statusByRule["X999"] != suppression.AuditUnknownRule {
		t.Errorf("X999 status = %s, want unknown-rule", statusByRule["X999"])
	}
}

func TestAuditSuppressions_EmptyFiles(t *testing.T) {
	t.Parallel()

	entries := suppression.AuditSuppressions(nil, nil, nil)
	if len(entries) != 0 {
		t.Errorf("got %d entries, want 0", len(entries))
	}
}

// TestDetectStaleSuppressions_BlankLineAboveCommentNotStale mirrors the
// suppression filter's blank-skip: a suppression comment separated from the
// finding by a blank line IS honored, so it must NOT be reported stale.
func TestDetectStaleSuppressions_BlankLineAboveCommentNotStale(t *testing.T) {
	tmp := t.TempDir()
	src := filepath.Join(tmp, "example.go")
	// comment at line 3, blank at line 4, finding at line 5.
	_ = os.WriteFile(
		src,
		[]byte("package main\n\n//cqrs-lint:ignore(D002)\n\ntype Foo struct{}\n"),
		0o644,
	)

	findings := []finding.Finding{
		{Rule: "D002", Position: finding.Position{File: finding.FilePath(src), Line: 5}},
	}

	stale := suppression.DetectStaleSuppressions([]string{src}, findings)
	if len(stale) != 0 {
		t.Fatalf("working suppression reported stale: %+v", stale)
	}
}

// TestDetectStaleSuppressions_NonBlankLineBetweenCommentAndFindingIsStale:
// when real code sits between the comment and the finding, the suppression
// filter does NOT honor the comment, so it must be reported stale.
func TestDetectStaleSuppressions_NonBlankLineBetweenCommentAndFindingIsStale(t *testing.T) {
	tmp := t.TempDir()
	src := filepath.Join(tmp, "example.go")
	// comment at line 3, unrelated code at line 4, finding at line 5.
	_ = os.WriteFile(
		src,
		[]byte("package main\n\n//cqrs-lint:ignore(D002)\nvar x = 1\ntype Foo struct{}\n"),
		0o644,
	)

	findings := []finding.Finding{
		{Rule: "D002", Position: finding.Position{File: finding.FilePath(src), Line: 5}},
	}

	stale := suppression.DetectStaleSuppressions([]string{src}, findings)
	if len(stale) != 1 {
		t.Fatalf("got %d stale suppressions, want 1: %+v", len(stale), stale)
	}
}
