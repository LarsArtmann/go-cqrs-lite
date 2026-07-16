// Package rules provides centralized rule registration for cqrs-lint.
package rules

import (
	"fmt"
	"strings"

	"github.com/larsartmann/go-finding"

	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/analyzer"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/api"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/architecture"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/boilerplate"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/consistency"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/correctness"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/security"
)

// RuleInfo describes a registered rule.
type RuleInfo struct {
	ID          string
	Name        string
	Category    string
	Severity    string
	Confidence  string
	Description string
	AutoFix     bool
}

// AllRules returns metadata for all available rules.
func AllRules() []RuleInfo {
	return []RuleInfo{
		{
			ID:          "C001",
			Name:        "missing-tx-commit",
			Category:    "correctness",
			Severity:    "critical",
			Confidence:  "high",
			Description: "Transaction wrapper returns nil instead of tx.Commit()",
			AutoFix:     true,
		},
		{
			ID:          "C002",
			Name:        "broken-command-id",
			Category:    "correctness",
			Severity:    "critical",
			Confidence:  "high",
			Description: "Command ID() returns zero value — breaks idempotency",
			AutoFix:     false,
		},
		{
			ID:          "C003",
			Name:        "silent-unknown-event-fold",
			Category:    "correctness",
			Severity:    "error",
			Confidence:  "high",
			Description: "Fold function silently ignores unknown event types",
			AutoFix:     true,
		},
		{
			ID:          "C005",
			Name:        "raw-json-unmarshal-payload",
			Category:    "correctness",
			Severity:    "error",
			Confidence:  "high",
			Description: "Raw json.Unmarshal on event payload instead of DecodePayloadAuto",
			AutoFix:     false,
		},
		{
			ID:          "C006",
			Name:        "manual-version-arithmetic",
			Category:    "correctness",
			Severity:    "warning",
			Confidence:  "high",
			Description: "Manual event.Version(x.Int()+1) instead of x.Increment()",
			AutoFix:     true,
		},
		{
			ID:          "C007",
			Name:        "time-now-in-decider",
			Category:    "correctness",
			Severity:    "warning",
			Confidence:  "medium",
			Description: "time.Now() inside decider — non-deterministic",
			AutoFix:     false,
		},
		{
			ID:          "C008",
			Name:        "float64-for-money",
			Category:    "correctness",
			Severity:    "warning",
			Confidence:  "medium",
			Description: "float64 field with monetary name — use decimal or cents",
			AutoFix:     false,
		},
		{
			ID:          "C009",
			Name:        "panic-in-production",
			Category:    "correctness",
			Severity:    "warning",
			Confidence:  "high",
			Description: "panic() in production code — use error returns",
			AutoFix:     false,
		},
		{
			ID:          "C010",
			Name:        "swallowed-error-in-fold",
			Category:    "correctness",
			Severity:    "warning",
			Confidence:  "high",
			Description: "Error from decode/unmarshal discarded in fold",
			AutoFix:     false,
		},
		{
			ID:          "C012",
			Name:        "missing-error-return-in-with-tx",
			Category:    "correctness",
			Severity:    "critical",
			Confidence:  "high",
			Description: "withTx ignores body error — failures silently lost",
			AutoFix:     false,
		},
		{
			ID:          "A001",
			Name:        "manual-command-interface",
			Category:    "api",
			Severity:    "error",
			Confidence:  "high",
			Description: "Manual Type()/ID()/AggregateID() instead of BasicCommand embedding",
			AutoFix:     false,
		},
		{
			ID:          "A002",
			Name:        "newevent-manual-marshal",
			Category:    "api",
			Severity:    "warning",
			Confidence:  "high",
			Description: "event.NewEvent with json.Marshal — use event.New for auto-marshal",
			AutoFix:     false,
		},
		{
			ID:          "A003",
			Name:        "explicit-codec-in-decode",
			Category:    "api",
			Severity:    "info",
			Confidence:  "medium",
			Description: "Explicit codec in DecodePayload — use DecodePayloadAuto",
			AutoFix:     false,
		},
		{
			ID:          "A004",
			Name:        "untyped-dispatch-register",
			Category:    "api",
			Severity:    "warning",
			Confidence:  "medium",
			Description: "Untyped handler with type assertion — use RegisterTyped",
			AutoFix:     false,
		},
		{
			ID:          "A005",
			Name:        "custom-projection-runner",
			Category:    "api",
			Severity:    "warning",
			Confidence:  "medium",
			Description: "Manual bus.SubscribeAll without projectionhost",
			AutoFix:     false,
		},
		{
			ID:          "A006",
			Name:        "adapter-layer-wrapping",
			Category:    "api",
			Severity:    "info",
			Confidence:  "low",
			Description: "WrapEvent/UnwrapEvent adapter methods",
			AutoFix:     false,
		},
		{
			ID:          "A007",
			Name:        "dual-model-oo-functional",
			Category:    "api",
			Severity:    "error",
			Confidence:  "medium",
			Description: "Both OO aggregates and functional deciders",
			AutoFix:     false,
		},
		{
			ID:          "A008",
			Name:        "parallel-type-system",
			Category:    "api",
			Severity:    "error",
			Confidence:  "high",
			Description: "Custom AggregateID/Version types duplicating go-cqrs-lite",
			AutoFix:     false,
		},
		{
			ID:          "B001",
			Name:        "single-event-helper",
			Category:    "boilerplate",
			Severity:    "info",
			Confidence:  "high",
			Description: "Single-event helper function — use event.Single()",
			AutoFix:     false,
		},
		{
			ID:          "B002",
			Name:        "manual-repository-wiring",
			Category:    "boilerplate",
			Severity:    "info",
			Confidence:  "medium",
			Description: "Manual store+bus+repository wiring — use stack preset",
			AutoFix:     false,
		},
		{
			ID:          "B003",
			Name:        "subscribeall-large-switch",
			Category:    "boilerplate",
			Severity:    "info",
			Confidence:  "medium",
			Description: "SubscribeAll with >5 switch cases — split into projections",
			AutoFix:     false,
		},
		{
			ID:          "D001",
			Name:        "inconsistent-event-naming",
			Category:    "consistency",
			Severity:    "info",
			Confidence:  "medium",
			Description: "Mixed dot notation and PascalCase event types",
			AutoFix:     false,
		},
		{
			ID:          "D002",
			Name:        "inconsistent-json-casing",
			Category:    "consistency",
			Severity:    "info",
			Confidence:  "low",
			Description: "Mixed camelCase and snake_case JSON tags",
			AutoFix:     false,
		},
		{
			ID:          "E004",
			Name:        "event-not-in-catalog",
			Category:    "architecture",
			Severity:    "info",
			Confidence:  "medium",
			Description: "Event type emitted but not in catalog",
			AutoFix:     false,
		},
		{
			ID:          "E005",
			Name:        "command-without-handler",
			Category:    "architecture",
			Severity:    "warning",
			Confidence:  "medium",
			Description: "Command type defined but never registered",
			AutoFix:     false,
		},
		{
			ID:          "S001",
			Name:        "hardcoded-secrets",
			Category:    "security",
			Severity:    "critical",
			Confidence:  "medium",
			Description: "Potential hardcoded secret in string literal",
			AutoFix:     false,
		},
	}
}

