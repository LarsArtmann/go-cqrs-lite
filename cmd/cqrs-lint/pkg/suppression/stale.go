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
	matched := make(map[suppressionLocation]bool)

	for _, f := range findings {
		file := string(f.Position.File)
		if file == "" {
			continue
		}

		rule := string(f.Rule)
		line := f.Position.Line

		// A suppression comment may be on the finding's line, the line above
		// (typical: comment above code), or the line below. Match all three.
		matched[suppressionLocation{file, line, rule}] = true
		matched[suppressionLocation{file, line - 1, rule}] = true
		matched[suppressionLocation{file, line + 1, rule}] = true
	}

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

		// Check inline suppressions.
		stale = append(stale, detectStaleInline(path, lines, matched)...)

		// Check block suppressions.
		stale = append(stale, detectStaleBlocks(path, lines, findings)...)
	}

	return stale
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

		for rule, reason := range suppressions {
			loc := suppressionLocation{path, lineNum, rule}
			if !matched[loc] {
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

	for lineIdx, line := range lines {
		lineNum := lineIdx + 1

		if strings.Contains(line, blockStartPrefix) {
			rules := parseBlockStart(line)
			openBlocks = append(openBlocks, block{
				startLine: lineNum,
				rules:     rules,
			})

			continue
		}

		if strings.Contains(line, blockEndPrefix) {
			if len(openBlocks) == 0 {
				continue // unmatched ignore-end; not a stale block issue
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

	return stale
}

// FormatStaleWarning renders a stale suppression as a user-facing warning.
func FormatStaleWarning(s StaleSuppression) string {
	return fmt.Sprintf(
		"warning: stale suppression at %s:%d — rule %s does not fire here; safe to remove",
		filepath.Base(s.File), s.Line, s.Rule,
	)
}
