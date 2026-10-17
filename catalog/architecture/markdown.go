package architecture

import "strings"

// linkerFunc resolves a "[[kind|id]]" reference found in prose into an
// in-document link, or a plain reference when the target is not in the file.
type linkerFunc func(kind, id string) resourceRef

// renderMarkdown renders the small markdown subset the catalog carries:
// headings, paragraphs, lists, block quotes, fenced code, inline code, bold,
// italic, links and [[kind|id]] resource references.
//
// The subset is deliberate. Resource bodies are authored by developers and are
// usually short; a full markdown engine would be a new dependency in a module
// whose dependency budget is full, and the escape hatches (raw HTML, tables)
// are the ones that make a shared document unsafe to hand to a third party.
func renderMarkdown(text string, link linkerFunc) markup {
	if strings.TrimSpace(text) == "" {
		return ""
	}

	var w htmlWriter

	renderBlocks(&w, strings.Split(normalizeNewlines(text), "\n"), link)

	return w.document()
}

// normalizeNewlines makes the line scanner indifferent to the author's editor.
func normalizeNewlines(text string) string {
	return strings.ReplaceAll(text, "\r\n", "\n")
}

// renderBlocks writes every markdown block in the given lines.
func renderBlocks(w *htmlWriter, lines []string, link linkerFunc) {
	for i := 0; i < len(lines); {
		line := lines[i]

		switch {
		case strings.TrimSpace(line) == "":
			i++
		case strings.HasPrefix(strings.TrimSpace(line), "```"):
			i = renderFence(w, lines, i)
		case headingLevel(line) > 0:
			i = renderHeading(w, lines, i)
		case isListItem(line):
			i = renderList(w, lines, i, link)
		case strings.HasPrefix(strings.TrimSpace(line), ">"):
			i = renderQuote(w, lines, i, link)
		case isRule(line):
			w.raw("<hr>")

			i++
		default:
			i = renderParagraph(w, lines, i, link)
		}
	}
}

// headingLevel returns the heading level of a line, or 0 when it is not one.
func headingLevel(line string) int {
	trimmed := strings.TrimSpace(line)

	level := 0

	for level < len(trimmed) && trimmed[level] == '#' {
		level++
	}

	if level == 0 || level > 6 || level == len(trimmed) {
		return 0
	}

	if trimmed[level] != ' ' {
		return 0
	}

	return level
}

// renderHeading writes one heading and returns the next line index.
func renderHeading(w *htmlWriter, lines []string, i int) int {
	level := headingLevel(lines[i])
	tag := "h" + string(rune('0'+level))

	w.open(tag)
	renderInline(w, strings.TrimSpace(strings.TrimSpace(lines[i])[level:]), nil)
	w.close(tag)

	return i + 1
}

// isListItem reports whether a line opens a bullet or ordered list item.
func isListItem(line string) bool {
	trimmed := strings.TrimSpace(line)

	if strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ") {
		return true
	}

	digits := 0

	for digits < len(trimmed) && trimmed[digits] >= '0' && trimmed[digits] <= '9' {
		digits++
	}

	return digits > 0 && strings.HasPrefix(trimmed[digits:], ". ")
}

// isRule reports whether a line is a horizontal rule.
func isRule(line string) bool {
	trimmed := strings.TrimSpace(line)

	return trimmed == "---" || trimmed == "***" || trimmed == "___"
}

// renderList writes a bullet or ordered list and returns the next index.
func renderList(w *htmlWriter, lines []string, i int, link linkerFunc) int {
	tag := "ul"
	if !strings.HasPrefix(strings.TrimSpace(lines[i]), "- ") &&
		!strings.HasPrefix(strings.TrimSpace(lines[i]), "* ") {
		tag = "ol"
	}

	w.open(tag)

	for i < len(lines) && isListItem(lines[i]) {
		w.open("li")
		renderInline(w, listItemText(lines[i]), link)
		w.close("li")

		i++
	}

	w.close(tag)

	return i
}

// listItemText strips a list item's marker.
func listItemText(line string) string {
	trimmed := strings.TrimSpace(line)

	if strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ") {
		return trimmed[2:]
	}

	if dot := strings.Index(trimmed, ". "); dot > 0 {
		return trimmed[dot+2:]
	}

	return trimmed
}

// renderQuote writes a run of block-quoted lines.
func renderQuote(w *htmlWriter, lines []string, i int, link linkerFunc) int {
	w.open("blockquote")

	for i < len(lines) && strings.HasPrefix(strings.TrimSpace(lines[i]), ">") {
		renderInline(
			w,
			strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(lines[i]), ">")),
			link,
		)
		w.raw(" ")

		i++
	}

	w.close("blockquote")

	return i
}

// renderFence writes a fenced code block and returns the next index.
func renderFence(w *htmlWriter, lines []string, i int) int {
	language := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(lines[i]), "```"))
	i++

	body := make([]string, 0, len(lines)-i)

	for i < len(lines) && !strings.HasPrefix(strings.TrimSpace(lines[i]), "```") {
		body = append(body, lines[i])

		i++
	}

	if i < len(lines) {
		i++
	}

	w.open("pre")
	w.element("code", strings.Join(body, "\n"), htmlClass("lang-"+language))
	w.close("pre")

	return i
}

// renderParagraph writes a run of prose lines as one paragraph.
func renderParagraph(w *htmlWriter, lines []string, i int, link linkerFunc) int {
	parts := make([]string, 0)
	parts = append(parts, strings.TrimSpace(lines[i]))

	i++

	for i < len(lines) && isParagraphLine(lines[i]) {
		parts = append(parts, strings.TrimSpace(lines[i]))

		i++
	}

	w.open("p")
	renderInline(w, strings.Join(parts, " "), link)
	w.close("p")

	return i
}

// isParagraphLine reports whether a line continues the current paragraph.
func isParagraphLine(line string) bool {
	trimmed := strings.TrimSpace(line)

	if trimmed == "" || isRule(line) || headingLevel(line) > 0 || isListItem(line) {
		return false
	}

	return !strings.HasPrefix(trimmed, "```") && !strings.HasPrefix(trimmed, ">") &&
		!strings.HasPrefix(trimmed, "<")
}
