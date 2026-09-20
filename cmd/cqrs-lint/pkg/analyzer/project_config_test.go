package analyzer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/larsartmann/go-finding"
)

func writeConfig(t *testing.T, dir, content string) {
	t.Helper()

	if err := os.WriteFile(filepath.Join(dir, ConfigFileName), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestLoadProjectConfig_MissingFile(t *testing.T) {
	cfg, found, err := LoadProjectConfig(t.TempDir())
	if err != nil {
		t.Fatalf("missing config must not be an error: %v", err)
	}
	if found {
		t.Fatal("found must be false when no config exists")
	}
	if cfg.Preset != PresetNone || len(cfg.Rules.Disable) != 0 {
		t.Fatalf("zero config expected, got %+v", cfg)
	}
}

func TestLoadProjectConfig_ValidJSONC(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, `{
		// adoption coaching is noise for a non-consumer
		"preset": "library-framework",
		"rules": {
			// D005 misattributes versions on shared lines
			"disable": ["D005"], /* trailing comma next */
		},
	}`)

	cfg, found, err := LoadProjectConfig(dir)
	if err != nil {
		t.Fatalf("parse JSONC: %v", err)
	}
	if !found {
		t.Fatal("found must be true")
	}
	if cfg.Preset != PresetLibraryFramework {
		t.Fatalf("preset = %q, want %q", cfg.Preset, PresetLibraryFramework)
	}
	if len(cfg.Rules.Disable) != 1 || cfg.Rules.Disable[0] != "D005" {
		t.Fatalf("disable = %v, want [D005]", cfg.Rules.Disable)
	}
}

func TestLoadProjectConfig_MalformedJSON(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, `{"preset": "library-framework",`)

	if _, _, err := LoadProjectConfig(dir); err == nil {
		t.Fatal("malformed config must be an error, not silently ignored")
	}
}

func TestLoadProjectConfig_UnknownPreset(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, `{"preset": "productionn"}`)

	_, _, err := LoadProjectConfig(dir)
	if err == nil {
		t.Fatal("unknown preset must be an error (ghost-config guard)")
	}
	if !strings.Contains(err.Error(), "productionn") {
		t.Fatalf("error must name the bad preset: %v", err)
	}
}

func TestLoadProjectConfig_UnreadableFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ConfigFileName)
	if err := os.WriteFile(path, []byte("{}"), 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(path, 0o644) })

	if _, _, err := LoadProjectConfig(dir); err == nil {
		t.Fatal("unreadable config must surface an error, not look missing")
	}
}

func TestEffectiveRules_NoPreset(t *testing.T) {
	cfg := ProjectConfig{Rules: RulesConfig{Disable: []string{"D005"}}}

	got := cfg.EffectiveRules()
	if len(got.Disable) != 1 || got.Disable[0] != "D005" {
		t.Fatalf("no-preset config must pass through unchanged, got %v", got.Disable)
	}
}

func TestEffectiveRules_PresetDisablesAreDefaults(t *testing.T) {
	cfg := ProjectConfig{
		Preset: PresetLibraryFramework,
		Rules:  RulesConfig{Disable: []string{"D005"}},
	}

	got := cfg.EffectiveRules()

	set := map[string]bool{}
	for _, id := range got.Disable {
		set[id] = true
	}
	for _, id := range []string{"F010", "E003", "V007"} {
		if !set[id] {
			t.Errorf("preset disable %s missing from union", id)
		}
	}
	if !set["D005"] {
		t.Error("config disable D005 missing from union")
	}
}

func TestEffectiveRules_SeverityOverridesMerge(t *testing.T) {
	cfg := ProjectConfig{
		Preset: PresetV5Ready, // {"V007": "error"}
		Rules: RulesConfig{
			SeverityOverrides: map[string]string{"C034": "critical"},
		},
	}

	got := cfg.EffectiveRules()
	if got.SeverityOverrides["V007"] != "error" {
		t.Errorf("preset override lost: %v", got.SeverityOverrides)
	}
	if got.SeverityOverrides["C034"] != "critical" {
		t.Errorf("config override lost: %v", got.SeverityOverrides)
	}
}

func TestEffectiveRules_ConfigWinsSeverityOverride(t *testing.T) {
	cfg := ProjectConfig{
		Preset: PresetV5Ready,
		Rules: RulesConfig{
			SeverityOverrides: map[string]string{"V007": "warning"},
		},
	}

	if got := cfg.EffectiveRules().SeverityOverrides["V007"]; got != "warning" {
		t.Fatalf("config must win per rule ID, got %q", got)
	}
}

func TestStripJSONC(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "line comments and trailing commas",
			in:   "{\n// comment\n\"a\": [1,],\n}",
			want: "{\n\n\"a\": [1]}",
		},
		{
			name: "block comment",
			in:   `{"a": /* c */ 1}`,
			want: `{"a":  1}`,
		},
		{
			name: "comment markers inside strings survive",
			in:   `{"a": "http://x /* y // z"}`,
			want: `{"a": "http://x /* y // z"}`,
		},
		{
			name: "escaped quote inside string",
			in:   `{"a": "say \"//" }`,
			want: `{"a": "say \"//" }`,
		},
		{
			name: "trailing comma before nested close",
			in:   `{"rules": {"disable": ["A",],},}`,
			want: `{"rules": {"disable": ["A"]}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := string(StripJSONC([]byte(tt.in))); got != tt.want {
				t.Fatalf("StripJSONC = %q, want %q", got, tt.want)
			}
		})
	}
}

func mkFinding(t *testing.T, rule string, sev finding.Severity) finding.Finding {
	b := finding.NewBuilder(
		finding.RuleName(rule),
		"cqrs-lint",
		"msg",
		sev,
		finding.Pos(finding.FilePath("x.go"), 1, 1),
	)
	f, err := b.Build()
	if err != nil {
		t.Fatal(err)
	}

	return f
}

func TestApplySeverityOverrides(t *testing.T) {
	findings := []finding.Finding{
		mkFinding(t, "V007", finding.SeverityWarning),
		mkFinding(t, "C034", finding.SeverityWarning),
	}

	got := ApplySeverityOverrides(findings, map[string]string{"v007": "error"})

	if got[0].Severity != finding.SeverityError {
		t.Fatalf("V007 severity = %v, want error", got[0].Severity)
	}
	if !strings.Contains(got[0].Message, "[severity overridden: error]") {
		t.Fatalf("V007 message = %q, want override annotation", got[0].Message)
	}
	if got[1].Severity != finding.SeverityWarning {
		t.Fatalf("unmentioned rule must be untouched, got %v", got[1].Severity)
	}
}

func TestApplySeverityOverrides_Empty(t *testing.T) {
	findings := []finding.Finding{mkFinding(t, "V007", finding.SeverityWarning)}

	if got := ApplySeverityOverrides(findings, nil); len(got) != 1 {
		t.Fatal("nil overrides must be a no-op")
	}
}

func TestFilterDisabledFindings(t *testing.T) {
	findings := []finding.Finding{
		mkFinding(t, "D005", finding.SeverityWarning),
		mkFinding(t, "C034", finding.SeverityWarning),
	}

	got := FilterDisabledFindings(findings, map[string]bool{"D005": true})

	if len(got) != 1 || string(got[0].Rule) != "C034" {
		t.Fatalf("disabled finding must be dropped, got %v", got)
	}

	if got := FilterDisabledFindings(findings, nil); len(got) != 2 {
		t.Fatal("nil disabled set must be a no-op")
	}
}
