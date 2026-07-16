// Package suppression provides inline-comment suppression for cqrs-lint findings.
package suppression

import (
	"strings"

	"github.com/larsartmann/go-finding"
	"github.com/larsartmann/go-finding/pipeline"
)

// commentPrefix is the inline suppression comment prefix.
const commentPrefix = "//cqrs-lint:ignore"

// NewSuppressionFilter creates a FindingTransformer that marks findings
// suppressed by inline //cqrs-lint:ignore(rule-id) comments.
func NewSuppressionFilter() pipeline.FindingTransformer {
	return pipeline.NamedTransformerFunc(
		"cqrs-suppression",
		func(findings []finding.Finding) []finding.Finding {
			if len(findings) == 0 {
				return findings
			}

			var result []finding.Finding

			for _, f := range findings {
				if f.Snippet != "" && strings.Contains(f.Snippet, commentPrefix) {
					ruleID := extractRuleID(f.Snippet)
					if ruleID == string(f.Rule) {
						f.Suppression = &finding.Suppression{
							Kind:   finding.SuppressionInSource,
							Rule:   f.Rule,
							Reason: "inline suppression comment",
						}
					}
				}

				result = append(result, f)
			}

			return result
		},
	)
}

// ParseSuppressions extracts suppressed rule IDs from comment text.
func ParseSuppressions(commentText string) map[string]string {
	result := make(map[string]string)

	lines := strings.SplitSeq(commentText, "\n")
	for line := range lines {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, commentPrefix) {
			continue
		}

		rest := strings.TrimPrefix(line, commentPrefix)

		rest = strings.TrimSpace(rest)
		if strings.HasPrefix(rest, "(") {
			end := strings.Index(rest, ")")
			if end > 0 {
				ruleID := rest[1:end]
				reason := strings.TrimSpace(rest[end+1:])
				result[ruleID] = reason
			}
		}
	}

	return result
}

func extractRuleID(snippet string) string {
	_, after, ok := strings.Cut(snippet, commentPrefix)
	if !ok {
		return ""
	}

	rest := after

	rest = strings.TrimSpace(rest)
	if strings.HasPrefix(rest, "(") {
		end := strings.Index(rest, ")")
		if end > 0 {
			return rest[1:end]
		}
	}

	return ""
}
