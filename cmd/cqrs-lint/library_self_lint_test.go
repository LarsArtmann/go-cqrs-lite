package main

import (
	"testing"

	"github.com/larsartmann/go-finding"
)

func TestFilterLibrarySelfLint_ConsumerProject(t *testing.T) {
	t.Parallel()

	findings := []finding.Finding{
		mustFinding("A001", "manual command interface"),
		mustFinding("C001", "missing tx commit"),
		mustFinding("E005", "uncataloged event"),
	}

	active, suppressed := filterLibrarySelfLint(findings, false)
	if len(suppressed) != 0 {
		t.Errorf("expected 0 suppressed for consumer project, got %d", len(suppressed))
	}
	if len(active) != 3 {
		t.Errorf("expected 3 active for consumer project, got %d", len(active))
	}
}

func TestFilterLibrarySelfLint_LibraryProject(t *testing.T) {
	t.Parallel()

	findings := []finding.Finding{
		mustFinding("A001", "manual command interface"),
		mustFinding("C001", "missing tx commit"),
		mustFinding("A020", "custom event bus"),
		mustFinding("E005", "uncataloged event"),
		mustFinding("S001", "hardcoded secret"),
	}

	active, suppressed := filterLibrarySelfLint(findings, true)
	if len(active) != 2 {
		t.Errorf("expected 2 active (C001 + S001), got %d", len(active))
	}
	if len(suppressed) != 3 {
		t.Errorf("expected 3 suppressed (A001 + A020 + E005), got %d", len(suppressed))
	}

	for _, s := range suppressed {
		if s.Suppression == nil {
			t.Errorf("suppressed finding %s has nil Suppression", s.Rule)
		}
	}
}

func TestFilterLibrarySelfLint_AllConsumerRulesSuppressed(t *testing.T) {
	t.Parallel()

	var findings []finding.Finding
	for _, rule := range []string{"A001", "A008", "A020", "A021", "A023", "E005", "E007"} {
		findings = append(findings, mustFinding(rule, "test"))
	}

	active, suppressed := filterLibrarySelfLint(findings, true)
	if len(active) != 0 {
		t.Errorf("expected 0 active in library mode, got %d", len(active))
	}
	if len(suppressed) != 7 {
		t.Errorf("expected 7 suppressed, got %d", len(suppressed))
	}
}

func mustFinding(rule, msg string) finding.Finding {
	f, err := finding.NewBuilder(
		finding.RuleName(rule), "cqrs-lint", msg,
		finding.SeverityWarning, finding.Pos("test.go", 1, 1),
	).Build()
	if err != nil {
		panic(err)
	}
	return f
}
