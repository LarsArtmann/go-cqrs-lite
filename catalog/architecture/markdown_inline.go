package architecture

import "strings"

// renderInline writes inline markdown. A nil linker disables nested
// [[kind|id]] resolution (emphasis and code spans do not nest links).
func renderInline(w *htmlWriter, text string, link linkerFunc) {
	for i := 0; i < len(text); {
		next := i + plainRun(text[i:])
		if next > i {
			w.text(text[i:next])

			i = next

			continue
		}

		i += renderSpan(w, text[i:], link)
	}
}

// plainRun returns how many bytes at the start of text are plain prose.
func plainRun(text string) int {
	for i := range len(text) {
		switch text[i] {
		case '`', '*', '[', '<':
			return i
		}
	}

	return len(text)
}

// renderSpan writes the inline span starting at text and returns its length.
func renderSpan(w *htmlWriter, text string, link linkerFunc) int {
	if text[0] == '<' {
		// Raw HTML is deliberately not supported: the file is shared with
		// people outside the team, so prose cannot inject markup.
		w.text("<")

		return 1
	}

	for _, span := range []struct {
		marker string
		tag    string
	}{
		{"**", "strong"},
		{"*", "em"},
	} {
		if consumed, ok := renderDelimited(w, text, span.marker, span.tag, link); ok {
			return consumed
		}
	}

	if consumed, ok := renderCodeSpan(w, text); ok {
		return consumed
	}

	if consumed, ok := renderLink(w, text); ok {
		return consumed
	}

	if consumed, ok := renderResourceLink(w, text, link); ok {
		return consumed
	}

	w.text(text[:1])

	return 1
}

// renderDelimited writes **bold** or *italic* and reports its byte length.
func renderDelimited(w *htmlWriter, text, marker, tag string, link linkerFunc) (int, bool) {
	if marker == "*" && strings.HasPrefix(text, "**") {
		return 0, false
	}

	if !strings.HasPrefix(text, marker) {
		return 0, false
	}

	rest := text[len(marker):]
	if end := strings.Index(rest, marker); end >= 0 {
		w.open(tag)
		renderInline(w, rest[:end], link)
		w.close(tag)

		return len(marker)*2 + end, true
	}

	return 0, false
}

// renderCodeSpan writes `code` and reports its byte length.
func renderCodeSpan(w *htmlWriter, text string) (int, bool) {
	if text[0] != '`' {
		return 0, false
	}

	if end := strings.IndexByte(text[1:], '`'); end >= 0 {
		w.element("code", text[1:1+end])

		return end + 2, true
	}

	return 0, false
}

// renderLink writes [label](url) and reports its byte length.
func renderLink(w *htmlWriter, text string) (int, bool) {
	if text[0] != '[' || strings.HasPrefix(text, "[[") {
		return 0, false
	}

	labelEnd := strings.Index(text, "](")
	if labelEnd < 0 {
		return 0, false
	}

	urlEnd := strings.IndexByte(text[labelEnd:], ')')
	if urlEnd < 0 {
		return 0, false
	}

	label := text[1:labelEnd]
	href := text[labelEnd+2 : labelEnd+urlEnd]

	href = safeHref(href)

	if strings.HasPrefix(href, "#") {
		w.open("a", htmlAttr("href", href))
	} else {
		w.open(
			"a",
			htmlAttr("href", href),
			htmlAttr("rel", "noreferrer noopener"),
			htmlAttr("target", "_blank"),
		)
	}

	w.text(label)
	w.close("a")

	return labelEnd + urlEnd + 1, true
}

// renderResourceLink writes an in-document [[kind|id]] reference.
func renderResourceLink(w *htmlWriter, text string, link linkerFunc) (int, bool) {
	if !strings.HasPrefix(text, "[[") {
		return 0, false
	}

	end := strings.Index(text, "]]")
	if end < 0 {
		return 0, false
	}

	body := text[2:end]

	kind, id, found := strings.Cut(body, "|")
	if !found || link == nil {
		w.element("code", body)

		return end + 2, true
	}

	link(strings.TrimSpace(kind), strings.TrimSpace(id)).writeRef(w)

	return end + 2, true
}
