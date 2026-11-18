package main

import (
	"maps"

	"github.com/larsartmann/go-finding"
)

// mergeStringSlices concatenates two slices into a fresh allocation. Used to
// merge preset defaults with config overrides without aliasing backing arrays.
func mergeStringSlices(preset, config []string) []string {
	result := make([]string, 0, len(preset)+len(config))
	result = append(result, preset...)
	result = append(result, config...)
	return result
}

// mergeSeverityOverrides merges two severity-override maps (base = preset or
// parent config, override = local config). Later wins per rule ID; inputs are
// never mutated. Returns an input unchanged when the other is empty.
func mergeSeverityOverrides(base, override map[string]string) map[string]string {
	if len(base) == 0 {
		return override
	}
	if len(override) == 0 {
		return base
	}

	merged := make(map[string]string, len(base)+len(override))
	maps.Copy(merged, base)
	maps.Copy(merged, override)
	return merged
}

// resolveMinSeverity applies the preset severity floor to the user's min-severity
// setting. The preset's value acts as a lower bound: users can raise it (e.g.
// "error" is stricter than "warning") but cannot go below the floor (e.g.
// local-cli's "warning" hides info-level noise for single-user CLI tools).
// When the preset has no severity floor (empty string), the user's value is
// returned unchanged.
func resolveMinSeverity(presetMinSeverity, userMinSeverity string) string {
	if presetMinSeverity == "" {
		return userMinSeverity
	}

	presetSev, err := finding.ParseSeverity(presetMinSeverity)
	if err != nil {
		return userMinSeverity
	}

	userSev, err := finding.ParseSeverity(userMinSeverity)
	if err != nil {
		return presetMinSeverity
	}

	if presetSev.GreaterThan(userSev) {
		return presetMinSeverity
	}

	return userMinSeverity
}

// isStrictMode reports whether either --strict or --strict-load is set.
// --strict is cmdguard's built-in strict-mode flag; --strict-load is the
// cqrs-lint-specific alias. Both cause load errors to fail the run.
func isStrictMode(cfg *AppConfig) bool {
	return cfg.StrictLoad || cfg.StrictMode
}
