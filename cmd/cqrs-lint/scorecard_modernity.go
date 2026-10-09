package main

import (
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/v4/pkg/analyzer"
)

// ModernityGrade classifies HOW MODERN the project's go-cqrs-lite usage is —
// a separate axis from the adoption grade. Adoption measures breadth of the
// directly-imported surface; modernity measures whether that usage sits on
// the canonical path. A focused app composed via system.New with zero
// v5-removed API uses is a perfect consumer and must not be coached toward
// feature bloat to raise a breadth score.
//
// Grades:
//   - "Legacy"  — v5-removed APIs or deprecated transports are in use; every
//     day on them compounds the v5 migration.
//   - "Modern"  — v5-clean AND on the canonical path: composed via
//     system.New, or metaengine with declarative pushdown adopted.
//   - "Partial" — v5-clean but neither composition signal present.
func ModernityGrade(deprecated *ScorecardDeprecated, fp analyzer.FeatureProfile) string {
	if deprecated != nil &&
		(deprecated.RemovedAPIUses > 0 || deprecated.DeprecatedTransport > 0) {
		return "Legacy"
	}
	if fp.HasSystemComposition || (fp.HasMetaengine && fp.MetaenginePushdown) {
		return "Modern"
	}
	return "Partial"
}

// ModernityHint is the one-line coaching attached to each modernity grade.
func ModernityHint(grade string) string {
	switch grade {
	case "Modern":
		return "canonical path (system.New composition or declarative pushdown)"
	case "Legacy":
		return "v5-removed APIs in use — migrate before the v5 cut (see DEPRECATED SURFACES)"
	case "Partial":
		return "v5-clean; adopt system.New composition or FilterOnField/SortOnField pushdown to modernize"
	default:
		return ""
	}
}
