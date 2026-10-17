package architecture

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

// renderDocument renders the finished document as one self-contained HTML page.
func renderDocument(doc *document) (string, error) {
	index, err := json.Marshal(doc.search)
	if err != nil {
		return "", fmt.Errorf("marshal search index: %w", err)
	}

	var w htmlWriter

	writeHead(&w, doc)
	writeTopbar(&w, doc)
	writeShell(&w, doc)
	writeSearchDialog(&w)
	writeSearchIndex(&w, string(index))
	w.open("script")
	w.raw(asset("reference.js"))
	w.close("script")
	w.close("body")
	w.close("html")

	return "<!doctype html>\n" + string(w.document()) + "\n", nil
}

// writeHead writes the document head, including the theme bootstrap (which
// must run before the body paints to avoid a flash of the wrong theme).
func writeHead(w *htmlWriter, doc *document) {
	w.open("html", htmlAttr("lang", "en"))
	w.open("head")
	w.open("meta", htmlAttr("charset", "utf-8"))
	w.open("meta", htmlAttr("name", "viewport"), htmlAttr("content", "width=device-width, initial-scale=1"))
	w.element("title", doc.title+" · Architecture reference")

	if doc.description != "" {
		w.open("meta", htmlAttr("name", "description"), htmlAttr("content", doc.description))
	}

	w.open("meta", htmlAttr("name", "generator"), htmlAttr("content", "go-cqrs-lite catalog architecture"))
	w.open("script")
	w.raw(asset("theme.js"))
	w.close("script")
	w.open("style")
	w.raw(asset("reference.css"))
	w.close("style")
	w.close("head")
}

// writeTopbar writes the sticky bar with the brand, search and theme controls.
func writeTopbar(w *htmlWriter, doc *document) {
	w.open("body")
	w.open("header", htmlClass("topbar"))
	w.open("a", htmlClass("brand"), htmlAttr("href", "#overview"))
	w.element("span", doc.title, htmlClass("brand-name"))
	w.close("a")
	w.open("div", htmlClass("top-spacer"))
	w.open("button", htmlClass("search-btn"), htmlAttr("id", "search-btn"), htmlAttr("type", "button"))
	w.element("span", "Search the reference")
	w.element("kbd", "Ctrl-K", htmlAttr("id", "kbd-hint"))
	w.close("button")
	w.element("span", generatedStamp(doc.generated), htmlClass("date-chip"))
	w.open("button", htmlClass("icon-btn"), htmlAttr("id", "theme-btn"), htmlAttr("type", "button"),
		htmlAttr("aria-label", "Switch between light and dark"))
	w.element("span", "◐")
	w.close("button")
	w.close("header")
}

// writeShell writes the sidebar and the document body.
func writeShell(w *htmlWriter, doc *document) {
	w.open("div", htmlClass("shell"))
	w.open("nav", htmlClass("sidenav"), htmlAttr("id", "sidenav"), htmlAttr("aria-label", "Reference"))
	w.raw(navHTML(doc))
	w.close("nav")
	w.open("main", htmlAttr("id", "main"))
	writeSections(w, doc)
	w.close("main")
	w.close("div")
}

// writeSections writes every section, opening a group heading when the
// resource kind changes.
func writeSections(w *htmlWriter, doc *document) {
	groups := make(map[resourceKind]navGroup, len(doc.nav))

	for _, group := range doc.nav {
		groups[group.kind] = group
	}

	last := kindOverview

	for _, sec := range doc.sections {
		if sec.kind != last {
			if group, ok := groups[sec.kind]; ok {
				w.raw(groupHeaderHTML(group))
			}

			last = sec.kind
		}

		w.raw(sectionHTML(sec))
	}
}

// sectionHTML renders one resource section.
func sectionHTML(sec section) markup {
	return renderInto(func(w *htmlWriter) {
		w.open("section", htmlClass("section"), htmlAttr("id", sec.anchor()), htmlAttr("data-kind", string(sec.kind)))
		writeSectionLeft(w, sec)
		writeSectionRight(w, sec)
		w.close("section")
	})
}

