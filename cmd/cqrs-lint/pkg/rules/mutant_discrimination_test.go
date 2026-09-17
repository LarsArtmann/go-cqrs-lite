package rules

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeMutantRule renders a syntactically valid rule file with the given
// severity into a fresh temp tree for scanBuilderDeclsFrom.
func writeMutantRule(t *testing.T, sev string) string {
	t.Helper()

	dir := t.TempDir()
	src := `package correctness

import "github.com/larsartmann/go-finding"

func mutant() {
	b := findingTemplate.Builder(
		"C003",
		"mutant message",
		` + sev + `,
		finding.Pos("x.go", 1, 1),
	)
	_ = b
}
`
	if err := os.WriteFile(filepath.Join(dir, "mutant.go"), []byte(src), 0o600); err != nil {
		t.Fatal(err)
	}

	return dir
}

// TestMutantRuleFailsMetaTest is the discrimination proof: a deliberately
// broken rule (right rule ID, wrong severity) MUST be flagged by the same
// scanner + catalog-check machinery that guards the real rules, and a
// correctly-severitied twin must pass clean. If this test ever fails, the
// severity meta-test has gone blind — it would pass a rule suite that no
// longer matches the catalog.
func TestMutantRuleFailsMetaTest(t *testing.T) {
	t.Parallel()

	entry, ok := LookupRule("C003")
	if !ok {
		t.Fatal("C003 missing from catalog — fixture invalid")
	}

	mutantSev := "finding.SeverityInfo"
	if entry.Severity == "info" {
		mutantSev = "finding.SeverityCritical"
	}

	// The mutant must be flagged.
	mutantDir := writeMutantRule(t, mutantSev)
	decls := scanBuilderDeclsFrom(t, mutantDir)
	if len(decls) != 1 {
		t.Fatalf("scanner found %d decls in mutant tree, want 1", len(decls))
	}

	problems := severityProblems(decls)
	if len(problems) == 0 {
		t.Fatal("mutant rule (wrong severity) was NOT flagged — meta-test is blind")
	}

	if !strings.Contains(problems[0], "C003") {
		t.Fatalf("flagged problem does not name the mutant rule: %q", problems[0])
	}

	// The healthy twin must be clean.
	healthyDir := writeMutantRule(t, "finding."+severityLiteralFor(entry.Severity))
	healthy := scanBuilderDeclsFrom(t, healthyDir)
	if len(healthy) != 1 {
		t.Fatalf("scanner found %d decls in healthy tree, want 1", len(healthy))
	}

	if problems := severityProblems(healthy); len(problems) != 0 {
		t.Fatalf("healthy rule incorrectly flagged: %v", problems)
	}
}

// severityProblems applies the severity half of TestRuleSeverityMatchesCatalog's
// catalog check to arbitrary decls.
func severityProblems(decls []builderDecl) []string {
	var problems []string

	for _, d := range decls {
		entry, ok := LookupRule(d.rule)
		if !ok {
			problems = append(problems, d.pos+": "+d.rule+" has no catalog entry")
			continue
		}

		if sev, isLiteral := sevLiterals[d.sev]; isLiteral && sev != entry.Severity {
			problems = append(problems, d.pos+": "+d.rule+" emits "+sev+
				" but catalog says "+entry.Severity)
		}
	}

	return problems
}

// severityLiteralFor maps a catalog severity string to its finding-package
// literal ("critical" -> "SeverityCritical").
func severityLiteralFor(sev string) string {
	return "Severity" + strings.ToUpper(sev[:1]) + sev[1:]
}
