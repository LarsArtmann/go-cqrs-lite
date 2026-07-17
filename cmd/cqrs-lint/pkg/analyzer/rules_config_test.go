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
