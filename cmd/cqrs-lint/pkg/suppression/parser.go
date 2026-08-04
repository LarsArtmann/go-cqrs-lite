// Package suppression provides inline-comment suppression for cqrs-lint findings.
package suppression

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/larsartmann/go-finding"
	"github.com/larsartmann/go-finding/pipeline"
)

// commentPrefix is the inline suppression comment prefix.
const commentPrefix = "//cqrs-lint:ignore"

// blockStartPrefix is the block suppression start comment prefix.
const blockStartPrefix = "//cqrs-lint:ignore-start"

// blockEndPrefix is the block suppression end comment prefix.
const blockEndPrefix = "//cqrs-lint:ignore-end"

// lineCache caches file contents to avoid re-reading for multiple findings.
type lineCache struct {
	mu    sync.Mutex
	files map[string][]string
}

func newLineCache() *lineCache {
	return &lineCache{files: make(map[string][]string)}
}

func (c *lineCache) getLines(path string) []string {
	c.mu.Lock()
	defer c.mu.Unlock()

	if lines, ok := c.files[path]; ok {
		return lines
	}

	f, err := os.Open(path)
	if err != nil {
		c.files[path] = nil

		return nil
	}
	defer func() { _ = f.Close() }()

	var lines []string

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	// Log scanner errors (e.g., bufio.ErrTooLong when a line exceeds the
	// 1MB buffer). Lines collected before the error are still valid for
	// suppression matching, so we cache partial results regardless.
	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: suppression scan of %s: %v\n", path, err)
	}

	c.files[path] = lines

	return lines
}

// normalizeCommentPrefix converts the Go-idiomatic "// cqrs-lint:..." (space
// after //) into the canonical "//cqrs-lint:..." form so both comment styles
// are recognized. gofmt does not normalize the space after //, so consumers
// naturally write "// cqrs-lint:ignore(C007)" — this must work.
func normalizeCommentPrefix(line string) string {
	return strings.Replace(line, "// cqrs-lint:", "//cqrs-lint:", 1)
}

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

// checkSuppressionInFile reads the source file and checks the finding's line
// and the line above for a suppression comment.
func checkSuppressionInFile(cache *lineCache, f finding.Finding) bool {
	filePath := string(f.Position.File)
	if filePath == "" {
		return false
	}

	lines := cache.getLines(filePath)
	if lines == nil {
		return false
	}

	ruleID := string(f.Rule)
	line := f.Position.Line // 1-based

	// Check the finding's own line.
	if line >= 1 && line <= len(lines) {
		suppressedRules := ParseSuppressions(lines[line-1])
		if _, ok := suppressedRules[ruleID]; ok {
			return true
		}
	}

	// Check the line above, skipping blank lines. A blank line is never
	// meaningful content — the suppression intent is clearly directed at
	// the next declaration. Without this skip, a blank line between a
	// suppression comment and the finding silently breaks suppression.
	for checkLine := line - 1; checkLine >= 1; checkLine-- {
		text := strings.TrimSpace(lines[checkLine-1])
		if text == "" {
			continue
		}

		suppressedRules := ParseSuppressions(lines[checkLine-1])
		if _, ok := suppressedRules[ruleID]; ok {
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
	filePath := string(f.Position.File)
	if filePath == "" {
		return false
	}

	lines := cache.getLines(filePath)
	if lines == nil {
		return false
	}

	ruleID := string(f.Rule)
	line := f.Position.Line // 1-based

	if line < 1 || line > len(lines) {
		return false
	}

	// Scan backward from the finding's line to find the nearest block start
	// or end. If we find a start first, we're inside a block. If we find an
	// end first (or run out of lines), we're not.
	for i := line; i >= 1; i-- {
		text := strings.TrimSpace(lines[i-1])
		// Normalize: accept "//cqrs-lint:ignore-start" and "// cqrs-lint:ignore-start"
		text = normalizeCommentPrefix(text)

		if strings.HasPrefix(text, blockEndPrefix) {
			return false // outside a block
		}

		if strings.HasPrefix(text, blockStartPrefix) {
			suppressedRules := parseBlockStart(text)
			if len(suppressedRules) == 0 {
				return true // suppresses all rules
			}

			_, ok := suppressedRules[ruleID]
			return ok
		}
	}

	return false
}

// parseBlockStart extracts the rule IDs from a block-start comment.
// Returns nil if no rule IDs are specified (suppresses all).
// Returns a map of rule IDs if specific rules are listed.
func parseBlockStart(text string) map[string]struct{} {
	rest := strings.TrimPrefix(text, blockStartPrefix)
	rest = strings.TrimSpace(rest)

	if !strings.HasPrefix(rest, "(") {
		return nil // no rule IDs = suppress all
	}

	end := strings.Index(rest, ")")
	if end <= 0 {
		return nil
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
		return nil
	}

	return result
}

// ParseSuppressions extracts suppressed rule IDs from comment text.
// Works on both single-line and multi-line comment text.
func ParseSuppressions(commentText string) map[string]string {
	result := make(map[string]string)

	lines := strings.SplitSeq(commentText, "\n")
	for line := range lines {
		line = strings.TrimSpace(line)
		// Normalize: accept both "//cqrs-lint:ignore" and "// cqrs-lint:ignore".
		line = normalizeCommentPrefix(line)
		// Find the suppression prefix anywhere in the line, not just at the
		// start. This recognizes end-of-line comments:
		//
		//	EventType = sdk.EventType //cqrs-lint:ignore(A008) re-export
		//
		// Without this, trailing suppressions after code were silently ignored
		// because HasPrefix requires the line to START with the comment prefix.
		idx := strings.Index(line, commentPrefix)
		if idx < 0 {
			continue
		}
		line = line[idx:]

		rest := strings.TrimPrefix(line, commentPrefix)

		rest = strings.TrimSpace(rest)
		if strings.HasPrefix(rest, "(") {
			end := strings.Index(rest, ")")
			if end > 0 {
				rawIDs := rest[1:end]
				reason := strings.TrimSpace(rest[end+1:])
				// Support comma-separated rule IDs: ignore(A001,E005).
				for id := range strings.SplitSeq(rawIDs, ",") {
					id = strings.TrimSpace(id)
					if id != "" {
						result[id] = reason
					}
				}
			}
		}
	}

	return result
}
