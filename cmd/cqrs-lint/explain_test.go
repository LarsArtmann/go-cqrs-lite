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
