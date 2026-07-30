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
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/performance"
	"github.com/larsartmann/go-cqrs-lite/cmd/cqrs-lint/pkg/rules/security"
)

func RegisterAll(ctx *analyzer.AnalysisContext) []finding.Detector {
	return []finding.Detector{
		// Correctness
		correctness.NewC001Detector(ctx),
		correctness.NewC002Detector(ctx),
		correctness.NewC003Detector(ctx),
		correctness.NewC004Detector(ctx),
		correctness.NewC005Detector(ctx),
		correctness.NewC006Detector(ctx),
		correctness.NewC007Detector(ctx),
		correctness.NewC008Detector(ctx),
		correctness.NewC009Detector(ctx),
		correctness.NewC010Detector(ctx),
		correctness.NewC011Detector(ctx),
		correctness.NewC012Detector(ctx),
		correctness.NewC013Detector(ctx),
		correctness.NewC014Detector(ctx),
		correctness.NewC015Detector(ctx),
		correctness.NewC016Detector(ctx),
		correctness.NewC017Detector(ctx),
		correctness.NewC019Detector(ctx),
		correctness.NewC020Detector(ctx),
		correctness.NewC022Detector(ctx),
		// API
		api.NewA001Detector(ctx),
		api.NewA002Detector(ctx),
		api.NewA003Detector(ctx),
		api.NewA004Detector(ctx),
		api.NewA005Detector(ctx),
		api.NewA006Detector(ctx),
		api.NewA007Detector(ctx),
		api.NewA008Detector(ctx),
		api.NewA009Detector(ctx),
		api.NewA010Detector(ctx),
		api.NewA011Detector(ctx),
		api.NewA012Detector(ctx),
		api.NewA013Detector(ctx),
		api.NewA014Detector(ctx),
		api.NewA015Detector(ctx),
		api.NewA016Detector(ctx),
		api.NewA017Detector(ctx),
		api.NewA018Detector(ctx),
		api.NewA019Detector(ctx),
		api.NewA027Detector(ctx),
		// Boilerplate
		boilerplate.NewB001Detector(ctx),
		boilerplate.NewB002Detector(ctx),
		boilerplate.NewB003Detector(ctx),
		boilerplate.NewB004Detector(ctx),
		boilerplate.NewB005Detector(ctx),
		boilerplate.NewB006Detector(ctx),
		boilerplate.NewB007Detector(ctx),
		boilerplate.NewB008Detector(ctx),
		boilerplate.NewB009Detector(ctx),
		boilerplate.NewB010Detector(ctx),
		boilerplate.NewB011Detector(ctx),
		boilerplate.NewB012Detector(ctx),
		boilerplate.NewB013Detector(ctx),
		boilerplate.NewB014Detector(ctx),
		boilerplate.NewB015Detector(ctx),
		boilerplate.NewB021Detector(ctx),
		boilerplate.NewB023Detector(ctx),
		boilerplate.NewB024Detector(ctx),
		// Performance
		performance.NewP001Detector(ctx),
		// Consistency
		consistency.NewD001Detector(ctx),
		consistency.NewD002Detector(ctx),
		consistency.NewD003Detector(ctx),
		consistency.NewD005Detector(ctx),
		consistency.NewD006Detector(ctx),
		// Architecture
		architecture.NewE001Detector(ctx),
		architecture.NewE002Detector(ctx),
		architecture.NewE003Detector(ctx),
		architecture.NewE004Detector(ctx),
		architecture.NewE005Detector(ctx),
		architecture.NewE006Detector(ctx),
		architecture.NewE007Detector(ctx),
		// Security
		security.NewS001Detector(ctx),
		security.NewS002Detector(ctx),
		security.NewS003Detector(ctx),
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
		catSet[strings.TrimSpace(c)] = true
	}

	var result []finding.Detector

	for _, d := range all {
		if catSet[detectorCategory(d.Name())] {
			result = append(result, d)
		}
	}

	return result
}

// FilterByRuleIDs returns only detectors whose name starts with one of the given rule IDs.
// Rule IDs are case-insensitive and matched as prefixes (e.g., "C001" matches "C001-broken-command-id").
func FilterByRuleIDs(all []finding.Detector, ruleIDs []string) []finding.Detector {
	if len(ruleIDs) == 0 {
		return all
	}

	idSet := make(map[string]bool)
	for _, id := range ruleIDs {
		idSet[strings.ToUpper(strings.TrimSpace(id))] = true
	}

	var result []finding.Detector
	for _, d := range all {
		for id := range idSet {
			if strings.HasPrefix(d.Name(), id) {
				result = append(result, d)

				break
			}
		}
	}

	return result
}

// IsRuleID returns true if s looks like a rule ID (uppercase letter + 3 digits, e.g., "C001").
func IsRuleID(s string) bool {
	s = strings.ToUpper(strings.TrimSpace(s))
	if len(s) < 4 {
		return false
	}
	if s[0] < 'A' || s[0] > 'Z' {
		return false
	}
	for i := 1; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}

	return true
}

func detectorCategory(name string) string {
	info := AllRules()

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
