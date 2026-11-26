package analyzer

import (
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"path/filepath"
	"strings"

	"github.com/larsartmann/go-finding"
)

// errUnknownPreset is the static sentinel wrapped by every unknown-preset
// error; the path and valid names are the dynamic context around it.
var errUnknownPreset = errors.New("unknown preset")

// ConfigFileName is the per-project configuration file cqrs-lint reads from
// the linted directory. Both the CLI and the embedded (toolspec) detection
// path honor it — before the embedded path did, hosts running cqrs-lint as a
// library silently ignored every setting in the file while the CLI honored
// the same file, making the config a no-op lie for SDK consumers.
const ConfigFileName = ".cqrs-lint.json"

// ProjectConfig is the project-behavior subset of .cqrs-lint.json: the named
// preset and the rules block. Output keys (format, color, group-by) and
// input-filter keys (exclude, min-confidence, min-severity) are CLI-surface
// concerns and intentionally NOT modeled here: the embedded detection path
// never renders output and must not drop findings a host (e.g. BuildFlow)
// aggregates itself.
type ProjectConfig struct {
	// Preset is the named preset whose feature flags and rule disables apply
	// as defaults. Empty (the zero value) means no preset: features are
	// auto-detected and no rules are pre-disabled.
	Preset ConfigPreset `json:"preset,omitzero"`

	// Rules carries rule-specific overrides, merged ON TOP of the preset's
	// rules (union for disable lists, config wins for severity overrides).
	Rules RulesConfig `json:"rules,omitzero"`

	// Scorecard carries scorecard behavior settings: recorded per-module
	// adoption waivers. See ScorecardSettings.
	Scorecard ScorecardSettings `json:"scorecard,omitzero"`
}

// LoadProjectConfig reads ConfigFileName from dir and decodes it as JSONC
// (// and /* */ comments, plus trailing commas, are allowed). The second
// return is false when no config file exists — not an error, most consumers
// have none. A config file that exists but cannot be parsed, or names an
// unknown preset, is an error: a typo must never silently lint unconfigured.
func LoadProjectConfig(dir string) (ProjectConfig, bool, error) {
	path := filepath.Join(dir, ConfigFileName)

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return ProjectConfig{}, false, nil
		}

		return ProjectConfig{}, true, fmt.Errorf("read %s: %w", path, err)
	}

	var cfg ProjectConfig
	if err := json.Unmarshal(StripJSONC(data), &cfg); err != nil {
		return ProjectConfig{}, true, fmt.Errorf("parse %s: %w", path, err)
	}

	if cfg.Preset != PresetNone && !IsKnownPreset(cfg.Preset) {
		return ProjectConfig{}, true, fmt.Errorf(
			"%s: %w %q (valid: %s)",
			path, errUnknownPreset, cfg.Preset, strings.Join(ValidPresetNames(), ", "),
		)
	}

	if err := ValidateScorecardWaivers(cfg.Scorecard.Waivers); err != nil {
		return ProjectConfig{}, true, fmt.Errorf("%s: %w", path, err)
	}

	if err := ValidateScorecardModulePins(cfg.Scorecard.Modules); err != nil {
		return ProjectConfig{}, true, fmt.Errorf("%s: %w", path, err)
	}

	// Same normalization the CLI runs on its decoded config (run.go:
	// Rules.Validate before filtering). Without it the embedded (toolspec)
	// path kept lowercase/whitespace disable IDs, which DisabledSet and
	// FilterDisabledFindings compare case-sensitively, kept invalid
	// severities (demoting the rule to info), and never warned about unknown
	// keys or orphaned disable-reasons — one file, two behaviors.
	cfg.Rules.Validate(configWarnings, rawRulesBlock(StripJSONC(data)))

	return cfg, true, nil
}

// configWarnings receives the rules-config validation warnings raised while
// loading a project config. The embedded (toolspec) path has no caller-owned
// writer to pass, so it defaults to stderr like the loader's other warnings.
//
//nolint:gochecknoglobals // injectable seam, swapped by tests
var configWarnings io.Writer = os.Stderr

// rawRulesBlock returns the re-encoded "rules" object of a stripped config
// document, or nil when there is none — the input Rules.Validate needs to
// report unknown keys.
func rawRulesBlock(stripped []byte) []byte {
	var top map[string]any
	if err := json.Unmarshal(stripped, &top); err != nil {
		return nil
	}

	raw, err := json.Marshal(top["rules"], json.Deterministic(true))
	if err != nil || string(raw) == "null" {
		return nil
	}

	return raw
}

