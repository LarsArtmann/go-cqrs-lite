// Package suppression provides inline-comment suppression for cqrs-lint findings.
//
// Alignment with go-finding's Suppression model (decision, 2026-09-17):
// cqrs-lint stamps only SuppressionInSource with a valid Kind/Rule/Reason —
// the other kinds map onto existing cqrs-lint mechanisms instead of
// duplicating them:
//
//   - go-finding "in-review" (accepted false positive) ≈ cqrs-lint's inline
//     ignore comment plus --show-suppressed for auditing.
//   - go-finding "in-config" ≈ cqrs-lint's rules.disable config and
//     --exclude-rules, which drop findings before suppression runs.
//   - go-finding ExpiresAt (time-based expiry) is deliberately NOT parsed
//     from ignore comments: staleness is handled structurally by
//     DetectStaleSuppressions (a directive that no longer suppresses is
//     reported) and --fail-on-stale-suppressions / doctor --audit-suppressions.
//     Date-stamped comments rot silently; a stale-directive gate fails loudly.
package suppression

import (
	"strings"

	"github.com/larsartmann/go-finding"
	"github.com/larsartmann/go-finding/pipeline"
)

// blockStartPrefix is the block suppression start comment prefix.
const blockStartPrefix = "//cqrs-lint:ignore-start"

// blockEndPrefix is the block suppression end comment prefix.
const blockEndPrefix = "//cqrs-lint:ignore-end"

// NewSuppressionFilter creates a FindingTransformer that marks findings
// suppressed by inline //cqrs-lint:ignore(rule-id) comments.
//
// The filter reads the source file at the finding's position and checks
// the finding's own line and the line above for a suppression comment.
// This is necessary because detectors do not populate the Snippet field.
// If the file cannot be read (e.g., in unit tests), it falls back to
// checking the Snippet field.
//
//nolint:ireturn // factory returns public interface
func NewSuppressionFilter() pipeline.FindingTransformer {
	cache := newLineCache()

	return pipeline.NamedTransformerFunc(
		"cqrs-suppression",
		func(findings []finding.Finding) []finding.Finding {
			if len(findings) == 0 {
				return findings
			}

			var result []finding.Finding

			for _, f := range findings {
				matched := checkSuppressionInFile(cache, f)
				if !matched {
					matched = checkBlockSuppressionInFile(cache, f)
				}

				if !matched {
					matched = checkSuppressionInSnippet(f)
				}

				if matched {
					f.Suppression = &finding.Suppression{
						Kind:   finding.SuppressionInSource,
						Rule:   f.Rule,
						Reason: "inline suppression comment",
					}
				}

				result = append(result, f)
			}

			return result
		},
	)
}

// findingLines holds the cached source lines and finding metadata needed by
// both checkSuppressionInFile and checkBlockSuppressionInFile.
type findingLines struct {
	lines    []string
	rawLines map[int]bool
	ruleID   string
	line     int
}

// loadFindingLines extracts the file path from the finding, loads cached
// source lines, and bundles them with the rule ID and line number. Returns
// ok=false when the file path is empty or the source could not be loaded.
func loadFindingLines(cache *lineCache, f finding.Finding) (findingLines, bool) {
	filePath := string(f.Position.File)
	if filePath == "" {
		return findingLines{}, false
	}

	lines := cache.getLines(filePath)
	if lines == nil {
		return findingLines{}, false
	}

	return findingLines{
		lines:    lines,
		rawLines: cache.getRawStringLines(filePath),
		ruleID:   string(f.Rule),
		line:     f.Position.Line, // 1-based
	}, true
}