// RegisterAll creates and returns all rule detectors.
func RegisterAll(ctx *analyzer.AnalysisContext) []finding.Detector {
	return []finding.Detector{
		// Correctness
		correctness.NewC001Detector(ctx),
		correctness.NewC002Detector(ctx),
		correctness.NewC003Detector(ctx),
		correctness.NewC005Detector(ctx),
		correctness.NewC006Detector(ctx),
		correctness.NewC007Detector(ctx),
		correctness.NewC008Detector(ctx),
		correctness.NewC009Detector(ctx),
		correctness.NewC010Detector(ctx),
		correctness.NewC012Detector(ctx),
		// API
		api.NewA001Detector(ctx),
		api.NewA002Detector(ctx),
		api.NewA003Detector(ctx),
		api.NewA004Detector(ctx),
		api.NewA005Detector(ctx),
		api.NewA006Detector(ctx),
		api.NewA007Detector(ctx),
		api.NewA008Detector(ctx),
		// Boilerplate
		boilerplate.NewB001Detector(ctx),
		boilerplate.NewB002Detector(ctx),
		boilerplate.NewB003Detector(ctx),
		// Consistency
		consistency.NewD001Detector(ctx),
		consistency.NewD002Detector(ctx),
		// Architecture
		architecture.NewE004Detector(ctx),
		architecture.NewE005Detector(ctx),
		// Security
		security.NewS001Detector(ctx),
	}
}

// RegisterCritical returns only Critical/High severity correctness rules (for --fast mode).
func RegisterCritical(ctx *analyzer.AnalysisContext) []finding.Detector {
	return []finding.Detector{
		correctness.NewC001Detector(ctx),
		correctness.NewC002Detector(ctx),
		correctness.NewC003Detector(ctx),
		correctness.NewC005Detector(ctx),
		correctness.NewC012Detector(ctx),
	}
}

// FilterByCategory returns only detectors matching the given categories.
func FilterByCategory(all []finding.Detector, categories []string) []finding.Detector {
	if len(categories) == 0 {
		return all
	}

	catSet := make(map[string]bool)
	for _, c := range categories {
		catSet[c] = true
	}

	var result []finding.Detector

	for _, d := range all {
		if catSet[detectorCategory(d.Name())] {
			result = append(result, d)
		}
	}

	return result
}

func detectorCategory(name string) string {
	info := AllRules()
	prefix := ""

	var prefixSb340 strings.Builder
	for i := 0; i < len(name) && i < len("C006"); i++ {
		c := name[i]
		if c >= 'A' && c <= 'Z' {
			prefixSb340.WriteString(string(c))
		} else {
			break
		}
	}
	prefix += prefixSb340.String()

	_ = prefix

	for _, r := range info {
		if r.ID != "" && len(name) >= len(r.ID) && name[:len(r.ID)] == r.ID {
			return r.Category
		}
	}

	return ""
}

// ListRules prints all available rules.
func ListRules() string {
	var sb strings.Builder
	sb.WriteString("Available cqrs-lint rules:\n\n")

	for _, r := range AllRules() {
		fixStr := ""
		if r.AutoFix {
			fixStr = " [auto-fixable]"
		}

		fmt.Fprintf(
			&sb,
			"  %s  %-35s  [%s]  %s%s\n",
			r.ID,
			r.Name,
			r.Severity,
			r.Description,
			fixStr,
		)
	}

	return sb.String()
}
