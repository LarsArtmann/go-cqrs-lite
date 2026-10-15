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
			body = make([]string, 0, strings.Count(md, "\n"))
		case open && trimmed == "```":
			blocks[len(blocks)-1].Code = strings.Join(body, "\n")
			open = false
		case open:
			body = append(body, line)
		}
	}

	return blocks
}

// splitTypeDecls hoists at-col-0 `type` and `func` declarations out of a
// snippet so the remaining statements can live inside a function body (a
// func body may not contain package-level type or func syntax).
//
// Decl recognition is signature-aware: a `func` line whose parameter list
// wraps (generic signatures do) opens signature mode, consuming lines until
// the body brace, and a body then runs until its braces balance — so
// `func Query[Q query.Query, R any](` stays one declaration instead of
// leaking its parameters into func main. One-line declarations
// (`type X string`, `func (T) M() string { return "x" }`) close immediately,
// preserving the historical single-line behavior.
func splitTypeDecls(code string) ([]string, []string) {
	var decls, stmts []string

	mode := 0 // 0 = scanning, 1 = inside a wrapped signature, 2 = inside a decl body
	depth := 0

	for line := range strings.SplitSeq(code, "\n") {
		trimmed := strings.TrimSpace(line)

		switch {
		case mode == 0 && (strings.HasPrefix(line, "type ") || strings.HasPrefix(line, "func ")):
			decls = append(decls, line)
			switch {
			case strings.HasSuffix(trimmed, "}"):
				mode = 0 // complete on this line
			case strings.HasSuffix(trimmed, "{"):
				mode, depth = 2, 1
			case strings.HasPrefix(line, "func "):
				mode = 1 // signature continues on the next line
			default:
				mode = 0 // one-line type alias
			}
		case mode == 1:
			decls = append(decls, line)

			switch {
			case strings.HasSuffix(trimmed, "{"):
				mode, depth = 2, 1
			case strings.HasSuffix(trimmed, "}"):
				mode = 0
			}
		case mode == 2:
			decls = append(decls, line)

			depth += strings.Count(line, "{") - strings.Count(line, "}")
			if depth <= 0 {
				mode = 0
			}
		default:
			stmts = append(stmts, line)
		}
	}

	return decls, stmts
}

// isWholeProgram reports whether a block carries its own package clause
// (a complete, runnable example).
func isWholeProgram(code string) bool {
	for ln := range strings.SplitSeq(code, "\n") {
		t := strings.TrimSpace(ln)
		if t == "" || strings.HasPrefix(t, "//") {
			continue
		}

		return strings.HasPrefix(t, "package ")
	}

	return false
}
