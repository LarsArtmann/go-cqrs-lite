package main

import "testing"

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
