package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
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

// TestLoadParentRulesConfig_IncludesImmediateParent pins the v4.22.1 fix:
// the CLI's ancestor walk consults the IMMEDIATE parent, not just the
// grandparent and beyond. The old walk skipped it on the assumption
// cmdguard's loader already applies that layer's rules.disable — false, so
// depth-1 modules ignored root-config disables that depth-2 modules honored
// (found 2026-11-17 when the cqrs-htmx gate escalated to fail-on-any-finding
// and six depth-1 modules resurfaced findings the toolsdk walk had
// inherited away; analyzer.LoadParentRulesConfig always included it).
func TestLoadParentRulesConfig_IncludesImmediateParent(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	mid := filepath.Join(root, "mid")
	leaf := filepath.Join(mid, "leaf")
	if err := os.MkdirAll(leaf, 0o755); err != nil {
		t.Fatal(err)
	}

	writeConfig := func(dir, rule string) {
		t.Helper()
		body := `{"rules":{"disable":["` + rule + `"]}}`
		if err := os.WriteFile(filepath.Join(dir, ".cqrs-lint.json"), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	writeConfig(root, "D005") // grandparent — was already inherited
	writeConfig(mid, "F005")  // immediate parent — SKIPPED before v4.22.1

	got := loadParentRulesConfig(leaf)
	has := func(rule string) bool {
		for _, d := range got.Disable {
			if d == rule {
				return true
			}
		}

		return false
	}
	if !has("D005") {
		t.Errorf("grandparent disable D005 must be inherited; got %v", got.Disable)
	}
	if !has("F005") {
		t.Errorf("immediate-parent disable F005 must be inherited (the v4.22.1 fix); got %v", got.Disable)
	}
}

// TestBuildContextForScope_ScopeSelection pins the --scope wiring (v4.22.2):
// the default/empty scope resolves workspace semantics, "module" resolves the
// single-module context, and an unknown scope fails loudly instead of
// silently degrading to a walk the caller did not ask for.
func TestBuildContextForScope_ScopeSelection(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/scopeprobe\n\ngo 1.21\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	for _, scope := range []string{"", "workspace", "module"} {
		actx, err := buildContextForScope(scope, root)
		if err != nil {
			t.Errorf("scope %q must resolve; got %v", scope, err)

			continue
		}
		if actx == nil || actx.ProjectRoot == "" {
			t.Errorf("scope %q must yield a context with a project root", scope)
		}
	}

	if _, err := buildContextForScope("bogus", root); err == nil {
		t.Error("an unknown scope must fail loudly, not degrade to a walk")
	}
}
