package suppression

import (
	"fmt"
	"os"
	"path/filepath"
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
// and returns any that don't correspond to a finding at that location. A
// suppression is "matched" if any finding exists for the same rule at the
// comment's line or the line below (suppression comments sit above findings).
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

		for lineIdx, line := range strings.Split(string(data), "\n") {
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
