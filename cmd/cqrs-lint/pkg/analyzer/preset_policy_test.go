package analyzer

import (
	"slices"
	"testing"
)

// TestPreset_DeprecatedSurfacePolicy locks the V007/F030 on/off policy:
// library and framework presets disable the two "deprecated surface"
// detectors (a library legitimately re-exports APIs that v5 removes while
// v4 still ships them — same rationale as IsLibrarySelfLint), while the
// application-facing presets and the default keep them enabled.
func TestPreset_DeprecatedSurfacePolicy(t *testing.T) {
	t.Parallel()

	disabled := func(name ConfigPreset, ruleID string) bool {
		return slices.Contains(ResolvePresetDefinition(name).Rules.Disable, ruleID)
	}

	for _, p := range []ConfigPreset{PresetLibrary, PresetLibraryFramework} {
		if !disabled(p, "V007") {
			t.Errorf("preset %q must disable V007 (compat-surface self-reference)", p)
		}

		if !disabled(p, "F030") {
			t.Errorf("preset %q must disable F030 (deprecated transport adoption)", p)
		}
	}

	for _, p := range []ConfigPreset{PresetNone, PresetProduction, PresetLocalCLI, PresetReadOnly} {
		if disabled(p, "V007") {
			t.Errorf("preset %q must keep V007 enabled for application code", p)
		}

		if disabled(p, "F030") {
			t.Errorf("preset %q must keep F030 enabled for application code", p)
		}
	}
}
