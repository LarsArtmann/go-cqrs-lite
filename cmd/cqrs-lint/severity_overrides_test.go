package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
)

func sevFinding(t *testing.T, rule string, sev finding.Severity) finding.Finding {
	t.Helper()

	f, err := finding.NewBuilder(
		finding.RuleName(rule), "cqrs-lint", "test message",
		sev, finding.Pos("test.go", 1, 1),
	).Build()
	if err != nil {
		t.Fatalf("build finding: %v", err)
	}
	return f
}

func TestApplySeverityOverrides_RewritesSeverity(t *testing.T) {
	t.Parallel()

	findings := []finding.Finding{
		sevFinding(t, "V007", finding.SeverityWarning),
		sevFinding(t, "S001", finding.SeverityWarning),
	}

	got := applySeverityOverrides(findings, map[string]string{"V007": "error"})

	if got[0].Severity != finding.SeverityError {
		t.Errorf("V007 severity = %v, want error", got[0].Severity)
	}
	if !strings.Contains(got[0].Message, "[severity overridden: error]") {
		t.Errorf("V007 message should carry the override marker, got %q", got[0].Message)
	}
	if got[1].Severity != finding.SeverityWarning {
		t.Errorf("S001 must be untouched, got %v", got[1].Severity)
	}
}

func TestApplySeverityOverrides_SameSeverityNoMarker(t *testing.T) {
	t.Parallel()

	findings := []finding.Finding{sevFinding(t, "V007", finding.SeverityError)}

	got := applySeverityOverrides(findings, map[string]string{"V007": "error"})

	if strings.Contains(got[0].Message, "severity overridden") {
		t.Errorf("no marker expected when severity is unchanged, got %q", got[0].Message)
	}
}

func TestApplySeverityOverrides_NilMapIsIdentity(t *testing.T) {
	t.Parallel()

	findings := []finding.Finding{sevFinding(t, "V007", finding.SeverityWarning)}

	got := applySeverityOverrides(findings, nil)

	if got[0].Severity != finding.SeverityWarning {
		t.Errorf("nil overrides must not change severity, got %v", got[0].Severity)
	}
}

func TestApplySeverityOverrides_RuleIDCaseInsensitive(t *testing.T) {
	t.Parallel()

	findings := []finding.Finding{sevFinding(t, "V007", finding.SeverityWarning)}

	got := applySeverityOverrides(findings, map[string]string{"v007": "critical"})

	if got[0].Severity != finding.SeverityCritical {
		t.Errorf("lowercase override key must match, got %v", got[0].Severity)
	}
}

func TestMergeSeverityOverrides_LaterWins(t *testing.T) {
	t.Parallel()

	base := map[string]string{"V007": "error", "S001": "warning"}
	override := map[string]string{"V007": "critical", "C008": "error"}

	merged := mergeSeverityOverrides(base, override)

	if merged["V007"] != "critical" {
		t.Errorf("override must win: got V007=%q", merged["V007"])
	}
	if merged["S001"] != "warning" || merged["C008"] != "error" {
		t.Errorf("merge lost entries: %v", merged)
	}
	if base["V007"] != "error" {
		t.Errorf("base map must not be mutated, got %v", base)
	}
}

func TestMergeSeverityOverrides_EmptySides(t *testing.T) {
	t.Parallel()

	base := map[string]string{"V007": "error"}
	if got := mergeSeverityOverrides(base, nil); len(got) != 1 {
		t.Errorf("nil override must return base, got %v", got)
	}
	if got := mergeSeverityOverrides(nil, base); len(got) != 1 {
		t.Errorf("nil base must return override, got %v", got)
	}
	if got := mergeSeverityOverrides(nil, nil); got != nil {
		t.Errorf("both empty must be nil, got %v", got)
	}
}

func TestValidateSeverityOverrideRuleIDs_WarnsOnUnknown(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	validateSeverityOverrideRuleIDs(&buf, map[string]string{
		"V007": "error",
		"C999": "error", // unknown rule ID
	})

	out := buf.String()
	if !strings.Contains(out, "C999") {
		t.Errorf("warning must name the unknown rule ID, got: %s", out)
	}
	if strings.Contains(out, "V007") {
		t.Errorf("known rule IDs must not warn, got: %s", out)
	}
}

func TestV5ReadyPreset_ConfigRoundTrip(t *testing.T) {
	t.Parallel()

	def := analyzer.ResolvePresetDefinition(analyzer.PresetV5Ready)
	if def.Rules.SeverityOverrides["V007"] != "error" {
		t.Fatalf("v5-ready preset must escalate V007 to error")
	}
}
