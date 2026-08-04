package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

func TestValidatePresetName_KnownPresetsNoWarning(t *testing.T) {
	t.Parallel()

	for _, name := range analyzer.ValidPresetNames() {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			validatePresetName(&buf, analyzer.ConfigPreset(name))
			if buf.Len() > 0 {
				t.Errorf("expected no warning for known preset %q, got: %s", name, buf.String())
			}
		})
	}
}

func TestValidatePresetName_EmptyNoWarning(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	validatePresetName(&buf, analyzer.PresetNone)
	if buf.Len() > 0 {
		t.Errorf("expected no warning for empty preset, got: %s", buf.String())
	}
}

func TestValidatePresetName_UnknownWarns(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	validatePresetName(&buf, analyzer.ConfigPreset("server"))
	output := buf.String()
	if !strings.Contains(output, "unknown preset") {
		t.Errorf("expected 'unknown preset' warning, got: %q", output)
	}
	if !strings.Contains(output, "local-cli") {
		t.Errorf("expected available preset names in warning, got: %q", output)
	}
}

func TestValidateDisabledRuleIDs_KnownIDsNoWarning(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	validateDisabledRuleIDs(&buf, []string{"C001", "A001", "E003"})
	if buf.Len() > 0 {
		t.Errorf("expected no warnings for known rule IDs, got: %s", buf.String())
	}
}

func TestValidateDisabledRuleIDs_UnknownIDWarns(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	validateDisabledRuleIDs(&buf, []string{"C999"})
	output := buf.String()
	if !strings.Contains(output, "C999") {
		t.Errorf("expected warning mentioning C999, got: %q", output)
	}
	if !strings.Contains(output, "not a known rule") {
		t.Errorf("expected 'not a known rule' in warning, got: %q", output)
	}
}

func TestValidateDisabledRuleIDs_EmptyListNoWarning(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	validateDisabledRuleIDs(&buf, nil)
	if buf.Len() > 0 {
		t.Errorf("expected no warnings for nil list, got: %s", buf.String())
	}
}
