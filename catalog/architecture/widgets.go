package architecture

import (
	"strconv"
	"strings"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4"
)

// resourceRef is a resolved in-document reference to another resource.
type resourceRef struct {
	anchor string
	name   string
	tag    string
	class  string
	note   string
}

// writeRef renders a resource reference as an in-document link.
func (r resourceRef) write(w *htmlWriter) {
	if r.anchor == "" {
		w.element("span", r.name, htmlClass("rl", "rl-plain"))

		return
	}

	w.open("a", htmlAttr("href", "#"+r.anchor), htmlClass("rl", r.class))
	w.element("span", r.tag, htmlClass("rl-tag"))
	w.element("span", r.name, htmlClass("rl-name"))

	if r.note != "" {
		w.element("span", r.note, htmlClass("rl-note"))
	}

	w.close("a")
}

// refBox renders a titled box of resource references, or nothing when empty.
func refBox(label string, refs []resourceRef) markup {
	if len(refs) == 0 {
		return ""
	}

	return renderInto(func(w *htmlWriter) {
		boxStart(w, label, len(refs))
		w.open("div", htmlClass("box-body"))

		for _, ref := range refs {
			ref.write(w)
		}

		w.close("div")
		w.close("div")
	})
}

// boxStart writes the chrome shared by every right-column box.
func boxStart(w *htmlWriter, label string, count int) {
	w.open("div", htmlClass("box"))
	w.open("div", htmlClass("box-head"))
	w.element("span", label)

	if count >= 0 {
		w.element("span", strconv.Itoa(count))
	}

	w.close("div")
}

// block writes a labelled left-column block.
func blockHTML(label string, body markup) block {
	return block{label: label, body: body}
}

// fieldList renders label/value pairs as a definition list.
func fieldList(pairs [][2]string) markup {
	return renderInto(func(w *htmlWriter) {
		w.open("dl", htmlClass("kv"))

		for _, pair := range pairs {
			if pair[1] == "" {
				continue
			}

			w.element("dt", pair[0])
			w.element("dd", pair[1])
		}

		w.close("dl")
	})
}

// pillRow renders the version pill and any badges under a section title.
func pillRow(version string, badges []catalog.Badge, deprecated string) markup {
	if version == "" && len(badges) == 0 && deprecated == "" {
		return ""
	}

	return renderInto(func(w *htmlWriter) {
		if version != "" {
			w.element("span", "v"+version, htmlClass("pill", "ver"))
		}

		if deprecated != "" {
			w.element("span", deprecated, htmlClass("pill", "warn"))
		}

		for _, badge := range badges {
			style := badgeStyle(badge)

			w.open("span", htmlClass("pill", "badge"), style)
			w.text(badge.Content)
			w.close("span")
		}
	})
}

// badgeStyle builds a safe inline style for a badge's colors.
func badgeStyle(badge catalog.Badge) string {
	fg := safeColor(string(badge.TextColor))
	bg := safeColor(string(badge.BackgroundColor))

	if fg == "" && bg == "" {
		return ""
	}

	rule := "--badge-fg:" + defaultIfEmpty(fg, "#ffffff") + ";--badge-bg:" + defaultIfEmpty(bg, "#1d4ed8")

	return htmlAttr("style", rule)
}

// safeColor strips anything that could escape a CSS declaration. Catalog
// colors are authored, not user input, but the reference is served to people
// outside the team, so the escape hatch stays closed.
func safeColor(value string) string {
	var b strings.Builder

	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '#', r == '(', r == ')', r == ',', r == '.', r == '%', r == ' ', r == '-':
			b.WriteRune(r)
		}
	}

	return strings.TrimSpace(b.String())
}

// summaryBlock renders a section's summary paragraph, when it has one.
func summaryBlock(summary string) markup {
	if summary == "" {
		return ""
	}

	return renderInto(func(w *htmlWriter) {
		w.element("p", summary, htmlClass("lead"))
	})
}

// defaultIfEmpty returns value when non-empty, else fallback.
func defaultIfEmpty(value, fallback string) string {
	if value == "" {
		return fallback
	}

	return value
}
