package suppression_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/suppression"
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
