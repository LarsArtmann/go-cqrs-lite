package analyzer

import (
	"bytes"
	"strings"
	"testing"
)

func TestRulesConfig_Validate_NormalizesPrefixes(t *testing.T) {
	t.Parallel()

	rc := &RulesConfig{
		ExternalAPIStructPrefixes: []string{"  Discord  ", "", "Stripe", "Discord", "discord"},
	}
	var buf bytes.Buffer
	rc.Validate(&buf, nil)

	want := []string{"Discord", "Stripe", "discord"}
	if len(rc.ExternalAPIStructPrefixes) != len(want) {
		t.Fatalf("got %v, want %v", rc.ExternalAPIStructPrefixes, want)
	}
	for i, p := range want {
		if rc.ExternalAPIStructPrefixes[i] != p {
			t.Errorf("prefix[%d] = %q, want %q", i, rc.ExternalAPIStructPrefixes[i], p)
		}
	}
}

func TestRulesConfig_Validate_WarnsOnUnknownKey(t *testing.T) {
	t.Parallel()

	rc := &RulesConfig{}
	raw := []byte(`{"external-api-prefixes": ["Discord"]}`)
	var buf bytes.Buffer
	rc.Validate(&buf, raw)

	out := buf.String()
	if !strings.Contains(out, "unknown rules config key") {
		t.Errorf("expected unknown-key warning, got: %s", out)
	}
	if !strings.Contains(out, "external-api-prefixes") {
		t.Errorf("warning should name the unknown key, got: %s", out)
	}
}

func TestRulesConfig_Validate_NoWarningForKnownKey(t *testing.T) {
	t.Parallel()

	rc := &RulesConfig{}
	raw := []byte(`{"external-api-struct-prefixes": ["Discord"]}`)
	var buf bytes.Buffer
	rc.Validate(&buf, raw)

	if buf.Len() > 0 {
		t.Errorf("expected no warnings for known key, got: %s", buf.String())
	}
}

func TestRulesConfig_Validate_NilIsSafe(t *testing.T) {
	t.Parallel()

	var rc *RulesConfig
	var buf bytes.Buffer
	rc.Validate(&buf, nil)

	if buf.Len() > 0 {
		t.Errorf("expected no output for nil config, got: %s", buf.String())
	}
}

func TestRulesConfig_Validate_NormalizesSeverityOverrides(t *testing.T) {
	t.Parallel()

	rc := &RulesConfig{
		SeverityOverrides: map[string]string{
			"  v007 ": " ERROR ",
			"c008": "warning",
			"":     "error",
			"S001": "",
		},
	}
	var buf bytes.Buffer
	rc.Validate(&buf, nil)

	if buf.Len() > 0 {
		t.Errorf("expected no warnings, got: %s", buf.String())
	}
	if len(rc.SeverityOverrides) != 2 {
		t.Fatalf("expected 2 normalized overrides, got %d: %v",
			len(rc.SeverityOverrides), rc.SeverityOverrides)
	}
	if rc.SeverityOverrides["V007"] != "error" || rc.SeverityOverrides["C008"] != "warning" {
		t.Errorf("normalized overrides wrong: %v", rc.SeverityOverrides)
	}
}

func TestRulesConfig_Validate_DropsInvalidSeverityWithWarning(t *testing.T) {
	t.Parallel()

	rc := &RulesConfig{
		SeverityOverrides: map[string]string{
			"V007": "fatal", // unknown severity — must be dropped, not demoted to info
			"S001": "error",
		},
	}
	var buf bytes.Buffer
	rc.Validate(&buf, nil)

	out := buf.String()
	if !strings.Contains(out, "invalid severity") || !strings.Contains(out, "fatal") {
		t.Errorf("expected invalid-severity warning naming %q, got: %s", "fatal", out)
	}
	if _, ok := rc.SeverityOverrides["V007"]; ok {
		t.Errorf("invalid override for V007 must be dropped, got: %v", rc.SeverityOverrides)
	}
	if rc.SeverityOverrides["S001"] != "error" {
		t.Errorf("valid override for S001 must survive, got: %v", rc.SeverityOverrides)
	}
}
