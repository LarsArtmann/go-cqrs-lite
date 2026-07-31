package main

import (
	"path/filepath"
	"strings"

	"github.com/larsartmann/go-finding"
	"github.com/larsartmann/go-finding/pipeline"
)

// consumerOnlyRules are rules that coach CONSUMERS of go-cqrs-lite to adopt
// features (use BasicCommand embedding, adopt the built-in event bus, etc.).
// When linting the library itself (self-lint mode), these rules are
// meaningless — the library defines these features, it can't "adopt" them.
// Auto-suppressing them eliminates the need for 181+ manual inline
// suppressions across the library's own source.
var consumerOnlyRules = map[string]bool{ //nolint:gochecknoglobals // static lookup table
	"A001": true, // manual-command-interface
	"A008": true, // parallel-type-system
	"A020": true, // custom-event-bus
	"A021": true, // custom-event-store
	"A023": true, // custom-snapshot-store
	"E005": true, // uncataloged-event-type (library defines types, not catalogs them)
	"E007": true, // unregistered-query-type (library defines query types)
}

// filterLibrarySelfLint drops consumer-only coaching rules when the analyzed
// code is the go-cqrs-lite library itself. The dropped findings are returned
// as suppressed so they appear in --show-suppressed auditing.
func filterLibrarySelfLint(
	findings []finding.Finding,
	isLibrary bool,
) (active, librarySuppressed []finding.Finding) {
	if !isLibrary {
		return findings, nil
	}

	active = make([]finding.Finding, 0, len(findings))

	for _, f := range findings {
		if consumerOnlyRules[string(f.Rule)] {
			f.Suppression = &finding.Suppression{
				Kind:   finding.SuppressionInSource,
				Rule:   f.Rule,
				Reason: "auto-suppressed: consumer-only rule in library self-lint mode",
			}
			librarySuppressed = append(librarySuppressed, f)
		} else {
			active = append(active, f)
		}
	}

	return active, librarySuppressed
}

func collectFindings(result *pipeline.PipelineResult) []finding.Finding {
	var all []finding.Finding
	for _, iter := range result.Iterations {
		all = append(all, iter.Findings()...)
	}

	if result.Verification != nil {
		all = append(all, result.Verification.Remaining...)
		all = append(all, result.Verification.NewFindings...)
	}

	seen := make(map[finding.ID]bool)

	unique := make([]finding.Finding, 0, len(all))

	for _, f := range all {
		if seen[f.ID] {
			continue
		}

		seen[f.ID] = true
		unique = append(unique, f)
	}

	return unique
}

func filterBySeverity(findings []finding.Finding, minSev string) []finding.Finding {
	minS := parseSeverity(minSev)

	result := make([]finding.Finding, 0, len(findings))

	for _, f := range findings {
		if f.Severity.Compare(minS) >= 0 {
			result = append(result, f)
		}
	}

	return result
}

func filterByConfidence(findings []finding.Finding, minConf string) []finding.Finding {
	minC := parseConfidence(minConf)

	result := make([]finding.Finding, 0, len(findings))

	for _, f := range findings {
		if f.Confidence >= minC {
			result = append(result, f)
		}
	}

	return result
}

// filterSuppressed splits findings into active and suppressed sets based
// on the Suppression field (set by //cqrs-lint:ignore(RULE) comments).
// Returns both slices: active findings feed severity/confidence filters,
// health score, and the error-exit check. Suppressed findings are retained
// for --show-suppressed auditing.
func filterSuppressed(findings []finding.Finding) (active, suppressed []finding.Finding) {
	active = make([]finding.Finding, 0, len(findings))
	suppressed = make([]finding.Finding, 0)

	for _, f := range findings {
		if f.Suppression != nil {
			suppressed = append(suppressed, f)
		} else {
			active = append(active, f)
		}
	}

	return active, suppressed
}

func filterByExcludedPaths(findings []finding.Finding, patterns []string) []finding.Finding {
	if len(patterns) == 0 {
		return findings
	}

	var result []finding.Finding
	for _, f := range findings {
		excluded := false
		for _, pattern := range patterns {
			pattern = strings.TrimSpace(pattern)
			if pattern == "" {
				continue
			}
			if matched, _ := filepath.Match(
				pattern,
				filepath.Base(string(f.Position.File)),
			); matched {
				excluded = true
				break
			}
			if strings.Contains(string(f.Position.File), pattern) {
				excluded = true
				break
			}
		}
		if !excluded {
			result = append(result, f)
		}
	}

	return result
}

// filterFPSuspects returns only findings with confidence below Medium —
// the ones most likely to be false positives. Used by --fp-suspects mode
// to help consumers batch-review low-confidence findings for suppression.
func filterFPSuspects(findings []finding.Finding) []finding.Finding {
	result := make([]finding.Finding, 0, len(findings))

	for _, f := range findings {
		if f.Confidence < finding.ConfidenceMedium {
			result = append(result, f)
		}
	}

	return result
}

// financialEscalatedRules are rules whose severity is escalated to Error when
// the project domain is financial. Security and money-handling bugs in
// financial systems are always errors, never warnings.
var financialEscalatedRules = map[string]bool{ //nolint:gochecknoglobals // static lookup table
	"S001": true, // hardcoded-secret
	"S002": true, // signing-disabled
	"S003": true, // encryption-disabled
	"S005": true, // hmac-secret-too-short
	"S006": true, // encryption-key-too-short
	"S007": true, // insecure-random
	"S008": true, // missing-event-signing
	"S009": true, // missing-event-encryption
	"S010": true, // encryption-signing-mismatch
	"C008": true, // money-as-float64
}

// applyDomainBias escalates finding severities based on the project domain.
// Financial domains escalate security and money-handling rules to Error so
// they cannot be filtered out by --min-severity=warning.
func applyDomainBias(
	findings []finding.Finding,
	domain analyzer.DomainKind,
) []finding.Finding {
	if domain != analyzer.DomainFinancial {
		return findings
	}

	result := make([]finding.Finding, len(findings))
	for i, f := range findings {
		if financialEscalatedRules[string(f.Rule)] && !f.Severity.AtLeast(finding.SeverityError) {
			f.Severity = finding.SeverityError
			if f.Message != "" {
				f.Message += " [escalated: financial domain]"
			}
		}
		result[i] = f
	}

	return result
}

func parseSeverity(s string) finding.Severity {
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

func parseConfidence(s string) finding.Confidence {
	switch strings.ToLower(s) {
	case "high":
		return finding.ConfidenceHigh
	case "medium":
		return finding.ConfidenceMedium
	case "low":
		return finding.ConfidenceLow
	default:
		return finding.ConfidenceLow
	}
}
