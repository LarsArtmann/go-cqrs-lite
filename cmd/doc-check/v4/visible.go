package main

import "strings"

// visibleLine is one markdown line outside fenced code blocks with inline
// code spans blanked to same-length spaces, so byte offsets of links and §
// tokens stay true while their content is excluded from matching.
type visibleLine struct {
	num  int
	text string
}

// isFenceToggle reports whether the line opens or closes a fenced block.
// Indented fences (list items) toggle too.
func isFenceToggle(line string) bool {
	return strings.HasPrefix(strings.TrimSpace(line), "```")
}

// visibleLines splits markdown into visible lines: fenced blocks are skipped
// entirely; inline code spans (`...`, “...“) become runs of spaces so
// generic brackets like [T](x, y) cannot masquerade as links or § refs.
func visibleLines(md string) []visibleLine {
	var out []visibleLine

	inFence := false

	for i, line := range strings.Split(md, "\n") {
		if isFenceToggle(line) {
			inFence = !inFence

			continue
		}

		if inFence {
			continue
		}

		out = append(out, visibleLine{num: i + 1, text: blankInlineCode(line)})
	}

	return out
}

// blankInlineCode replaces every well-formed inline code span with spaces of
// identical length. An unmatched backtick run is left alone.
func blankInlineCode(line string) string {
	runes := []rune(line)

	out := []rune(line)

	for i := 0; i < len(runes); {
		if runes[i] != '`' {
			i++

			continue
		}

		n := backtickRun(runes, i)

		end := findClosingRun(runes, i+n, n)
		if end < 0 {
			i += n

			continue
		}

		for j := i; j < end+n; j++ {
			out[j] = ' '
		}

		i = end + n
	}

	return string(out)
}

func backtickRun(runes []rune, i int) int {
	n := 0

	for i+n < len(runes) && runes[i+n] == '`' {
		n++
	}

	return n
}

// findClosingRun returns the index of a backtick run of exactly n runes at
// or after start, or -1.
func findClosingRun(runes []rune, start, n int) int {
	for i := start; i < len(runes); {
		if runes[i] != '`' {
			i++

			continue
		}

		m := backtickRun(runes, i)
		if m == n {
			return i
		}

		i += m
	}

	return -1
}
