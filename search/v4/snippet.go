package search

import "strings"

// DefaultSnippetRadius is the number of characters kept on each side of the
// first match when Snippet is called with radius 0.
const DefaultSnippetRadius = 60

// snippetSnap is how far a cut may move to land on a word boundary.
const snipLimit = 12

// ellipsis marks a window that was cut.
const ellipsis = "…"

// Snippet returns the text around the first term occurrence, with every term
// inside the window marked.
//
// The window keeps up to radius characters on each side of the first match
// and moves outward to a word boundary when one lies within a few
// characters. Cuts are made on rune boundaries only; a cut edge starts or
// ends with an unmarked ellipsis. An empty query or a text that matches no
// term yields the start of the text as one unmarked segment.
func Snippet(text, query string, radius int) []Segment {
	return SnippetTerms(text, strings.Fields(collapseWhitespace(query)), radius)
}

// SnippetTerms is [Snippet] over terms the caller already split.
func SnippetTerms(text string, terms []string, radius int) []Segment {
	if radius <= 0 {
		radius = DefaultSnippetRadius
	}
	runes := []rune(text)
	marked, first, matchEnd := markTerms(runes, terms)

	if first < 0 {
		end := min(len(runes), radius)
		return Plain(string(runes[:snapEnd(runes, end)]))
	}

	start := snapStart(runes, max(0, first-radius))
	end := snapEnd(runes, min(len(runes), matchEnd+radius))
	start, end = trimEdges(runes, start, end, first)
	return segmentsOf(runes, marked, start, max(start, end))
}

// trimEdges drops whitespace at the window edges and returns the new bounds.
func trimEdges(runes []rune, start, end, first int) (int, int) {
	for start < first && isSpace(runes[start]) {
		start++
	}
	for end > first+1 && isSpace(runes[end-1]) {
		end--
	}
	return start, end
}

// isSpace reports whether a rune is whitespace.
func isSpace(r rune) bool {
	return r == ' ' || r == '\t' || r == '\n' || r == '\r'
}

// markTerms marks every case-folded term occurrence in the text. It
// returns the mark per rune, the index of the earliest marked rune (-1 when
// none) and the index just past the marked run that starts there, so a
// window never cuts inside a match.
func markTerms(runes []rune, terms []string) (marked []bool, first, matchEnd int) {
	marked = make([]bool, len(runes))
	folded := []rune(Fold(string(runes)))
	first, matchEnd = -1, 0
	for _, term := range terms {
		needle := []rune(Fold(term))
		if len(needle) == 0 || len(needle) > len(folded) {
			continue
		}
		for i := 0; i+len(needle) <= len(folded); i++ {
			if !runesEqual(folded[i:i+len(needle)], needle) {
				continue
			}
			if first < 0 || i < first {
				first, matchEnd = i, i+len(needle)
			}
			for j := i; j < i+len(needle); j++ {
				marked[j] = true
			}
		}
	}
	if first >= 0 {
		for matchEnd < len(marked) && marked[matchEnd] {
			matchEnd++
		}
	}
	return marked, first, matchEnd
}

// runesEqual compares two rune slices.
func runesEqual(a, b []rune) bool {
	for i := range b {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// snapStart moves a cut start backward to the word boundary before it when
// the cut lands inside a word, so the first word is whole.
func snapStart(runes []rune, start int) int {
	if start == 0 || isWordBoundary(runes[start-1]) {
		return start
	}
	for i := start - 1; i > max(0, start-snipLimit); i-- {
		if isWordBoundary(runes[i]) {
			return i + 1
		}
	}
	return start
}

// snapEnd moves a cut end forward to the word boundary after it when the cut
// lands inside a word, so the last word is whole.
func snapEnd(runes []rune, end int) int {
	if end >= len(runes) || end == 0 || isWordBoundary(runes[end-1]) {
		return end
	}
	for i := end; i < min(len(runes), end+snipLimit); i++ {
		if isWordBoundary(runes[i]) {
			return i + 1
		}
	}
	return end
}

// isWordBoundary reports whether a rune separates words.
func isWordBoundary(r rune) bool {
	return r == ' ' || r == '\t' || r == '\n' || r == '\r' ||
		r == '.' || r == ',' || r == ';' || r == '!' || r == '?' || r == ')' || r == ']' ||
		r == '-' || r == '|' || r == '/'
}

// segmentsOf groups a window into runs of equal marking and adds the
// ellipses at cut edges.
func segmentsOf(runes []rune, marked []bool, start, end int) []Segment {
	var segments []Segment
	if start > 0 {
		segments = append(segments, Segment{Text: ellipsis})
	}
	for i := start; i < end; {
		j := i + 1
		for j < end && marked[j] == marked[i] {
			j++
		}
		segments = append(segments, Segment{
			Text:   string(runes[i:j]),
			Marked: marked[i],
		})
		i = j
	}
	if end < len(runes) {
		segments = append(segments, Segment{Text: ellipsis})
	}
	return segments
}
