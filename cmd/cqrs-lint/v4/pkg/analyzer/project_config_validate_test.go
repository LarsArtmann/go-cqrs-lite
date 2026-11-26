package analyzer

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestLoadProjectConfig_NormalizesRulesConfig: the embedded (toolspec) path
// loads the config through LoadProjectConfig only, so the normalization the
// CLI gets from Rules.Validate must happen here. Before the fix " d005 "
// stayed lowercase (DisabledSet is case-sensitive, so the disable was a
// no-op), "fatal" demoted C001 to info instead of being dropped with a
// warning, and the unknown key and orphaned disable-reason were silent.
func TestLoadProjectConfig_NormalizesRulesConfig(t *testing.T) {
	dir := t.TempDir()

	config := `{
		"rules": {
			"disable": [" d005 ", "D005"],
			"severity-overrides": {"c001": "fatal", "c002": " ERROR "},
			"disable-reasons": {"D005": "kept", "C999": "orphan"},
			"external-api-struct-prefixes": [" a ", "a", ""],
			"disable-typo": true
		}
	}`
	if err := os.WriteFile(filepath.Join(dir, ConfigFileName), []byte(config), 0o600); err != nil {
		t.Fatal(err)
	}

	var warnings bytes.Buffer
	old := configWarnings
	configWarnings = &warnings
	t.Cleanup(func() { configWarnings = old })

	cfg, found, err := LoadProjectConfig(dir)
	if err != nil {
		t.Fatalf("LoadProjectConfig: %v", err)
	}

	if !found {
		t.Fatal("config must be found")
	}

	if got := strings.Join(cfg.Rules.Disable, ","); got != "D005" {
		t.Errorf("disable = %q, want normalized deduplicated \"D005\"", got)
	}

	if _, kept := cfg.Rules.SeverityOverrides["C002"]; !kept || cfg.Rules.SeverityOverrides["C002"] != "error" {
		t.Errorf("valid override must be normalized, got %v", cfg.Rules.SeverityOverrides)
	}

	if _, dropped := cfg.Rules.SeverityOverrides["C001"]; dropped {
		t.Errorf("invalid severity \"fatal\" must be dropped, got %v", cfg.Rules.SeverityOverrides)
	}

	if cfg.Rules.DisableReasons["D005"] != "kept" || len(cfg.Rules.DisableReasons) != 1 {
		t.Errorf("disable-reasons = %v, want only the D005 entry", cfg.Rules.DisableReasons)
	}

	if got := strings.Join(cfg.Rules.ExternalAPIStructPrefixes, ","); got != "a" {
		t.Errorf("prefixes = %q, want \"a\"", got)
	}

	for _, want := range []string{"disable-typo", `"fatal"`, "C999"} {
		if !strings.Contains(warnings.String(), want) {
			t.Errorf("warnings must mention %s, got:\n%s", want, warnings.String())
		}
	}

	if disabled := cfg.Rules.DisabledSet(); !disabled["D005"] || len(disabled) != 1 {
		t.Errorf("DisabledSet = %v, want {D005}", disabled)
	}
}
