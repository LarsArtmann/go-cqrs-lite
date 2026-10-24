package main

import (
	"errors"
	"testing"

	"github.com/larsartmann/go-finding"
)

func TestResolveMinSeverity_PresetFloorWins(t *testing.T) {
	t.Parallel()

	got := resolveMinSeverity("warning", "info")
	if got != "warning" {
		t.Errorf("resolveMinSeverity(warning, info) = %q, want warning (preset floor wins)", got)
	}
}

func TestResolveMinSeverity_UserStricterWins(t *testing.T) {
	t.Parallel()

	got := resolveMinSeverity("warning", "error")
	if got != "error" {
		t.Errorf("resolveMinSeverity(warning, error) = %q, want error (user stricter wins)", got)
	}
}

func TestResolveMinSeverity_EqualStaysUserValue(t *testing.T) {
	t.Parallel()

	got := resolveMinSeverity("warning", "warning")
	if got != "warning" {
		t.Errorf("resolveMinSeverity(warning, warning) = %q, want warning", got)
	}
}

func TestResolveMinSeverity_NoPresetFloor(t *testing.T) {
	t.Parallel()

	got := resolveMinSeverity("", "info")
	if got != "info" {
		t.Errorf("resolveMinSeverity(empty, info) = %q, want info", got)
	}
}

func TestResolveMinSeverity_BothEmpty(t *testing.T) {
	t.Parallel()

	got := resolveMinSeverity("", "")
	if got != "" {
		t.Errorf("resolveMinSeverity(empty, empty) = %q, want empty", got)
	}
}

func TestResolveMinSeverity_PresetCriticalUserWarning(t *testing.T) {
	t.Parallel()

	got := resolveMinSeverity("critical", "warning")
	if got != "critical" {
		t.Errorf("resolveMinSeverity(critical, warning) = %q, want critical", got)
	}
}

// TestShouldExitWithError_InfoAdvisoryNeverGates pins the advisory contract
// for info-severity rules (E023 satellite-owned SQL is the reference case):
// info and warning findings must never drive the exit code non-zero — only
// error-and-above does. A consumer running the default floor sees E023 as a
// visible advisory, never a red build.
func TestShouldExitWithError_InfoAdvisoryNeverGates(t *testing.T) {
	t.Parallel()

	cfg := &AppConfig{}

	info := finding.Finding{
		ID: "1", Rule: "E023-satellite-owned-sql", Severity: finding.SeverityInfo,
	}
	if err := shouldExitWithError(cfg, []finding.Finding{info}); err != nil {
		t.Errorf("info-only findings must not gate the exit code, got %v", err)
	}

	infoAndWarning := []finding.Finding{info, {ID: "2", Severity: finding.SeverityWarning}}
	if err := shouldExitWithError(cfg, infoAndWarning); err != nil {
		t.Errorf("info+warning findings must not gate the exit code, got %v", err)
	}

	withError := append(
		append([]finding.Finding{}, infoAndWarning...),
		finding.Finding{ID: "3", Severity: finding.SeverityError},
	)
	if err := shouldExitWithError(cfg, withError); !errors.Is(err, errFindingsWithErrors) {
		t.Errorf("an error-severity finding must gate the exit code, got %v", err)
	}
}
