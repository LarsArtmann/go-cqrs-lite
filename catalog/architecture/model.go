package architecture

import (
	"time"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4"
)

// document is a fully collected architecture reference, ready to render.
type document struct {
	title       string
	description string
	generated   time.Time
	command     string
	counts      []countRow
	nav         []navGroup
	sections    []section
	search      []searchEntry
}

// countRow is one entry of the catalog summary ("Services 12").
type countRow struct {
	label string
	value int
}

// section is one resource's page: a heading, its two columns, and the
// sidebar/search metadata that points at it.
type section struct {
	kind       resourceKind
	anchor     string
	name       string
	version    string
	summary    string
	badges     []catalog.Badge
	owners     []string
	deprecated string // non-empty renders a deprecation notice
	meta       []markup
	left       []block
	right      []block
}

// noCount marks a block heading that shows no item count.
const noCount = -1

// block is a titled region inside a section column.
type block struct {
	label string
	count int
	body  markup
}

// navGroup is one sidebar group: a heading, a count and its entries.
type navGroup struct {
	kind    resourceKind
	title   string
	entries []navEntry
}

// navEntry is one sidebar link.
type navEntry struct {
	anchor string
	tag    string
	class  string
	name   string
	alt    string
}

// searchEntry is one record of the in-file search index. The JSON field names
// are short because the index ships inside every generated file.
type searchEntry struct {
	Group   string `json:"g"`
	Key     string `json:"k"`
	Class   string `json:"c"`
	Title   string `json:"t"`
	Alt     string `json:"a"`
	Summary string `json:"s"`
	Domain  string `json:"d"`
	Href    string `json:"h"`
}

// docBuilder accumulates the document while resources are collected. It owns
// anchor allocation, so every link and section agrees on one id per resource.
type docBuilder struct {
	title     string
	generated time.Time
	command   string
	counts    []countRow
	nav       []navGroup
	sections  []section
	search    []searchEntry
}

// builder creates an empty document builder.
func newDocBuilder(title, command string, generated time.Time) *docBuilder {
	return &docBuilder{title: title, command: command, generated: generated}
}

// addSection appends a section and, when it is not empty, its sidebar entry.
func (d *docBuilder) addSection(sec section, entry navEntry) {
	d.sections = append(d.sections, sec)

	if entry.anchor == "" {
		return
	}

	d.navAt(sec.kind, entry)
}

// navAt appends an entry to the sidebar group for a kind, creating it on first use.
func (d *docBuilder) navAt(kind resourceKind, entry navEntry) {
	for i := range d.nav {
		if d.nav[i].kind == kind {
			d.nav[i].entries = append(d.nav[i].entries, entry)

			return
		}
	}

	d.nav = append(d.nav, navGroup{
		kind:    kind,
		title:   kindInfos()[kind].group,
		entries: []navEntry{entry},
	})
}

// addCount records one summary row.
func (d *docBuilder) addCount(label string, value int) {
	d.counts = append(d.counts, countRow{label: label, value: value})
}
