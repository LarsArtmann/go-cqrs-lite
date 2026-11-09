package architecture

import "strconv"

// navHTML renders the sidebar: a filter box and one group per resource kind.
func navHTML(doc *document) markup {
	return renderInto(func(w *htmlWriter) {
		w.open("input",
			htmlClass("nav-filter"),
			htmlAttr("id", "nav-filter"),
			htmlAttr("type", "search"),
			htmlAttr("placeholder", "Filter resources"),
			htmlAttr("autocomplete", "off"),
			htmlAttr("aria-label", "Filter the navigation"),
		)

		for _, group := range doc.nav {
			writeNavGroup(w, group)
		}

		w.element("p", "Nothing matches that filter.",
			htmlClass("nav-empty"), htmlAttr("id", "nav-empty"), htmlAttr("hidden", "hidden"))
	})
}

// writeNavGroup renders one sidebar group and its entries.
func writeNavGroup(w *htmlWriter, group navGroup) {
	if len(group.entries) == 0 {
		return
	}

	w.open("div", htmlClass("nav-group"))
	w.open("div", htmlClass("nav-label"))
	w.element("a", group.title, htmlAttr("href", groupAnchor(group)))
	w.element("small", strconv.Itoa(len(group.entries)))
	w.close("div")

	for _, entry := range group.entries {
		writeNavEntry(w, entry)
	}

	w.close("div")
}

// groupAnchor points a sidebar group heading at its body heading, or at the
// overview section when the group has no heading of its own.
func groupAnchor(group navGroup) string {
	if group.kind == kindOverview {
		return "#overview"
	}

	return "#group-" + slug(string(group.kind))
}

// writeNavEntry renders one sidebar link with its kind tag.
func writeNavEntry(w *htmlWriter, entry navEntry) {
	w.open("a",
		htmlClass("item"),
		htmlAttr("href", "#"+entry.anchor),
		htmlAttr("data-alt", entry.alt),
	)
	w.element("span", entry.tag, htmlClass("tag", entry.class))
	w.element("span", entry.name, htmlClass("t"))
	w.close("a")
}

// groupHeaderHTML renders the heading that opens a resource group in the body.
func groupHeaderHTML(group navGroup) markup {
	info := kindInfos()[group.kind]

	return renderInto(func(w *htmlWriter) {
		w.open("header", htmlClass("group-head"), htmlAttr("id", "group-"+slug(string(group.kind))))
		w.element("h2", group.title)

		if info.blurb != "" {
			w.element("p", info.blurb)
		}

		w.close("header")
	})
}
