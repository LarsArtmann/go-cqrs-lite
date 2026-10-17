package architecture

// tableCell is one cell of a rendered table: escaped text plus an optional
// CSS class ("" renders a plain cell).
type tableCell struct {
	text  string
	class string
}

// plainCell builds a table cell without a class.
func plainCell(text string) tableCell {
	return tableCell{text: text}
}

// monoCell builds a table cell rendered in the monospace style.
func monoCell(text string) tableCell {
	return tableCell{text: text, class: "mono"}
}

// writeTable renders headers and rows as the standard .md-table markup. An
// empty row set renders nothing, matching the optional-block convention of
// the section builders.
func writeTable(headers []string, rows [][]tableCell) markup {
	if len(rows) == 0 {
		return ""
	}

	return renderInto(func(w *htmlWriter) {
		w.open("div", htmlClass("md-table"))
		w.open("table")
		w.open("thead")
		w.open("tr")

		for _, header := range headers {
			w.element("th", header)
		}

		w.close("tr")
		w.close("thead")
		w.open("tbody")

		for _, row := range rows {
			w.open("tr")

			for _, cell := range row {
				if cell.class != "" {
					w.element("td", cell.text, htmlClass(cell.class))
				} else {
					w.element("td", cell.text)
				}
			}

			w.close("tr")
		}

		w.close("tbody")
		w.close("table")
		w.close("div")
	})
}
