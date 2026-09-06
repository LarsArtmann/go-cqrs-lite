package suppression

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/larsartmann/go-finding"
)

// StaleSuppression describes a //cqrs-lint:ignore(RULE) comment that
// references a rule which does not fire at the comment's location.
type StaleSuppression struct {
	File   string
	Line   int // 1-based
	Rule   string
	Reason string
}

// suppressionLocation identifies a (file, line, rule) triple where a
// suppression comment exists and may or may not match a finding.
type suppressionLocation struct {
	file string
	line int
	rule string
}

// DetectStaleSuppressions scans Go files for //cqrs-lint:ignore(RULE) comments
// and //cqrs-lint:ignore-start / ignore-end block suppressions, returning any
// that don't correspond to a finding at that location. A suppression is
// "matched" if any finding exists for the same rule at the comment's line or
// the line below (suppression comments sit above findings).
func DetectStaleSuppressions(
	goFiles []string,
	findings []finding.Finding,
) []StaleSuppression {
	var stale []StaleSuppression

	for _, path := range goFiles {
		if !strings.HasSuffix(path, ".go") {
			continue
		}

		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		lines := strings.Split(string(data), "\n")

		// Build the matched-location set from this file's own lines: whether a
		// suppression comment is honored depends on the lines between it and
		// the finding (the suppression filter skips blank lines when looking
		// for the comment above a finding). Stale detection must mirror that
		// semantic exactly, or it reports working suppressions as stale.
		matched := matchedSuppressionsForFile(findings, path, lines)

		// Check inline suppressions.
		stale = append(stale, detectStaleInline(path, lines, matched)...)

		// Check block suppressions.
		stale = append(stale, detectStaleBlocks(path, lines, findings)...)
	}

	return stale
}

// matchedSuppressionsForFile marks every (file, line, rule) location whose
// //cqrs-lint:ignore(RULE) directive the suppression filter would honor for
// the given findings in this file: the finding's own line, the line below
// (leniency for trailing styles), and the first non-blank line above the
// finding — the same blank-skipping walk the suppression filter performs.
func matchedSuppressionsForFile(
	findings []finding.Finding,
	path string,
	lines []string,
) map[suppressionLocation]bool {
	matched := make(map[suppressionLocation]bool)

	for _, f := range findings {
		if string(f.Position.File) != path {
			continue
		}

		rule := string(f.Rule)
		line := f.Position.Line

		matched[suppressionLocation{path, line, rule}] = true
		matched[suppressionLocation{path, line + 1, rule}] = true

		// Mirror the suppression filter: scan up past blank lines and mark the
		// first non-blank line (the comment-above-the-code position).
		for checkLine := line - 1; checkLine >= 1; checkLine-- {
			if checkLine > len(lines) {
				continue
			}

			if strings.TrimSpace(lines[checkLine-1]) == "" {
				continue
			}

			matched[suppressionLocation{path, checkLine, rule}] = true

			break
		}
	}

	return matched
}

// detectStaleInline checks each line for inline //cqrs-lint:ignore(RULE)
// comments that don't match any finding.
func detectStaleInline(
	path string,
	lines []string,
	matched map[suppressionLocation]bool,
) []StaleSuppression {
	var stale []StaleSuppression

	for lineIdx, line := range lines {
		suppressions := ParseSuppressions(line)
		if len(suppressions) == 0 {
			continue
		}

		lineNum := lineIdx + 1

		// For combined directives (//cqrs-lint:ignore(A,B,C)), suppress the
		// stale warning for individual rules when at least one rule in the
		// directive DOES fire at this location. Combined directives are often
		// applied uniformly across parallel code paths (e.g., SQLite vs
		// in-memory implementations) where different detectors fire in each.
		// Reporting the non-firing rules as stale is noise — the directive is
		// still useful because at least one rule matches.
		anyMatched := false
		for rule := range suppressions {
			if matched[suppressionLocation{path, lineNum, rule}] {
				anyMatched = true
				break
			}
		}

		for rule, reason := range suppressions {
			loc := suppressionLocation{path, lineNum, rule}
			if !matched[loc] && !anyMatched {
				stale = append(stale, StaleSuppression{
					File:   path,
					Line:   lineNum,
					Rule:   rule,
					Reason: reason,
				})
			}
		}
	}

	return stale
}

