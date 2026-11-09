package architecture

import (
	"html"
	"strings"
)

// markup is a fragment of HTML that has already been escaped by an
// [htmlWriter]. The only way to obtain one is to render through a writer, so
// callers never interpolate untrusted data into raw output.
type markup string

// htmlWriter accumulates an HTML document. Every character of catalog data
// passes through [htmlWriter.text] or [htmlAttr]; nothing else writes strings.
type htmlWriter struct {
	b strings.Builder
}

// raw appends an already-escaped fragment.
func (w *htmlWriter) raw(fragment markup) {
	w.b.WriteString(string(fragment))
}

// text appends text with HTML metacharacters escaped.
func (w *htmlWriter) text(value string) {
	w.b.WriteString(html.EscapeString(value))
}

// open writes an opening tag with the given pre-built attributes.
func (w *htmlWriter) open(name string, attrs ...string) {
	w.b.WriteByte('<')
	w.b.WriteString(name)

	for _, a := range attrs {
		w.b.WriteString(a)
	}

	w.b.WriteByte('>')
}

// close writes a closing tag.
func (w *htmlWriter) close(name string) {
	w.b.WriteString("</")
	w.b.WriteString(name)
	w.b.WriteByte('>')
}

// element writes an open tag, escaped content, and a close tag.
func (w *htmlWriter) element(name, content string, attrs ...string) {
	w.open(name, attrs...)
	w.text(content)
	w.close(name)
}

// container writes an open tag, an already-escaped fragment, and a close tag.
func (w *htmlWriter) container(name string, content markup, attrs ...string) {
	w.open(name, attrs...)
	w.raw(content)
	w.close(name)
}

// document returns everything written so far.
func (w *htmlWriter) document() markup {
	return markup(w.b.String())
}

// htmlAttr builds an escaped HTML attribute (leading space included so it can
// be appended to any tag helper).
func htmlAttr(name, value string) string {
	return " " + name + `="` + html.EscapeString(value) + `"`
}

// htmlClass builds a class attribute from non-empty class names.
func htmlClass(names ...string) string {
	kept := make([]string, 0, len(names))

	for _, name := range names {
		if name != "" {
			kept = append(kept, name)
		}
	}

	if len(kept) == 0 {
		return ""
	}

	return htmlAttr("class", strings.Join(kept, " "))
}

// renderInto runs fn against a fresh writer and returns the escaped fragment.
// It lets small renderers compose without threading a writer through every
// helper's signature.
func renderInto(fn func(*htmlWriter)) markup {
	var w htmlWriter

	fn(&w)

	return w.document()
}
