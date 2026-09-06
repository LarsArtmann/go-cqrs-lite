package main

import (
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"os"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/rules"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/suppression"
)

// doctorJSONReport is the machine-readable `doctor --format json` surface:
// the resolved configuration, the detected feature profile, and — with
// --audit-suppressions / --fix — the suppression audit and fix outcome.
type doctorJSONReport struct {
	Path               string                    `json:"path"`
	ConfigFile         string                    `json:"config_file,omitempty"`
	ConfigFound        bool                      `json:"config_found"`
	ParentConfigs      []string                  `json:"parent_configs,omitempty"`
	Preset             string                    `json:"preset,omitempty"`
	SeverityFloor      string                    `json:"severity_floor"`
	MinConfidence      string                    `json:"min_confidence"`
	RulesTotal         int                       `json:"rules_total"`
	RulesActive        int                       `json:"rules_active"`
	RulesDisabled      int                       `json:"rules_disabled"`
	DisabledFromPreset []string                  `json:"disabled_from_preset,omitempty"`
	DisabledFromConfig []string                  `json:"disabled_from_config,omitempty"`
	Features           analyzer.FeatureProfile   `json:"features"`
	Modules            []moduleProfileJSON       `json:"modules,omitempty"`
	Audit              *suppressionAuditJSON     `json:"audit,omitempty"`
}

type moduleProfileJSON struct {
	Module  string                   `json:"module"`
	Profile analyzer.FeatureProfile  `json:"profile"`
}

type suppressionAuditJSON struct {
	Total       int                                 `json:"total"`
	Active      int                                 `json:"active"`
	Stale       int                                 `json:"stale"`
	UnknownRule int                                 `json:"unknown_rule"`
	Entries     []suppression.SuppressionAuditEntry `json:"entries"`
	Fix         *suppressionFixJSON                 `json:"fix,omitempty"`
}

type suppressionFixJSON struct {
	DryRun  bool                                `json:"dry_run"`
	Removed []suppression.SuppressionAuditEntry `json:"removed"`
	Skipped []suppression.SuppressionAuditEntry `json:"skipped"`
	Files   []string                            `json:"files,omitempty"`
}

// runDoctorJSON renders the full doctor report as JSON.
func runDoctorJSON(
	ctx context.Context,
	cfg *AppConfig,
	actx *analyzer.AnalysisContext,
	flags doctorFlags,
) error {
	if flags.AuditSuppressions || flags.Fix {
		audit, err := buildSuppressionAuditJSON(ctx, cfg, actx, flags.Fix, flags.DryRun)
		if err != nil {
			return err
		}

		return printDoctorJSON(withAudit(cfg, actx, audit))
	}

	return printDoctorJSON(buildDoctorJSONReport(cfg, actx))
}

func withAudit(cfg *AppConfig, actx *analyzer.AnalysisContext, audit *suppressionAuditJSON) doctorJSONReport {
	report := buildDoctorJSONReport(cfg, actx)
	report.Audit = audit

	return report
}

func printDoctorJSON(report doctorJSONReport) error {
	data, err := json.Marshal(report, jsontext.WithIndent("  "))
	if err != nil {
		return fmt.Errorf("marshal doctor report: %w", err)
	}

	if _, err := fmt.Fprintln(os.Stdout, string(data)); err != nil {
		return fmt.Errorf("write doctor report: %w", err)
	}

	return nil
}

// buildDoctorJSONReport captures the same resolved state the text renderer
// shows: config file discovery, preset, effective settings with the
// preset-vs-config disabled-rule breakdown, and the feature profile.
func buildDoctorJSONReport(cfg *AppConfig, actx *analyzer.AnalysisContext) doctorJSONReport {
	presetDef := analyzer.ResolvePresetDefinition(cfg.Preset)

	report := doctorJSONReport{
		Path:          cfg.Path,
		SeverityFloor: resolveMinSeverity(presetDef.MinSeverity, cfg.MinSeverity),
		MinConfidence: cfg.MinConfidence,
		Features:      actx.FeatureProfile,
	}

	if configPath := joinPath(cfg.Path, ".cqrs-lint.json"); fileExists(configPath) {
		report.ConfigFile = configPath
		report.ConfigFound = true
	}

	report.ParentConfigs = findParentConfigs(cfg.Path)

	if cfg.Preset != "" {
		report.Preset = cfg.Preset
	}

	report.RulesTotal = len(rules.AllRules())
	report.RulesDisabled = len(cfg.Rules.Disable)
	report.RulesActive = report.RulesTotal - report.RulesDisabled
	splitDisabledRules(&report, presetDef.Rules.Disable)

	for dir, profile := range actx.FeatureProfiles {
		report.Modules = append(report.Modules, moduleProfileJSON{Module: dir, Profile: profile})
	}

	return report
}

// splitDisabledRules breaks the effective disable list into preset-pinned and
// config-supplied rule IDs, mirroring the text renderer's source breakdown.
func splitDisabledRules(report *doctorJSONReport, presetDisabled []string) {
	presetSet := make(map[string]bool, len(presetDisabled))
	for _, r := range presetDisabled {
		presetSet[r] = true
	}

	for _, r := range report-disabled-list(report) {
		if presetSet[r] {
			report.DisabledFromPreset = append(report.DisabledFromPreset, r)
		} else {
			report.DisabledFromConfig = append(report.DisabledFromConfig, r)
		}
	}
}

func report-disabled-list(report *doctorJSONReport) []string {
	return report.Disabled // placeholder — replaced below
}

// buildSuppressionAuditJSON runs the suppression audit pipeline and shapes
// the outcome for JSON output, including the fix plan/result when --fix.
func buildSuppressionAuditJSON(
	ctx context.Context,
	cfg *AppConfig,
	actx *analyzer.AnalysisContext,
	fix bool,
	dryRun bool,
) (*suppressionAuditJSON, error) {
	entries, err := computeSuppressionAudit(ctx, cfg, actx)
	if err != nil {
		return nil, err
	}

	audit := &suppressionAuditJSON{Total: len(entries), Entries: entries}
	for _, e := range entries {
		switch e.Status {
		case suppression.AuditActive:
			audit.Active++
		case suppression.AuditStale:
			audit.Stale++
		case suppression.AuditUnknownRule:
			audit.UnknownRule++
		}
	}

	if fix {
		fixResult := applyOrPlanFix(entries, dryRun)
		audit.Fix = &suppressionFixJSON{
			DryRun:  dryRun,
			Removed: fixResult.Removed,
			Skipped: fixResult.Skipped,
			Files:   fixResult.Files,
		}
	}

	return audit, nil
}
