package main

import (
	"regexp"
	"strings"
	"unicode"
)

// headingLink strips markdown links/images from heading text — GitHub slugs
// the RENDERED text, so [Domain Language](DOMAIN_LANGUAGE.md) contributes
// "Domain Language" to the slug, not its URL.
var headingLink = regexp.MustCompile(`!?\[([^\]]*)\]\([^)]*\)`)

// heading is one ATX markdown heading with its computed GitHub anchor slug
// and leading section number ("" when unnumbered).
type heading struct {
	text   string
	slug   string
	number string
	line   int
}

// githubSlug converts heading text to a GitHub.com anchor slug. Rules are
// GitHub-exact (empirically pinned against this repo's hand-written TOCs):
// lowercase; keep letters, digits, underscores, and hyphens; drop all other
// punctuation (periods, parens, backticks, en/em dashes, arrows, ...); every
// space becomes a hyphen — consecutive spaces therefore yield consecutive
// hyphens ("a — b" -> "a--b"). Inline-code content is KEPT (only the
// backticks drop), and underscores are never stripped.
func githubSlug(text string) string {
	var b strings.Builder

	for _, r := range strings.TrimSpace(headingLink.ReplaceAllString(text, "$1")) {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-':
			b.WriteRune(unicode.ToLower(r))
		case r == ' ' || r == '\t':
			b.WriteByte('-')
		}
	}

	return b.String()
}

// sectionNumberRe matches the leading dotted section number of a heading:
// "2.13b Retry with Backoff" -> "2.13b", "2. Composition Recipes" -> "2".
var sectionNumberRe = regexp.MustCompile(`^\d+(?:\.\d+)*[a-z]*`)

// sectionNumberPrefix returns the leading dotted section number of heading
// text (with its lowercase suffix, if any), "" when there is none. The
// number must end at a word boundary ("2.13b Retry" yes, "2nd" no).
func sectionNumberPrefix(text string) string {
	m := sectionNumberRe.FindString(text)
	if m == "" {
		return ""
	}

	if end := len(m); end < len(text) {
		switch text[end] {
		case '.', ':', ' ', '\t':
			return m
		}

		return ""
	}

	return m
}

// parseHeadings extracts ATX headings outside fenced code blocks, in order.
// Closing hash sequences ("## foo ##") are stripped like GitHub does.
func parseHeadings(md string) []heading {
	var (
		headings []heading
		inFence  bool
	)

	for i, line := range strings.Split(md, "\n") {
		if isFenceToggle(line) {
			inFence = !inFence

			continue
		}

		if inFence {
			continue
		}

		h, ok := parseHeadingLine(line, i+1)
		if ok {
			headings = append(headings, h)
		}
	}

	return headings
}

func parseHeadingLine(line string, lineNo int) (heading, bool) {
	depth := 0

	for depth < len(line) && line[depth] == '#' {
		depth++
	}

	if depth == 0 || depth > 6 || depth >= len(line) ||
		(line[depth] != ' ' && line[depth] != '\t') {
		return heading{}, false //nolint:exhaustruct_v5 // zero heading = not found
	}

	text := strings.TrimRight(strings.TrimLeft(line[depth:], " \t"), " \t")
	text = trimClosingHashes(text)

	return heading{
		text:   text,
		slug:   githubSlug(text),
		number: sectionNumberPrefix(text),
		line:   lineNo,
	}, true
}

func trimClosingHashes(text string) string {
	trimmed := strings.TrimRight(text, " \t")

	end := len(trimmed)

	for end > 0 && trimmed[end-1] == '#' {
		end--
	}

	if end == len(trimmed) || end == 0 {
		return text
	}

	// Only a trailing hash run separated by spaces is a closing sequence.
	if trimmed[end-1] == ' ' || trimmed[end-1] == '\t' {
		return strings.TrimRight(trimmed[:end], " \t")
	}

	return text
}

// slugCounts maps every heading slug in a document to its occurrence count
// (GitHub disambiguates repeats with -1/-2 suffixes that link targets use).
func slugCounts(headings []heading) map[string]int {
	counts := make(map[string]int, len(headings))

	for _, h := range headings {
		counts[h.slug]++
	}

	return counts
}

// numberSet returns the set of leading section numbers of a document.
func numberSet(headings []heading) map[string]bool {
	set := make(map[string]bool, len(headings))

	for _, h := range headings {
		if h.number != "" {
			set[h.number] = true
		}
	}

	return set
}
