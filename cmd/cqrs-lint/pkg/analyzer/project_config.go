package analyzer

import (
	"encoding/json/v2"
	"errors"
	"fmt"
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

	return cfg, true, nil
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

// StripJSONC removes // line comments and /* block */ comments from JSON data
// while respecting string literals, then strips trailing commas (allowed by
// the JSONC spec but rejected by strict JSON parsers). This is the single
// JSONC decoder for both config paths (CLI loader and embedded loader).
func StripJSONC(data []byte) []byte {
	var result []byte
	result = make([]byte, 0, len(data))

	i := 0
	inString := false

	for i < len(data) {
		c := data[i]

		if inString {
			if c == '\\' && i+1 < len(data) {
				result = append(result, c, data[i+1])
				i += 2
				continue
			}
			if c == '"' {
				inString = false
			}
			result = append(result, c)
			i++
			continue
		}

		switch {
		case c == '"':
			inString = true
			result = append(result, c)
			i++

		case c == '/' && i+1 < len(data) && data[i+1] == '/':
			for i < len(data) && data[i] != '\n' {
				i++
			}

		case c == '/' && i+1 < len(data) && data[i+1] == '*':
			i += 2
			for i+1 < len(data) && (data[i] != '*' || data[i+1] != '/') {
				i++
			}
			if i+1 < len(data) {
				i += 2
			} else {
				i = len(data)
			}

		default:
			result = append(result, c)
			i++
		}
	}

	return stripTrailingCommas(result)
}

// stripTrailingCommas removes commas that are followed only by whitespace
// until a closing } or ]. This handles the JSONC trailing comma convention
// while respecting string literals (a comma inside a string followed by } is
// NOT removed).
func stripTrailingCommas(data []byte) []byte {
	result := make([]byte, 0, len(data))
	inString := false
	pendingComma := -1 // index in result where a comma may be trailing

	for i := 0; i < len(data); i++ {
		c := data[i]

		if inString {
			if c == '\\' && i+1 < len(data) {
				result = append(result, c, data[i+1])
				i++
				continue
			}
			if c == '"' {
				inString = false
			}
			result = append(result, c)
			pendingComma = -1
			continue
		}

		switch c {
		case '"':
			inString = true
			result = append(result, c)
			pendingComma = -1
		case ',':
			result = append(result, c)
			pendingComma = len(result) - 1
		case '}', ']':
			if pendingComma >= 0 {
				result = result[:pendingComma]
			}
			result = append(result, c)
			pendingComma = -1
		case ' ', '\t', '\n', '\r':
			result = append(result, c)
		default:
			result = append(result, c)
			pendingComma = -1
		}
	}

	return result
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
