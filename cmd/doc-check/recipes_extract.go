package main

import (
	"strings"
)

// RecipeBlock is one fenced Go block from a markdown reference doc.
type RecipeBlock struct {
	Heading string // nearest preceding at-col-0 heading line
	Ordinal int    // 1-based block number within the same heading
	Line    int    // 1-based line of the opening fence
	Code    string // block content without the fences
}

func recipeKey(b RecipeBlock) string {
	return b.Heading + " #" + itoa(b.Ordinal)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	digits := ""
	for n > 0 {
		digits = string(rune('0'+n%10)) + digits
		n /= 10
	}
	return digits
}

// extractGoBlocks scans markdown for ```go fenced blocks. Non-Go fences
// (```sql, ```yaml, ...) are ignored.
func extractGoBlocks(md string) []RecipeBlock {
	var blocks []RecipeBlock
	heading := ""
	ordinal := 0
	var body []string
	open := false
	for i, line := range strings.Split(md, "\n") {
		trimmed := strings.TrimSpace(line)
		switch {
		case !open && strings.HasPrefix(line, "#"):
			heading = strings.TrimSpace(line)
			ordinal = 0
		case !open && strings.HasPrefix(trimmed, "```go"):
			ordinal++
			blocks = append(blocks, RecipeBlock{Heading: heading, Ordinal: ordinal, Line: i + 1})
			open = true
			body = nil
		case open && trimmed == "```":
			blocks[len(blocks)-1].Code = strings.Join(body, "\n")
			open = false
		case open:
			body = append(body, line)
		}
	}
	return blocks
}

// splitTypeDecls hoists at-col-0 `type` declarations out of a snippet so the
// remaining statements can live inside a function body (Go forbids type
// declarations mid-function in older consumers' mental models — and
// practically, a func body may not contain package-level type syntax).
func splitTypeDecls(code string) (decls []string, stmts []string) {
	inType := false
	for _, ln := range strings.Split(code, "\n") {
		switch {
		case inType:
			decls = append(decls, ln)
			if ln == "}" {
				inType = false
			}
		case strings.HasPrefix(ln, "type "):
			decls = append(decls, ln)
			if !strings.HasSuffix(strings.TrimSpace(ln), "}") {
				inType = true
			}
		default:
			stmts = append(stmts, ln)
		}
	}
	return decls, stmts
}

// isWholeProgram reports whether a block carries its own package clause
// (a complete, runnable example).
func isWholeProgram(code string) bool {
	for _, ln := range strings.Split(code, "\n") {
		t := strings.TrimSpace(ln)
		if t == "" || strings.HasPrefix(t, "//") {
			continue
		}
		return strings.HasPrefix(t, "package ")
	}
	return false
}