// checkSuppressionInFile reads the source file and checks the finding's line
// and the line above for a suppression comment.
func checkSuppressionInFile(cache *lineCache, f finding.Finding) bool {
	fl, ok := loadFindingLines(cache, f)
	if !ok {
		return false
	}

	// Check the finding's own line.
	if fl.line >= 1 && fl.line <= len(fl.lines) && !fl.rawLines[fl.line-1] {
		suppressedRules := ParseSuppressions(fl.lines[fl.line-1])
		if _, ok := suppressedRules[fl.ruleID]; ok {
			return true
		}
	}

	// Check the line above, skipping blank lines. A blank line is never
	// meaningful content — the suppression intent is clearly directed at
	// the next declaration. Without this skip, a blank line between a
	// suppression comment and the finding silently breaks suppression.
	// The line is clamped to the file length first: a finding whose position
	// is beyond EOF (file truncated after the AST was loaded) must not
	// panic — mirror the bounds guard in checkBlockSuppressionInFile.
	for checkLine := min(fl.line, len(fl.lines)) - 1; checkLine >= 1; checkLine-- {
		if fl.rawLines[checkLine-1] {
			continue // line is inside a multi-line raw string
		}
		text := strings.TrimSpace(fl.lines[checkLine-1])
		if text == "" {
			continue
		}

		suppressedRules := ParseSuppressions(fl.lines[checkLine-1])
		if _, ok := suppressedRules[fl.ruleID]; ok {
			return true
		}

		break // first non-blank line above — stop scanning
	}

	return false
}

// checkSuppressionInSnippet checks the Snippet field as a fallback for unit tests.
func checkSuppressionInSnippet(f finding.Finding) bool {
	if f.Snippet == "" {
		return false
	}

	suppressedRules := ParseSuppressions(f.Snippet)
	_, ok := suppressedRules[string(f.Rule)]

	return ok
}

// checkBlockSuppressionInFile scans backward from the finding's line to
// determine if it falls inside a //cqrs-lint:ignore-start / ignore-end block.
// If the block start specifies rule IDs (e.g. ignore-start(A001)), only
// those rules are suppressed. If no IDs are specified, all rules are suppressed.
func checkBlockSuppressionInFile(cache *lineCache, f finding.Finding) bool {
	fl, ok := loadFindingLines(cache, f)
	if !ok {
		return false
	}

	if fl.line < 1 || fl.line > len(fl.lines) {
		return false
	}

	// Scan backward from the finding's line to find the nearest block start
	// or end. If we find a start first, we're inside a block. If we find an
	// end first (or run out of lines), we're not.
	for i := fl.line; i >= 1; i-- {
		if fl.rawLines[i-1] {
			continue // line is inside a multi-line raw string
		}
		text := strings.TrimSpace(fl.lines[i-1])
		// Normalize: accept "//cqrs-lint:ignore-start" and "// cqrs-lint:ignore-start"
		text = normalizeCommentPrefix(text)

		if strings.HasPrefix(text, blockEndPrefix) {
			return false // outside a block
		}

		if strings.HasPrefix(text, blockStartPrefix) {
			suppressedRules, valid := parseBlockStart(text)
			if !valid {
				return false // malformed directive — suppress nothing (fail closed)
			}

			if len(suppressedRules) == 0 {
				return true // suppresses all rules
			}

			_, ok := suppressedRules[fl.ruleID]
			return ok
		}
	}

	return false
}

// parseBlockStart extracts the rule IDs from a block-start comment.
// A bare blockStartPrefix (no ID list) suppresses ALL rules: (nil, true).
// A parenthesized ID list returns the parsed rule IDs: (ids, true).
// Malformed directives — unclosed parenthesis or an ID list with no valid
// IDs — return (nil, false): the block suppresses NOTHING. Fail-closed beats
// fail-open: a typo like ignore-start(A01 must never silently disable every
// rule inside the block.
func parseBlockStart(text string) (map[string]struct{}, bool) {
	rest := strings.TrimPrefix(text, blockStartPrefix)
	rest = strings.TrimSpace(rest)

	if !strings.HasPrefix(rest, "(") {
		return nil, true // no rule IDs = suppress all
	}

	end := strings.Index(rest, ")")
	if end <= 0 {
		return nil, false // malformed: unclosed parenthesis
	}

	rawIDs := rest[1:end]
	result := make(map[string]struct{})

	for id := range strings.SplitSeq(rawIDs, ",") {
		id = strings.TrimSpace(id)
		if id != "" {
			result[id] = struct{}{}
		}
	}

	if len(result) == 0 {
		return nil, false // malformed: empty ID list
	}

	return result, true
}
