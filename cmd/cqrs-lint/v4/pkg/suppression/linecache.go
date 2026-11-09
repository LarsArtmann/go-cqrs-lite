// Source-line scanning utilities for suppression matching: the lineCache
// that memoizes file contents, and the lexical classifiers (raw-string,
// block-comment, line-comment tracking) that decide which lines can carry a
// //cqrs-lint: directive.
package suppression

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"sync"
)

// lineCache caches file contents to avoid re-reading for multiple findings.
type lineCache struct {
	mu         sync.Mutex
	files      map[string][]string
	rawStrings map[string]map[int]bool // cached raw-string-line sets
}

func newLineCache() *lineCache {
	return &lineCache{
		files:      make(map[string][]string),
		rawStrings: make(map[string]map[int]bool),
	}
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

	// Bare defer (C015-exempt): the module sits at its dep budget and a
	// read-only os.File close error is not actionable — record.DeferClose
	// is not worth a runtime dep in a static-analysis tool (ADR-0144 §4).
	defer f.Close()

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

// getRawStringLines returns a set of 0-based line indices on which a
// //cqrs-lint: directive is inert: lines entirely inside a multi-line raw
// string literal (backtick) or entirely inside a multi-line /* */ block
// comment. Lines containing the opening/closing delimiter are NOT included —
// commentTextStart handles those via its within-line tracking. The result is
// cached alongside the file's lines.
func (c *lineCache) getRawStringLines(path string) map[int]bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if m, ok := c.rawStrings[path]; ok {
		return m
	}

	lines := c.files[path]
	m := computeRawStringLines(lines)
	for ln := range computeBlockCommentLines(lines) {
		m[ln] = true
	}

	c.rawStrings[path] = m

	return m
}

// computeRawStringLines scans lines forward and returns a set of 0-based line
// indices that fall entirely inside a multi-line raw string literal (backtick
// string that spans lines). This is needed because commentTextStart operates
// per-line and cannot know that a line is inside a raw string opened on a
// previous line. Without this, a //cqrs-lint:ignore directive that appears
// as literal text inside a multi-line raw string would be falsely treated as
// a real suppression comment.
func computeRawStringLines(lines []string) map[int]bool {
	result := make(map[int]bool)
	inRawString := false

	for i, line := range lines {
		if inRawString && !strings.Contains(line, "`") {
			result[i] = true
		}

		backtickCount := strings.Count(line, "`")
		if backtickCount%2 != 0 {
			inRawString = !inRawString
		}
	}

	return result
}

// computeBlockCommentLines scans lines forward and returns the set of 0-based
// line indices that lie entirely inside a multi-line /* */ block comment
// (opened on an earlier line and not closed on this one). A directive written
// as literal text inside such a comment must never suppress anything. Lines
// that OPEN or CLOSE the comment keep per-line handling: a directive can
// legitimately follow a closing "*/".
func computeBlockCommentLines(lines []string) map[int]bool {
	result := make(map[int]bool)
	inBlock := false

	for i, line := range lines {
		if inBlock && !strings.Contains(line, "*/") {
			result[i] = true
		}

		opens := strings.Count(line, "/*")
		closes := strings.Count(line, "*/")
		if (opens+closes)%2 != 0 {
			inBlock = !inBlock
		}
	}

	return result
}

// normalizeCommentPrefix converts any "// cqrs-lint:..." spelling (one or
// more spaces/tabs after //) into the canonical "//cqrs-lint:..." form so
// every comment style is recognized. gofmt does not normalize the space
// after //, so consumers naturally write "// cqrs-lint:ignore(C007)" — this
// must work, including with multiple spaces.
func normalizeCommentPrefix(line string) string {
	text := strings.TrimSpace(line)
	if !strings.HasPrefix(text, "//") {
		return text
	}

	rest := strings.TrimLeft(text[2:], " \t")
	if !strings.HasPrefix(rest, "cqrs-lint:") {
		return text // not a directive comment — leave untouched
	}

	return "//" + rest
}

// commentTextStart returns the byte index in line of the first "//" that
// begins a Go line comment AND is not inside a string literal (double- or
// backtick-quoted) or a block comment (/* */) opened earlier on the line.
// Only this first "//" starts the comment; everything after it is comment
// text, so a later "//cqrs-lint:ignore" appearing in an already-open comment
// or a doc string is literal text, not a directive.
// Returns -1 when the line has no out-of-string line comment.
func commentTextStart(line string) int {
	inDouble := false
	inBacktick := false
	inBlock := false

	for i := 0; i < len(line); i++ {
		c := line[i]

		switch {
		case inBacktick:
			if c == '`' {
				inBacktick = false
			}
		case inBlock:
			if c == '*' && i+1 < len(line) && line[i+1] == '/' {
				inBlock = false
				i++
			}
		case inDouble:
			if c == '\\' { // skip the next (escaped) byte
				i++
				continue
			}
			if c == '"' {
				inDouble = false
			}
		default:
			switch c {
			case '`':
				inBacktick = true
			case '"':
				inDouble = true
			case '/':
				if i+1 < len(line) {
					switch line[i+1] {
					case '/':
						return i
					case '*':
						inBlock = true
						i++
					}
				}
			}
		}
	}

	return -1
}
