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

	// Check the finding's own line and the line above.
	for _, checkLine := range []int{line, line - 1} {
		if checkLine < 1 || checkLine > len(lines) {
			continue
		}

		suppressedRules := ParseSuppressions(lines[checkLine-1])
		if _, ok := suppressedRules[ruleID]; ok {
			return true
		}
	}

	return false
}

// checkSuppressionInSnippet checks the Snippet field as a fallback for unit tests.
func checkSuppressionInSnippet(f finding.Finding) bool {
	if f.Snippet == "" {
		return false
	}

	ruleID := extractRuleID(f.Snippet)
	if ruleID == "" {
		return false
	}

	return ruleID == string(f.Rule)
}

// ParseSuppressions extracts suppressed rule IDs from comment text.
// Works on both single-line and multi-line comment text.
func ParseSuppressions(commentText string) map[string]string {
	result := make(map[string]string)

	lines := strings.SplitSeq(commentText, "\n")
	for line := range lines {
		line = strings.TrimSpace(line)
		// Accept both "//cqrs-lint:ignore" and "// cqrs-lint:ignore".
		line = strings.TrimPrefix(line, "// ")
		if !strings.HasPrefix(line, commentPrefix) {
			continue
		}

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

func extractRuleID(snippet string) string {
	// Accept both "//cqrs-lint:ignore" and "// cqrs-lint:ignore".
	snippet = strings.ReplaceAll(snippet, "// cqrs-lint:ignore", commentPrefix)

	_, after, ok := strings.Cut(snippet, commentPrefix)
	if !ok {
		return ""
	}

	rest := strings.TrimSpace(after)
	if strings.HasPrefix(rest, "(") {
		end := strings.Index(rest, ")")
		if end > 0 {
			// Return only the first rule for comma-separated IDs.
			first := strings.SplitN(rest[1:end], ",", 2)[0]
			return strings.TrimSpace(first)
		}
	}

	return ""
}
