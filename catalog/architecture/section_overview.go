package architecture

import (
	"strconv"
	"strings"
)

// overviewSection builds the document's opening section: what this file is,
// how much it documents, and how to regenerate it.
func overviewSection(cfg config, counts []countRow, idx *refIndex) section {
	sec := newSection(kindOverview, "", cfg.title)
	sec.summary = cfg.description
	sec.left = []block{blockHTML("", overviewProse(cfg.title, counts))}

	if cfg.introduction != "" {
		sec.left = append(
			sec.left,
			blockHTML("", renderMarkdown(cfg.introduction, idx.link)),
		)
	}

	sec.right = []block{blockHTML("Catalog", countTable(counts))}

	if cfg.command != "" {
		sec.right = append(
			sec.right,
			blockHTML("", codeBlock("Regenerate this reference", cfg.command)),
		)
	}

	return sec
}

// overviewProse writes the two-paragraph introduction.
func overviewProse(title string, counts []countRow) markup {
	return renderInto(func(w *htmlWriter) {
		w.open("div", htmlClass("prose"))
		w.open("p")
		w.text("This reference is generated from the " + title + " catalog. It documents ")

		for i, row := range counts {
			if row.value == 0 {
				continue
			}

			if i > 0 {
				w.raw(", ")
			}

			w.element("strong", strconv.Itoa(row.value)+" "+strings.ToLower(row.word()))
		}

		w.raw(".")
		w.close("p")
		w.open("p")
		w.text("Every resource is linked to everything it relates to. Press ")
		w.element("kbd", "Ctrl-K")
		w.text(" to jump to any of them, or filter the sidebar on the left.")
		w.close("p")
		w.close("div")
	})
}

// countTable renders the summary numbers as a definition list.
func countTable(counts []countRow) markup {
	pairs := make([][2]string, 0, len(counts))

	for _, row := range counts {
		if row.value == 0 {
			continue
		}

		pairs = append(pairs, [2]string{row.label, strconv.Itoa(row.value)})
	}

	if len(pairs) == 0 {
		return ""
	}

	return fieldList(pairs)
}
