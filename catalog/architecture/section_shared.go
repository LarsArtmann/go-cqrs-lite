package architecture

import "github.com/dustin/go-humanize/english"

// metaLine renders the small "Owned by … · 5 services" line under a title.
func metaLine(items []markup) markup {
	kept := make([]markup, 0, len(items))

	for _, item := range items {
		if item != "" {
			kept = append(kept, item)
		}
	}

	if len(kept) == 0 {
		return ""
	}

	return renderInto(func(w *htmlWriter) {
		w.open("div", htmlClass("meta-line"))

		for _, item := range kept {
			w.container("span", item)
		}

		w.close("div")
	})
}

// ownerMeta renders the "Owned by …" meta item with links to the owners.
func ownerMeta(idx *refIndex, owners []string) markup {
	if len(owners) == 0 || idx.hide.owners {
		return ""
	}

	refs := idx.owners(owners)

	return renderInto(func(w *htmlWriter) {
		w.text("Owned by ")

		for i, ref := range refs {
			if i > 0 {
				w.text(", ")
			}

			ref.writeRef(w)
		}
	})
}

// countMeta renders a counted meta item such as "5 services".
func countMeta(count int, singular, plural string) markup {
	if count == 0 {
		return ""
	}

	return renderInto(func(w *htmlWriter) { w.text(english.Plural(count, singular, plural)) })
}

// textMeta renders a plain meta item, or nothing for empty text.
func textMeta(text string) markup {
	if text == "" {
		return ""
	}

	return renderInto(func(w *htmlWriter) { w.text(text) })
}

// idLine renders the resource's identifier under its title.
func idLine(id string) markup {
	if id == "" {
		return ""
	}

	return renderInto(func(w *htmlWriter) { w.element("div", id, htmlClass("res-id")) })
}
