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

// headingNumberRe picks the leading dotted section number of a heading text:
// "2.13b Retry with Backoff" -> "2.13b", "2. Composition Recipes" -> "2".
// A trailing "." or ":" separator is consumed; a following letter suffix
// (2.21b) is part of the number.
var sectionNumberPrefix = func(text string) string {
	i := 0

	for i < len(text) && text[i] >= '0' && text[i] <= '9' {
		for i < len(text) && text[i] >= '0' && text[i] <= '9' {
			i++
		}

		if i < len(text) && text[i] == '.' && i+1 < len(text) && text[i+1] >= '0' && text[i+1] <= '9' {
			i++ // consume the dot, keep scanning digits

			continue
		}

		break
	}

	if i == 0 {
		return ""
	}

	// Letter suffix (2.21b), then the number must end at a word boundary.
	end := i

	for end < len(text) && text[end] >= 'a' && text[end] <= 'z' {
		end++
	}

	if end < len(text) && text[end] != '.' && text[end] != ':' && text[end] != ' ' && text[end] != '\t' {
		return ""
	}

	return text[:end]
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
		return heading{}, false
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
