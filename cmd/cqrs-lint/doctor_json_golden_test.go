package main

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"os"
	"path/filepath"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
)

// The doctor JSON surface had no shape guard: renaming a field, dropping an
// omitempty, or changing nesting silently broke every consumer script. This
// golden pins the SHAPE. Volatile values (paths, rule counts) are stamped to
// constants so the golden only changes when the surface itself changes;
// regenerate with UPDATE_GOLDEN=1.
func TestDoctorJSONReport_Golden(t *testing.T) {
	t.Parallel()

	cfg := &AppConfig{
		Path:          t.TempDir(),
		Preset:        analyzer.PresetV5Ready,
		MinConfidence: "medium",
		Rules: analyzer.RulesConfig{
			Disable:           []string{"B001", "V007"},
			SeverityOverrides: map[string]string{"V007": "error", "S011": "error"},
		},
	}

	actx := &analyzer.AnalysisContext{
		FeatureProfile: analyzer.FeatureProfile{
			Store:       analyzer.StoreSQLite,
			CommandFlow: analyzer.CommandFlowCommands,
			Tracing:     analyzer.TracingOn,
			Snapshot:    analyzer.SnapshotOff,
		},
		FeatureProfiles: map[string]analyzer.FeatureProfile{
			"/example/project":          {Store: analyzer.StoreSQLite, CommandFlow: analyzer.CommandFlowCommands},
			"/example/project/examples": {Store: analyzer.StoreMemory, HasServer: true},
		},
	}

	report := buildDoctorJSONReport(cfg, actx)

	report.Path = "/example/project"
	report.ParentConfigs = []string{"/example/.cqrs-lint.json"}
	report.RulesTotal = 185
	report.RulesActive = 183
	report.RulesDisabled = 2

	data, err := json.Marshal(report, jsontext.WithIndent("  "))
	if err != nil {
		t.Fatalf("marshal report: %v", err)
	}

	golden := filepath.Join("testdata", "doctor_json_report.golden")

	if os.Getenv("UPDATE_GOLDEN") == "1" {
		if err := os.WriteFile(golden, data, 0o644); err != nil {
			t.Fatalf("write golden: %v", err)
		}

		return
	}

	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatalf("read golden (run UPDATE_GOLDEN=1 once): %v", err)
	}

	if string(data) != string(want) {
		t.Errorf("doctor JSON shape drifted from golden.\n--- want ---\n%s\n--- got ---\n%s", want, data)
	}
}
