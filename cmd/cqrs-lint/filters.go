package main

import (
	"strings"

	"github.com/larsartmann/go-finding"
	"github.com/larsartmann/go-finding/pipeline"
)

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

	var unique []finding.Finding

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

	var result []finding.Finding

	for _, f := range findings {
		if f.Severity.Compare(minS) >= 0 {
			result = append(result, f)
		}
	}

	return result
}

func filterByConfidence(findings []finding.Finding, minConf string) []finding.Finding {
	minC := parseConfidence(minConf)

	var result []finding.Finding

	for _, f := range findings {
		if f.Confidence >= minC {
			result = append(result, f)
		}
	}

	return result
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