// detectStaleBlocks scans for //cqrs-lint:ignore-start / ignore-end pairs and
// reports any block where no suppressed rule actually fires within the range.
func detectStaleBlocks(
	path string,
	lines []string,
	findings []finding.Finding,
) []StaleSuppression {
	var stale []StaleSuppression

	type block struct {
		startLine int
		rules     map[string]struct{} // nil = all rules
	}

	var openBlocks []block

	for lineIdx, raw := range lines {
		lineNum := lineIdx + 1

		// Only consider real Go comments, not string literals that mention the
		// block syntax (e.g. help-text examples in fmt.Println strings). The
		// block marker must be at the start of the comment text.
		cs := commentTextStart(raw)
		if cs < 0 {
			continue
		}

		commentText := normalizeCommentPrefix(strings.TrimSpace(raw[cs:]))

		if strings.HasPrefix(commentText, blockStartPrefix) {
			rules, valid := parseBlockStart(commentText)
			if !valid {
				continue // malformed directive — inert for suppression and staleness
			}

			openBlocks = append(openBlocks, block{
				startLine: lineNum,
				rules:     rules,
			})

			continue
		}

		if strings.HasPrefix(commentText, blockEndPrefix) {
			if len(openBlocks) == 0 {
				// Unmatched end: the directive suppresses nothing and almost
				// certainly signals a typo (wrong rule-ID list on the start,
				// deleted start, or stray end). Surface it like any other
				// dead suppression instead of silently ignoring it.
				stale = append(stale, StaleSuppression{
					File:   path,
					Line:   lineNum,
					Rule:   "block:end",
					Reason: "ignore-end without a matching ignore-start",
				})

				continue
			}

			blk := openBlocks[len(openBlocks)-1]
			openBlocks = openBlocks[:len(openBlocks)-1]

			// Check if any finding fires within this block's range.
			hasFinding := false

			for _, f := range findings {
				if string(f.Position.File) != path {
					continue
				}

				if f.Position.Line < blk.startLine || f.Position.Line > lineNum {
					continue
				}

				if blk.rules == nil {
					hasFinding = true

					break // all-rules block; any finding counts
				}

				if _, ok := blk.rules[string(f.Rule)]; ok {
					hasFinding = true

					break
				}
			}

			if !hasFinding {
				ruleDesc := "all"
				if len(blk.rules) > 0 {
					keys := make([]string, 0, len(blk.rules))
					for r := range blk.rules {
						keys = append(keys, r)
					}
					sort.Strings(keys)
					ruleDesc = strings.Join(keys, ",")
				}

				stale = append(stale, StaleSuppression{
					File:   path,
					Line:   blk.startLine,
					Rule:   "block:" + ruleDesc,
					Reason: "no findings within ignore-start/ignore-end block",
				})
			}
		}
	}

	// Unterminated starts: a block still open at EOF suppresses everything to
	// the end of the file — almost certainly not what the author wanted.
	for _, blk := range openBlocks {
		ruleDesc := "all"
		if len(blk.rules) > 0 {
			keys := make([]string, 0, len(blk.rules))
			for r := range blk.rules {
				keys = append(keys, r)
			}
			sort.Strings(keys)
			ruleDesc = strings.Join(keys, ",")
		}

		stale = append(stale, StaleSuppression{
			File:   path,
			Line:   blk.startLine,
			Rule:   "block:" + ruleDesc,
			Reason: "ignore-start without a matching ignore-end — suppresses to EOF",
		})
	}

	return stale
}

// FormatStaleWarning renders a stale suppression as a user-facing warning.
func FormatStaleWarning(s StaleSuppression) string {
	if s.Reason == "unknown rule" {
		return fmt.Sprintf(
			"warning: suppression at %s:%d references unknown rule %s — possible typo or stale rule ID",
			filepath.Base(s.File),
			s.Line,
			s.Rule,
		)
	}

	return fmt.Sprintf(
		"warning: stale suppression at %s:%d — rule %s does not fire here; safe to remove",
		filepath.Base(s.File), s.Line, s.Rule,
	)
}

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
func AuditSuppressions(
	goFiles []string,
	findings []finding.Finding,
	knownRuleIDs map[string]bool,
) []SuppressionAuditEntry {
	matched := make(map[suppressionLocation]bool)

	for _, f := range findings {
		file := string(f.Position.File)
		if file == "" {
			continue
		}

		rule := string(f.Rule)
		line := f.Position.Line

		matched[suppressionLocation{file, line, rule}] = true
		matched[suppressionLocation{file, line - 1, rule}] = true
		matched[suppressionLocation{file, line + 1, rule}] = true
	}

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