// LoadParentRulesConfig resolves ancestor .cqrs-lint.json rules for config
// inheritance (monorepo support): walking from the immediate parent of dir
// upward to the filesystem root, every ancestor's rules.disable and
// rules.external-api-struct-prefixes are unioned into the result. The module's
// own config (if any) is NOT consulted here — LoadProjectConfig covers it, and
// the caller merges this result underneath it.
//
// Ancestor severity overrides are intentionally NOT inherited (parity with the
// CLI's ancestor walk, which collects disables and prefixes only). Ancestor
// configs are best-effort: an unreadable or malformed file contributes
// nothing and never blocks the run — only the module's own config errors
// loudly.
//
// This includes the immediate parent, unlike the CLI's ancestor walk, which
// skips it (a cmdguard-copied comment admits its own config loader covers
// that layer; the toolsdk host has no such second loader, and skipping it
// made depth-1 modules ignore a root disable list that depth-2 modules
// honored).
func LoadParentRulesConfig(dir string) (RulesConfig, error) {
	var merged RulesConfig

	absDir, err := filepath.Abs(dir)
	if err != nil {
		return RulesConfig{}, fmt.Errorf("resolve %s: %w", dir, err)
	}

	for cursor := filepath.Dir(absDir); ; {
		data, readErr := os.ReadFile(filepath.Join(cursor, ConfigFileName))
		if readErr == nil {
			var top struct {
				Rules RulesConfig `json:"rules,omitzero"`
			}
			if json.Unmarshal(StripJSONC(data), &top) == nil {
				for _, id := range top.Rules.Disable {
					// Ancestor disables are unioned raw; normalize to the
					// uppercase form DisabledSet compares against.
					if id = strings.ToUpper(strings.TrimSpace(id)); id != "" {
						merged.Disable = append(merged.Disable, id)
					}
				}

				merged.ExternalAPIStructPrefixes = append(
					merged.ExternalAPIStructPrefixes,
					top.Rules.ExternalAPIStructPrefixes...,
				)
			}
		}

		parent := filepath.Dir(cursor)
		if parent == cursor {
			break
		}

		cursor = parent
	}

	return merged, nil
}

// EffectiveRules merges the preset's rule defaults with this config's rules
// block, mirroring the CLI's merge order: preset disable lists apply as
// defaults and config disables are appended (union, never subtracted);
// severity overrides merge per rule ID with config winning. Returns the
// config unchanged when there is no preset.
func (p ProjectConfig) EffectiveRules() RulesConfig {
	effective := p.Rules

	def, ok := PresetDefinitions[p.Preset]
	if !ok {
		return effective
	}

	effective.Disable = append(append([]string{}, def.Rules.Disable...), effective.Disable...)

	switch {
	case len(def.Rules.SeverityOverrides) == 0:
		// config overrides only — keep as-is
	case len(effective.SeverityOverrides) == 0:
		effective.SeverityOverrides = def.Rules.SeverityOverrides
	default:
		merged := make(
			map[string]string,
			len(def.Rules.SeverityOverrides)+len(effective.SeverityOverrides),
		)
		maps.Copy(merged, def.Rules.SeverityOverrides)
		maps.Copy(merged, effective.SeverityOverrides)

		effective.SeverityOverrides = merged
	}

	return effective
}

// ApplySeverityOverrides rewrites finding severities according to a
// rules.severity-overrides map (preset defaults merged with explicit config).
// Runs post-detection, pre-filter so triage and host-side gating observe the
// overridden severity. Rule-ID lookup is case-insensitive. An override whose
// value parses to the finding's current severity leaves the finding
// untouched; unknown severity values fall back to info, matching the CLI.
func ApplySeverityOverrides(
	findings []finding.Finding,
	overrides map[string]string,
) []finding.Finding {
	if len(overrides) == 0 {
		return findings
	}

	normalized := make(map[string]string, len(overrides))
	for id, sev := range overrides {
		normalized[strings.ToUpper(strings.TrimSpace(id))] = sev
	}

	result := make([]finding.Finding, len(findings))
	for i, f := range findings {
		sev, ok := normalized[string(f.Rule)]
		if ok {
			parsed := parseOverrideSeverity(sev)
			if parsed != f.Severity {
				f.Severity = parsed
				if f.Message != "" {
					f.Message += " [severity overridden: " + sev + "]"
				}
			}
		}
		result[i] = f
	}

	return result
}

// parseOverrideSeverity maps an override value to a Severity. Mirrors the
// CLI's parseSeverity: unknown values fall back to info rather than dropping
// the override silently.
func parseOverrideSeverity(s string) finding.Severity {
	switch strings.ToLower(s) {
	case "critical":
		return finding.SeverityCritical
	case "error":
		return finding.SeverityError
	case "warning":
		return finding.SeverityWarning
	case "info":
		return finding.SeverityInfo
	default:
		return finding.SeverityInfo
	}
}

// FilterDisabledFindings drops findings whose rule ID is in the disabled set
// (from "rules": {"disable"} and preset defaults). Disabled findings are
// dropped entirely — they do not reach the host's aggregation.
func FilterDisabledFindings(
	findings []finding.Finding,
	disabled map[string]bool,
) []finding.Finding {
	if len(disabled) == 0 {
		return findings
	}

	result := make([]finding.Finding, 0, len(findings))
	for _, f := range findings {
		if !disabled[string(f.Rule)] {
			result = append(result, f)
		}
	}

	return result
}
