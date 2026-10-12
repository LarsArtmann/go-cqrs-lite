package suppression

import (
	"os"
	"sort"
	"strings"

	"github.com/larsartmann/go-finding"
)

// AuditStatus describes the health of a single inline suppression.
type AuditStatus string

const (
	// AuditActive means a finding for this rule fires at this location.
	AuditActive AuditStatus = "active"
	// AuditStale means no finding fires here; the suppression is dead weight.
	AuditStale AuditStatus = "stale"
	// AuditUnknownRule means the rule ID is not registered (typo or removed).
	AuditUnknownRule AuditStatus = "unknown-rule"
)

// SuppressionAuditEntry describes one inline suppression comment and its
// cross-referenced audit status. Used by `cqrs-lint doctor --audit-suppressions`.
type SuppressionAuditEntry struct {
	File   string
	Line   int
	Rule   string
	Reason string
	Status AuditStatus
}

// AuditSuppressions collects ALL inline //cqrs-lint:ignore(RULE) comments and
// cross-references them with findings and known rule IDs to classify each as
// active, stale, or unknown-rule. Unlike DetectStaleSuppressions (which only
// returns stale ones), this returns every suppression so developers can see
// the full picture during periodic suppression health checks.
//
// The active/stale verdict uses the SAME matcher as the suppression filter and
// DetectStaleSuppressions (matchedSuppressionsForFile: finding line, trailing
// line, first non-blank line above) so a directive the filter honors can never
// classify stale here. The doctor audit and the lint run must agree by
// construction (2026-10-12 browser-history false-stale lesson).
func AuditSuppressions(
	goFiles []string,
	findings []finding.Finding,
	knownRuleIDs map[string]bool,
) []SuppressionAuditEntry {
	var entries []SuppressionAuditEntry

	for _, path := range goFiles {
		if !strings.HasSuffix(path, ".go") {
			continue
		}

		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		lines := strings.Split(string(data), "\n")
		matched := matchedSuppressionsForFile(findings, path, lines)

		for lineIdx, line := range lines {
			suppressions := ParseSuppressions(line)
			if len(suppressions) == 0 {
				continue
			}

			lineNum := lineIdx + 1

			anyMatched := false
			for rule := range suppressions {
				if matched[suppressionLocation{path, lineNum, rule}] {
					anyMatched = true
					break
				}
			}

			for rule, reason := range suppressions {
				entry := SuppressionAuditEntry{
					File:   path,
					Line:   lineNum,
					Rule:   rule,
					Reason: reason,
				}

				switch {
				case len(knownRuleIDs) > 0 && !knownRuleIDs[rule]:
					entry.Status = AuditUnknownRule
				case matched[suppressionLocation{path, lineNum, rule}]:
					entry.Status = AuditActive
				case anyMatched:
					entry.Status = AuditActive
				default:
					entry.Status = AuditStale
				}

				entries = append(entries, entry)
			}
		}
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].File != entries[j].File {
			return entries[i].File < entries[j].File
		}
		return entries[i].Line < entries[j].Line
	})

	return entries
}

// DetectUnknownRuleSuppressions scans Go files for //cqrs-lint:ignore(XYZ)
// comments where XYZ is not a registered rule ID. These are likely typos
// (e.g. PO12 with letter O instead of zero) or stale references to rules that
// were renamed/removed. knownRuleIDs is the set of all currently-registered
// rule IDs (from rules.AllRules or rules.LookupRule).
func DetectUnknownRuleSuppressions(
	goFiles []string,
	knownRuleIDs map[string]bool,
) []StaleSuppression {
	if len(knownRuleIDs) == 0 {
		return nil
	}

	var unknown []StaleSuppression

	for _, path := range goFiles {
		if !strings.HasSuffix(path, ".go") {
			continue
		}

		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		lines := strings.Split(string(data), "\n")

		for lineIdx, line := range lines {
			suppressions := ParseSuppressions(line)
			if len(suppressions) == 0 {
				continue
			}

			lineNum := lineIdx + 1

			for rule := range suppressions {
				if !knownRuleIDs[rule] {
					unknown = append(unknown, StaleSuppression{
						File:   path,
						Line:   lineNum,
						Rule:   rule,
						Reason: "unknown rule",
					})
				}
			}
		}
	}

	return unknown
}
