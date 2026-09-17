package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/output"
)

func groupedFixture() []finding.Finding {
	order := finding.NewBuilder(
		"C019", "cqrs-lint",
		"Multiple Repository instances for Order — wastes singleflight/cache, share one instance",
		finding.SeverityWarning,
		finding.Pos("cmd/a/main.go", 10, 2),
	).
		WithGroupID("c019:Order").
		MustBuild()

	order2 := finding.NewBuilder(
		"C019", "cqrs-lint",
		"Multiple Repository instances for Order — wastes singleflight/cache, share one instance",
		finding.SeverityWarning,
		finding.Pos("cmd/b/other.go", 20, 3),
	).
		WithGroupID("c019:Order").
		MustBuild()

	ungrouped := finding.NewBuilder(
		"C003", "cqrs-lint",
		"Fold apply silently ignores unknown event types in default case",
		finding.SeverityError,
		finding.Pos("internal/x.go", 5, 1),
	).MustBuild()

	return []finding.Finding{ungrouped, order2, order}
}

// TestPrintRelatedGroupsGolden pins the exact text rendering of the GroupID
// summary section: only grouped findings appear, groups sorted by ID, members
// in report order, ungrouped findings never leak into the section.
func TestPrintRelatedGroupsGolden(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	printRelatedGroups(&buf, groupedFixture(), output.ColorModeNever)

	want := strings.Join([]string{
		"Related findings (1 group(s) — findings share a root cause):",
		"  c019:Order (2 finding(s))",
		"    WARNING cmd/b/other.go:20:3  Multiple Repository instances for Order — wastes singleflight/cache, share one instance",
		"    WARNING cmd/a/main.go:10:2  Multiple Repository instances for Order — wastes singleflight/cache, share one instance",
		"",
		"",
	}, "\n")
	if got := buf.String(); got != want {
		t.Fatalf("grouped text mismatch\ngot:\n%q\nwant:\n%q", got, want)
	}
}

// TestPrintRelatedGroupsSkipsWhenNoGroups: no GroupIDs means no section.
func TestPrintRelatedGroupsSkipsWhenNoGroups(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	printRelatedGroups(&buf, groupedFixture()[:1], output.ColorModeNever)

	if buf.Len() != 0 {
		t.Fatalf("expected no output without grouped findings, got %q", buf.String())
	}
}

// TestRelatedGroupsMarkdownGolden pins the markdown variant of the section.
func TestRelatedGroupsMarkdownGolden(t *testing.T) {
	t.Parallel()

	section, ok := relatedGroupsMarkdown(groupedFixture())
	if !ok {
		t.Fatal("expected markdown section for grouped findings")
	}

	want := strings.Join([]string{
		"## Related findings (1 group(s))",
		"",
		"### `c019:Order` (2 finding(s))",
		"",
		"- `cmd/b/other.go:20:3` — Multiple Repository instances for Order — wastes singleflight/cache, share one instance",
		"- `cmd/a/main.go:10:2` — Multiple Repository instances for Order — wastes singleflight/cache, share one instance",
		"",
		"",
	}, "\n")
	if section != want {
		t.Fatalf("grouped markdown mismatch\ngot:\n%q\nwant:\n%q", section, want)
	}

	if _, ok := relatedGroupsMarkdown(groupedFixture()[:1]); ok {
		t.Fatal("expected no markdown section without grouped findings")
	}
}

// TestC019StampsGroupID drives the real C019 detector over a two-Repository
// fixture and asserts both findings carry the c019:<Type> group ID.
func TestC019StampsGroupID(t *testing.T) {
	t.Parallel()

	src := `package main

import "example.com/x/decider"

type Order struct{ N int }

func setup() {
	_ = decider.NewRepository[Order](nil)
	_ = decider.NewRepository[Order](nil)
}
`
	actx, cleanup := analyzer.BuildContextFromTempFiles(t, map[string]string{"main.go": src})
	defer cleanup()

	findings := ruletest.RunDetector(t, correctness.NewC019Detector(actx))
	if len(findings) != 2 {
		t.Fatalf("expected 2 C019 findings, got %d", len(findings))
	}

	for _, f := range findings {
		if f.GroupID != "c019:Order" {
			t.Fatalf("expected group c019:Order, got %q", f.GroupID)
		}
	}
}
