package analyzer

import (
	"encoding/json/v2"
	"fmt"
	"io"
	"strings"
)

// RulesConfig holds rule-specific configuration that detectors consult to
// suppress domain-specific false positives. Populated from the "rules" key in
// .cqrs-lint.json. Fields are intentionally narrow: each one targets a
// concrete false-positive pattern documented in consumer feedback.
//
// Zero value (the default when no config is present) disables every override,
// so detectors behave exactly as before — this is the contract
// BuildContextFromSource relies on for rule unit tests.
type RulesConfig struct {
	// Disable lists rule IDs to suppress project-wide. A disabled rule never
	// fires, so its findings neither appear in the output nor count toward the
	// health score. Use this for rules that are known false positives in the
	// project's architecture (e.g. P012/P013 when WAL is applied in a shared
	// storage package the linter cannot trace across files). For one-off
	// cases prefer inline //cqrs-lint:ignore(RULE) comments.
	//
	// Example:
	//
	//	{"rules": {"disable": ["P012", "P013"]}}
	Disable []string `json:"disable,omitempty"`

	// ExternalAPIStructPrefixes lists struct-name prefixes whose JSON tags
	// mirror an external API (Discord, Stripe, GitHub, ...) and must NOT count
	// toward D002's mixed-casing check. Example: ["Discord", "Stripe"] marks
	// every struct whose name starts with "Discord" or "Stripe" as an external
	// mirror, so its snake_case tags no longer trigger "mixes camelCase and
	// snake_case" on files that also define camelCase consumer types.
	//
	// For one-off cases prefer the in-source marker
	//
	//	//cqrs-lint:external-api
	//
	// on the struct's doc comment; it suppresses the same rule without needing
	// config. Both mechanisms stack: a struct is excluded if either matches.
	ExternalAPIStructPrefixes []string `json:"external-api-struct-prefixes,omitempty"` //nolint:tagliatelle // CLI config key

	// IgnoreFloatFields lists field names to exclude from C008 (float64-for-
	// money). Use for fields that are intentionally float64 — cost estimates,
	// performance metrics, observability counters — where the monetary name
	// pattern (amount, price, cost) matches but the value is not exact money.
	// Matching is case-insensitive.
	//
	// Example:
	//
	//	{"rules": {"c008-ignore-fields": ["CostEstimate", "PriceIndex"]}}
	IgnoreFloatFields []string `json:"c008-ignore-fields,omitempty"` //nolint:tagliatelle // CLI config key
}

// DisabledSet returns the set of disabled rule IDs as a map for O(1) lookup.
// Returns nil when no rules are disabled.
func (rc *RulesConfig) DisabledSet() map[string]bool {
	if rc == nil || len(rc.Disable) == 0 {
		return nil
	}

	result := make(map[string]bool, len(rc.Disable))
	for _, r := range rc.Disable {
		result[r] = true
	}

	return result
}

// knownRulesConfigKeys is the set of valid keys under "rules" in
// .cqrs-lint.json. Any other key is a likely typo and is reported by Validate.
//
//nolint:gochecknoglobals // read-only lookup table
var knownRulesConfigKeys = map[string]bool{
	"disable":                      true,
	"external-api-struct-prefixes": true,
	"c008-ignore-fields":           true,
}

// Validate checks the rules config for common misconfigurations and writes
// human-readable warnings to w. It also normalizes the config in place
// (trimming whitespace, dropping empty/duplicate prefixes).
//
// rawRulesJSON is the unparsed JSON of the "rules" object from the config file
// (may be nil/empty when no config file is present). When non-empty, unknown
// keys are flagged as likely typos so a misspelled field name doesn't silently
// disable an override.
//
// This closes the silent-failure gap noted in the round-2 self-critique (§d-6):
// a typo like "external-api-prefixes" (missing "struct") or
// "external-api-struct-prefixes": "Discord" (string instead of array) previously
// failed without any diagnostic.
func (rc *RulesConfig) Validate(w io.Writer, rawRulesJSON []byte) {
	if rc == nil {
		return
	}

	// Normalize disable list: trim, uppercase rule IDs, drop empties, deduplicate.
	seenDisable := make(map[string]bool)
	cleanedDisable := rc.Disable[:0]

	for _, r := range rc.Disable {
		id := strings.ToUpper(strings.TrimSpace(r))
		if id == "" || seenDisable[id] {
			continue
		}

		seenDisable[id] = true
		cleanedDisable = append(cleanedDisable, id)
	}

	rc.Disable = cleanedDisable

	// Normalize prefix list: trim, drop empties, deduplicate (order-preserving).
	seen := make(map[string]bool)
	cleaned := rc.ExternalAPIStructPrefixes[:0]

	for _, p := range rc.ExternalAPIStructPrefixes {
		trimmed := strings.TrimSpace(p)
		if trimmed == "" || seen[trimmed] {
			continue
		}
		seen[trimmed] = true
		cleaned = append(cleaned, trimmed)
	}

	rc.ExternalAPIStructPrefixes = cleaned

	// Normalize ignore-float-fields: trim, lowercase, drop empties, deduplicate.
	seenIgnore := make(map[string]bool)
	cleanedIgnore := rc.IgnoreFloatFields[:0]
	for _, f := range rc.IgnoreFloatFields {
		trimmed := strings.ToLower(strings.TrimSpace(f))
		if trimmed == "" || seenIgnore[trimmed] {
			continue
		}
		seenIgnore[trimmed] = true
		cleanedIgnore = append(cleanedIgnore, trimmed)
	}
	rc.IgnoreFloatFields = cleanedIgnore

	// Check for unknown keys in the raw JSON (catches typos).
	if len(rawRulesJSON) > 0 {
		var raw map[string]any
		if err := json.Unmarshal(rawRulesJSON, &raw); err == nil {
			for key := range raw {
				if !knownRulesConfigKeys[key] {
					_, _ = fmt.Fprintf(
						w,
						"warning: unknown rules config key %q (known: disable, external-api-struct-prefixes, c008-ignore-fields)\n",
						key,
					)
				}
			}
		}
	}
}
