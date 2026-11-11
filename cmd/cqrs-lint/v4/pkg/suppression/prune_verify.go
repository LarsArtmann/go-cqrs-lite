package suppression

import (
	"strings"

	"github.com/larsartmann/go-finding"
)

// WouldSuppress reports whether an inline directive for rule at (file, line)
// would suppress any of the given findings. It is the position-matcher
// inverse of the stale detector: a finding is suppressible by a directive on
// its own line, the line below (trailing styles), or the first non-blank
// line above it (comment-above-the-code).
//
// The prune verification path uses this to fail loudly when a removed
// "stale" directive turns out to suppress a real finding — the belt-and-
// braces guard for false-stale verdicts (issue #126).
func WouldSuppress(findings []finding.Finding, file string, line int, rule string) bool {
	lines, linesOK := readGoFileLines(file)

	for _, f := range findings {
		if string(f.Rule) != rule || string(f.Position.File) != file {
			continue
		}

		findingLine := f.Position.Line
		if findingLine == line || findingLine+1 == line {
			return true
		}

		if !linesOK {
			continue
		}

		for check := findingLine - 1; check >= 1; check-- {
			if check > len(lines) {
				continue
			}

			if strings.TrimSpace(lines[check-1]) == "" {
				continue
			}

			return check == line
		}
	}

	return false
}
