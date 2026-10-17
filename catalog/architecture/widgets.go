package architecture

import (
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

// Table column headers. The same labels head many tables, so they are named
// once here instead of being spelled out again in every renderer.
const (
	headerField       = "Field"
	headerProperty    = "Property"
	headerType        = "Type"
	headerRequired    = "Required"
	headerDescription = "Description"
	headerVersion     = "Version"
	headerDate        = "Date"
	headerSummary     = "Summary"
)

// plainRef builds a reference to something with no section in the document,
// which therefore renders as text rather than a link.
func plainRef(name string) resourceRef {
	return resourceRef{anchor: "", name: name, tag: "", class: "", note: ""}
}

// writeRef renders a resource reference as an in-document link, or as plain
// text when the target is not part of the document.
func (r resourceRef) writeRef(w *htmlWriter) {
	if r.anchor == "" {
		if r.name == "" {
			return
		}

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

// refList renders references as a vertical list.
func refList(refs []resourceRef) markup {
	if len(refs) == 0 {
		return ""
	}

	return renderInto(func(w *htmlWriter) {
		for _, ref := range refs {
			ref.writeRef(w)
		}
	})
}

// refBlock builds a block (left column) from a titled list of references and
// omits itself entirely when the list is empty.
func refBlock(label string, refs []resourceRef) (block, bool) {
	if len(refs) == 0 {
		return block{label: "", count: 0, body: ""}, false
	}

	return blockCounted(label, len(refs), refList(refs)), true
}

// appendRefs appends a reference block when it has content.
func appendRefs(blocks []block, label string, refs []resourceRef) []block {
	got, ok := refBlock(label, refs)
	if !ok {
		return blocks
	}

	return append(blocks, got)
}

// blockHTML builds an uncounted block.
func blockHTML(label string, body markup) block {
	return block{label: label, count: noCount, body: body}
}

// blockCounted builds a block whose heading shows how many items it lists.
func blockCounted(label string, count int, body markup) block {
	return block{label: label, count: count, body: body}
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
			w.open("span", htmlClass("pill", "badge"), badgeStyle(badge))
			w.text(badge.Content)
			w.close("span")
		}
	})
}

// badgeStyle builds a safe inline style for a badge's colors.
func badgeStyle(badge catalog.Badge) string {
	foreground := safeColor(string(badge.TextColor))
	background := safeColor(string(badge.BackgroundColor))

	if foreground == "" && background == "" {
		return ""
	}

	rule := "--badge-fg:" + defaultIfEmpty(
		foreground,
		"#ffffff",
	) + ";--badge-bg:" + defaultIfEmpty(
		background,
		"#1d4ed8",
	)

	return htmlAttr("style", rule)
}

// safeColor strips anything that could escape a CSS declaration. Catalog
// colors are authored, not user input, but the reference is read by people
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

	return renderInto(func(w *htmlWriter) { w.element("p", summary, htmlClass("lead")) })
}

// proseBlock renders markdown prose inside a documentation container.
func proseBlock(text string, link linkerFunc) markup {
	body := renderMarkdown(text, link)
	if body == "" {
		return ""
	}

	return renderInto(func(w *htmlWriter) {
		w.open("div", htmlClass("md"))
		w.raw(body)
		w.close("div")
	})
}

// appendProse appends a markdown block when the prose is non-empty.
func appendProse(blocks []block, label, text string, link linkerFunc) []block {
	body := proseBlock(text, link)
	if body == "" {
		return blocks
	}

	return append(blocks, blockHTML(label, body))
}

// codeBlock renders a titled code block with a copy button.
func codeBlock(title, code string) markup {
	return renderInto(func(w *htmlWriter) {
		w.open("div", htmlClass("code"))
		w.open("div", htmlClass("code-head"))

		if title != "" {
			w.element("span", title, htmlClass("code-title"))
		}

		w.open(
			"button",
			htmlClass("copy"),
			htmlAttr("type", "button"),
			htmlAttr("aria-label", "Copy code"),
		)
		w.element("span", "Copy")
		w.close("button")
		w.close("div")
		w.open("pre")
		w.element("code", code, htmlClass("code-body"))
		w.close("pre")
		w.close("div")
	})
}

// externalLink renders a link that leaves the document.
func externalLink(href, label string) markup {
	return renderInto(func(w *htmlWriter) {
		if href == "" {
			w.element("span", label)

			return
		}

		w.open(
			"a",
			htmlAttr("href", safeHref(href)),
			htmlAttr("rel", "noreferrer noopener"),
			htmlAttr("target", "_blank"),
		)
		w.text(label)
		w.close("a")
	})
}

// safeImageSrc allowlists image sources for the brand logo: only remote
// http(s) images and inline data images are rendered; anything else (notably
// javascript:) is dropped entirely rather than rewritten, so a hostile logo
// leaves no trace in the shared document.
func safeImageSrc(src string) string {
	lower := strings.ToLower(strings.TrimSpace(src))

	for _, prefix := range []string{"http://", "https://", "data:image/"} {
		if strings.HasPrefix(lower, prefix) {
			return src
		}
	}

	return ""
}

// safeHref allowlists URL schemes: external links in a shared document must
// not be able to run script when a reader clicks them.
func safeHref(href string) string {
	lower := strings.ToLower(strings.TrimSpace(href))

	for _, prefix := range []string{"http://", "https://", "mailto:", "data:image/", "#", "/"} {
		if strings.HasPrefix(lower, prefix) {
			return href
		}
	}

	return "#"
}

// defaultIfEmpty returns value when non-empty, else fallback.
func defaultIfEmpty(value, fallback string) string {
	if value == "" {
		return fallback
	}

	return value
}
