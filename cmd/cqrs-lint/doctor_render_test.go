package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
)

func TestRenderDoctorPreset_NoPreset(t *testing.T) {
	t.Parallel()

	cfg := &AppConfig{}
	buf := &bytes.Buffer{}
	renderDoctorPreset(buf, cfg)

	out := buf.String()
	if !strings.Contains(out, "(none)") {
		t.Errorf("expected '(none)' in output, got:\n%s", out)
	}
	if !strings.Contains(out, "ACTIVE PRESET") {
		t.Errorf("expected 'ACTIVE PRESET' header, got:\n%s", out)
	}
}

func TestRenderDoctorPreset_WithPreset(t *testing.T) {
	t.Parallel()

	cfg := &AppConfig{Preset: "local-cli"}
	buf := &bytes.Buffer{}
	renderDoctorPreset(buf, cfg)

	out := buf.String()
	if !strings.Contains(out, "local-cli") {
		t.Errorf("expected 'local-cli' in output, got:\n%s", out)
	}
	if !strings.Contains(out, "Features pinned") {
		t.Errorf("expected 'Features pinned' in output, got:\n%s", out)
	}
}

func TestRenderDoctorFeatureProfile(t *testing.T) {
	t.Parallel()

	actx := &analyzer.AnalysisContext{
		FeatureProfile: analyzer.FeatureProfile{
			Store:       analyzer.StoreSQLite,
			HasServer:   true,
			CommandFlow: analyzer.CommandFlowCommands,
		},
	}
	buf := &bytes.Buffer{}
	renderDoctorFeatureProfile(buf, actx)

	out := buf.String()
	if !strings.Contains(out, "FEATURE PROFILE") {
		t.Errorf("expected 'FEATURE PROFILE' header, got:\n%s", out)
	}
	if !strings.Contains(out, "sqlite") {
		t.Errorf("expected 'sqlite' store in output, got:\n%s", out)
	}
	if !strings.Contains(out, "server:        true") {
		t.Errorf("expected 'server: true' in output, got:\n%s", out)
	}
}

func TestRenderDoctorEffectiveSettings(t *testing.T) {
	t.Parallel()

	cfg := &AppConfig{
		Format:        "text",
		MinSeverity:   "info",
		MinConfidence: "low",
		Color:         "auto",
	}
	buf := &bytes.Buffer{}
	renderDoctorEffectiveSettings(buf, cfg)

	out := buf.String()
	if !strings.Contains(out, "EFFECTIVE SETTINGS") {
		t.Errorf("expected 'EFFECTIVE SETTINGS' header, got:\n%s", out)
	}
	if !strings.Contains(out, "format:          text") {
		t.Errorf("expected format line, got:\n%s", out)
	}
	if !strings.Contains(out, "rules:") {
		t.Errorf("expected rules count line, got:\n%s", out)
	}
}

func TestRenderDoctorSuppressions_NoSuppressions(t *testing.T) {
	t.Parallel()

	actx := &analyzer.AnalysisContext{}
	buf := &bytes.Buffer{}
	renderDoctorSuppressions(buf, actx)

	if buf.Len() != 0 {
		t.Errorf("expected empty output for no suppressions, got:\n%s", buf.String())
	}
}

func TestRenderDoctorSuppressions_WithSuppressions(t *testing.T) {
	t.Parallel()

	actx := analyzer.BuildContextFromSource(t, map[string]string{
		"main.go": `package main

//cqrs-lint:ignore(C007) false positive in generated code
var x = 1
`,
	})
	buf := &bytes.Buffer{}
	renderDoctorSuppressions(buf, actx)

	out := buf.String()
	if !strings.Contains(out, "INLINE SUPPRESSIONS") {
		t.Errorf("expected 'INLINE SUPPRESSIONS' header, got:\n%s", out)
	}
	if !strings.Contains(out, "C007") {
		t.Errorf("expected 'C007' rule in output, got:\n%s", out)
	}
}

func TestRenderDoctorPerModuleProfiles_SingleModule(t *testing.T) {
	t.Parallel()

	actx := &analyzer.AnalysisContext{
		FeatureProfiles: map[string]analyzer.FeatureProfile{
			"/repo": {Store: analyzer.StoreSQLite},
		},
	}
	buf := &bytes.Buffer{}
	renderDoctorPerModuleProfiles(buf, actx)

	if buf.Len() != 0 {
		t.Errorf("expected empty output for single module, got:\n%s", buf.String())
	}
}

func TestRenderDoctorPerModuleProfiles_MultipleModules(t *testing.T) {
	t.Parallel()

	actx := &analyzer.AnalysisContext{
		FeatureProfiles: map[string]analyzer.FeatureProfile{
			"/repo/lib":          {Store: analyzer.StoreMemory, HasServer: false},
			"/repo/examples/app": {Store: analyzer.StoreSQLite, HasServer: true},
		},
	}
	buf := &bytes.Buffer{}
	renderDoctorPerModuleProfiles(buf, actx)

	out := buf.String()
	if !strings.Contains(out, "PER-MODULE PROFILES") {
		t.Errorf("expected 'PER-MODULE PROFILES' header, got:\n%s", out)
	}
	if !strings.Contains(out, "/repo/lib") {
		t.Errorf("expected '/repo/lib' in output, got:\n%s", out)
	}
	if !strings.Contains(out, "/repo/examples/app") {
		t.Errorf("expected '/repo/examples/app' in output, got:\n%s", out)
	}
}

func TestRenderDoctorLoadErrors_NoErrors(t *testing.T) {
	t.Parallel()

	actx := &analyzer.AnalysisContext{}
	buf := &bytes.Buffer{}
	renderDoctorLoadErrors(buf, actx)

	if buf.Len() != 0 {
		t.Errorf("expected empty output for no errors, got:\n%s", buf.String())
	}
}

func TestRenderDoctorLoadErrors_WithErrors(t *testing.T) {
	t.Parallel()

	actx := &analyzer.AnalysisContext{
		LoadErrors: []analyzer.PackageLoadError{
			{Module: "test-mod", Errors: []string{"syntax error"}},
		},
	}
	buf := &bytes.Buffer{}
	renderDoctorLoadErrors(buf, actx)

	out := buf.String()
	if !strings.Contains(out, "WARNING") {
		t.Errorf("expected 'WARNING' in output, got:\n%s", out)
	}
	if !strings.Contains(out, "test-mod") {
		t.Errorf("expected 'test-mod' in output, got:\n%s", out)
	}
}
