// Package rules provides centralized rule registration for cqrs-lint.
package rules

import (
	"slices"
	"sync"
)

type RuleInfo struct {
	ID          string
	Name        string
	Category    string
	Severity    string
	Confidence  string
	Description string
	AutoFix     bool
	// DocURL is an optional link to detailed documentation for this rule.
	// When set, findings include it in Metadata for SARIF/JSON output.
	DocURL string
}

// allRulesCache memoizes the full rule catalog so AllRules is only built once.
// Called from detectorCategory, renderRulesTable, ListRules, and the meta-test.
//
//nolint:gochecknoglobals // intentional memoized cache
var allRulesCache = sync.OnceValue(func() []RuleInfo {
	return slices.Concat(
		correctnessRules(),
		apiRules(),
		boilerplateRules(),
		consistencyRules(),
		architectureRules(),
		securityRules(),
		performanceRules(),
		versionRules(),
		testingRules(),
		adoptionRules(),
	)
})

// AllRules returns metadata for all available rules, organized by category.
// The result is cached after the first call.
func AllRules() []RuleInfo {
	return allRulesCache()
}

// ruleLookupCache memoizes a rule-ID → RuleInfo map for LookupRule.
//
//nolint:gochecknoglobals // intentional memoized cache
var ruleLookupCache = sync.OnceValue(func() map[string]RuleInfo {
	m := make(map[string]RuleInfo, 180)

	for _, r := range allRulesCache() {
		m[r.ID] = r
	}

	return m
})

// LookupRule returns the catalog entry for the given rule ID (e.g., "C001").
// The boolean is false if the rule is not in the catalog.
func LookupRule(id string) (RuleInfo, bool) {
	r, ok := ruleLookupCache()[id]
	return r, ok
}


// correctnessRules aggregates both halves of the correctness family table.
func correctnessRules() []RuleInfo {
	return slices.Concat(correctnessRulesPart1(), correctnessRulesPart2())
}
