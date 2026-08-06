package main

import (
	"strings"
	"testing"
)

func TestRenderExplain_ContainsAllSections(t *testing.T) {
	t.Parallel()

	output := renderExplain()

	requiredSections := []string{
		"CONFIG FILE",
		"TOP-LEVEL KEYS",
		"PRESETS",
		"FEATURES",
		"RULES",
		"HEALTH",
		"RESOLUTION ORDER",
		"SUPPRESSION",
	}

	for _, section := range requiredSections {
		if !strings.Contains(output, section) {
			t.Errorf("renderExplain() output missing section %q", section)
		}
	}
}

func TestRenderExplain_ContainsAllPresetDescriptions(t *testing.T) {
	t.Parallel()

	output := renderExplain()

	for preset, desc := range presetDescriptions {
		if !strings.Contains(output, string(preset)) {
			t.Errorf("renderExplain() output missing preset %q", preset)
		}
		if desc != "" && !strings.Contains(output, desc) {
			t.Errorf("renderExplain() output missing description for preset %q: %q", preset, desc)
		}
	}
}

func TestRenderExplain_ContainsJSONCInfo(t *testing.T) {
	t.Parallel()

	output := renderExplain()

	if !strings.Contains(output, "JSONC") && !strings.Contains(output, "JSON with Comments") {
		t.Error("renderExplain() should mention JSONC / JSON with Comments support")
	}
	if !strings.Contains(output, "line comment") {
		t.Error("renderExplain() should document line comment support")
	}
}

// TestFeatureKeys_DerivedValidValuesPopulated guards the init() that derives
// store/command-flow/tracing/snapshot/domain valid values from Kind constants.
// Each derived featureKey carries a non-nil derive field (a closure over the
// All*Kind() enumerator). After init, every feature key must have non-empty
// validValues — a nil derive or a broken enumerator would leave it blank.
func TestFeatureKeys_DerivedValidValuesPopulated(t *testing.T) {
	t.Parallel()

	// Every string-typed feature key should have a derive closure.
	derivedKeys := []string{"store", "command-flow", "tracing", "snapshot", "domain"}
	for _, key := range derivedKeys {
		fk := findFeatureKey(key)
		if fk == nil {
			t.Fatalf("feature key %q not found", key)
		}
		if fk.derive == nil {
			t.Errorf("feature key %q is missing its derive closure", key)
		}
	}

	// After init(), no feature key should have empty validValues.
	for _, fk := range featureKeys {
		if len(fk.validValues) == 0 {
			t.Errorf(
				"feature key %q has empty validValues after init() — derivation likely failed",
				fk.key,
			)
		}
	}

	// Spot-check that the derived values contain known constants.
	storeEntry := findFeatureKey("store")
	if storeEntry == nil {
		t.Fatal("store feature key not found")
	}
	for _, want := range []string{"sqlite", "postgres", "duckdb"} {
		if !strings.Contains(strings.Join(storeEntry.validValues, ","), want) {
			t.Errorf("store validValues should contain %q, got %v", want, storeEntry.validValues)
		}
	}
}

func findFeatureKey(key string) *featureKey {
	for i, fk := range featureKeys {
		if fk.key == key {
			return &featureKeys[i]
		}
	}
	return nil
}