// writeSectionLeft writes the main column of a section.
func writeSectionLeft(w *htmlWriter, sec section) {
	w.open("div", htmlClass("col-l"))
	w.open("div", htmlClass("eyebrow"))
	w.element("span", kindLabel(sec.kind))
	w.close("div")
	w.open("div", htmlClass("title-row"))
	w.element("h2", sec.name)
	w.raw(pillRow(sec.version, sec.badges, sec.deprecated))
	w.close("div")

	if sec.id != "" && sec.kind != kindDoc {
		w.raw(idLine(sec.id))
	}

	w.raw(summaryBlock(sec.summary))
	w.raw(sec.meta)

	for _, item := range sec.left {
		writeLeftBlock(w, item)
	}

	w.close("div")
}

// writeSectionRight writes the sticky relationship column of a section.
func writeSectionRight(w *htmlWriter, sec section) {
	if len(sec.right) == 0 {
		return
	}

	w.open("div", htmlClass("col-r"))
	w.open("div", htmlClass("sticky"))

	for _, item := range sec.right {
		writeRightBox(w, item)
	}

	w.close("div")
	w.close("div")
}

// writeLeftBlock writes one labelled block in the main column.
func writeLeftBlock(w *htmlWriter, item block) {
	w.open("div", htmlClass("block"))

	if item.label != "" {
		w.element("div", item.label, htmlClass("label"))
	}

	w.raw(item.body)
	w.close("div")
}

// writeRightBox writes one titled box in the relationship column.
func writeRightBox(w *htmlWriter, item block) {
	w.open("div", htmlClass("box"))
	w.open("div", htmlClass("box-head"))
	w.element("span", item.label)

	if item.count > 0 {
		w.element("span", strconv.Itoa(item.count))
	}

	w.close("div")
	w.open("div", htmlClass("box-body"))
	w.raw(item.body)
	w.close("div")
	w.close("div")
}

// writeSearchIndex writes the JSON payload the search dialog reads.
func writeSearchIndex(w *htmlWriter, index string) {
	w.open("script", htmlAttr("type", "application/json"), htmlAttr("id", "search-index"))
	w.raw(markup(index))
	w.close("script")
}

// writeSearchDialog writes the command palette markup.
func writeSearchDialog(w *htmlWriter) {
	w.open("div", htmlClass("cmdk-scrim"), htmlAttr("id", "cmdk-scrim"), htmlAttr("hidden", "hidden"))
	w.open("div", htmlClass("cmdk"), htmlAttr("id", "cmdk"), htmlAttr("role", "dialog"),
		htmlAttr("aria-modal", "true"), htmlAttr("aria-label", "Search the reference"), htmlAttr("hidden", "hidden"))
	w.open("input", htmlClass("cmdk-input"), htmlAttr("id", "cmdk-input"), htmlAttr("type", "text"),
		htmlAttr("placeholder", "Search resources by name, id or summary"), htmlAttr("autocomplete", "off"),
		htmlAttr("aria-label", "Search the reference"))
	w.open("div", htmlClass("cmdk-results"), htmlAttr("id", "cmdk-results"), htmlAttr("role", "listbox"))
	w.close("div")
	w.close("div")
	w.close("div")
}

// kindLabel returns the display name of a resource kind.
func kindLabel(kind resourceKind) string {
	switch kind {
	case kindOverview:
		return "Architecture reference"
	case kindDomain:
		return "Domain"
	case kindService:
		return "Service"
	case kindCommand:
		return "Command"
	case kindEvent:
		return "Event"
	case kindQuery:
		return "Query"
	case kindChannel:
		return "Channel"
	case kindDataStore:
		return "Data store"
	case kindFlow:
		return "Flow"
	case kindEntity:
		return "Entity"
	case kindDataProduct:
		return "Data product"
	case kindAgent:
		return "Agent"
	case kindADR:
		return "Decision record"
	case kindTeam:
		return "Team"
	case kindUser:
		return "Person"
	case kindDoc:
		return "Documentation"
	default:
		return string(kind)
	}
}

// generatedStamp formats the generation time shown in the top bar.
func generatedStamp(at time.Time) string {
	if at.IsZero() {
		return ""
	}

	return at.UTC().Format(time.DateOnly)
}
